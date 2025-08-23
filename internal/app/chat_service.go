package app

import (
	"fmt"
	"sync"
	"time"

	"OwlWhisper/internal/core"
	"OwlWhisper/internal/protocol"
	"OwlWhisper/internal/storage"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/protobuf/proto"
)

// ChatService управляет логикой чата, связывая сетевой слой с хранилищем
type ChatService struct {
	coreController    core.ICoreController
	messageRepository storage.IMessageRepository
	contactRepository storage.IContactRepository

	// Каналы для UI клиентов
	messagesChan chan ChatMessage
	peersChan    chan []peer.ID

	// Внутренние каналы
	stopChan chan struct{}

	// Состояние
	isRunning bool
	mutex     sync.RWMutex
}

// ChatMessage представляет собой сообщение для UI клиентов
type ChatMessage struct {
	ID          string    `json:"id"`
	Text        string    `json:"text"`
	SenderID    string    `json:"sender_id"`
	SenderName  string    `json:"sender_name"`
	Timestamp   time.Time `json:"timestamp"`
	ChatType    string    `json:"chat_type"`
	RecipientID string    `json:"recipient_id"`
	IsOutgoing  bool      `json:"is_outgoing"`
}

// NewChatService создает новый сервис чата
func NewChatService(
	coreController core.ICoreController,
	messageRepository storage.IMessageRepository,
	contactRepository storage.IContactRepository,
) *ChatService {
	return &ChatService{
		coreController:    coreController,
		messageRepository: messageRepository,
		contactRepository: contactRepository,
		messagesChan:      make(chan ChatMessage, 100),
		peersChan:         make(chan []peer.ID, 10),
		stopChan:          make(chan struct{}),
	}
}

// Start запускает сервис чата
func (cs *ChatService) Start() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if cs.isRunning {
		return fmt.Errorf("сервис чата уже запущен")
	}

	cs.isRunning = true
	core.Info("🚀 ChatService запущен")

	// Запускаем горутину для обработки входящих сообщений
	go cs.handleIncomingMessages()

	// Запускаем горутину для мониторинга пиров
	go cs.monitorPeers()

	return nil
}

// Stop останавливает сервис чата
func (cs *ChatService) Stop() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if !cs.isRunning {
		return nil
	}

	cs.isRunning = false
	close(cs.stopChan)

	// Закрываем каналы
	close(cs.messagesChan)
	close(cs.peersChan)

	core.Info("🛑 ChatService остановлен")
	return nil
}

// handleIncomingMessages обрабатывает входящие сообщения из сети
func (cs *ChatService) handleIncomingMessages() {
	for {
		select {
		case rawMsg, ok := <-cs.coreController.Messages():
			if !ok {
				return // Канал закрыт
			}
			cs.processIncomingMessage(rawMsg)

		case <-cs.stopChan:
			return
		}
	}
}

// processIncomingMessage обрабатывает одно входящее сообщение
func (cs *ChatService) processIncomingMessage(rawMsg core.RawMessage) {
	core.Info("📥 Получено сообщение от %s", rawMsg.SenderID.ShortString())

	// Декодируем Protobuf сообщение
	envelope := &protocol.Envelope{}
	if err := proto.Unmarshal(rawMsg.Data, envelope); err != nil {
		core.Error("❌ Ошибка декодирования сообщения: %v", err)
		return
	}

	// Проверяем тип сообщения
	switch payload := envelope.Payload.(type) {
	case *protocol.Envelope_ProfileInfo:
		// Обрабатываем информацию о профиле
		cs.handleProfileInfo(rawMsg.SenderID, payload.ProfileInfo)
		return
	case *protocol.Envelope_Content:
		// Обрабатываем обычное сообщение
		cs.handleContentMessage(envelope, payload.Content, rawMsg.SenderID)
	default:
		core.Warn("⚠️ Неизвестный тип сообщения от %s", rawMsg.SenderID.ShortString())
		return
	}
}

// handleProfileInfo обрабатывает информацию о профиле
func (cs *ChatService) handleProfileInfo(senderID peer.ID, profileInfo *protocol.ProfileInfo) {
	core.Info("👤 Получен профиль от %s: %s%s", senderID.ShortString(), profileInfo.Nickname, profileInfo.Discriminator)

	// TODO: Сохранить профиль в базу данных
	// TODO: Обновить UI с новой информацией о пире

	// Автоматически отправляем наш профиль в ответ
	go cs.sendMyProfileToPeer(senderID)
}

