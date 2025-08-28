package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	webrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"

	"github.com/multiformats/go-multiaddr"
)

// PROTOCOL_ID - уникальный идентификатор нашего чат-протокола
const PROTOCOL_ID = "/owl-whisper/1.0.0"

// Лимиты соединений для ConnectionManager
const (
	// Максимальное количество инфраструктурных соединений (bootstrap, DHT, mDNS)
	MAX_INFRASTRUCTURE_CONNECTIONS = 100
	// Максимальное количество защищенных соединений (контакты)
	MAX_PROTECTED_CONNECTIONS = 100
	// Общий лимит соединений
	MAX_TOTAL_CONNECTIONS = 200
)

// Настройки автопереподключения
const (
	// Интервал между попытками переподключения
	RECONNECT_INTERVAL = 30 * time.Second
	// Максимальное количество попыток переподключения
	MAX_RECONNECT_ATTEMPTS = 5
)

// NodeConfig содержит настройки для создания Node
type NodeConfig struct {
	// Транспорты
	EnableTCP       bool
	EnableQUIC      bool
	EnableWebSocket bool
	EnableWebRTC    bool

	// Шифрование
	EnableNoise bool
	EnableTLS   bool

	// NAT и Relay
	EnableNATPortMap   bool
	EnableHolePunching bool
	EnableAutoNATv2    bool
	EnableRelay        bool
	EnableAutoRelay    bool

	// Relay настройки
	StaticRelays           []string
	UseBootstrapAsRelay    bool
	AutoRelayBootDelay     time.Duration
	AutoRelayMaxCandidates int

	// Discovery
	EnableMDNS bool
	EnableDHT  bool

	// Порт и адреса
	ListenAddresses []string

	// NAT Reachability
	ForceReachabilityPublic  bool
	ForceReachabilityPrivate bool

	// Таймауты для стримов
	StreamCreationTimeout time.Duration
	StreamReadTimeout     time.Duration
	StreamWriteTimeout    time.Duration
}

// DefaultNodeConfig возвращает дефолтную конфигурацию на основе рабочего poc.go
func DefaultNodeConfig() *NodeConfig {
	return &NodeConfig{
		EnableTCP:              true,
		EnableQUIC:             true,
		EnableWebSocket:        true,
		EnableWebRTC:           true,
		EnableNoise:            true,
		EnableTLS:              true,
		EnableNATPortMap:       true,
		EnableHolePunching:     true,
		EnableAutoNATv2:        true,
		EnableRelay:            true,
		EnableAutoRelay:        true,
		UseBootstrapAsRelay:    true,
		AutoRelayBootDelay:     2 * time.Second,
		AutoRelayMaxCandidates: 10,
		EnableMDNS:             true,
		EnableDHT:              true,
		ListenAddresses: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/tcp/0/ws",
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip4/0.0.0.0/udp/0/webrtc-direct",
		},
		StaticRelays: []string{
			"/dns4/relay.dev.svcs.d.foundation/tcp/443/wss/p2p/12D3KooWCKd2fU1g4k15u3J5i6pGk26h3g68d3amEa2S71G5v1jS",
		},
		ForceReachabilityPublic:  true,
		ForceReachabilityPrivate: false,
		StreamCreationTimeout:    60 * time.Second, // как в poc.go
		StreamReadTimeout:        30 * time.Second,
		StreamWriteTimeout:       10 * time.Second,
	}
}

