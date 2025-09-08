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
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multihash"

	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	webrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
)

const protocolID = "/p2p-chat/1.0.0"
const appPrefix = "owlwhisper" // Префикс для Rendezvous

// --- Helper-функция для создания CID из ника ---
func getCIDForNick(nick string) (cid.Cid, error) {
	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   multihash.SHA2_256,
		MhLength: -1,
	}
	c, err := pref.Sum([]byte(nick))
	if err != nil {
		return cid.Undef, err
	}
	return c, nil
}

func main() {
	// --- Определение флагов командной строки ---
	myNick := flag.String("my-nick", "", "Ваш уникальный ник (обязательно)")

	findNick := flag.String("find-nick", "", "Ник пользователя для поиска")
	findPeerID := flag.String("find-peer-id", "", "Peer ID для прямого поиска")

	// Булевые флаги для выбора методов поиска (по умолчанию true)
	useRendezvous := flag.Bool("find-rendezvous", true, "Использовать Rendezvous для поиска по нику")
	useCID := flag.Bool("find-cid", true, "Использовать CID для поиска по нику")

	forcePrivate := flag.Bool("forcePrivate", true, "Принудительный режим за NAT")
	forcePublic := flag.Bool("forcePublic", false, "Принудительный режим без NAT")

	flag.Parse()

	if *myNick == "" {
		log.Fatalln("Ошибка: необходимо указать ваш ник с помощью флага -my-nick")
	}

	// Определяем, есть ли хотя бы одна задача на поиск
	isDiscovering := *findNick != "" || *findPeerID != ""

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		log.Fatalf("Ошибка генерации ключа: %v", err)
	}

	// --- Сборка списка Relay-узлов ---
	staticRelaysStrings := []string{
		"/dns4/relay.dev.svcs.d.foundation/tcp/443/wss/p2p/12D3KooWCKd2fU1g4k15u3J5i6pGk26h3g68d3amEa2S71G5v1jS",
	}
	var allRelays []peer.AddrInfo
	for _, addrStr := range staticRelaysStrings {
		pi, err := peer.AddrInfoFromString(addrStr)
		if err != nil {
			log.Printf("Не удалось распарсить статический relay-адрес: %v", err)
			continue
		}
		allRelays = append(allRelays, *pi)
	}
	bootstrapPeers := dht.GetDefaultBootstrapPeerAddrInfos()
	allRelays = append(allRelays, bootstrapPeers...)
	log.Printf("🔗 Всего relay-кандидатов: %d (статические: %d + bootstrap: %d)",
		len(allRelays), len(staticRelaysStrings), len(bootstrapPeers))

	var kademliaDHT *dht.IpfsDHT

	// --- "Ультимативная" конфигурация хоста ---
	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/tcp/0/ws",
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip4/0.0.0.0/udp/0/webrtc-direct",
		),
		// Явно включаем все транспорты
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		libp2p.Transport(ws.New),
		libp2p.Transport(webrtc.New),
		// Двойное шифрование для максимальной совместимости
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(tls.ID, tls.New),
		// Все механизмы обхода NAT
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
		libp2p.EnableAutoNATv2(),
		// Настройки Relay
		libp2p.EnableRelay(),
		// Узел сам попытается найти и использовать релей, если обнаружит, что он за NAT
		libp2p.EnableAutoRelayWithStaticRelays(allRelays),
		libp2p.EnableAutoRelayWithPeerSource(func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
			ch := make(chan peer.AddrInfo)
			go func() {
				defer close(ch)

				// Проверяем, инициализирован ли DHT
				if kademliaDHT == nil {
					return
				}

				// Используем routing table самого DHT как источник пиров.
				for _, pi := range kademliaDHT.RoutingTable().ListPeers() {
					if numPeers <= 0 {
						break
					}
					// Проверяем, есть ли у нас адреса для этого пира
					addrs := kademliaDHT.Host().Peerstore().Addrs(pi)
					if len(addrs) > 0 {
						ch <- peer.AddrInfo{ID: pi, Addrs: addrs}
						numPeers--
					}
				}
			}()
			return ch
		},
			autorelay.WithBootDelay(2*time.Second),
			autorelay.WithMaxCandidates(10),
		),
	}

	if *forcePrivate {
		opts = append(opts, libp2p.ForceReachabilityPrivate())
	} else if *forcePublic {
		opts = append(opts, libp2p.ForceReachabilityPublic())
	}

	node, err := libp2p.New(opts...)

	if err != nil {
		log.Fatalf("Ошибка создания хоста: %v", err)
	}
	defer node.Close()

	log.Printf("[*] ID нашего узла: %s", node.ID())
	log.Println("[*] Наши адреса:")
	for _, addr := range node.Addrs() {
		fmt.Printf("    - %s\n", addr)
	}
	log.Printf("[*] Анонсируем себя под ником: %s", *myNick)
	fmt.Println()

	// --- Настройка DHT и сервисов обнаружения ---
	log.Println("Подключение к DHT...")
	kademliaDHT, err = dht.New(ctx, node, dht.Mode(dht.ModeServer))
	if err != nil {
		log.Fatalf("Ошибка создания DHT: %v", err)
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		log.Fatalf("Ошибка bootstrap DHT: %v", err)
	}

	var wg sync.WaitGroup
	for _, peerAddr := range bootstrapPeers {
		wg.Add(1)
		go func(pi peer.AddrInfo) {
			defer wg.Done()
			ctxConnect, cancelConnect := context.WithTimeout(ctx, 15*time.Second)
			defer cancelConnect()
			if err := node.Connect(ctxConnect, pi); err != nil {
				log.Printf("Не удалось подключиться к bootstrap-пиру %s: %s", pi.ID.ShortString(), err)
			} else {
				log.Printf("Успешное подключение к bootstrap-пиру: %s", pi.ID.ShortString())
			}
		}(peerAddr)
	}
	wg.Wait()

	// --- Основная логика ---
	node.SetStreamHandler(protocolID, handleStream)
	routingDiscovery := routing.NewRoutingDiscovery(kademliaDHT)

	// --- Слой анонса (всегда активен) ---
	go func() {
		rendezvousPoint := fmt.Sprintf("%s-%s", appPrefix, *myNick)
		myNickCID, err := getCIDForNick(*myNick)
		if err != nil {
			log.Printf("Ошибка создания CID для своего ника: %v", err)
			return // Не можем анонсировать через CID
		}

		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			log.Printf("📢 Анонсируем себя по Rendezvous: %s", rendezvousPoint)
			dutil.Advertise(ctx, routingDiscovery, rendezvousPoint)

			log.Printf("📢 Анонсируем себя по CID: %s", myNickCID)
			if err := kademliaDHT.Provide(ctx, myNickCID, true); err != nil {
				log.Printf("Ошибка анонса по CID: %v", err)
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	log.Println("✅ Слой анонса запущен. Ожидание подключений...")

	// --- Слой поиска (если нужно) ---
	if isDiscovering {
		peerChan := make(chan peer.AddrInfo)
		var searchWg sync.WaitGroup

		// Запускаем поиск по Peer ID
		if *findPeerID != "" {
			searchWg.Add(1)
			go func() {
				defer searchWg.Done()
				log.Printf("🔍 Начинаем поиск по Peer ID: %s", *findPeerID)
				peerID, err := peer.Decode(*findPeerID)
				if err != nil {
					log.Printf("Ошибка декодирования Peer ID: %v", err)
					return
				}

				// FindPeer может быть очень медленным, даем ему время
				ctxFind, cancelFind := context.WithTimeout(ctx, 2*time.Minute)
				defer cancelFind()
				pi, err := kademliaDHT.FindPeer(ctxFind, peerID)
				if err != nil {
					log.Printf("Не удалось найти пир по Peer ID %s: %v", *findPeerID, err)
					return
				}
				log.Printf("✅ Найден пир по Peer ID: %s", pi.ID)
				peerChan <- pi
			}()
		}

		// Запускаем поиск по Нику
		if *findNick != "" {
			// Поиск по Rendezvous
			if *useRendezvous {
				searchWg.Add(1)
				go func() {
					defer searchWg.Done()
					rendezvousPoint := fmt.Sprintf("%s-%s", appPrefix, *findNick)
					log.Printf("🔍 Начинаем поиск по Rendezvous: %s", rendezvousPoint)
					foundPeers, err := routingDiscovery.FindPeers(ctx, rendezvousPoint)
					if err != nil {
						log.Printf("Ошибка поиска по Rendezvous: %v", err)
						return
					}
					for p := range foundPeers {
						if p.ID == node.ID() {
							continue
						}
						log.Printf("✅ Найден пир через Rendezvous: %s", p.ID)
						peerChan <- p
					}
				}()
			}
			// Поиск по CID
			if *useCID {
				searchWg.Add(1)
				go func() {
					defer searchWg.Done()
					nickCID, err := getCIDForNick(*findNick)
					if err != nil {
						log.Printf("Ошибка создания CID для поиска: %v", err)
						return
					}
					log.Printf("🔍 Начинаем поиск по CID: %s", nickCID)
					providers := kademliaDHT.FindProvidersAsync(ctx, nickCID, 0)
					for p := range providers {
						if p.ID == node.ID() {
							continue
						}
						log.Printf("✅ Найден пир через CID: %s", p.ID)
						peerChan <- p
					}
				}()
			}
		}

		// Горутина для закрытия канала после завершения всех поисков
		go func() {
			searchWg.Wait()
			close(peerChan)
		}()

		// --- Логика подключения к найденным пирам ---
		var connectedPeer peer.ID
		var connectedPeerLock sync.Mutex

		for p := range peerChan {
			// Проверяем, не подключились ли мы уже к кому-то
			connectedPeerLock.Lock()
			if connectedPeer != "" {
				connectedPeerLock.Unlock()
				break
			}
			connectedPeerLock.Unlock()

			log.Printf("🌀 Попытка подключения к пиру: %s. Адреса: %v", p.ID, p.Addrs)
			ctxConnect, cancelConnect := context.WithTimeout(ctx, 90*time.Second)
			if err := node.Connect(ctxConnect, p); err != nil {
				log.Printf("❌ Не удалось подключиться к %s: %v", p.ID.ShortString(), err)
				cancelConnect()
				continue
			}
			cancelConnect()

			log.Printf("✅ УСПЕШНОЕ СОЕДИНЕНИЕ с %s!", p.ID.ShortString())
			streamCtx, streamCancel := context.WithTimeout(ctx, 60*time.Second)
			stream, err := node.NewStream(streamCtx, p.ID, protocolID)
			streamCancel()

			if err != nil {
				log.Printf("   ❌ Не удалось открыть стрим: %v", err)
				continue
			}

			// Устанавливаем флаг, что мы подключились, чтобы остановить другие попытки
			connectedPeerLock.Lock()
			connectedPeer = p.ID
			connectedPeerLock.Unlock()

			log.Printf("   ✅ Стрим успешно создан! Соединение через: %s", stream.Conn().RemoteMultiaddr())
			log.Println("🎉 Начинаем чат!")
			runChat(stream)
			goto End // Выходим из программы после успешного чата
		}
	}

End:
	// Ожидаем сигнала завершения (Ctrl+C), если мы в режиме слушателя или после чата
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\nПолучен сигнал завершения, закрываем узел...")
}

func handleStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	log.Printf("📡 Получен новый стрим от %s", remotePeer.ShortString())
	runChat(stream)
}

