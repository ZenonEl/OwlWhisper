package network

import (
	"context"
	"fmt"
	"log"
	"sync"

	"OwlWhisper/pkg/interfaces"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
)

// Service реализует NetworkService интерфейс
type Service struct {
	host             host.Host
	dht              *dht.IpfsDHT
	routingDiscovery *routing.RoutingDiscovery
	mdnsService      mdns.Service

	peersMutex sync.RWMutex
	peers      map[peer.ID]*interfaces.PeerInfo

	ctx    context.Context
	cancel context.CancelFunc

	config *interfaces.NetworkConfig
}

// NewService создает новый сетевой сервис
func NewService(config *interfaces.NetworkConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем libp2p узел с опциями для глобальной сети
	opts := []libp2p.Option{
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
	}

	if config.ListenPort > 0 {
		opts = append(opts, libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.ListenPort)))
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать libp2p узел: %w", err)
	}

	service := &Service{
		host:   h,
		ctx:    ctx,
		cancel: cancel,
		config: config,
		peers:  make(map[peer.ID]*interfaces.PeerInfo),
	}

	// Устанавливаем Network Notifiee для мониторинга
	h.Network().Notify(service)

	return service, nil
}

// Start запускает сетевой сервис
func (s *Service) Start(ctx context.Context) error {
	// Создаем DHT
	var err error
	s.dht, err = dht.New(ctx, s.host)
	if err != nil {
		return fmt.Errorf("не удалось создать DHT: %w", err)
	}

	// Подключаемся к bootstrap узлам
	if err = s.dht.Bootstrap(ctx); err != nil {
		return fmt.Errorf("не удалось подключиться к bootstrap узлам: %w", err)
	}

	// Создаем routing discovery
	s.routingDiscovery = routing.NewRoutingDiscovery(s.dht)

	// Запускаем mDNS discovery
	notifee := &mdnsNotifee{service: s}
	s.mdnsService = mdns.NewMdnsService(s.host, "owl-whisper-mdns", notifee)
	if err := s.mdnsService.Start(); err != nil {
		return fmt.Errorf("не удалось запустить mDNS: %w", err)
	}

	// Запускаем DHT discovery в фоне
	go s.startDHTDiscovery()

	log.Printf("✅ Сетевой сервис запущен. PeerID: %s", s.host.ID().String())
	return nil
}

// Stop останавливает сетевой сервис
func (s *Service) Stop(ctx context.Context) error {
	if s.mdnsService != nil {
		s.mdnsService.Close()
	}

	if s.host != nil {
		return s.host.Close()
	}

	return nil
}

// GetPeers возвращает список подключенных пиров
func (s *Service) GetPeers() []peer.ID {
	s.peersMutex.RLock()
	defer s.peersMutex.RUnlock()

	peers := make([]peer.ID, 0, len(s.peers))
	for peerID := range s.peers {
		peers = append(peers, peerID)
	}

	return peers
}

// IsConnected проверяет, подключен ли пир
func (s *Service) IsConnected(peerID peer.ID) bool {
	s.peersMutex.RLock()
	defer s.peersMutex.RUnlock()

	_, exists := s.peers[peerID]
	return exists
}

// GetPeerInfo возвращает информацию о пире
func (s *Service) GetPeerInfo(peerID peer.ID) (interfaces.PeerInfo, error) {
	s.peersMutex.RLock()
	defer s.peersMutex.RUnlock()

	info, exists := s.peers[peerID]
	if !exists {
		return interfaces.PeerInfo{}, fmt.Errorf("пир %s не найден", peerID.String())
	}

	return *info, nil
}

// startDHTDiscovery запускает поиск через DHT
func (s *Service) startDHTDiscovery() {
	// Анонсируемся в глобальной сети
	ttl, err := s.routingDiscovery.Advertise(s.ctx, "owl-whisper-global")
	if err != nil {
		log.Printf("⚠️ Не удалось анонсироваться в глобальной сети: %v", err)
	} else {
		log.Printf("📢 Анонсировались в глобальной сети, TTL: %v", ttl)
	}

	// Поиск других участников
	peerChan, err := s.routingDiscovery.FindPeers(s.ctx, "owl-whisper-global")
	if err != nil {
		log.Printf("⚠️ Ошибка поиска в глобальной сети: %v", err)
		return
	}

	for p := range peerChan {
		if p.ID == s.host.ID() {
			continue
		}

		log.Printf("🌐 Найден участник в глобальной сети: %s", p.ID.ShortString())
		s.handlePeerFound(p)
	}
}

// handlePeerFound обрабатывает найденного пира
func (s *Service) handlePeerFound(pi peer.AddrInfo) {
	// Пытаемся подключиться
	if err := s.host.Connect(s.ctx, pi); err != nil {
		log.Printf("❌ Не удалось подключиться к %s: %v", pi.ID.ShortString(), err)
		return
	}

	log.Printf("✅ Успешное подключение к %s", pi.ID.ShortString())

	// Добавляем в список пиров
	s.peersMutex.Lock()
	s.peers[pi.ID] = &interfaces.PeerInfo{
		ID:      pi.ID,
		Address: pi.Addrs[0].String(),
		Latency: 0, // TODO: измерить латентность
	}
	s.peersMutex.Unlock()
}

// Network Notifiee методы
func (s *Service) Listen(network.Network, multiaddr.Multiaddr)      {}
func (s *Service) ListenClose(network.Network, multiaddr.Multiaddr) {}

func (s *Service) Connected(net network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	log.Printf("🔗 EVENT: Успешное соединение с %s", peerID.ShortString())
}

func (s *Service) Disconnected(net network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	log.Printf("🔌 EVENT: Соединение с %s разорвано", peerID.ShortString())

	// Удаляем из списка пиров
	s.peersMutex.Lock()
	delete(s.peers, peerID)
	s.peersMutex.Unlock()
}

func (s *Service) OpenedStream(network.Network, network.Stream) {}
func (s *Service) ClosedStream(network.Network, network.Stream) {}

// mdnsNotifee для mDNS discovery
type mdnsNotifee struct {
	service *Service
}

func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.service.handlePeerFound(pi)
}