// buildLibp2pOptions создает опции libp2p на основе конфигурации
func buildLibp2pOptions(privKey crypto.PrivKey, config *NodeConfig) []libp2p.Option {
	opts := []libp2p.Option{
		libp2p.Identity(privKey),
	}

	// Добавляем адреса для прослушивания
	if len(config.ListenAddresses) > 0 {
		opts = append(opts, libp2p.ListenAddrStrings(config.ListenAddresses...))
	}

	// Добавляем транспорты
	if config.EnableTCP {
		opts = append(opts, libp2p.Transport(tcp.NewTCPTransport))
	}
	if config.EnableQUIC {
		opts = append(opts, libp2p.Transport(quic.NewTransport))
	}
	if config.EnableWebSocket {
		opts = append(opts, libp2p.Transport(ws.New))
	}
	if config.EnableWebRTC {
		opts = append(opts, libp2p.Transport(webrtc.New))
	}

	// Добавляем шифрование
	if config.EnableNoise {
		opts = append(opts, libp2p.Security(noise.ID, noise.New))
	}
	if config.EnableTLS {
		opts = append(opts, libp2p.Security(tls.ID, tls.New))
	}

	// Добавляем NAT и hole punching
	if config.EnableNATPortMap {
		opts = append(opts, libp2p.NATPortMap())
	}
	if config.EnableHolePunching {
		opts = append(opts, libp2p.EnableHolePunching())
	}
	if config.EnableAutoNATv2 {
		opts = append(opts, libp2p.EnableAutoNATv2())
	}

	// Добавляем relay настройки
	if config.EnableRelay {
		opts = append(opts, libp2p.EnableRelay())
	}

	// Добавляем autorelay настройки
	if config.EnableAutoRelay {
		// Создаем список всех relay узлов: статические + bootstrap
		var allRelays []peer.AddrInfo

		// Добавляем статические relay
		for _, addrStr := range config.StaticRelays {
			pi, err := peer.AddrInfoFromString(addrStr)
			if err != nil {
				Warn("⚠️ Не удалось распарсить статический relay-адрес: %v", err)
				continue
			}
			allRelays = append(allRelays, *pi)
		}

		// Добавляем bootstrap узлы как relay если включено
		if config.UseBootstrapAsRelay {
			bootstrapPeers := dht.GetDefaultBootstrapPeerAddrInfos()
			allRelays = append(allRelays, bootstrapPeers...)
		}

		// Включаем autorelay с настройками
		opts = append(opts,
			libp2p.EnableAutoRelayWithStaticRelays(allRelays),
			libp2p.EnableAutoRelayWithPeerSource(func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
				ch := make(chan peer.AddrInfo)
				go func() {
					defer close(ch)
					// Используем bootstrap узлы как источник пиров для autorelay
					bootstrapPeers := dht.GetDefaultBootstrapPeerAddrInfos()
					for _, pi := range bootstrapPeers {
						if numPeers <= 0 {
							break
						}
						select {
						case ch <- pi:
							numPeers--
						case <-ctx.Done():
							return
						}
					}
				}()
				return ch
			},
				autorelay.WithBootDelay(config.AutoRelayBootDelay),
				autorelay.WithMaxCandidates(config.AutoRelayMaxCandidates),
			),
		)
	}

	// Настройки NAT Reachability
	if config.ForceReachabilityPublic {
		opts = append(opts, libp2p.ForceReachabilityPublic())
	}
	if config.ForceReachabilityPrivate {
		opts = append(opts, libp2p.ForceReachabilityPrivate())
	}

	return opts
}

// NetworkEventLogger логирует сетевые события и отправляет их в EventManager
type NetworkEventLogger struct {
	node *Node
}

func (nel *NetworkEventLogger) Listen(network.Network, multiaddr.Multiaddr)      {}
func (nel *NetworkEventLogger) ListenClose(network.Network, multiaddr.Multiaddr) {}

func (nel *NetworkEventLogger) Connected(net network.Network, conn network.Conn) {
	peerID := conn.RemotePeer().String()
	Info("🔗 EVENT: Успешное соединение с %s", conn.RemotePeer().ShortString())

	// Отправляем событие в EventManager
	if nel.node != nil && nel.node.eventManager != nil {
		event := PeerConnectedEvent(peerID)
		if err := nel.node.eventManager.PushEvent(event); err != nil {
			Warn("⚠️ Не удалось отправить событие PeerConnected: %v", err)
		}
	}
}

func (nel *NetworkEventLogger) Disconnected(net network.Network, conn network.Conn) {
	peerID := conn.RemotePeer().String()
	Info("🔌 EVENT: Соединение с %s разорвано", conn.RemotePeer().ShortString())

	// Отправляем событие в EventManager
	if nel.node != nil && nel.node.eventManager != nil {
		event := PeerDisconnectedEvent(peerID)
		if err := nel.node.eventManager.PushEvent(event); err != nil {
			Warn("⚠️ Не удалось отправить событие PeerDisconnected: %v", err)
		}
	}
}

func (nel *NetworkEventLogger) OpenedStream(network.Network, network.Stream) {}
func (nel *NetworkEventLogger) ClosedStream(network.Network, network.Stream) {}

