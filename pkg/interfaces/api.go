package interfaces

import (
	"context"
	"time"
)

// ChatAPI определяет API для чата
type ChatAPI interface {
	// SendMessage отправляет сообщение конкретному пиру
	SendMessage(ctx context.Context, peerID string, content string) error

	// SendBroadcast отправляет сообщение всем подключенным пирам
	SendBroadcast(ctx context.Context, content string) error

	// GetMessageHistory возвращает историю сообщений
	GetMessageHistory(peerID string, limit int) ([]Message, error)

	// SubscribeToMessages подписывается на входящие сообщения
	SubscribeToMessages() <-chan Message
}

// NetworkAPI определяет API для сетевых операций
type NetworkAPI interface {
	// GetPeers возвращает список подключенных пиров
	GetPeers() []PeerInfo

	// GetPeerInfo возвращает информацию о конкретном пире
	GetPeerInfo(peerID string) (PeerInfo, error)

	// ConnectToPeer подключается к пиру по ID
	ConnectToPeer(ctx context.Context, peerID string) error

	// DisconnectFromPeer отключается от пира
	DisconnectFromPeer(ctx context.Context, peerID string) error

	// SubscribeToConnections подписывается на события подключений
	SubscribeToConnections() <-chan ConnectionEvent
}

// FileAPI определяет API для работы с файлами
type FileAPI interface {
	// SendFile отправляет файл конкретному пиру
	SendFile(ctx context.Context, peerID string, filePath string) error

	// GetFileInfo возвращает информацию о файле
	GetFileInfo(fileID string) (FileInfo, error)

	// DownloadFile скачивает файл
	DownloadFile(ctx context.Context, fileID string, savePath string) error

	// SubscribeToFiles подписывается на входящие файлы
	SubscribeToFiles() <-chan FileTransfer
}

// SystemAPI определяет API для системных операций
type SystemAPI interface {
	// GetStatus возвращает статус системы
	GetStatus() SystemStatus

	// GetConfig возвращает конфигурацию
	GetConfig() Config

	// UpdateConfig обновляет конфигурацию
	UpdateConfig(ctx context.Context, config Config) error

	// Shutdown корректно останавливает систему
	Shutdown(ctx context.Context) error
}

// ConnectionEvent представляет событие подключения
type ConnectionEvent struct {
	Type      string    `json:"type"` // "connected", "disconnected"
	PeerID    string    `json:"peerId"`
	Timestamp time.Time `json:"timestamp"`
}

// SystemStatus представляет статус системы
type SystemStatus struct {
	Running     bool   `json:"running"`
	PeersCount  int    `json:"peersCount"`
	NetworkType string `json:"networkType"`
	Uptime      int64  `json:"uptime"`
	Version     string `json:"version"`
}
