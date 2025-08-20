package interfaces

import (
	"context"
	"time"
)

// MessageService определяет интерфейс для работы с сообщениями
type MessageService interface {
	// SendMessage отправляет сообщение конкретному пиру
	SendMessage(ctx context.Context, peerID string, content string) error

	// SendBroadcast отправляет сообщение всем подключенным пирам
	SendBroadcast(ctx context.Context, content string) error

	// GetMessageHistory возвращает историю сообщений
	GetMessageHistory(peerID string, limit int) ([]Message, error)

	// SubscribeToMessages подписывается на входящие сообщения
	SubscribeToMessages() <-chan Message
}

// Message представляет собой сообщение
type Message struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Content   string    `json:"content"`
	Type      string    `json:"type"` // text, file, image, etc.
	Timestamp time.Time `json:"timestamp"`
	Encrypted bool      `json:"encrypted"`
}
