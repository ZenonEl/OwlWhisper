package app

import (
	"fmt"
	"log"
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
	log.Println("🚀 ChatService запущен")

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

	log.Println("🛑 ChatService остановлен")
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
	log.Printf("📥 Получено сообщение от %s", rawMsg.SenderID.ShortString())

	// Декодируем Protobuf сообщение
	envelope := &protocol.Envelope{}
	if err := proto.Unmarshal(rawMsg.Data, envelope); err != nil {
		log.Printf("❌ Ошибка декодирования сообщения: %v", err)
		return
	}

	// Проверяем тип контента
	if envelope.GetContent() == nil {
		log.Printf("⚠️ Получено сообщение без контента от %s", rawMsg.SenderID.ShortString())
		return
	}

	content := envelope.GetContent()
	if content.GetText() == nil {
		log.Printf("⚠️ Получено сообщение без текста от %s", rawMsg.SenderID.ShortString())
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
	if err := cs.saveMessageToStorage(envelope, rawMsg.SenderID); err != nil {
		log.Printf("❌ Ошибка сохранения сообщения в БД: %v", err)
	}

	// Отправляем в канал для UI
	select {
	case cs.messagesChan <- chatMsg:
		log.Printf("✅ Сообщение отправлено в UI: %s", chatMsg.Text[:min(len(chatMsg.Text), 50)])
	default:
		log.Printf("⚠️ Канал UI переполнен, сообщение потеряно")
	}
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
		log.Printf("✅ Сообщение отправлено: %s", text[:min(len(text), 50)])
	default:
		log.Printf("⚠️ Канал UI переполнен, исходящее сообщение потеряно")
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

	for {
		select {
		case <-ticker.C:
			peers := cs.coreController.GetPeers()
			select {
			case cs.peersChan <- peers:
				// Пиры отправлены в канал
			default:
				// Канал переполнен, пропускаем
			}

		case <-cs.stopChan:
			return
		}
	}
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
