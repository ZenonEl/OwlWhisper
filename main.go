package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// PROTOCOL_ID - это уникальный идентификатор нашего чат-протокола
const PROTOCOL_ID = "/owl-whisper/1.0.0"

// DISCOVERY_TAG - это "секретное слово", по которому наши узлы будут находить друг друга в локальной сети через mDNS
const DISCOVERY_TAG = "owl-whisper-mdns"

// discoveryNotifee обрабатывает события обнаружения новых участников сети
type discoveryNotifee struct {
	node host.Host
	ctx  context.Context
}

// HandlePeerFound - этот метод вызывается, когда mDNS находит нового участника
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	// Пропускаем, если нашли самого себя
	if pi.ID == n.node.ID() {
		return
	}
	log.Printf("📢 Обнаружен новый участник: %s", pi.ID.String())

	// Пытаемся подключиться к найденному участнику
	err := n.node.Connect(n.ctx, pi)
	if err != nil {
		log.Printf("❌ Не удалось подключиться к %s: %v", pi.ID.String(), err)
	} else {
		log.Printf("✅ Успешное подключение к %s", pi.ID.String())
	}
}

func handleStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	log.Printf("ℹ️ Получен новый поток от %s", remotePeer.String())

	// Создаем 'reader' для чтения данных из потока
	reader := bufio.NewReader(stream)
	for {
		// Читаем сообщение до символа новой строки
		str, err := reader.ReadString('\n')
		if err != nil {
			// Ошибка EOF означает, что собеседник закрыл поток. Это нормально.
			// log.Printf("⚠️ Поток с %s закрыт: %v", remotePeer.ShortString(), err)
			stream.Close()
			return
		}
		// Выводим полученное сообщение
		fmt.Printf("📥 От %s: %s", remotePeer.ShortString(), str)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Создаем новый узел libp2p.
	// Мы убрали опции для NAT, так как в локальной сети они не нужны.
	node, err := libp2p.New()
	if err != nil {
		log.Fatalf("Не удалось создать узел libp2p: %v", err)
	}

	log.Printf("✅ Узел создан. Ваш PeerID: %s", node.ID().String())
	log.Println("Адреса для прослушивания:")
	for _, addr := range node.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, node.ID().String())
	}

	// Устанавливаем обработчик для нашего протокола
	node.SetStreamHandler(PROTOCOL_ID, handleStream)

	// --- Запускаем mDNS для обнаружения в локальной сети ---
	// Это самая важная часть, которая решает вашу проблему.
	notifee := &discoveryNotifee{node: node, ctx: ctx}
	mdnsService := mdns.NewMdnsService(node, DISCOVERY_TAG, notifee)
	if err := mdnsService.Start(); err != nil {
		log.Fatalf("Не удалось запустить mDNS: %v", err)
	}
	log.Println("📡 Сервис mDNS запущен. Идет поиск других участников...")

	// --- Чтение сообщений из консоли и отправка ---
	// Используем sync.Mutex для безопасного доступа к списку пиров из разных горутин
	var peersMux sync.Mutex

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		message := scanner.Text()

		peersMux.Lock()
		peers := node.Network().Peers()
		peersMux.Unlock()

		if len(peers) == 0 {
			log.Println("Нет подключенных участников для отправки сообщения.")
			continue
		}

		// Отправляем сообщение всем, с кем установлено соединение
		for _, p := range peers {
			// Открываем новый поток для каждого сообщения
			stream, err := node.NewStream(ctx, p, PROTOCOL_ID)
			if err != nil {
				log.Printf("Не удалось открыть поток к %s: %v", p.ShortString(), err)
				continue
			}

			_, err = stream.Write([]byte(message + "\n"))
			if err != nil {
				log.Printf("Не удалось отправить сообщение к %s: %v\n", p.ShortString(), err)
			} else {
				log.Printf("📤 Вам -> %s: %s", p.ShortString(), message)
			}
			// Важно: закрываем поток после отправки одного сообщения
			stream.Close()
		}
	}
}
