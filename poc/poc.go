package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
)

const protocolID = "/p2p-chat/1.0.0"

// Главная функция
func main() {
	// Определяем флаги командной строки
	// -rendezvous: секретное слово/имя комнаты, по которому узлы будут находить друг друга
	// -discover: режим поиска (если не указан, узел будет "слушать")
	rendezvous := flag.String("rendezvous", "my-super-secret-rendezvous-point", "Уникальная строка для поиска пиров")
	discoverMode := flag.Bool("discover", false, "Запустить в режиме поиска пиров")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- 1. Создание libp2p хоста ---
	// Создаем новый приватный ключ для идентификации нашего узла
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		log.Fatalf("Ошибка генерации ключа: %v", err)
	}

	// Создаем хост, включая все необходимые опции для работы в реальной сети
	// Новый, более надежный блок
	node, err := libp2p.New(
		libp2p.Identity(priv),
		// 1. Оставляем слушать только TCP
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		// 2. Явно указываем, что хотим использовать ТОЛЬКО TCP транспорт
		libp2p.Transport(tcp.NewTCPTransport),
		// 3. Остальные опции очень важны и остаются
		libp2p.NATPortMap(),
		libp2p.EnableAutoNATv2(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelayService(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		log.Fatalf("Ошибка создания хоста: %v", err)
	}
	defer node.Close()

	// Выводим ID и адреса нашего узла. Другой узел может подключиться напрямую, используя один из этих адресов.
	fmt.Printf("[*] ID нашего узла: %s\n", node.ID())
	fmt.Println("[*] Наши адреса:")
	for _, addr := range node.Addrs() {
		fmt.Printf("    - %s/p2p/%s\n", addr, node.ID())
	}
	fmt.Println()

	// --- 2. Настройка обработчика входящих соединений ---
	// Эта функция будет вызываться каждый раз, когда кто-то подключится к нам по нашему protocolID
	node.SetStreamHandler(protocolID, handleStream)

	// --- 3. Настройка DHT для обнаружения пиров ---
	// Подключаемся к bootstrap-узлам IPFS, чтобы войти в общую сеть
	// Это КЛЮЧЕВОЙ шаг для работы через интернет
	log.Println("Подключение к DHT...")
	kademliaDHT, err := dht.New(ctx, node, dht.Mode(dht.ModeServer))
	if err != nil {
		log.Fatalf("Ошибка создания DHT: %v", err)
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		log.Fatalf("Ошибка bootstrap DHT: %v", err)
	}

	// Подключаемся к известным bootstrap-пирам
	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		pi, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func(pi peer.AddrInfo) {
			defer wg.Done()
			if err := node.Connect(ctx, pi); err != nil {
				log.Printf("Не удалось подключиться к bootstrap-пиру %s: %s", pi.ID, err)
			} else {
				log.Printf("Успешное подключение к bootstrap-пиру: %s", pi.ID)
			}
		}(*pi)
	}
	wg.Wait()

	// --- 4. Логика Анонсирования или Поиска ---
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)

	if !*discoverMode {
		// Режим "Слушателя/Анонсера"
		log.Printf("Анонсируем себя по rendezvous-строке: %s", *rendezvous)
		dutil.Advertise(ctx, routingDiscovery, *rendezvous)
		log.Println("Успешно анонсировано. Ожидание подключений...")
	} else {
		// Режим "Искателя/Подключающегося"
		log.Printf("Ищем пиров по rendezvous-строке: %s", *rendezvous)
		peerChan, err := routingDiscovery.FindPeers(ctx, *rendezvous)
		if err != nil {
			log.Fatalf("Ошибка поиска пиров: %v", err)
		}

		// Список надежных TCP и WSS ретрансляторов
		relayAddrs := []string{
			"/ip4/139.178.68.125/tcp/4001/p2p/12D3KooWL1V2Wp155eQtKork2S51RNCyX55K2iA6Ln52a83f23tt",
			"/dns4/relay.dev.svcs.d.foundation/tcp/443/wss/p2p/12D3KooWCKd2fU1g4k15u3J5i6pGk26h3g68d3amEa2S71G5v1jS",
		}

		for p := range peerChan {
			if p.ID == node.ID() {
				continue
			}
			log.Printf("Найден пир: %s.", p.ID)

			var connected bool
			// Пробуем подключиться через каждый ретранслятор из списка
			for _, relayAddrStr := range relayAddrs {
				log.Printf("...попытка через ретранслятор %s", relayAddrStr)
				relayAddr, err := multiaddr.NewMultiaddr(relayAddrStr + "/p2p-circuit/p2p/" + p.ID.String())
				if err != nil {
					log.Printf("Ошибка создания адреса: %v", err)
					continue
				}

				relayPeerInfo := peer.AddrInfo{
					ID:    p.ID,
					Addrs: []multiaddr.Multiaddr{relayAddr},
				}

				if err := node.Connect(ctx, relayPeerInfo); err != nil {
					log.Printf("Не удалось подключиться через этот ретранслятор: %v", err)
					continue // Пробуем следующий
				}

				log.Printf("УСПЕШНО подключено к %s через ретранслятор!", p.ID)
				connected = true
				break // Выходим из цикла ретрансляторов
			}

			if connected {
				stream, err := node.NewStream(ctx, p.ID, protocolID)
				if err != nil {
					log.Printf("Не удалось открыть стрим: %v", err)
					continue
				}
				log.Println("Начинаем чат.")
				runChat(stream)
				goto End
			}
		}
		log.Println("Не найдено активных пиров, к которым удалось бы подключиться.")
	}

End:
	// Ожидаем сигнала завершения (Ctrl+C)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\nПолучен сигнал завершения, закрываем узел...")
}

// handleStream обрабатывает входящие потоки данных
func handleStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	log.Printf("Получено новое входящее соединение от %s", remotePeer)
	// Запускаем чат
	runChat(stream)
}

// runChat запускает двунаправленный обмен сообщениями
func runChat(stream network.Stream) {
	// Создаем читателя и писателя для потока
	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)

	// Горутина для чтения входящих сообщений и вывода их в stdout
	go func() {
		for {
			str, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					log.Printf("Ошибка чтения от %s: %v", stream.Conn().RemotePeer(), err)
				}
				return
			}
			fmt.Printf("\x1b[32m%s\x1b[0m> %s", stream.Conn().RemotePeer().ShortString(), str)
		}
	}()

	// Горутина для чтения ввода из stdin и отправки его в поток
	go func() {
		stdReader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("> ")
			sendData, err := stdReader.ReadString('\n')
			if err != nil {
				log.Printf("Ошибка чтения stdin: %v", err)
				return
			}
			_, err = writer.WriteString(sendData)
			if err != nil {
				log.Printf("Ошибка записи для %s: %v", stream.Conn().RemotePeer(), err)
				return
			}
			err = writer.Flush()
			if err != nil {
				log.Printf("Ошибка flush для %s: %v", stream.Conn().RemotePeer(), err)
				return
			}
		}
	}()
}