// Node представляет собой libp2p узел
type Node struct {
	host host.Host
	ctx  context.Context

	// Канал для входящих сообщений
	messagesChan chan RawMessage

	// Мьютекс для безопасного доступа к пирам
	peersMutex sync.RWMutex
	peers      map[peer.ID]bool

	// Менеджер персистентности для управления ключами
	persistence *PersistenceManager

	// DiscoveryManager для работы с DHT
	discovery *DiscoveryManager

	// Мьютекс для защищенных пиров
	protectedPeersMutex sync.RWMutex
	protectedPeers      map[peer.ID]bool

	// ConnectionManager для управления соединениями
	connManager interface {
		Protect(peer.ID, string)
		Unprotect(peer.ID, string) bool
		IsProtected(peer.ID, string) bool
	}

	// Лимиты соединений
	connectionLimits struct {
		infrastructure int // Текущее количество инфраструктурных соединений
		protected      int // Текущее количество защищенных соединений
		total          int // Общее количество соединений
	}
	limitsMutex sync.RWMutex

	// Автопереподключение к защищенным пирам
	reconnectManager struct {
		enabled     bool
		interval    time.Duration
		maxAttempts int
		attempts    map[peer.ID]int
	}
	reconnectMutex sync.RWMutex

	// EventManager для управления событиями
	eventManager *EventManager

	// StreamHandler для обработки стримов и чата
	streamHandler *StreamHandler
}

// NewNode создает новый libp2p узел (для обратной совместимости)
func NewNode(ctx context.Context) (*Node, error) {
	// Создаем менеджер персистентности
	persistence, err := NewPersistenceManager()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать менеджер персистентности: %w", err)
	}

	// Загружаем или создаем ключ идентичности
	privKey, err := persistence.LoadOrCreateIdentity()
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить/создать ключ идентичности: %w", err)
	}

	return NewNodeWithKey(ctx, privKey, persistence)
}

// NewNodeWithKeyBytes создает новый libp2p узел с переданными байтами ключа
func NewNodeWithKeyBytes(ctx context.Context, keyBytes []byte, persistence *PersistenceManager) (*Node, error) {
	// Десериализуем ключ из байтов
	privKey, err := crypto.UnmarshalPrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("не удалось десериализовать ключ из байтов: %w", err)
	}

	return NewNodeWithKey(ctx, privKey, persistence)
}

// NewNodeWithKey создает новый libp2p узел с переданным ключом
func NewNodeWithKey(ctx context.Context, privKey crypto.PrivKey, persistence *PersistenceManager) (*Node, error) {
	return NewNodeWithKeyAndConfig(ctx, privKey, persistence, DefaultNodeConfig())
}

// NewNodeWithKeyAndConfig создает новый libp2p узел с переданным ключом и конфигурацией
func NewNodeWithKeyAndConfig(ctx context.Context, privKey crypto.PrivKey, persistence *PersistenceManager, config *NodeConfig) (*Node, error) {
	// Получаем PeerID из ключа
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить PeerID из ключа: %w", err)
	}

	Info("🔑 Создаем узел с ключом для PeerID: %s", peerID.String())

	// Создаем опции libp2p на основе конфигурации
	opts := buildLibp2pOptions(privKey, config)

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать libp2p узел: %w", err)
	}

	// Создаем канал для сообщений
	messagesChan := make(chan RawMessage, 100)

	node := &Node{
		host:           h,
		ctx:            ctx,
		messagesChan:   messagesChan,
		peers:          make(map[peer.ID]bool), // 🔧 ИНИЦИАЛИЗАЦИЯ MAP!
		persistence:    persistence,
		protectedPeers: make(map[peer.ID]bool),
		connManager:    h.ConnManager(),
		eventManager:   NewEventManager(1000), // Очередь на 1000 событий
		streamHandler:  NewStreamHandler(h, PROTOCOL_ID, config),
	}

	// Инициализируем менеджер автопереподключения
	node.reconnectManager.enabled = true
	node.reconnectManager.interval = RECONNECT_INTERVAL
	node.reconnectManager.maxAttempts = MAX_RECONNECT_ATTEMPTS
	node.reconnectManager.attempts = make(map[peer.ID]int)

	// StreamHandler уже инициализирован в структуре Node

	// Добавляем логирование сетевых событий
	h.Network().Notify(&NetworkEventLogger{node: node})

	// Создаем DiscoveryManager
	discovery, err := NewDiscoveryManager(ctx, h, func(pi peer.AddrInfo) {
		node.AddPeer(pi.ID)
	}, node.eventManager)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать DiscoveryManager: %w", err)
	}

	node.discovery = discovery

	return node, nil
}

