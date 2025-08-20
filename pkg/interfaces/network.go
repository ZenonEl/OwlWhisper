package interfaces

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// NetworkConfig содержит конфигурацию сети
type NetworkConfig struct {
	ListenPort       int           `json:"listenPort"`
	EnableNAT        bool          `json:"enableNAT"`
	EnableHolePunch  bool          `json:"enableHolePunch"`
	EnableRelay      bool          `json:"enableRelay"`
	DiscoveryTimeout time.Duration `json:"discoveryTimeout"`
}

// NetworkService определяет интерфейс для сетевого взаимодействия
type NetworkService interface {
	// Start запускает сетевой сервис
	Start(ctx context.Context) error

	// Stop останавливает сетевой сервис
	Stop(ctx context.Context) error

	// GetPeers возвращает список подключенных пиров
	GetPeers() []peer.ID

	// IsConnected проверяет, подключен ли пир
	IsConnected(peerID peer.ID) bool

	// GetPeerInfo возвращает информацию о пире
	GetPeerInfo(peerID peer.ID) (PeerInfo, error)
}

// PeerInfo содержит информацию о пире
type PeerInfo struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Latency int64  `json:"latency"` // в миллисекундах
}