func runChat(stream network.Stream) {
	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)
	out := make(chan string)
	done := make(chan struct{})

	// Горутина для чтения из сети
	go func() {
		defer close(done)
		for {
			str, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					log.Printf("Ошибка чтения от %s: %v", stream.Conn().RemotePeer().ShortString(), err)
				}
				return
			}
			fmt.Printf("\x1b[32m%s\x1b[0m> %s", stream.Conn().RemotePeer().ShortString(), str)
		}
	}()

	// Горутина для записи в сеть
	go func() {
		for msg := range out {
			_, err := writer.WriteString(msg)
			if err != nil {
				log.Printf("Ошибка записи для %s: %v", stream.Conn().RemotePeer().ShortString(), err)
				return
			}
			err = writer.Flush()
			if err != nil {
				log.Printf("Ошибка flush для %s: %v", stream.Conn().RemotePeer().ShortString(), err)
				return
			}
		}
	}()

	// Основной цикл для чтения из stdin
	stdReader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-done:
			log.Println("Собеседник отключился.")
			close(out)
			return
		default:
			fmt.Print("> ")
			sendData, err := stdReader.ReadString('\n')
			if err != nil {
				log.Printf("Ошибка чтения stdin: %v", err)
				close(out)
				return
			}
			// Неблокирующая отправка в канал
			select {
			case out <- sendData:
			case <-done:
			}
		}
	}
}