// Start запускает узел
func (n *Node) Start() error {
	// Запускаем DiscoveryManager
	if n.discovery != nil {
		if err := n.discovery.Start(); err != nil {
			return fmt.Errorf("не удалось запустить DiscoveryManager: %w", err)
		}
	}

	Info("🚀 Узел запущен")
	return nil
}

// Stop останавливает узел
func (n *Node) Stop() error {
	// Сохраняем DHT routing table перед остановкой
	if err := n.SaveDHTRoutingTable(); err != nil {
		Warn("⚠️ Не удалось сохранить DHT routing table: %v", err)
	}

	// Останавливаем DiscoveryManager
	if n.discovery != nil {
		if err := n.discovery.Stop(); err != nil {
			Warn("⚠️ Ошибка остановки DiscoveryManager: %v", err)
		}
	}

	// Останавливаем EventManager
	if n.eventManager != nil {
		n.eventManager.Stop()
	}

	if err := n.host.Close(); err != nil {
		return fmt.Errorf("ошибка остановки узла: %w", err)
	}
	close(n.messagesChan)
	Info("🛑 Узел остановлен")
	return nil
}

// GetHost возвращает host.Host для внутреннего использования
func (n *Node) GetHost() host.Host {
	return n.host
}

// GetEventManager возвращает EventManager для управления событиями
func (n *Node) GetEventManager() *EventManager {
	return n.eventManager
}

// GetMyID возвращает ID текущего узла
func (n *Node) GetMyID() string {
	return n.host.ID().String()
}

// GetConnectedPeers возвращает список подключенных пиров
func (n *Node) GetConnectedPeers() []peer.ID {
	n.peersMutex.RLock()
	defer n.peersMutex.RUnlock()

	peers := make([]peer.ID, 0, len(n.peers))
	for peerID := range n.peers {
		peers = append(peers, peerID)
	}
	return peers
}

// IsConnected проверяет, подключен ли пир
func (n *Node) IsConnected(peerID peer.ID) bool {
	n.peersMutex.RLock()
	defer n.peersMutex.RUnlock()

	return n.peers[peerID]
}

// AddPeer добавляет пира в список
func (n *Node) AddPeer(peerID peer.ID) {
	n.peersMutex.Lock()
	defer n.peersMutex.Unlock()

	// Проверяем, можно ли добавить соединение
	if !n.canAddInfrastructureConnection() {
		Warn("⚠️ Достигнут лимит инфраструктурных соединений для пира %s", peerID.ShortString())
		return
	}

	n.peers[peerID] = true

	// Увеличиваем счетчик инфраструктурных соединений
	if n.addInfrastructureConnection() {
		Info("🔗 Добавлен инфраструктурный пир %s (всего: %d/%d)",
			peerID.ShortString(), n.connectionLimits.infrastructure, MAX_INFRASTRUCTURE_CONNECTIONS)
	}

	// Сохраняем пира в кэш
	go func() {
		addrs := n.host.Peerstore().Addrs(peerID)
		var addrStrings []string
		for _, addr := range addrs {
			addrStrings = append(addrStrings, addr.String())
		}

		// Определяем, является ли пир "здоровым" (есть адреса)
		healthy := len(addrStrings) > 0

		if err := n.SavePeerToCache(peerID, addrStrings, healthy); err != nil {
			Warn("⚠️ Не удалось сохранить пира %s в кэш: %v", peerID.ShortString(), err)
		} else {
			Info("💾 Пир %s сохранен в кэш", peerID.ShortString())
		}
	}()
}

// RemovePeer удаляет пира из списка
func (n *Node) RemovePeer(peerID peer.ID) {
	n.peersMutex.Lock()
	defer n.peersMutex.Unlock()

	// Проверяем, был ли пир в списке
	if n.peers[peerID] {
		delete(n.peers, peerID)

		// Уменьшаем счетчик инфраструктурных соединений
		n.removeInfrastructureConnection()

		Info("🔌 Удален инфраструктурный пир %s (осталось: %d/%d)",
			peerID.ShortString(), n.connectionLimits.infrastructure, MAX_INFRASTRUCTURE_CONNECTIONS)
	}
}

