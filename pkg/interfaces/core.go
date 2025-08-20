package interfaces

import (
	"context"
)

// CoreService определяет основной интерфейс для CORE сервиса
type CoreService interface {
	// Start запускает CORE сервис
	Start(ctx context.Context) error

	// Stop останавливает CORE сервис
	Stop(ctx context.Context) error

	// GetStatus возвращает статус сервиса
	GetStatus() ServiceStatus

	// Network возвращает сетевой сервис
	Network() NetworkService

	// Messages возвращает сервис сообщений
	Messages() MessageService

	// Files возвращает сервис файлов
	Files() FileService
}

// ServiceStatus представляет статус сервиса
type ServiceStatus struct {
	Running     bool   `json:"running"`
	PeersCount  int    `json:"peersCount"`
	NetworkType string `json:"networkType"` // local, global, mixed
	Uptime      int64  `json:"uptime"`      // в секундах
}
