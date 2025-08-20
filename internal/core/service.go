package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"OwlWhisper/internal/core/network"
	"OwlWhisper/pkg/interfaces"
)

// Service реализует CoreService интерфейс
type Service struct {
	networkService interfaces.NetworkService
	messageService interfaces.MessageService
	fileService    interfaces.FileService

	config *interfaces.Config

	startTime time.Time
	running   bool
	mu        sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

// NewService создает новый CORE сервис
func NewService(config *interfaces.Config) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем сетевой сервис
	networkService, err := network.NewService(&interfaces.NetworkConfig{
		ListenPort:       config.Network.ListenPort,
		EnableNAT:        config.Network.EnableNAT,
		EnableHolePunch:  config.Network.EnableHolePunch,
		EnableRelay:      config.Network.EnableRelay,
		DiscoveryTimeout: config.Network.DiscoveryTimeout,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать сетевой сервис: %w", err)
	}

	// TODO: Создать сервисы сообщений и файлов
	// messageService := messages.NewService(config)
	// fileService := files.NewService(config)

	service := &Service{
		networkService: networkService,
		// messageService: messageService,
		// fileService:    fileService,
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	return service, nil
}

// Start запускает CORE сервис
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("сервис уже запущен")
	}

	// Запускаем сетевой сервис
	if err := s.networkService.Start(ctx); err != nil {
		return fmt.Errorf("не удалось запустить сетевой сервис: %w", err)
	}

	// TODO: Запустить сервисы сообщений и файлов
	// if err := s.messageService.Start(ctx); err != nil {
	//     return fmt.Errorf("не удалось запустить сервис сообщений: %w", err)
	// }
	// if err := s.fileService.Start(ctx); err != nil {
	//     return fmt.Errorf("не удалось запустить сервис файлов: %w", err)
	// }

	s.startTime = time.Now()
	s.running = true

	log.Println("🚀 CORE сервис запущен")
	return nil
}

// Stop останавливает CORE сервис
func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	// Останавливаем все сервисы
	if err := s.networkService.Stop(ctx); err != nil {
		log.Printf("⚠️ Ошибка остановки сетевого сервиса: %v", err)
	}

	// TODO: Остановить сервисы сообщений и файлов
	// if err := s.messageService.Stop(ctx); err != nil {
	//     log.Printf("⚠️ Ошибка остановки сервиса сообщений: %v", err)
	// }
	// if err := s.fileService.Stop(ctx); err != nil {
	//     log.Printf("⚠️ Ошибка остановки сервиса файлов: %v", err)
	// }

	s.running = false
	s.cancel()

	log.Println("🛑 CORE сервис остановлен")
	return nil
}

// GetStatus возвращает статус сервиса
func (s *Service) GetStatus() interfaces.ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var uptime int64
	if s.running {
		uptime = int64(time.Since(s.startTime).Seconds())
	}

	return interfaces.ServiceStatus{
		Running:     s.running,
		PeersCount:  len(s.networkService.GetPeers()),
		NetworkType: "mixed", // TODO: определить тип сети
		Uptime:      uptime,
	}
}

// Network возвращает сетевой сервис
func (s *Service) Network() interfaces.NetworkService {
	return s.networkService
}

// Messages возвращает сервис сообщений
func (s *Service) Messages() interfaces.MessageService {
	return s.messageService
}

// Files возвращает сервис файлов
func (s *Service) Files() interfaces.FileService {
	return s.fileService
}