// AddProtectedPeer добавляет пира в список защищенных
func (n *Node) AddProtectedPeer(peerID peer.ID) {
	n.protectedPeersMutex.Lock()
	defer n.protectedPeersMutex.Unlock()

	// Проверяем, можно ли добавить защищенное соединение
	if !n.canAddProtectedConnection() {
		Warn("⚠️ Достигнут лимит защищенных соединений для пира %s", peerID.ShortString())
		return
	}

	n.protectedPeers[peerID] = true

	// Защищаем соединение с этим пиром
	if n.connManager != nil {
		n.connManager.Protect(peerID, "owl-whisper-protected")
	}

	// Увеличиваем счетчик защищенных соединений
	if n.addProtectedConnection() {
		Info("🔒 Пир %s добавлен в защищенные (всего: %d/%d)",
			peerID.ShortString(), n.connectionLimits.protected, MAX_PROTECTED_CONNECTIONS)
	}

	// Сохраняем защищенного пира в кэш как "здорового"
	go func() {
		addrs := n.host.Peerstore().Addrs(peerID)
		var addrStrings []string
		for _, addr := range addrs {
			addrStrings = append(addrStrings, addr.String())
		}

		// Защищенные пиры всегда считаются "здоровыми"
		if err := n.SavePeerToCache(peerID, addrStrings, true); err != nil {
			Warn("⚠️ Не удалось сохранить защищенного пира %s в кэш: %v", peerID.ShortString(), err)
		} else {
			Info("💾 Защищенный пир %s сохранен в кэш", peerID.ShortString())
		}
	}()
}

// RemoveProtectedPeer удаляет пира из списка защищенных
func (n *Node) RemoveProtectedPeer(peerID peer.ID) {
	n.protectedPeersMutex.Lock()
	defer n.protectedPeersMutex.Unlock()

	// Проверяем, был ли пир в списке
	if n.protectedPeers[peerID] {
		delete(n.protectedPeers, peerID)

		// Снимаем защиту с соединения
		if n.connManager != nil {
			n.connManager.Unprotect(peerID, "owl-whisper-protected")
		}

		// Уменьшаем счетчик защищенных соединений
		n.removeProtectedConnection()

		Info("🔓 Пир %s удален из защищенных (осталось: %d/%d)",
			peerID.ShortString(), n.connectionLimits.protected, MAX_PROTECTED_CONNECTIONS)
	}
}

// IsProtectedPeer проверяет, является ли пир защищенным
func (n *Node) IsProtectedPeer(peerID peer.ID) bool {
	n.protectedPeersMutex.RLock()
	defer n.protectedPeersMutex.RUnlock()

	return n.protectedPeers[peerID]
}

// GetProtectedPeers возвращает список защищенных пиров
func (n *Node) GetProtectedPeers() []peer.ID {
	n.protectedPeersMutex.RLock()
	defer n.protectedPeersMutex.RUnlock()

	peers := make([]peer.ID, 0, len(n.protectedPeers))
	for peerID := range n.protectedPeers {
		peers = append(peers, peerID)
	}
	return peers
}

// GetConnectionLimits возвращает текущие лимиты соединений
func (n *Node) GetConnectionLimits() map[string]interface{} {
	n.limitsMutex.RLock()
	defer n.limitsMutex.RUnlock()

	return map[string]interface{}{
		"infrastructure": map[string]interface{}{
			"current": n.connectionLimits.infrastructure,
			"max":     MAX_INFRASTRUCTURE_CONNECTIONS,
		},
		"protected": map[string]interface{}{
			"current": n.connectionLimits.protected,
			"max":     MAX_PROTECTED_CONNECTIONS,
		},
		"total": map[string]interface{}{
			"current": n.connectionLimits.total,
			"max":     MAX_TOTAL_CONNECTIONS,
		},
	}
}

// canAddInfrastructureConnection проверяет, можно ли добавить инфраструктурное соединение
func (n *Node) canAddInfrastructureConnection() bool {
	n.limitsMutex.RLock()
	defer n.limitsMutex.RUnlock()

	return n.connectionLimits.infrastructure < MAX_INFRASTRUCTURE_CONNECTIONS &&
		n.connectionLimits.total < MAX_TOTAL_CONNECTIONS
}

