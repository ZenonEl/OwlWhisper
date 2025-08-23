package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ICoreController - это публичный интерфейс для всего Core слоя
type ICoreController interface {
	// Start запускает Core контроллер
	Start() error

	// Stop останавливает Core контроллер
	Stop() error

	// Broadcast отправляет данные всем подключенным пирам
	Broadcast(data []byte) error

	// Send отправляет данные конкретному пиру
	Send(peerID peer.ID, data []byte) error

	// GetMyID возвращает ID текущего узла
	GetMyID() string

	// GetConnectedPeers возвращает список подключенных пиров
	GetConnectedPeers() []peer.ID

	// GetProtectedPeers возвращает список защищенных пиров
	GetProtectedPeers() []peer.ID

	// AddProtectedPeer добавляет пира в защищенные
	AddProtectedPeer(peerID peer.ID) error

	// RemoveProtectedPeer удаляет пира из защищенных
	RemoveProtectedPeer(peerID peer.ID) error

	// IsProtectedPeer проверяет, является ли пир защищенным
	IsProtectedPeer(peerID peer.ID) bool

	// GetConnectionLimits возвращает текущие лимиты соединений
	GetConnectionLimits() map[string]interface{}

	// Автопереподключение к защищенным пирам
	EnableAutoReconnect()
	DisableAutoReconnect()
	IsAutoReconnectEnabled() bool
	GetReconnectAttempts(peerID peer.ID) int

	// GetNetworkStats возвращает статистику сети для отладки
	GetNetworkStats() map[string]interface{}

	// FindPeer ищет пира в сети по PeerID
	FindPeer(peerID peer.ID) (*peer.AddrInfo, error)

	// GetConnectionQuality возвращает качество соединения с пиром
	GetConnectionQuality(peerID peer.ID) map[string]interface{}

	// Messages возвращает канал для получения ВСЕХ входящих данных
	Messages() <-chan RawMessage

	// GetHost возвращает узел
	GetHost() host.Host

	// Новые методы для работы с профилями

}

// CoreController реализует ICoreController интерфейс
type CoreController struct {
	node      *Node
	discovery *DiscoveryManager

	ctx    context.Context
	cancel context.CancelFunc

	// Мьютекс для безопасного доступа
	mu sync.RWMutex

	// Статус работы
	running bool
}

// NewCoreController создает новый Core контроллер (для обратной совместимости)
func NewCoreController(ctx context.Context) (*CoreController, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Создаем Node
	node, err := NewNode(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать Node: %w", err)
	}

	return createControllerFromNode(ctx, cancel, node)
}

// NewCoreControllerWithKey создает новый Core контроллер с переданным ключом
func NewCoreControllerWithKey(ctx context.Context, privKey crypto.PrivKey) (*CoreController, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Создаем PersistenceManager
	persistence, err := NewPersistenceManager()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать PersistenceManager: %w", err)
	}

	// Создаем Node с переданным ключом
	node, err := NewNodeWithKey(ctx, privKey, persistence)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать Node с ключом: %w", err)
	}

	return createControllerFromNode(ctx, cancel, node)
}

// NewCoreControllerWithKeyBytes создает новый Core контроллер с переданными байтами ключа
func NewCoreControllerWithKeyBytes(ctx context.Context, keyBytes []byte) (*CoreController, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Создаем PersistenceManager
	persistence, err := NewPersistenceManager()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать PersistenceManager: %w", err)
	}

	// Создаем Node с переданными байтами ключа
	node, err := NewNodeWithKeyBytes(ctx, keyBytes, persistence)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать Node с байтами ключа: %w", err)
	}

	return createControllerFromNode(ctx, cancel, node)
}

// createControllerFromNode создает контроллер из готового узла
func createControllerFromNode(ctx context.Context, cancel context.CancelFunc, node *Node) (*CoreController, error) {
	// Создаем DiscoveryManager с callback для новых пиров
	discovery, err := NewDiscoveryManager(ctx, node.GetHost(), func(pi peer.AddrInfo) {
		// Когда найден новый пир, добавляем его в Node
		node.AddPeer(pi.ID)
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать DiscoveryManager: %w", err)
	}

	controller := &CoreController{
		node:      node,
		discovery: discovery,
		ctx:       ctx,
		cancel:    cancel,
	}

	return controller, nil
}

// Start запускает Core контроллер
func (c *CoreController) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("контроллер уже запущен")
	}

	// Запускаем Node
	if err := c.node.Start(); err != nil {
		return fmt.Errorf("не удалось запустить Node: %w", err)
	}

	// Запускаем Discovery
	if err := c.discovery.Start(); err != nil {
		return fmt.Errorf("не удалось запустить Discovery: %w", err)
	}

	c.running = true
	Info("🚀 Core контроллер запущен")

	return nil
}

