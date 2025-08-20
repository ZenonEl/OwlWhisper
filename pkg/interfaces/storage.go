package interfaces

import (
	"context"
	"time"
)

// Message представляет сообщение в чате
type Message struct {
	ID        string    `json:"id"`
	FromPeer  string    `json:"from_peer"`
	ToPeer    string    `json:"to_peer"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "text", "file", etc.
	IsRead    bool      `json:"is_read"`
}

// Contact представляет контакт пользователя
type Contact struct {
	PeerID   string    `json:"peer_id"`
	Nickname string    `json:"nickname"`
	AddedAt  time.Time `json:"added_at"`
	LastSeen time.Time `json:"last_seen"`
	IsOnline bool      `json:"is_online"`
}

// IMessageRepository определяет интерфейс для работы с сообщениями
type IMessageRepository interface {
	// SaveMessage сохраняет сообщение
	SaveMessage(ctx context.Context, message *Message) error

	// GetMessages возвращает сообщения между двумя пирами
	GetMessages(ctx context.Context, peer1, peer2 string, limit int, offset int) ([]*Message, error)

	// GetLastMessage возвращает последнее сообщение между двумя пирами
	GetLastMessage(ctx context.Context, peer1, peer2 string) (*Message, error)

	// DeleteMessage удаляет сообщение по ID
	DeleteMessage(ctx context.Context, messageID string) error

	// GetUnreadCount возвращает количество непрочитанных сообщений
	GetUnreadCount(ctx context.Context, peerID string) (int, error)

	// MarkAsRead отмечает сообщения как прочитанные
	MarkAsRead(ctx context.Context, peerID string) error
}

// IContactRepository определяет интерфейс для работы с контактами
type IContactRepository interface {
	// SaveContact сохраняет контакт
	SaveContact(ctx context.Context, contact *Contact) error

	// GetContact возвращает контакт по PeerID
	GetContact(ctx context.Context, peerID string) (*Contact, error)

	// GetAllContacts возвращает все контакты
	GetAllContacts(ctx context.Context) ([]*Contact, error)

	// UpdateContact обновляет контакт
	UpdateContact(ctx context.Context, contact *Contact) error

	// DeleteContact удаляет контакт
	DeleteContact(ctx context.Context, peerID string) error

	// UpdateLastSeen обновляет время последнего появления пира
	UpdateLastSeen(ctx context.Context, peerID string, lastSeen time.Time) error
}