// canAddProtectedConnection проверяет, можно ли добавить защищенное соединение
func (n *Node) canAddProtectedConnection() bool {
	n.limitsMutex.RLock()
	defer n.limitsMutex.RUnlock()

	return n.connectionLimits.protected < MAX_PROTECTED_CONNECTIONS &&
		n.connectionLimits.total < MAX_TOTAL_CONNECTIONS
}

// addInfrastructureConnection добавляет инфраструктурное соединение
func (n *Node) addInfrastructureConnection() bool {
	n.limitsMutex.Lock()
	defer n.limitsMutex.Unlock()

	if n.connectionLimits.infrastructure < MAX_INFRASTRUCTURE_CONNECTIONS &&
		n.connectionLimits.total < MAX_TOTAL_CONNECTIONS {
		n.connectionLimits.infrastructure++
		n.connectionLimits.total++
		return true
	}
	return false
}

// removeInfrastructureConnection удаляет инфраструктурное соединение
func (n *Node) removeInfrastructureConnection() {
	n.limitsMutex.Lock()
	defer n.limitsMutex.Unlock()

	if n.connectionLimits.infrastructure > 0 {
		n.connectionLimits.infrastructure--
	}
	if n.connectionLimits.total > 0 {
		n.connectionLimits.total--
	}
}

// addProtectedConnection добавляет защищенное соединение
func (n *Node) addProtectedConnection() bool {
	n.limitsMutex.Lock()
	defer n.limitsMutex.Unlock()

	if n.connectionLimits.protected < MAX_PROTECTED_CONNECTIONS &&
		n.connectionLimits.total < MAX_TOTAL_CONNECTIONS {
		n.connectionLimits.protected++
		n.connectionLimits.total++
		return true
	}
	return false
}

// removeProtectedConnection удаляет защищенное соединение
func (n *Node) removeProtectedConnection() {
	n.limitsMutex.Lock()
	defer n.limitsMutex.Unlock()

	if n.connectionLimits.protected > 0 {
		n.connectionLimits.protected--
	}
	if n.connectionLimits.total > 0 {
		n.connectionLimits.total--
	}
}

// EnableAutoReconnect включает автопереподключение к защищенным пирам
func (n *Node) EnableAutoReconnect() {
	n.reconnectMutex.Lock()
	defer n.reconnectMutex.Unlock()

	n.reconnectManager.enabled = true
	Info("🔄 Автопереподключение к защищенным пирам включено")
}

// DisableAutoReconnect отключает автопереподключение к защищенным пирам
func (n *Node) DisableAutoReconnect() {
	n.reconnectMutex.Lock()
	defer n.reconnectMutex.Unlock()

	n.reconnectManager.enabled = false
	Info("⏸️ Автопереподключение к защищенным пирам отключено")
}

// IsAutoReconnectEnabled проверяет, включено ли автопереподключение
func (n *Node) IsAutoReconnectEnabled() bool {
	n.reconnectMutex.RLock()
	defer n.reconnectMutex.RUnlock()

	return n.reconnectManager.enabled
}

// GetReconnectAttempts возвращает количество попыток переподключения для пира
func (n *Node) GetReconnectAttempts(peerID peer.ID) int {
	n.reconnectMutex.RLock()
	defer n.reconnectMutex.RUnlock()

	return n.reconnectManager.attempts[peerID]
}

// SavePeerToCache сохраняет пира в кэш
func (n *Node) SavePeerToCache(peerID peer.ID, addresses []string, healthy bool) error {
	if n.persistence == nil {
		return fmt.Errorf("PersistenceManager недоступен")
	}
	return n.persistence.SavePeerToCache(peerID, addresses, healthy)
}

// LoadPeerFromCache загружает пира из кэша
func (n *Node) LoadPeerFromCache(peerID peer.ID) (*PeerCacheEntry, error) {
	if n.persistence == nil {
		return nil, fmt.Errorf("PersistenceManager недоступен")
	}
	return n.persistence.LoadPeerFromCache(peerID)
}

// GetAllCachedPeers возвращает всех кэшированных пиров
func (n *Node) GetAllCachedPeers() ([]PeerCacheEntry, error) {
	if n.persistence == nil {
		return nil, fmt.Errorf("PersistenceManager недоступен")
	}
	return n.persistence.GetAllCachedPeers()
}