// Stop останавливает Core контроллер
func (c *CoreController) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	// Останавливаем Discovery
	if err := c.discovery.Stop(); err != nil {
		Warn("⚠️ Ошибка остановки Discovery: %v", err)
	}

	// Останавливаем Node
	if err := c.node.Stop(); err != nil {
		Warn("⚠️ Ошибка остановки Discovery: %v", err)
	}

	// Отменяем контекст
	c.cancel()

	c.running = false
	Info("🛑 Core контроллер остановлен")

	return nil
}

// Broadcast отправляет данные всем подключенным пирам
func (c *CoreController) Broadcast(data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return fmt.Errorf("контроллер не запущен")
	}

	return c.node.Broadcast(data)
}

// Send отправляет данные конкретному пиру
func (c *CoreController) Send(peerID peer.ID, data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return fmt.Errorf("контроллер не запущен")
	}

	return c.node.Send(peerID, data)
}

// GetMyID возвращает ID текущего узла
func (c *CoreController) GetMyID() string {
	return c.node.GetMyID()
}

// GetConnectedPeers возвращает список подключенных пиров
func (c *CoreController) GetConnectedPeers() []peer.ID {
	return c.node.GetConnectedPeers()
}

// GetProtectedPeers возвращает список защищенных пиров
func (c *CoreController) GetProtectedPeers() []peer.ID {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return nil
	}

	return c.node.GetProtectedPeers()
}

// AddProtectedPeer добавляет пира в защищенные
func (c *CoreController) AddProtectedPeer(peerID peer.ID) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return fmt.Errorf("контроллер не запущен")
	}

	c.node.AddProtectedPeer(peerID)
	return nil
}

// RemoveProtectedPeer удаляет пира из защищенных
func (c *CoreController) RemoveProtectedPeer(peerID peer.ID) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return fmt.Errorf("контроллер не запущен")
	}

	if !c.node.IsProtectedPeer(peerID) {
		return fmt.Errorf("пир %s не является защищенным", peerID.ShortString())
	}

	c.node.RemoveProtectedPeer(peerID)
	return nil
}

// IsProtectedPeer проверяет, является ли пир защищенным
func (c *CoreController) IsProtectedPeer(peerID peer.ID) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return false
	}

	return c.node.IsProtectedPeer(peerID)
}

// GetConnectionLimits возвращает текущие лимиты соединений
func (c *CoreController) GetConnectionLimits() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return map[string]interface{}{
			"status": "not_running",
		}
	}

	return c.node.GetConnectionLimits()
}

// EnableAutoReconnect включает автопереподключение к защищенным пирам
func (c *CoreController) EnableAutoReconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	c.node.EnableAutoReconnect()
}

// DisableAutoReconnect отключает автопереподключение к защищенным пирам
func (c *CoreController) DisableAutoReconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	c.node.DisableAutoReconnect()
}

// IsAutoReconnectEnabled проверяет, включено ли автопереподключение
func (c *CoreController) IsAutoReconnectEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return false
	}

	return c.node.IsAutoReconnectEnabled()
}

// GetReconnectAttempts возвращает количество попыток переподключения для пира
func (c *CoreController) GetReconnectAttempts(peerID peer.ID) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return 0
	}

	return c.node.GetReconnectAttempts(peerID)
}

