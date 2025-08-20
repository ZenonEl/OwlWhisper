package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"OwlWhisper/pkg/interfaces"

	"github.com/libp2p/go-libp2p/core/peer"
)

// NetworkService реализует INetworkService интерфейс
type NetworkService struct {
	transport      interfaces.ITransport
	contactService *ContactService
	chatService    *ChatService
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewNetworkService создает новый экземпляр NetworkService
func NewNetworkService(transport interfaces.ITransport, contactService *ContactService, chatService *ChatService) *NetworkService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &NetworkService{
		transport:      transport,
		contactService: contactService,
		chatService:    chatService,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Устанавливаем обработчик сообщений
	transport.SetMessageHandler(service.handleIncomingMessage)

	return service
}

// Start запускает сетевой сервис
func (s *NetworkService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Запускаем транспорт
	if err := s.transport.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	// Запускаем мониторинг подключений
	go s.monitorConnections()

	log.Printf("🌐 Сетевой сервис запущен")
	return nil
}

// Stop останавливает сетевой сервис
func (s *NetworkService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	if err := s.transport.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop transport: %w", err)
	}

	log.Printf("🌐 Сетевой сервис остановлен")
	return nil
}

// ConnectToPeer подключается к указанному пиру
func (s *NetworkService) ConnectToPeer(ctx context.Context, peerID peer.ID) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Подключаемся через транспорт
	if err := s.transport.Connect(ctx, peerID); err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	// Обновляем статус контакта
	if err := s.contactService.UpdateLastSeen(ctx, peerID); err != nil {
		log.Printf("Warning: failed to update contact status: %v", err)
	}

	log.Printf("🔗 Подключились к пиру: %s", peerID.ShortString())
	return nil
}

// ConnectDirectly подключается к пиру по multiaddr
func (s *NetworkService) ConnectDirectly(ctx context.Context, multiaddrStr string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Подключаемся через транспорт
	if err := s.transport.ConnectDirectly(ctx, multiaddrStr); err != nil {
		return fmt.Errorf("failed to connect directly: %w", err)
	}

	log.Printf("🔗 Прямое подключение установлено к %s", multiaddrStr)
	return nil
}

// DisconnectFromPeer отключается от указанного пира
func (s *NetworkService) DisconnectFromPeer(ctx context.Context, peerID peer.ID) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Отключаемся через транспорт
	if err := s.transport.Disconnect(ctx, peerID); err != nil {
		return fmt.Errorf("failed to disconnect from peer: %w", err)
	}

	// Отмечаем контакт как оффлайн
	if err := s.contactService.SetOffline(ctx, peerID); err != nil {
		log.Printf("Warning: failed to set contact offline: %v", err)
	}

	log.Printf("🔌 Отключились от пира: %s", peerID.ShortString())
	return nil
}

// GetConnectedPeers возвращает список подключенных пиров
func (s *NetworkService) GetConnectedPeers() []peer.ID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.transport.GetConnectedPeers()
}

// GetPeerInfo возвращает информацию о пире
func (s *NetworkService) GetPeerInfo(peerID peer.ID) *interfaces.Contact {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contact, err := s.contactService.GetContact(context.Background(), peerID)
	if err != nil {
		log.Printf("Warning: failed to get peer info: %v", err)
		return nil
	}

	return contact
}

// SetMessageHandler устанавливает обработчик входящих сообщений
func (s *NetworkService) SetMessageHandler(handler func(peer.ID, []byte)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Обновляем обработчик в транспорте
	s.transport.SetMessageHandler(handler)
}

// handleIncomingMessage обрабатывает входящие сообщения
func (s *NetworkService) handleIncomingMessage(fromPeer peer.ID, content []byte) {
	// Обновляем статус контакта
	if err := s.contactService.UpdateLastSeen(context.Background(), fromPeer); err != nil {
		log.Printf("Warning: failed to update contact status: %v", err)
	}

	// Обрабатываем сообщение через чат-сервис
	if err := s.chatService.HandleIncomingMessage(fromPeer, content); err != nil {
		log.Printf("Error handling incoming message: %v", err)
	}
}

// monitorConnections мониторит подключения и обновляет статусы контактов
func (s *NetworkService) monitorConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.updateConnectionStatuses()
		}
	}
}

// updateConnectionStatuses обновляет статусы подключений
func (s *NetworkService) updateConnectionStatuses() {
	s.mu.RLock()
	connectedPeers := s.transport.GetConnectedPeers()
	s.mu.RUnlock()

	// Получаем все контакты
	contacts, err := s.contactService.GetContacts(context.Background())
	if err != nil {
		log.Printf("Warning: failed to get contacts: %v", err)
		return
	}

	// Создаем множество подключенных пиров для быстрого поиска
	connectedSet := make(map[string]bool)
	for _, peerID := range connectedPeers {
		connectedSet[peerID.String()] = true
	}

	// Обновляем статусы
	for _, contact := range contacts {
		isConnected := connectedSet[contact.PeerID]
		if contact.IsOnline != isConnected {
			contact.IsOnline = isConnected
			if err := s.contactService.UpdateContact(context.Background(), contact); err != nil {
				log.Printf("Warning: failed to update contact status: %v", err)
			}
		}
	}
}
