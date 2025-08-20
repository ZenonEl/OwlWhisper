package api

import (
	"time"
)

// Message представляет собой сообщение для внешних клиентов
type Message struct {
	ID          string    `json:"id"`
	Sender      string    `json:"sender"` // Может быть никнейм или PeerID
	Text        string    `json:"text"`
	Timestamp   time.Time `json:"timestamp"`
	ChatType    string    `json:"chat_type"` // "private", "group", "broadcast"
	RecipientID string    `json:"recipient_id,omitempty"`
	IsOutgoing  bool      `json:"is_outgoing"` // true если сообщение от нас
}

// Peer представляет собой участника сети
type Peer struct {
	ID       string    `json:"id"`
	Nickname string    `json:"nickname,omitempty"`
	Status   string    `json:"status"` // "online", "offline", "away"
	LastSeen time.Time `json:"last_seen"`
}

// ConnectionStatus представляет собой статус подключения
type ConnectionStatus struct {
	IsConnected bool      `json:"is_connected"`
	PeerCount   int       `json:"peer_count"`
	MyPeerID    string    `json:"my_peer_id"`
	LastUpdate  time.Time `json:"last_update"`
	NetworkType string    `json:"network_type"` // "local", "global", "relay"
}

// ChatHistory представляет собой историю сообщений
type ChatHistory struct {
	Messages   []Message `json:"messages"`
	TotalCount int       `json:"total_count"`
	HasMore    bool      `json:"has_more"`
}

// SendMessageRequest представляет собой запрос на отправку сообщения
type SendMessageRequest struct {
	Text        string `json:"text"`
	ChatType    string `json:"chat_type"` // "private", "group", "broadcast"
	RecipientID string `json:"recipient_id,omitempty"`
}

// APIConfig представляет собой конфигурацию API
type APIConfig struct {
	EnableTUI      bool   `json:"enable_tui"`       // Включить TUI интерфейс
	DatabasePath   string `json:"database_path"`    // Путь к базе данных
	LogLevel       string `json:"log_level"`        // "debug", "info", "warn", "error"
	MaxMessageSize int    `json:"max_message_size"` // Максимальный размер сообщения
	HistoryLimit   int    `json:"history_limit"`    // Лимит истории сообщений
}

// DefaultAPIConfig возвращает конфигурацию по умолчанию
func DefaultAPIConfig() *APIConfig {
	return &APIConfig{
		EnableTUI:      true,
		DatabasePath:   "owlwhisper.db",
		LogLevel:       "info",
		MaxMessageSize: 4096, // 4KB
		HistoryLimit:   100,
	}
}
