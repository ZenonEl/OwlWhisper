package network

import (
	"context"
	"fmt"

	"OwlWhisper/pkg/interfaces"

	"github.com/libp2p/go-libp2p/core/peer"
)

// NetworkAPIAdapter адаптирует NetworkService для работы с API
type NetworkAPIAdapter struct {
	service *Service
}

// NewNetworkAPIAdapter создает новый адаптер
func NewNetworkAPIAdapter(service *Service) *NetworkAPIAdapter {
	return &NetworkAPIAdapter{
		service: service,
	}
}

// GetPeers возвращает список подключенных пиров для API
func (a *NetworkAPIAdapter) GetPeers() []interfaces.PeerInfo {
	peerIDs := a.service.GetPeers()
	peers := make([]interfaces.PeerInfo, 0, len(peerIDs))

	for _, peerID := range peerIDs {
		info, err := a.service.GetPeerInfo(peerID)
		if err != nil {
			continue
		}

		// Конвертируем peer.ID в string для API
		apiInfo := interfaces.PeerInfo{
			ID:      peerID.String(),
			Address: info.Address,
			Latency: info.Latency,
		}
		peers = append(peers, apiInfo)
	}

	return peers
}

// GetPeerInfo возвращает информацию о пире для API
func (a *NetworkAPIAdapter) GetPeerInfo(peerID string) (interfaces.PeerInfo, error) {
	// Конвертируем string в peer.ID
	pid, err := peer.Decode(peerID)
	if err != nil {
		return interfaces.PeerInfo{}, fmt.Errorf("неверный формат PeerID: %w", err)
	}

	info, err := a.service.GetPeerInfo(pid)
	if err != nil {
		return interfaces.PeerInfo{}, err
	}

	// Конвертируем для API
	return interfaces.PeerInfo{
		ID:      peerID,
		Address: info.Address,
		Latency: info.Latency,
	}, nil
}

// ConnectToPeer подключается к пиру по ID
func (a *NetworkAPIAdapter) ConnectToPeer(ctx context.Context, peerID string) error {
	// TODO: Реализовать подключение к пиру
	// Пока что просто возвращаем ошибку
	return fmt.Errorf("подключение к пиру пока не реализовано")
}

// DisconnectFromPeer отключается от пира
func (a *NetworkAPIAdapter) DisconnectFromPeer(ctx context.Context, peerID string) error {
	// TODO: Реализовать отключение от пира
	return fmt.Errorf("отключение от пира пока не реализовано")
}

// SubscribeToConnections подписывается на события подключений
func (a *NetworkAPIAdapter) SubscribeToConnections() <-chan interfaces.ConnectionEvent {
	// TODO: Реализовать подписку на события подключений
	// Пока что возвращаем пустой канал
	ch := make(chan interfaces.ConnectionEvent)
	close(ch)
	return ch
}
