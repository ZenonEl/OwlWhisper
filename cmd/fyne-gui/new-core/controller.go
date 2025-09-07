// cmd/fyne-gui/new-core/controller.go

package newcore

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	// PROTOCOL_ID - уникальный идентификатор нашего чат-протокола.
	PROTOCOL_ID = "/owl-whisper/1.0.0"
)

// --- Структуры Событий (Контракт с GUI) ---

// Event - это универсальная структура для всех асинхронных событий,
// отправляемых из Core в GUI.
type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// NewMessagePayload содержит данные для события "NewMessage".
type NewMessagePayload struct {
	SenderID string `json:"sender_id"`
	Data     []byte `json:"data"`
}

// PeerStatusPayload содержит данные для событий "PeerConnected" и "PeerDisconnected".
type PeerStatusPayload struct {
	PeerID string `json:"peer_id"`
}

// --- Интерфейс Контроллера (Публичный API) ---

// ICoreController определяет публичный API для управления P2P-узлом.
// Весь остальной код (GUI, TUI) будет работать только с этим интерфейсом.
type ICoreController interface {
	Start() error
	Stop() error
	GetMyPeerID() string
	GetConnectedPeers() []string
	SendDataToPeer(peerID string, data []byte) error
	BroadcastData(data []byte) error
	FindPeer(peerID string) (*peer.AddrInfo, error)
	FindProvidersForContent(contentID string) ([]peer.AddrInfo, error)
	ProvideContent(contentID string) error
	GetDHTTableSize() int
	Events() <-chan Event
}

// --- Реализация Контроллера ---

// CoreController - это конкретная реализация ICoreController.
type CoreController struct {
	ctx            context.Context
	cancel         context.CancelFunc
	cfg            Config
	privKey        crypto.PrivKey
	node           *Node
	discovery      *DiscoveryManager
	eventChan      chan Event
	connectedPeers map[peer.ID]bool
	mu             sync.RWMutex
}

// NewCoreController - конструктор для нашего контроллера.
func NewCoreController(privKey crypto.PrivKey, cfg Config) (ICoreController, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &CoreController{
		ctx:            ctx,
		cancel:         cancel,
		cfg:            cfg,
		privKey:        privKey,
		eventChan:      make(chan Event, 100), // Буферизированный канал
		connectedPeers: make(map[peer.ID]bool),
	}, nil
}

func (c *CoreController) Start() error {
	var err error

	// 1. Создаем узел
	c.node, err = NewNode(c.ctx, c.privKey, c.cfg)
	if err != nil {
		return fmt.Errorf("ошибка создания узла: %w", err)
	}

	// 2. Регистрируем обработчик входящих потоков
	c.node.SetStreamHandler(PROTOCOL_ID, c.handleStream)
	// Регистрируем обработчик сетевых событий для отслеживания подключений
	c.node.Host().Network().Notify(c.newNetworkNotifee())

	// 3. Создаем и запускаем менеджер обнаружения
	c.discovery, err = NewDiscoveryManager(c.ctx, c.node.Host(), c.cfg, c.onPeerFound)
	if err != nil {
		return fmt.Errorf("ошибка создания DiscoveryManager: %w", err)
	}
	c.discovery.Start()

	log.Println("INFO: [Controller] Ядро успешно запущено.")
	return nil
}

func (c *CoreController) Stop() error {
	log.Println("INFO: [Controller] Остановка ядра...")
	c.cancel() // Отменяем контекст, чтобы остановить все горутины
	close(c.eventChan)
	return c.node.Close()
}

func (c *CoreController) GetMyPeerID() string {
	// ИСПРАВЛЕНО: Добавляем проверку, чтобы избежать паники
	if c.node == nil || c.node.Host() == nil {
		return ""
	}
	return c.node.Host().ID().String()
}

func (c *CoreController) GetConnectedPeers() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	peers := make([]string, 0, len(c.connectedPeers))
	for p := range c.connectedPeers {
		peers = append(peers, p.String())
	}
	return peers
}

func (c *CoreController) SendDataToPeer(peerIDStr string, data []byte) error {
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("неверный PeerID: %w", err)
	}

	stream, err := c.node.Host().NewStream(c.ctx, peerID, PROTOCOL_ID)
	if err != nil {
		return fmt.Errorf("не удалось открыть поток: %w", err)
	}
	defer stream.Close()

	_, err = stream.Write(data)
	return err
}

func (c *CoreController) BroadcastData(data []byte) error {
	peersStr := c.GetConnectedPeers()
	for _, pStr := range peersStr {
		// Запускаем отправку в горутинах, чтобы не блокировать
		go func(p string) {
			if err := c.SendDataToPeer(p, data); err != nil {
				log.Printf("WARN: Не удалось отправить broadcast-сообщение пиру %s: %v", p, err)
			}
		}(pStr)
	}
	return nil
}

