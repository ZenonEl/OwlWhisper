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
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

const (
	// CHAT_PROTOCOL_ID - для обычных сообщений (текст, анонсы файлов, сигнализация)
	CHAT_PROTOCOL_ID = protocol.ID("/owl-whisper/chat/1.0.0")
	// FILE_PROTOCOL_ID - для высокопроизводительной передачи файлов
	FILE_PROTOCOL_ID = "/owl-whisper/file/1.0.0"
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

type CoreReadyPayload struct {
	PeerID string `json:"peer_id"`
}

// PeerStatusPayload содержит данные для событий "PeerConnected" и "PeerDisconnected".
type PeerStatusPayload struct {
	PeerID string `json:"peer_id"`
}

// NewIncomingStreamPayload содержит данные для события "NewIncomingStream".
type NewIncomingStreamPayload struct {
	StreamID   uint64 `json:"stream_id"`
	PeerID     string `json:"peer_id"`
	ProtocolID string `json:"protocol_id"`
}

// StreamDataReceivedPayload содержит данные для события "StreamDataReceived".
type StreamDataReceivedPayload struct {
	StreamID uint64 `json:"stream_id"`
	Data     []byte `json:"data"`
}

// StreamClosedPayload содержит данные для события "StreamClosed".
type StreamClosedPayload struct {
	StreamID uint64 `json:"stream_id"`
	PeerID   string `json:"peer_id"`
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

	OpenStream(peerID string, protocolID string) (uint64, error)
	WriteToStream(streamID uint64, data []byte) error
	CloseStream(streamID uint64) error

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
	streamCounter  uint64                    // Простой счетчик для генерации уникальных ID
	activeStreams  map[uint64]network.Stream // Хранилище активных стримов
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
		activeStreams:  make(map[uint64]network.Stream),
	}, nil
}

func (c *CoreController) Start() error {
	var err error

	// 1. Создаем узел
	c.node, err = NewNode(c.ctx, c.privKey, c.cfg)
	if err != nil {
		return fmt.Errorf("ошибка создания узла: %w", err)
	}
	c.pushEvent("CoreReady", CoreReadyPayload{
		PeerID: c.node.Host().ID().String(),
	})
	// 2. Регистрируем обработчик входящих потоков
	c.node.SetStreamHandler(CHAT_PROTOCOL_ID, c.handleGenericStream)
	c.node.SetStreamHandler(FILE_PROTOCOL_ID, c.handleFileTransferStream)
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
	Info("[Controller] Остановка ядра...")
	c.cancel()
	if c.node != nil {
		if err := c.node.Close(); err != nil {
			return err
		}
	}
	close(c.eventChan)
	return nil
}

func (c *CoreController) GetMyPeerID() string {
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

	stream, err := c.node.Host().NewStream(c.ctx, peerID, CHAT_PROTOCOL_ID)
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

func (c *CoreController) handleGenericStream(stream network.Stream) {
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

func (c *CoreController) handleFileTransferStream(stream network.Stream) {
	// Это новый входящий файловый стрим.
	c.mu.Lock()
	c.streamCounter++
	streamID := c.streamCounter
	c.activeStreams[streamID] = stream
	c.mu.Unlock()

	peerID := stream.Conn().RemotePeer()

	// 1. Уведомляем GUI о новом стриме. GUI должен будет сам "связать" его с передачей.
	c.pushEvent("NewIncomingStream", NewIncomingStreamPayload{
		StreamID:   streamID,
		PeerID:     peerID.String(),
		ProtocolID: string(stream.Protocol()),
	})

	// 2. Запускаем горутину, которая просто читает "куски" и пересылает их наверх.
	// Core не знает, что это за куски - Protobuf-сообщения или что-то еще.
	go func() {
		defer func() {
			c.mu.Lock()
			delete(c.activeStreams, streamID)
			c.mu.Unlock()
			stream.Close()
			c.pushEvent("StreamClosed", StreamClosedPayload{StreamID: streamID, PeerID: peerID.String()})
		}()

		// Просто читаем сырые байты и отправляем их как событие.
		// GUI сам будет отвечать за их "сборку" в Protobuf-сообщения.
		buffer := make([]byte, 65536) // 64KB
		for {
			n, err := stream.Read(buffer)
			if err != nil {
				if err != io.EOF {
					Error("[Controller] Ошибка чтения из файлового стрима %d: %v", streamID, err)
				}
				break
			}
			// Копируем данные, чтобы избежать гонок
			dataToSend := make([]byte, n)
			copy(dataToSend, buffer[:n])

			c.pushEvent("StreamDataReceived", StreamDataReceivedPayload{
				StreamID: streamID,
				Data:     dataToSend,
			})
		}
	}()
}

func (c *CoreController) OpenStream(peerIDStr string, protocolID string) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		return 0, fmt.Errorf("неверный PeerID: %w", err)
	}

	stream, err := c.node.Host().NewStream(c.ctx, peerID, protocol.ID(protocolID))
	if err != nil {
		return 0, err
	}

	// Генерируем новый ID для стрима
	c.streamCounter++
	streamID := c.streamCounter
	c.activeStreams[streamID] = stream

	Info("[Controller] Открыт новый исходящий стрим %d к %s", streamID, peerID.ShortString())
	return streamID, nil
}

func (c *CoreController) WriteToStream(streamID uint64, data []byte) error {
	c.mu.RLock()
	stream, ok := c.activeStreams[streamID]
	c.mu.RUnlock()

	if !ok {
		return fmt.Errorf("стрим с ID %d не найден или уже закрыт", streamID)
	}

	_, err := stream.Write(data)
	return err
}

func (c *CoreController) CloseStream(streamID uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	stream, ok := c.activeStreams[streamID]
	if !ok {
		return fmt.Errorf("стрим с ID %d не найден", streamID)
	}

	delete(c.activeStreams, streamID) // Удаляем из нашего хранилища
	return stream.Close()
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
func (n *networkNotifee) ClosedStream(net network.Network, s network.Stream)      {}