// GetHealthyCachedPeers возвращает только "здоровых" кэшированных пиров
func (n *Node) GetHealthyCachedPeers() ([]PeerCacheEntry, error) {
	if n.persistence == nil {
		return nil, fmt.Errorf("PersistenceManager недоступен")
	}
	return n.persistence.GetHealthyCachedPeers()
}

// RemovePeerFromCache удаляет пира из кэша
func (n *Node) RemovePeerFromCache(peerID peer.ID) error {
	if n.persistence == nil {
		return fmt.Errorf("PersistenceManager недоступен")
	}
	return n.persistence.RemovePeerFromCache(peerID)
}

// ClearPeerCache очищает весь кэш пиров
func (n *Node) ClearPeerCache() error {
	if n.persistence == nil {
		return fmt.Errorf("PersistenceManager недоступен")
	}
	return n.persistence.ClearPeerCache()
}

// SaveDHTRoutingTable сохраняет DHT routing table в кэш
func (n *Node) SaveDHTRoutingTable() error {
	if n.discovery == nil {
		return fmt.Errorf("DiscoveryManager недоступен")
	}
	return n.discovery.SaveDHTRoutingTable(n.persistence)
}

// LoadDHTRoutingTableFromCache загружает DHT routing table из кэша
func (n *Node) LoadDHTRoutingTableFromCache() error {
	if n.discovery == nil {
		return fmt.Errorf("DiscoveryManager недоступен")
	}
	return n.discovery.LoadDHTRoutingTableFromCache(n.persistence)
}

// GetRoutingTableStats возвращает статистику DHT routing table
func (n *Node) GetRoutingTableStats() map[string]interface{} {
	if n.discovery == nil {
		return map[string]interface{}{
			"status": "discovery_unavailable",
		}
	}
	return n.discovery.GetRoutingTableStats()
}

// startReconnectLoop запускает цикл автопереподключения
func (n *Node) startReconnectLoop() {
	go func() {
		ticker := time.NewTicker(n.reconnectManager.interval)
		defer ticker.Stop()

		for {
			select {
			case <-n.ctx.Done():
				return
			case <-ticker.C:
				n.reconnectProtectedPeers()
			}
		}
	}()
}

// reconnectProtectedPeers пытается переподключиться к отключенным защищенным пирам
func (n *Node) reconnectProtectedPeers() {
	n.reconnectMutex.RLock()
	enabled := n.reconnectManager.enabled
	n.reconnectMutex.RUnlock()

	if !enabled {
		return
	}

	// Получаем список защищенных пиров
	protectedPeers := n.GetProtectedPeers()

	for _, peerID := range protectedPeers {
		// Проверяем, подключен ли пир
		if !n.IsConnected(peerID) {
			n.attemptReconnect(peerID)
		}
	}
}

// attemptReconnect пытается переподключиться к конкретному пиру
func (n *Node) attemptReconnect(peerID peer.ID) {
	n.reconnectMutex.Lock()
	attempts := n.reconnectManager.attempts[peerID]
	maxAttempts := n.reconnectManager.maxAttempts
	n.reconnectMutex.Unlock()

	if attempts >= maxAttempts {
		Warn("⚠️ Превышен лимит попыток переподключения к пиру %s (%d/%d)",
			peerID.ShortString(), attempts, maxAttempts)
		return
	}

	Info("🔄 Попытка переподключения к защищенному пиру %s (%d/%d)",
		peerID.ShortString(), attempts+1, maxAttempts)

	// Здесь должна быть логика переподключения через libp2p
	// Пока просто увеличиваем счетчик попыток
	n.reconnectMutex.Lock()
	n.reconnectManager.attempts[peerID]++
	n.reconnectMutex.Unlock()

	// TODO: Реализовать реальное переподключение через host.Connect()
	// Для этого нужно сохранять адреса защищенных пиров
}

// Send отправляет данные конкретному пиру
func (n *Node) Send(peerID peer.ID, data []byte) error {
	if n.streamHandler == nil {
		return fmt.Errorf("StreamHandler недоступен")
	}
	return n.streamHandler.Send(peerID, data)
}

// GetStreamHandler возвращает StreamHandler для работы со стримами
func (n *Node) GetStreamHandler() *StreamHandler {
	return n.streamHandler
}

// CreateStream создает исходящий стрим к пиру
func (n *Node) CreateStream(ctx context.Context, peerID peer.ID, timeout time.Duration) (network.Stream, error) {
	if n.streamHandler == nil {
		return nil, fmt.Errorf("StreamHandler недоступен")
	}
	return n.streamHandler.CreateStream(ctx, peerID, timeout)
}

