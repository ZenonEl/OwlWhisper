package interfaces

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
)

// ITransport определяет интерфейс для транспортного слоя
type ITransport interface {
	// Start запускает транспорт
	Start(ctx context.Context) error

	// Stop останавливает транспорт
	Stop(ctx context.Context) error

	// Connect подключается к указанному пиру
	Connect(ctx context.Context, peerID peer.ID) error

	// ConnectDirectly подключается к пиру по multiaddr
	ConnectDirectly(ctx context.Context, multiaddrStr string) error

	// Disconnect отключается от указанного пира
	Disconnect(ctx context.Context, peerID peer.ID) error

	// SendMessage отправляет сообщение указанному пиру
	SendMessage(ctx context.Context, peerID peer.ID, message []byte) error

	// GetConnectedPeers возвращает список подключенных пиров
	GetConnectedPeers() []peer.ID

	// GetPeerID возвращает ID текущего пира
	GetPeerID() peer.ID

	// GetMultiaddrs возвращает адреса текущего пира
	GetMultiaddrs() []string

	// DiscoverPeers ищет пиров в глобальной сети
	DiscoverPeers(ctx context.Context) (<-chan peer.AddrInfo, error)

	// Advertise анонсирует себя в глобальной сети
	Advertise(ctx context.Context) error

	// SetMessageHandler устанавливает обработчик входящих сообщений
	SetMessageHandler(handler func(peer.ID, []byte))
}
