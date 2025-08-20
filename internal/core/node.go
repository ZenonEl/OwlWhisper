package core

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// PROTOCOL_ID - уникальный идентификатор нашего чат-протокола
const PROTOCOL_ID = "/owl-whisper/1.0.0"

// NetworkEventLogger логирует сетевые события
type NetworkEventLogger struct{}

func (nel *NetworkEventLogger) Listen(network.Network, multiaddr.Multiaddr)      {}
func (nel *NetworkEventLogger) ListenClose(network.Network, multiaddr.Multiaddr) {}

func (nel *NetworkEventLogger) Connected(net network.Network, conn network.Conn) {
	log.Printf("🔗 EVENT: Успешное соединение с %s", conn.RemotePeer().ShortString())
}

func (nel *NetworkEventLogger) Disconnected(net network.Network, conn network.Conn) {
	log.Printf("🔌 EVENT: Соединение с %s разорвано", conn.RemotePeer().ShortString())
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
}

// NewNode создает новый libp2p узел
func NewNode(ctx context.Context) (*Node, error) {
	// Создаем libp2p узел с опциями для глобальной сети
	opts := []libp2p.Option{
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать узел libp2p: %w", err)
	}

	node := &Node{
		host:         h,
		ctx:          ctx,
		messagesChan: make(chan RawMessage, 100), // Буферизованный канал
		peers:        make(map[peer.ID]bool),
	}

	// Устанавливаем обработчик потоков
	h.SetStreamHandler(PROTOCOL_ID, node.handleStream)

	// Устанавливаем Network Notifiee для мониторинга
	h.Network().Notify(&NetworkEventLogger{})

	log.Printf("✅ Узел создан. Ваш PeerID: %s", h.ID().String())
	log.Println("Адреса для прослушивания:")
	for _, addr := range h.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, h.ID().String())
	}

	return node, nil
}

// Start запускает узел
func (n *Node) Start() error {
	log.Println("🚀 Узел запущен")
	return nil
}

// Stop останавливает узел
func (n *Node) Stop() error {
	if err := n.host.Close(); err != nil {
		return fmt.Errorf("ошибка остановки узла: %w", err)
	}
	close(n.messagesChan)
	log.Println("🛑 Узел остановлен")
	return nil
}

// GetHost возвращает host.Host для внутреннего использования
func (n *Node) GetHost() host.Host {
	return n.host
}

// GetMyID возвращает ID текущего узла
func (n *Node) GetMyID() string {
	return n.host.ID().String()
}

// GetPeers возвращает список подключенных пиров
func (n *Node) GetPeers() []peer.ID {
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

	n.peers[peerID] = true
}

// RemovePeer удаляет пира из списка
func (n *Node) RemovePeer(peerID peer.ID) {
	n.peersMutex.Lock()
	defer n.peersMutex.Unlock()

	delete(n.peers, peerID)
}

// Send отправляет данные конкретному пиру
func (n *Node) Send(peerID peer.ID, data []byte) error {
	// Открываем поток к пиру
	stream, err := n.host.NewStream(n.ctx, peerID, PROTOCOL_ID)
	if err != nil {
		return fmt.Errorf("не удалось открыть поток к %s: %w", peerID.ShortString(), err)
	}
	defer stream.Close()

	// Отправляем данные
	_, err = stream.Write(data)
	if err != nil {
		return fmt.Errorf("не удалось отправить данные к %s: %w", peerID.ShortString(), err)
	}

	log.Printf("📤 Отправлено %d байт к %s", len(data), peerID.ShortString())
	return nil
}

// Broadcast отправляет данные всем подключенным пирам
func (n *Node) Broadcast(data []byte) error {
	peers := n.GetPeers()
	if len(peers) == 0 {
		log.Println("⚠️ Нет подключенных пиров для broadcast")
		return nil
	}

	var lastError error
	for _, peerID := range peers {
		if err := n.Send(peerID, data); err != nil {
			log.Printf("❌ Ошибка отправки к %s: %v", peerID.ShortString(), err)
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
	log.Printf("📥 Получен поток от %s", remotePeer.ShortString())

	// Добавляем пира в список
	n.AddPeer(remotePeer)

	// Читаем данные из потока
	buffer := make([]byte, 1024)
	bytesRead, err := stream.Read(buffer)
	if err != nil {
		log.Printf("❌ Ошибка чтения потока от %s: %v", remotePeer.ShortString(), err)
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
		log.Printf("📨 Сообщение от %s добавлено в очередь", remotePeer.ShortString())
	default:
		log.Printf("⚠️ Канал сообщений переполнен, сообщение от %s потеряно", remotePeer.ShortString())
	}

	stream.Close()
}