func (c *CoreController) FindPeer(peerIDStr string) (*peer.AddrInfo, error) {
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return nil, fmt.Errorf("неверный PeerID: %w", err)
	}

	findCtx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	// ИСПРАВЛЕНО: dht.FindPeer возвращает peer.AddrInfo, а не *peer.AddrInfo
	addrInfo, err := c.discovery.DHT().FindPeer(findCtx, peerID)
	if err != nil {
		return nil, err // dht.FindPeer уже возвращает понятные ошибки
	}
	return &addrInfo, nil
}

func (c *CoreController) ProvideContent(contentID string) error {
	cid, err := cid.Decode(contentID)
	if err != nil {
		return err
	}
	provideCtx, cancel := context.WithTimeout(c.ctx, 60*time.Second)
	defer cancel()
	return c.discovery.DHT().Provide(provideCtx, cid, true)
}

func (c *CoreController) FindProvidersForContent(contentID string) ([]peer.AddrInfo, error) {
	cid, err := cid.Decode(contentID)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования CID: %w", err)
	}

	findCtx, cancel := context.WithTimeout(c.ctx, 60*time.Second)
	defer cancel()

	// ИСПРАВЛЕНО: Используем актуальную блокирующую функцию FindProviders,
	// которая сразу возвращает слайс.
	allProviders, err := c.discovery.DHT().FindProviders(findCtx, cid)
	if err != nil {
		return nil, fmt.Errorf("ошибка при поиске провайдеров в DHT: %w", err)
	}

	// Теперь allProviders - это []peer.AddrInfo, как и должно быть.
	// Фильтруем самих себя из списка.
	var filteredProviders []peer.AddrInfo
	myID := c.node.Host().ID()
	for _, p := range allProviders {
		if p.ID != myID {
			filteredProviders = append(filteredProviders, p)
		}
	}

	if len(filteredProviders) == 0 {
		// Это не ошибка, а нормальный результат, если никто не найден.
		return nil, fmt.Errorf("провайдеры для контента не найдены")
	}

	return filteredProviders, nil
}
func (c *CoreController) GetDHTTableSize() int {
	return c.discovery.DHT().RoutingTable().Size()
}

func (c *CoreController) Events() <-chan Event {
	return c.eventChan
}

// handleStream - это наш главный обработчик входящих сообщений.
func (c *CoreController) handleStream(stream network.Stream) {
	defer stream.Close()
	senderID := stream.Conn().RemotePeer()
	data, err := io.ReadAll(stream)
	if err != nil {
		log.Printf("ERROR: Не удалось прочитать данные из потока от %s: %v", senderID.ShortString(), err)
		return
	}

	c.pushEvent("NewMessage", NewMessagePayload{
		SenderID: senderID.String(),
		Data:     data,
	})
}

// onPeerFound - колбэк, который вызывается DiscoveryManager'ом.
func (c *CoreController) onPeerFound(pi peer.AddrInfo) {
	// Пытаемся подключиться к найденному пиру в фоновом режиме.
	go func() {
		if err := c.node.Host().Connect(c.ctx, pi); err != nil {
			// log.Printf("WARN: Не удалось подключиться к найденному пиру %s: %v", pi.ID.ShortString(), err)
		}
	}()
}

// pushEvent - потокобезопасный способ отправить событие в GUI.
func (c *CoreController) pushEvent(eventType string, payload interface{}) {
	select {
	case c.eventChan <- Event{Type: eventType, Payload: payload}:
	default:
		log.Printf("WARN: Очередь событий переполнена. Событие типа '%s' было отброшено.", eventType)
	}
}

// --- Обработчик сетевых событий ---

// networkNotifee реализует интерфейс network.Notifiee для отслеживания подключений.
type networkNotifee struct {
	c *CoreController
}

func (c *CoreController) newNetworkNotifee() network.Notifiee {
	return &networkNotifee{c: c}
}

func (n *networkNotifee) Connected(net network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	n.c.mu.Lock()
	n.c.connectedPeers[peerID] = true
	n.c.mu.Unlock()
	n.c.pushEvent("PeerConnected", PeerStatusPayload{PeerID: peerID.String()})
}

func (n *networkNotifee) Disconnected(net network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()
	n.c.mu.Lock()
	delete(n.c.connectedPeers, peerID)
	n.c.mu.Unlock()
	n.c.pushEvent("PeerDisconnected", PeerStatusPayload{PeerID: peerID.String()})
}

// Пустые реализации остальных методов интерфейса
func (n *networkNotifee) Listen(net network.Network, ma multiaddr.Multiaddr)      {}
func (n *networkNotifee) ListenClose(net network.Network, ma multiaddr.Multiaddr) {}
func (n *networkNotifee) OpenedStream(net network.Network, s network.Stream)      {}
