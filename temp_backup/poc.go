package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
)

// PROTOCOL_ID - это уникальный идентификатор нашего чат-протокола
const PROTOCOL_ID = "/owl-whisper/1.0.0"

// DISCOVERY_TAG - это "секретное слово", по которому наши узлы будут находить друг друга в сети
const DISCOVERY_TAG = "owl-whisper-rendezvous-point"

func handleStream(stream network.Stream) {
	log.Println("Получен новый поток от", stream.Conn().RemotePeer().String())
	// Создаем 'reader' для чтения данных из потока
	reader := bufio.NewReader(stream)
	for {
		// Читаем сообщение до символа новой строки
		str, err := reader.ReadString('\n')
		if err != nil {
			log.Println("Ошибка чтения из потока:", err)
			stream.Close()
			return
		}
		// Выводим полученное сообщение
		fmt.Printf("📥 От %s: %s", stream.Conn().RemotePeer().ShortString(), str)
	}
}

func main() {
	// --- Настройка и создание узла libp2p ---
	destAddr := flag.String("d", "", "Адрес для прямого подключения (если нужно)")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Создаем новый узел libp2p. Это самая важная часть.
	// Мы включаем все необходимые опции для обхода NAT.
	node, err := libp2p.New(
		// Включить автоматическое определение и обход NAT
		libp2p.EnableNATService(),
		// Включить "пробивание дыр" в NAT (Hole Punching)
		libp2p.EnableHolePunching(),
		// Включить ретрансляцию (Relay) как запасной вариант
		libp2p.EnableRelay(),
	)
	if err != nil {
		log.Fatalf("Не удалось создать узел libp2p: %v", err)
	}

	log.Printf("✅ Узел создан. Ваш PeerID: %s\n", node.ID().String())
	log.Println("Адреса для прослушивания:")
	for _, addr := range node.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, node.ID().String())
	}

	// Устанавливаем обработчик для нашего протокола
	node.SetStreamHandler(PROTOCOL_ID, handleStream)

	// --- Запуск DHT для обнаружения других узлов ---
	go startDHT(ctx, node)

	// --- Логика для прямого подключения (если указан адрес) ---
	if *destAddr != "" {
		go connectDirectly(ctx, node, *destAddr)
	}

	// --- Чтение сообщений из консоли и отправка ---
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		message := scanner.Text()
		// Отправляем сообщение всем, с кем установлено соединение
		for _, p := range node.Network().Peers() {
			stream, err := node.NewStream(ctx, p, PROTOCOL_ID)
			if err != nil {
				log.Printf("Не удалось открыть поток к %s: %v\n", p.ShortString(), err)
				continue
			}
			_, err = stream.Write([]byte(message + "\n"))
			if err != nil {
				log.Printf("Не удалось отправить сообщение к %s: %v\n", p.ShortString(), err)
			}
			log.Printf("📤 Вам -> %s: %s", p.ShortString(), message)
			stream.Close() // Закрываем поток после отправки
		}
	}
}

func startDHT(ctx context.Context, node host.Host) {
	// Создаем новый DHT-клиент
	kadDHT, err := dht.New(ctx, node)
	if err != nil {
		log.Fatalf("Не удалось создать DHT: %v", err)
	}

	// Подключаемся к bootstrap-узлам IPFS
	log.Println("Подключение к bootstrap-узлам...")
	if err = kadDHT.Bootstrap(ctx); err != nil {
		log.Fatalf("Не удалось подключиться к bootstrap-узлам: %v", err)
	}

	// Начинаем поиск других узлов
	log.Println("Ищем других участников сети...")
	routingDiscovery := routing.NewRoutingDiscovery(kadDHT)
	routingDiscovery.Advertise(ctx, DISCOVERY_TAG) // "Анонсируем" себя в сети

	// Ищем других, кто анонсировал себя с тем же тегом
	peerChan, err := routingDiscovery.FindPeers(ctx, DISCOVERY_TAG)
	if err != nil {
		log.Fatalf("Не удалось начать поиск пиров: %v", err)
	}

	for p := range peerChan {
		// Пропускаем, если нашли самого себя
		if p.ID == node.ID() {
			continue
		}
		log.Printf("Найден участник: %s. Попытка подключения...\n", p.ID.String())
		// Пытаемся подключиться. libp2p сам разберется с NAT.
		if err := node.Connect(ctx, p); err != nil {
			log.Printf("Не удалось подключиться к %s: %v\n", p.ID.String(), err)
		} else {
			log.Printf("✅ Успешное подключение к %s\n", p.ID.String())
		}
	}
}

func connectDirectly(ctx context.Context, node host.Host, destAddr string) {
	log.Printf("Попытка прямого подключения к %s", destAddr)
	maddr, err := multiaddr.NewMultiaddr(destAddr)
	if err != nil {
		log.Printf("Неверный формат multiaddr: %v", err)
		return
	}
	pinfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		log.Printf("Не удалось извлечь AddrInfo: %v", err)
		return
	}
	// Ждем немного, чтобы основной узел успел запуститься
	time.Sleep(2 * time.Second)
	if err := node.Connect(ctx, *pinfo); err != nil {
		log.Printf("Не удалось подключиться к %s: %v", destAddr, err)
	} else {
		log.Printf("✅ Успешное прямое подключение к %s", destAddr)
	}
}
