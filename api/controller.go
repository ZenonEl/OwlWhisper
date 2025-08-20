package api

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"OwlWhisper/internal/app"
	"OwlWhisper/internal/core"
	"OwlWhisper/internal/storage/sqlite"
)

// OwlWhisperAPI определяет главный интерфейс платформы
type OwlWhisperAPI interface {
	// Start запускает платформу
	Start() error

	// Stop останавливает платформу
	Stop() error

	// SendMessage отправляет сообщение
	SendMessage(request SendMessageRequest) error

	// GetPeers возвращает список подключенных пиров
	GetPeers() []Peer

	// GetConnectionStatus возвращает статус подключения
	GetConnectionStatus() ConnectionStatus

	// GetHistory возвращает историю сообщений
	GetHistory(limit int) (ChatHistory, error)

	// MessageChannel возвращает канал для получения сообщений
	MessageChannel() <-chan Message

	// PeerChannel возвращает канал для получения обновлений пиров
	PeerChannel() <-chan []Peer

	// GetMyPeerID возвращает ID нашего пира
	GetMyPeerID() string
}

// owlWhisperAPI реализует OwlWhisperAPI
type owlWhisperAPI struct {
	// Внутренние сервисы
	coreController *core.CoreController
	chatService    *app.ChatService
	messageRepo    *sqlite.MessageRepository

	// Конфигурация
	config *APIConfig

	// Каналы для клиентов
	messagesChan chan Message
	peersChan    chan []Peer

	// Контекст и состояние
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	mutex     sync.RWMutex
}

// NewOwlWhisperAPI создает новый экземпляр API
func NewOwlWhisperAPI(config *APIConfig) (OwlWhisperAPI, error) {
	if config == nil {
		config = DefaultAPIConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Создаем Core Controller
	coreController, err := core.NewCoreController(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать Core Controller: %w", err)
	}

	// Создаем репозиторий сообщений
	messageRepo, err := sqlite.NewMessageRepository(config.DatabasePath)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать репозиторий сообщений: %w", err)
	}

	// Создаем Chat Service
	chatService := app.NewChatService(coreController, messageRepo, nil)

	api := &owlWhisperAPI{
		coreController: coreController,
		chatService:    chatService,
		messageRepo:    messageRepo,
		config:         config,
		messagesChan:   make(chan Message, 100),
		peersChan:      make(chan []Peer, 10),
		ctx:            ctx,
		cancel:         cancel,
	}

	return api, nil
}

// Start запускает платформу
func (api *owlWhisperAPI) Start() error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if api.isRunning {
		return fmt.Errorf("API уже запущен")
	}

	log.Println("🚀 Запуск OwlWhisper API...")

	// Запускаем Core Controller
	if err := api.coreController.Start(); err != nil {
		return fmt.Errorf("не удалось запустить Core Controller: %w", err)
	}

	// Запускаем Chat Service
	if err := api.chatService.Start(); err != nil {
		api.coreController.Stop()
		return fmt.Errorf("не удалось запустить Chat Service: %w", err)
	}

	api.isRunning = true

	// Запускаем горутины для обработки событий
	go api.handleMessages()
	go api.handlePeers()

	log.Printf("✅ OwlWhisper API запущен. PeerID: %s", api.coreController.GetMyID())
	return nil
}

// Stop останавливает платформу
func (api *owlWhisperAPI) Stop() error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if !api.isRunning {
		return nil
	}

	log.Println("🛑 Остановка OwlWhisper API...")

	// Останавливаем сервисы
	if err := api.chatService.Stop(); err != nil {
		log.Printf("⚠️ Ошибка остановки Chat Service: %v", err)
	}

	if err := api.coreController.Stop(); err != nil {
		log.Printf("⚠️ Ошибка остановки Core Controller: %v", err)
	}

	// Закрываем репозиторий
	if err := api.messageRepo.Close(); err != nil {
		log.Printf("⚠️ Ошибка закрытия репозитория: %v", err)
	}

	// Отменяем контекст
	api.cancel()

	// Закрываем каналы
	close(api.messagesChan)
	close(api.peersChan)

	api.isRunning = false
	log.Println("✅ OwlWhisper API остановлен")
	return nil
}

// SendMessage отправляет сообщение
func (api *owlWhisperAPI) SendMessage(request SendMessageRequest) error {
	api.mutex.RLock()
	if !api.isRunning {
		api.mutex.RUnlock()
		return fmt.Errorf("API не запущен")
	}
	api.mutex.RUnlock()

	// Проверяем размер сообщения
	if len(request.Text) > api.config.MaxMessageSize {
		return fmt.Errorf("сообщение слишком длинное: %d > %d", len(request.Text), api.config.MaxMessageSize)
	}

	// Отправляем через Chat Service
	return api.chatService.Send(request.Text, request.ChatType, request.RecipientID)
}

