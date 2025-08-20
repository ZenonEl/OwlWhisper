package interfaces

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
)

// IChatService определяет интерфейс для сервиса чата
type IChatService interface {
	// SendMessage отправляет сообщение указанному пиру
	SendMessage(ctx context.Context, toPeer peer.ID, content string) error

	// GetMessages возвращает историю сообщений с указанным пиром
	GetMessages(ctx context.Context, peerID peer.ID, limit, offset int) ([]*Message, error)

	// GetUnreadCount возвращает количество непрочитанных сообщений
	GetUnreadCount(ctx context.Context, peerID peer.ID) (int, error)

	// MarkAsRead отмечает сообщения как прочитанные
	MarkAsRead(ctx context.Context, peerID peer.ID) error
}

// IContactService определяет интерфейс для сервиса контактов
type IContactService interface {
	// AddContact добавляет новый контакт
	AddContact(ctx context.Context, peerID peer.ID, nickname string) error

	// RemoveContact удаляет контакт
	RemoveContact(ctx context.Context, peerID peer.ID) error

	// GetContacts возвращает все контакты
	GetContacts(ctx context.Context) ([]*Contact, error)

	// UpdateContact обновляет информацию о контакте
	UpdateContact(ctx context.Context, contact *Contact) error

	// GetContact возвращает контакт по PeerID
	GetContact(ctx context.Context, peerID peer.ID) (*Contact, error)
}

// INetworkService определяет интерфейс для сетевого сервиса
type INetworkService interface {
	// Start запускает сетевой сервис
	Start(ctx context.Context) error

	// Stop останавливает сетевой сервис
	Stop(ctx context.Context) error

	// ConnectToPeer подключается к указанному пиру
	ConnectToPeer(ctx context.Context, peerID peer.ID) error

	// DisconnectFromPeer отключается от указанного пира
	DisconnectFromPeer(ctx context.Context, peerID peer.ID) error

	// GetConnectedPeers возвращает список подключенных пиров
	GetConnectedPeers() []peer.ID

	// GetPeerInfo возвращает информацию о пире
	GetPeerInfo(peerID peer.ID) *Contact

	// SetMessageHandler устанавливает обработчик входящих сообщений
	SetMessageHandler(handler func(peer.ID, []byte))
}
