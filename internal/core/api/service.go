package api

import (
	"context"
	"fmt"
	"log"
	"sync"

	"OwlWhisper/pkg/interfaces"
)

// Service реализует все API интерфейсы
type Service struct {
	chatAPI    interfaces.ChatAPI
	networkAPI interfaces.NetworkAPI
	fileAPI    interfaces.FileAPI
	systemAPI  interfaces.SystemAPI

	// Каналы для событий
	messageChan    chan interfaces.Message
	connectionChan chan interfaces.ConnectionEvent
	fileChan       chan interfaces.FileTransfer

	mu sync.RWMutex
}

// NewService создает новый API сервис
func NewService(
	chatAPI interfaces.ChatAPI,
	networkAPI interfaces.NetworkAPI,
	fileAPI interfaces.FileAPI,
	systemAPI interfaces.SystemAPI,
) *Service {
	service := &Service{
		chatAPI:    chatAPI,
		networkAPI: networkAPI,
		fileAPI:    fileAPI,
		systemAPI:  systemAPI,

		messageChan:    make(chan interfaces.Message, 100),
		connectionChan: make(chan interfaces.ConnectionEvent, 100),
		fileChan:       make(chan interfaces.FileTransfer, 100),
	}

	return service
}

// ChatAPI методы
func (s *Service) SendMessage(ctx context.Context, peerID string, content string) error {
	if s.chatAPI == nil {
		return fmt.Errorf("сервис чата не доступен")
	}
	return s.chatAPI.SendMessage(ctx, peerID, content)
}

func (s *Service) SendBroadcast(ctx context.Context, content string) error {
	if s.chatAPI == nil {
		return fmt.Errorf("сервис чата не доступен")
	}
	return s.chatAPI.SendBroadcast(ctx, content)
}

func (s *Service) GetMessageHistory(peerID string, limit int) ([]interfaces.Message, error) {
	if s.chatAPI == nil {
		return nil, fmt.Errorf("сервис чата не доступен")
	}
	return s.chatAPI.GetMessageHistory(peerID, limit)
}

func (s *Service) SubscribeToMessages() <-chan interfaces.Message {
	return s.messageChan
}

// NetworkAPI методы
func (s *Service) GetPeers() []interfaces.PeerInfo {
	if s.networkAPI == nil {
		return nil
	}
	return s.networkAPI.GetPeers()
}

func (s *Service) GetPeerInfo(peerID string) (interfaces.PeerInfo, error) {
	if s.networkAPI == nil {
		return interfaces.PeerInfo{}, fmt.Errorf("сетевой сервис не доступен")
	}
	return s.networkAPI.GetPeerInfo(peerID)
}

func (s *Service) ConnectToPeer(ctx context.Context, peerID string) error {
	if s.networkAPI == nil {
		return fmt.Errorf("сетевой сервис не доступен")
	}
	return s.networkAPI.ConnectToPeer(ctx, peerID)
}

func (s *Service) DisconnectFromPeer(ctx context.Context, peerID string) error {
	if s.networkAPI == nil {
		return fmt.Errorf("сетевой сервис не доступен")
	}
	return s.networkAPI.DisconnectFromPeer(ctx, peerID)
}

func (s *Service) SubscribeToConnections() <-chan interfaces.ConnectionEvent {
	return s.connectionChan
}

// FileAPI методы
func (s *Service) SendFile(ctx context.Context, peerID string, filePath string) error {
	if s.fileAPI == nil {
		return fmt.Errorf("сервис файлов не доступен")
	}
	return s.fileAPI.SendFile(ctx, peerID, filePath)
}

func (s *Service) GetFileInfo(fileID string) (interfaces.FileInfo, error) {
	if s.fileAPI == nil {
		return interfaces.FileInfo{}, fmt.Errorf("сервис файлов не доступен")
	}
	return s.fileAPI.GetFileInfo(fileID)
}

func (s *Service) DownloadFile(ctx context.Context, fileID string, savePath string) error {
	if s.fileAPI == nil {
		return fmt.Errorf("сервис файлов не доступен")
	}
	return s.fileAPI.DownloadFile(ctx, fileID, savePath)
}

func (s *Service) SubscribeToFiles() <-chan interfaces.FileTransfer {
	return s.fileChan
}

// SystemAPI методы
func (s *Service) GetStatus() interfaces.SystemStatus {
	if s.systemAPI == nil {
		return interfaces.SystemStatus{
			Running:     false,
			PeersCount:  0,
			NetworkType: "unknown",
			Uptime:      0,
			Version:     "0.1.0",
		}
	}
	return s.systemAPI.GetStatus()
}

func (s *Service) GetConfig() interfaces.Config {
	if s.systemAPI == nil {
		return interfaces.Config{}
	}
	return s.systemAPI.GetConfig()
}

func (s *Service) UpdateConfig(ctx context.Context, config interfaces.Config) error {
	if s.systemAPI == nil {
		return fmt.Errorf("системный сервис не доступен")
	}
	return s.systemAPI.UpdateConfig(ctx, config)
}

func (s *Service) Shutdown(ctx context.Context) error {
	if s.systemAPI == nil {
		return fmt.Errorf("системный сервис не доступен")
	}
	return s.systemAPI.Shutdown(ctx)
}

// Внутренние методы для отправки событий
func (s *Service) SendMessageEvent(message interfaces.Message) {
	select {
	case s.messageChan <- message:
	default:
		log.Printf("⚠️ Канал сообщений переполнен, сообщение потеряно")
	}
}

func (s *Service) SendConnectionEvent(event interfaces.ConnectionEvent) {
	select {
	case s.connectionChan <- event:
	default:
		log.Printf("⚠️ Канал подключений переполнен, событие потеряно")
	}
}

func (s *Service) SendFileEvent(event interfaces.FileTransfer) {
	select {
	case s.fileChan <- event:
	default:
		log.Printf("⚠️ Канал файлов переполнен, событие потеряно")
	}
}