// handleContentMessage обрабатывает сообщение с контентом
func (cs *ChatService) handleContentMessage(envelope *protocol.Envelope, content *protocol.Content, senderID peer.ID) {
	// Проверяем тип контента
	if content.GetText() == nil {
		core.Warn("⚠️ Получено сообщение без текста от %s", senderID.ShortString())
		return
	}

	textMsg := content.GetText()

	// Создаем ChatMessage для UI
	chatMsg := ChatMessage{
		ID:          envelope.MessageId,
		Text:        textMsg.Body,
		SenderID:    envelope.SenderId,
		SenderName:  cs.getPeerName(envelope.SenderId),
		Timestamp:   time.Unix(envelope.TimestampUnix, 0),
		ChatType:    envelope.ChatType.String(),
		RecipientID: envelope.ChatId,
		IsOutgoing:  false,
	}

	// Сохраняем в базу данных
	if err := cs.saveMessageToStorage(envelope, senderID); err != nil {
		core.Error("❌ Ошибка сохранения сообщения в БД: %v", err)
	}

	// Отправляем в канал для UI
	select {
	case cs.messagesChan <- chatMsg:
		core.Info("✅ Сообщение отправлено в UI: %s", chatMsg.Text[:min(len(chatMsg.Text), 50)])
	default:
		core.Warn("⚠️ Канал UI переполнен, сообщение потеряно")
	}
}

// generateMessageID создает уникальный ID для сообщения
func generateMessageID() string {
	return uuid.New().String()
}

// sendMyProfileToPeer отправляет наш профиль указанному пиру
func (cs *ChatService) sendMyProfileToPeer(peerID peer.ID) {
	// Получаем наш профиль
	myProfile := cs.coreController.GetMyProfile()

	// Создаем Protobuf сообщение с профилем
	profileInfo := &protocol.ProfileInfo{
		Nickname:      myProfile.Nickname,
		Discriminator: myProfile.Discriminator,
		DisplayName:   myProfile.DisplayName,
		LastSeen:      time.Now().Unix(),
		IsOnline:      true,
	}

	envelope := &protocol.Envelope{
		MessageId:     generateMessageID(),
		SenderId:      cs.coreController.GetMyID(),
		TimestampUnix: time.Now().Unix(),
		ChatType:      protocol.Envelope_PRIVATE,
		ChatId:        peerID.String(),
		Payload:       &protocol.Envelope_ProfileInfo{ProfileInfo: profileInfo},
	}

	// Сериализуем в Protobuf
	data, err := proto.Marshal(envelope)
	if err != nil {
		core.Error("❌ Ошибка сериализации профиля: %v", err)
		return
	}

	// Отправляем через core controller
	if err := cs.coreController.Send(peerID, data); err != nil {
		core.Error("❌ Ошибка отправки профиля к %s: %v", peerID.ShortString(), err)
		return
	}

	core.Info("📤 Профиль отправлен к %s", peerID.ShortString())
}

