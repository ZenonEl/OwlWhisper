package storage

import (
	"OwlWhisper/internal/protocol/protocol"
	"time"
)

// IMessageRepository определяет интерфейс для работы с сообщениями
type IMessageRepository interface {
	// Save сохраняет сообщение в базу данных
	Save(message *protocol.ChatMessage, senderID string) error

	// GetHistory возвращает историю сообщений
	GetHistory(limit int) ([]StoredMessage, error)

	// GetMessagesByPeer возвращает сообщения от конкретного пира
	GetMessagesByPeer(peerID string, limit int) ([]StoredMessage, error)

	// GetMessageByID возвращает сообщение по ID
	GetMessageByID(messageID string) (*StoredMessage, error)

	// DeleteMessage удаляет сообщение по ID
	DeleteMessage(messageID string) error

	// Close закрывает соединение с базой данных
	Close() error
}

// StoredMessage представляет собой сообщение, хранимое в базе данных
type StoredMessage struct {
	ID          string    `json:"id"`
	Text        string    `json:"text"`
	Timestamp   time.Time `json:"timestamp"`
	SenderID    string    `json:"sender_id"`
	ChatType    string    `json:"chat_type"`
	RecipientID string    `json:"recipient_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IContactRepository определяет интерфейс для работы с контактами
type IContactRepository interface {
	// SaveContact сохраняет контакт
	SaveContact(contact *Contact) error

	// GetContact возвращает контакт по ID
	GetContact(contactID string) (*Contact, error)

	// GetAllContacts возвращает все контакты
	GetAllContacts() ([]Contact, error)

	// UpdateContact обновляет контакт
	UpdateContact(contact *Contact) error

	// DeleteContact удаляет контакт
	DeleteContact(contactID string) error

	// Close закрывает соединение с базой данных
	Close() error
}

// Contact представляет собой контакт
type Contact struct {
	ID         string    `json:"id"`
	PeerID     string    `json:"peer_id"`
	Nickname   string    `json:"nickname"`
	AvatarHash string    `json:"avatar_hash"`
	Status     string    `json:"status"` // online, offline, away
	LastSeen   time.Time `json:"last_seen"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// DatabaseConfig содержит конфигурацию базы данных
type DatabaseConfig struct {
	Driver   string `json:"driver"`    // sqlite, postgres, mysql
	DSN      string `json:"dsn"`       // строка подключения
	MaxConns int    `json:"max_conns"` // максимальное количество соединений
}