// GetPeers возвращает список подключенных пиров
func (api *owlWhisperAPI) GetPeers() []Peer {
	api.mutex.RLock()
	if !api.isRunning {
		api.mutex.RUnlock()
		return []Peer{}
	}
	api.mutex.RUnlock()

	peerIDs := api.coreController.GetPeers()
	var peers []Peer

	for _, peerID := range peerIDs {
		peer := Peer{
			ID:       peerID.String(),
			Nickname: shortenPeerID(peerID.String()),
			Status:   "online",
			LastSeen: time.Now(),
		}
		peers = append(peers, peer)
	}

	return peers
}

// GetConnectionStatus возвращает статус подключения
func (api *owlWhisperAPI) GetConnectionStatus() ConnectionStatus {
	api.mutex.RLock()
	if !api.isRunning {
		api.mutex.RUnlock()
		return ConnectionStatus{
			IsConnected: false,
			PeerCount:   0,
			MyPeerID:    "",
			LastUpdate:  time.Now(),
			NetworkType: "offline",
		}
	}
	api.mutex.RUnlock()

	peers := api.coreController.GetPeers()
	return ConnectionStatus{
		IsConnected: len(peers) > 0,
		PeerCount:   len(peers),
		MyPeerID:    api.coreController.GetMyID(),
		LastUpdate:  time.Now(),
		NetworkType: "p2p",
	}
}

// GetHistory возвращает историю сообщений
func (api *owlWhisperAPI) GetHistory(limit int) (ChatHistory, error) {
	api.mutex.RLock()
	if !api.isRunning {
		api.mutex.RUnlock()
		return ChatHistory{}, fmt.Errorf("API не запущен")
	}
	api.mutex.RUnlock()

	if limit <= 0 || limit > api.config.HistoryLimit {
		limit = api.config.HistoryLimit
	}

	chatMessages, err := api.chatService.GetHistory(limit)
	if err != nil {
		return ChatHistory{}, fmt.Errorf("не удалось получить историю: %w", err)
	}

	// Конвертируем в API Message
	var messages []Message
	for _, chatMsg := range chatMessages {
		msg := Message{
			ID:          chatMsg.ID,
			Sender:      chatMsg.SenderName,
			Text:        chatMsg.Text,
			Timestamp:   chatMsg.Timestamp,
			ChatType:    chatMsg.ChatType,
			RecipientID: chatMsg.RecipientID,
			IsOutgoing:  chatMsg.IsOutgoing,
		}
		messages = append(messages, msg)
	}

	return ChatHistory{
		Messages:   messages,
		TotalCount: len(messages),
		HasMore:    len(messages) == limit,
	}, nil
}

// MessageChannel возвращает канал для получения сообщений
func (api *owlWhisperAPI) MessageChannel() <-chan Message {
	return api.messagesChan
}

// PeerChannel возвращает канал для получения обновлений пиров
func (api *owlWhisperAPI) PeerChannel() <-chan []Peer {
	return api.peersChan
}

// GetMyPeerID возвращает ID нашего пира
func (api *owlWhisperAPI) GetMyPeerID() string {
	api.mutex.RLock()
	if !api.isRunning {
		api.mutex.RUnlock()
		return ""
	}
	api.mutex.RUnlock()

	return api.coreController.GetMyID()
}

// handleMessages обрабатывает сообщения от Chat Service
func (api *owlWhisperAPI) handleMessages() {
	for {
		select {
		case chatMsg, ok := <-api.chatService.GetMessages():
			if !ok {
				return // Канал закрыт
			}

			// Конвертируем в API Message
			msg := Message{
				ID:          chatMsg.ID,
				Sender:      chatMsg.SenderName,
				Text:        chatMsg.Text,
				Timestamp:   chatMsg.Timestamp,
				ChatType:    chatMsg.ChatType,
				RecipientID: chatMsg.RecipientID,
				IsOutgoing:  chatMsg.IsOutgoing,
			}

			// Отправляем в канал API
			select {
			case api.messagesChan <- msg:
				// Сообщение отправлено
			default:
				log.Printf("⚠️ Канал API переполнен, сообщение потеряно")
			}

		case <-api.ctx.Done():
			return
		}
	}
}

// handlePeers обрабатывает обновления пиров
func (api *owlWhisperAPI) handlePeers() {
	for {
		select {
		case peerIDs, ok := <-api.chatService.GetPeers():
			if !ok {
				return // Канал закрыт
			}

			// Конвертируем в API Peer
			var peers []Peer
			for _, peerID := range peerIDs {
				peer := Peer{
					ID:       peerID.String(),
					Nickname: shortenPeerID(peerID.String()),
					Status:   "online",
					LastSeen: time.Now(),
				}
				peers = append(peers, peer)
			}

			// Отправляем в канал API
			select {
			case api.peersChan <- peers:
				// Пиры отправлены
			default:
				// Канал переполнен, пропускаем
			}

		case <-api.ctx.Done():
			return
		}
	}
}

// shortenPeerID сокращает PeerID для отображения
func shortenPeerID(peerID string) string {
	if len(peerID) > 8 {
		return peerID[:8] + "..."
	}
	return peerID
}