// CreateStreamWithRetry создает стрим с повторными попытками
func (n *Node) CreateStreamWithRetry(ctx context.Context, peerID peer.ID, timeout time.Duration, maxRetries int) (network.Stream, error) {
	if n.streamHandler == nil {
		return nil, fmt.Errorf("StreamHandler недоступен")
	}
	return n.streamHandler.CreateStreamWithRetry(ctx, peerID, timeout, maxRetries)
}

// SetMessageCallback устанавливает callback для входящих сообщений
func (n *Node) SetMessageCallback(callback func(peer.ID, []byte)) {
	if n.streamHandler != nil {
		n.streamHandler.SetMessageCallback(callback)
	}
}

// SetStreamOpenCallback устанавливает callback для открытия стримов
func (n *Node) SetStreamOpenCallback(callback func(peer.ID, network.Stream)) {
	if n.streamHandler != nil {
		n.streamHandler.SetStreamOpenCallback(callback)
	}
}

// SetStreamCloseCallback устанавливает callback для закрытия стримов
func (n *Node) SetStreamCloseCallback(callback func(peer.ID)) {
	if n.streamHandler != nil {
		n.streamHandler.SetStreamCloseCallback(callback)
	}
}

// CreateStreamWithDefaultTimeout создает стрим с дефолтным таймаутом из конфига
func (n *Node) CreateStreamWithDefaultTimeout(ctx context.Context, peerID peer.ID) (network.Stream, error) {
	if n.streamHandler == nil {
		return nil, fmt.Errorf("StreamHandler недоступен")
	}
	return n.streamHandler.CreateStream(ctx, peerID, 0) // 0 = использовать дефолт из конфига
}

// GetStreamTimeouts возвращает текущие настройки таймаутов для стримов
func (n *Node) GetStreamTimeouts() map[string]time.Duration {
	if n.streamHandler == nil || n.streamHandler.config == nil {
		return map[string]time.Duration{
			"creation": 60 * time.Second,
			"read":     30 * time.Second,
			"write":    10 * time.Second,
		}
	}

	config := n.streamHandler.config
	return map[string]time.Duration{
		"creation": config.StreamCreationTimeout,
		"read":     config.StreamReadTimeout,
		"write":    config.StreamWriteTimeout,
	}
}

// Broadcast отправляет данные всем подключенным пирам
func (n *Node) Broadcast(data []byte) error {
	peers := n.GetConnectedPeers()
	if len(peers) == 0 {
		Warn("⚠️ Нет подключенных пиров для broadcast")
		return nil
	}

	var lastError error
	for _, peerID := range peers {
		if err := n.Send(peerID, data); err != nil {
			Error("❌ Ошибка отправки к %s: %v", peerID.ShortString(), err)
			lastError = err
		}
	}

	return lastError
}

// Messages возвращает канал для получения входящих сообщений
func (n *Node) Messages() <-chan RawMessage {
	return n.messagesChan
}

// handleStream обрабатывает входящие потоки
func (n *Node) handleStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	Info("📥 Получен поток от %s", remotePeer.ShortString())

	// Добавляем пира в список
	n.AddPeer(remotePeer)

	// Читаем данные из потока
	buffer := make([]byte, 1024)
	bytesRead, err := stream.Read(buffer)
	if err != nil {
		Error("❌ Ошибка чтения потока от %s: %v", remotePeer.ShortString(), err)
		stream.Close()
		return
	}

	// Создаем RawMessage
	message := RawMessage{
		SenderID: remotePeer,
		Data:     buffer[:bytesRead],
	}

	// Отправляем в канал сообщений
	select {
	case n.messagesChan <- message:
		Info("📨 Сообщение от %s добавлено в очередь", remotePeer.ShortString())
	default:
		Warn("⚠️ Канал сообщений переполнен, сообщение от %s потеряно", remotePeer.ShortString())
	}

	// Отправляем событие в EventManager
	if n.eventManager != nil {
		event := NewMessageEvent(remotePeer.String(), buffer[:bytesRead])
		if err := n.eventManager.PushEvent(event); err != nil {
			Warn("⚠️ Не удалось отправить событие NewMessage: %v", err)
		}
	}

	stream.Close()
}