// saveMessageToStorage сохраняет сообщение в базу данных
func (cs *ChatService) saveMessageToStorage(envelope *protocol.Envelope, senderID peer.ID) error {
	// Создаем StoredMessage для сохранения
	chatMsg := &storage.StoredMessage{
		ID:          envelope.MessageId,
		Text:        envelope.GetContent().GetText().Body,
		Timestamp:   time.Unix(envelope.TimestampUnix, 0),
		SenderID:    envelope.SenderId,
		ChatType:    envelope.ChatType.String(),
		RecipientID: envelope.ChatId,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Сохраняем в базу данных
	return cs.messageRepository.Save(chatMsg)
}

// Send отправляет текстовое сообщение
func (cs *ChatService) Send(text string, chatType string, recipientID string) error {
	cs.mutex.RLock()
	if !cs.isRunning {
		cs.mutex.RUnlock()
		return fmt.Errorf("сервис чата не запущен")
	}
	cs.mutex.RUnlock()

	// Создаем уникальный ID сообщения
	messageID := uuid.New().String()

	// Определяем тип чата
	var chatTypeEnum protocol.Envelope_ChatType
	switch chatType {
	case "private":
		chatTypeEnum = protocol.Envelope_PRIVATE
	case "group":
		chatTypeEnum = protocol.Envelope_GROUP
	default:
		chatTypeEnum = protocol.Envelope_PRIVATE
	}

	// Создаем Envelope сообщение
	envelope := &protocol.Envelope{
		MessageId:     messageID,
		SenderId:      cs.coreController.GetMyID(),
		TimestampUnix: time.Now().Unix(),
		ChatType:      chatTypeEnum,
		ChatId:        recipientID,
		Payload: &protocol.Envelope_Content{
			Content: &protocol.Content{
				Type: &protocol.Content_Text{
					Text: &protocol.TextMessage{
						Body: text,
					},
				},
			},
		},
	}

	// Кодируем в Protobuf
	data, err := proto.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("ошибка кодирования сообщения: %w", err)
	}

	// Отправляем в сеть
	if err := cs.coreController.Broadcast(data); err != nil {
		return fmt.Errorf("ошибка отправки в сеть: %w", err)
	}

	// Создаем ChatMessage для UI
	chatMsg := ChatMessage{
		ID:          messageID,
		Text:        text,
		SenderID:    cs.coreController.GetMyID(),
		SenderName:  "Вы",
		Timestamp:   time.Now(),
		ChatType:    chatType,
		RecipientID: recipientID,
		IsOutgoing:  true,
	}

	// Отправляем в канал для UI
	select {
	case cs.messagesChan <- chatMsg:
		core.Info("✅ Сообщение отправлено: %s", text[:min(len(text), 50)])
	default:
		core.Warn("⚠️ Канал UI переполнен, исходящее сообщение потеряно")
	}

	return nil
}

// GetMessages возвращает канал для получения сообщений
func (cs *ChatService) GetMessages() <-chan ChatMessage {
	return cs.messagesChan
}

// GetPeers возвращает канал для получения списка пиров
func (cs *ChatService) GetPeers() <-chan []peer.ID {
	return cs.peersChan
}

// monitorPeers мониторит изменения в списке пиров
func (cs *ChatService) monitorPeers() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastPeers []peer.ID

	for {
		select {
		case <-ticker.C:
			currentPeers := cs.coreController.GetConnectedPeers()

			// Проверяем новых пиров
			for _, peerID := range currentPeers {
				if !containsPeer(lastPeers, peerID) {
					core.Info("🆕 Обнаружен новый пир: %s", peerID.ShortString())

					// Автоматически отправляем наш профиль новому пиру
					go cs.sendMyProfileToPeer(peerID)
				}
			}

			// Обновляем список пиров
			lastPeers = make([]peer.ID, len(currentPeers))
			copy(lastPeers, currentPeers)

			// Отправляем обновленный список в UI
			select {
			case cs.peersChan <- currentPeers:
				// Список пиров отправлен в UI
			default:
				// Канал переполнен, пропускаем
			}

		case <-cs.stopChan:
			return
		}
	}
}

// containsPeer проверяет, содержится ли пир в списке
func containsPeer(peers []peer.ID, target peer.ID) bool {
	for _, p := range peers {
		if p == target {
			return true
		}
	}
	return false
}

// getPeerName возвращает имя пира (никнейм или PeerID)
func (cs *ChatService) getPeerName(peerID string) string {
	// TODO: Реализовать получение никнейма из ContactRepository
	// Пока возвращаем короткий PeerID
	if len(peerID) > 8 {
		return peerID[:8] + "..."
	}
	return peerID
}

// GetHistory возвращает историю сообщений
func (cs *ChatService) GetHistory(limit int) ([]ChatMessage, error) {
	storedMessages, err := cs.messageRepository.GetHistory(limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории: %w", err)
	}

	// Конвертируем StoredMessage в ChatMessage
	var chatMessages []ChatMessage
	for _, stored := range storedMessages {
		chatMsg := ChatMessage{
			ID:          stored.ID,
			Text:        stored.Text,
			SenderID:    stored.SenderID,
			SenderName:  cs.getPeerName(stored.SenderID),
			Timestamp:   stored.Timestamp,
			ChatType:    stored.ChatType,
			RecipientID: stored.RecipientID,
			IsOutgoing:  stored.SenderID == cs.coreController.GetMyID(),
		}
		chatMessages = append(chatMessages, chatMsg)
	}

	return chatMessages, nil
}

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