// GetNetworkStats возвращает статистику сети для отладки
func (c *CoreController) GetNetworkStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return map[string]interface{}{
			"status": "not_running",
		}
	}

	host := c.node.GetHost()
	if host == nil {
		return map[string]interface{}{
			"status": "no_host",
		}
	}

	// Получаем статистику из libp2p
	network := host.Network()
	peers := network.Peers()
	connections := network.Conns()

	// Подсчитываем активные соединения по протоколам
	protocolStats := make(map[string]int)
	for _, conn := range connections {
		for _, stream := range conn.GetStreams() {
			protocol := string(stream.Protocol())
			protocolStats[protocol]++
		}
	}

	// Получаем информацию о DHT
	dhtStats := map[string]interface{}{
		"status": "unknown",
	}
	if c.discovery != nil {
		// TODO: Добавить реальную статистику DHT
		dhtStats["status"] = "active"
	}

	stats := map[string]interface{}{
		"status":            "running",
		"total_peers":       len(peers),
		"connected_peers":   len(c.node.GetConnectedPeers()),
		"total_connections": len(connections),
		"protocols":         protocolStats,
		"dht":               dhtStats,
		"my_peer_id":        c.GetMyID(),
		"listening_addrs":   host.Addrs(),
	}

	return stats
}

// FindPeer ищет пира в сети по PeerID
func (c *CoreController) FindPeer(peerID peer.ID) (*peer.AddrInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return nil, fmt.Errorf("контроллер не запущен")
	}

	// Сначала проверяем, подключены ли мы уже к этому пиру
	if c.node.IsConnected(peerID) {
		host := c.node.GetHost()
		addrs := host.Peerstore().Addrs(peerID)
		return &peer.AddrInfo{
			ID:    peerID,
			Addrs: addrs,
		}, nil
	}

	// Если не подключены, ищем через DHT
	if c.discovery != nil {
		// Получаем DHT из discovery manager
		dht := c.discovery.GetDHT()
		if dht == nil {
			return nil, fmt.Errorf("DHT недоступен")
		}

		// Создаем контекст с таймаутом для DHT поиска
		// 30 секунд - разумное значение для публичной DHT
		findCtx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
		defer cancel()

		// Ищем пира через DHT
		addrInfo, err := dht.FindPeer(findCtx, peerID)
		if err != nil {
			// Проверяем, является ли это ошибкой "не найден"
			if err.Error() == "routing: not found" {
				return nil, fmt.Errorf("пир %s не найден в DHT (вероятно, офлайн)", peerID.ShortString())
			}
			return nil, fmt.Errorf("ошибка при поиске в DHT: %w", err)
		}

		Info("SUCCESS: Пир %s успешно найден в DHT", addrInfo.ID.ShortString())
		return &addrInfo, nil
	}

	return nil, fmt.Errorf("discovery manager не доступен")
}

// GetConnectionQuality возвращает качество соединения с пиром
func (c *CoreController) GetConnectionQuality(peerID peer.ID) map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return map[string]interface{}{
			"status": "not_running",
		}
	}

	// Проверяем, подключены ли мы к этому пиру
	if !c.node.IsConnected(peerID) {
		return map[string]interface{}{
			"status": "not_connected",
		}
	}

	host := c.node.GetHost()
	if host == nil {
		return map[string]interface{}{
			"status": "no_host",
		}
	}

	// Получаем информацию о соединении
	network := host.Network()
	connections := network.ConnsToPeer(peerID)

	if len(connections) == 0 {
		return map[string]interface{}{
			"status": "no_connections",
		}
	}

	// Анализируем качество соединения
	var totalStreams int
	var activeStreams int
	protocols := make(map[string]int)

	for _, conn := range connections {
		streams := conn.GetStreams()
		totalStreams += len(streams)

		for _, stream := range streams {
			protocol := string(stream.Protocol())
			protocols[protocol]++

			// Проверяем, активен ли стрим
			if !stream.Stat().Opened.IsZero() {
				activeStreams++
			}
		}
	}

	// Получаем адреса пира
	addrs := host.Peerstore().Addrs(peerID)

	quality := map[string]interface{}{
		"status":            "connected",
		"peer_id":           peerID.String(),
		"total_connections": len(connections),
		"total_streams":     totalStreams,
		"active_streams":    activeStreams,
		"protocols":         protocols,
		"addresses":         addrs,
		"latency_ms":        -1, // TODO: Реализовать измерение латентности
	}

	return quality
}

// Messages возвращает канал для получения входящих сообщений
func (c *CoreController) Messages() <-chan RawMessage {
	return c.node.Messages()
}

// GetHost возвращает узел
func (c *CoreController) GetHost() host.Host {
	return c.node.GetHost()
}

// IsRunning проверяет, запущен ли контроллер
func (c *CoreController) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.running
}

// IsConnected проверяет, подключен ли указанный пир
func (c *CoreController) IsConnected(peerID peer.ID) bool {
	return c.node.IsConnected(peerID)
}
