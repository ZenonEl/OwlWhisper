package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"OwlWhisper/pkg/interfaces"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ChatService реализует IChatService интерфейс
type ChatService struct {
	messageRepo interfaces.IMessageRepository
	transport   interfaces.ITransport
}

// NewChatService создает новый экземпляр ChatService
func NewChatService(messageRepo interfaces.IMessageRepository, transport interfaces.ITransport) *ChatService {
	return &ChatService{
		messageRepo: messageRepo,
		transport:   transport,
	}
}

// generateMessageID генерирует уникальный ID для сообщения
func (s *ChatService) generateMessageID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// SendMessage отправляет сообщение указанному пиру
func (s *ChatService) SendMessage(ctx context.Context, toPeer peer.ID, content string) error {
	// Проверяем длину сообщения
	if len(content) == 0 {
		return fmt.Errorf("message content cannot be empty")
	}

	if len(content) > 1000 { // Максимальная длина сообщения
		return fmt.Errorf("message too long, maximum length is 1000 characters")
	}

	// Создаем сообщение
	message := &interfaces.Message{
		ID:        s.generateMessageID(),
		FromPeer:  s.transport.GetPeerID().String(),
		ToPeer:    toPeer.String(),
		Content:   content,
		Timestamp: time.Now(),
		Type:      "text",
		IsRead:    false,
	}

	// Сохраняем сообщение локально
	if err := s.messageRepo.SaveMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to save message locally: %w", err)
	}

	// Отправляем сообщение через транспорт
	messageBytes := []byte(content)
	if err := s.transport.SendMessage(ctx, toPeer, messageBytes); err != nil {
		return fmt.Errorf("failed to send message via transport: %w", err)
	}

	log.Printf("📤 Сообщение отправлено к %s: %s", toPeer.ShortString(), content)
	return nil
}

// GetMessages возвращает историю сообщений с указанным пиром
func (s *ChatService) GetMessages(ctx context.Context, peerID peer.ID, limit, offset int) ([]*interfaces.Message, error) {
	if limit <= 0 {
		limit = 50 // Значение по умолчанию
	}
	if offset < 0 {
		offset = 0
	}

	// Получаем сообщения из репозитория
	messages, err := s.messageRepo.GetMessages(ctx, s.transport.GetPeerID().String(), peerID.String(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	return messages, nil
}

// GetUnreadCount возвращает количество непрочитанных сообщений
func (s *ChatService) GetUnreadCount(ctx context.Context, peerID peer.ID) (int, error) {
	count, err := s.messageRepo.GetUnreadCount(ctx, peerID.String())
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// MarkAsRead отмечает сообщения как прочитанные
func (s *ChatService) MarkAsRead(ctx context.Context, peerID peer.ID) error {
	if err := s.messageRepo.MarkAsRead(ctx, peerID.String()); err != nil {
		return fmt.Errorf("failed to mark messages as read: %w", err)
	}

	return nil
}

// HandleIncomingMessage обрабатывает входящее сообщение
func (s *ChatService) HandleIncomingMessage(fromPeer peer.ID, content []byte) error {
	// Создаем сообщение
	message := &interfaces.Message{
		ID:        s.generateMessageID(),
		FromPeer:  fromPeer.String(),
		ToPeer:    s.transport.GetPeerID().String(),
		Content:   string(content),
		Timestamp: time.Now(),
		Type:      "text",
		IsRead:    false,
	}

	// Сохраняем сообщение
	if err := s.messageRepo.SaveMessage(context.Background(), message); err != nil {
		return fmt.Errorf("failed to save incoming message: %w", err)
	}

	log.Printf("📥 Получено сообщение от %s: %s", fromPeer.ShortString(), string(content))
	return nil
}
