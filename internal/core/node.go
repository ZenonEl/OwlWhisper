package core

import (
	"bufio"
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// PROTOCOL_ID - уникальный идентификатор нашего чат-протокола
const PROTOCOL_ID = "/owl-whisper/1.0.0"

// Node представляет собой libp2p узел
type Node struct {
	host host.Host
	ctx  context.Context
}

// NewNode создает новый libp2p узел
func NewNode(ctx context.Context) (*Node, error) {
	// Создаем новый узел libp2p
	// Убираем опции для NAT, так как в локальной сети они не нужны
	h, err := libp2p.New()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать узел libp2p: %w", err)
	}

	node := &Node{
		host: h,
		ctx:  ctx,
	}

	// Устанавливаем обработчик для нашего протокола
	h.SetStreamHandler(PROTOCOL_ID, node.handleStream)

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

// Close останавливает узел
func (n *Node) Close() error {
	return n.host.Close()
}

// GetHost возвращает libp2p host
func (n *Node) GetHost() host.Host {
	return n.host
}

// GetPeers возвращает список подключенных пиров
func (n *Node) GetPeers() []peer.ID {
	return n.host.Network().Peers()
}

// SendMessage отправляет сообщение конкретному пиру
func (n *Node) SendMessage(peerID peer.ID, message string) error {
	// Открываем новый поток для каждого сообщения
	stream, err := n.host.NewStream(n.ctx, peerID, PROTOCOL_ID)
	if err != nil {
		return fmt.Errorf("не удалось открыть поток к %s: %w", peerID.ShortString(), err)
	}
	defer stream.Close()

	// Отправляем сообщение
	_, err = stream.Write([]byte(message + "\n"))
	if err != nil {
		return fmt.Errorf("не удалось отправить сообщение к %s: %w", peerID.ShortString(), err)
	}

	log.Printf("📤 Вам -> %s: %s", peerID.ShortString(), message)
	return nil
}

// BroadcastMessage отправляет сообщение всем подключенным пирам
func (n *Node) BroadcastMessage(message string) {
	peers := n.GetPeers()
	if len(peers) == 0 {
		log.Println("Нет подключенных участников для отправки сообщения.")
		return
	}

	for _, p := range peers {
		if err := n.SendMessage(p, message); err != nil {
			log.Printf("⚠️ Не удалось отправить сообщение к %s: %v", p.ShortString(), err)
		}
	}
}

// handleStream обрабатывает входящие потоки
func (n *Node) handleStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	log.Printf("ℹ️ Получен новый поток от %s", remotePeer.String())

	// Создаем 'reader' для чтения данных из потока
	reader := bufio.NewReader(stream)
	for {
		// Читаем сообщение до символа новой строки
		str, err := reader.ReadString('\n')
		if err != nil {
			// Ошибка EOF означает, что собеседник закрыл поток. Это нормально.
			stream.Close()
			return
		}
		// Выводим полученное сообщение
		fmt.Printf("📥 От %s: %s", remotePeer.ShortString(), str)
	}
}
