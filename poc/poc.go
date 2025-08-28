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

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"

	// Актуальные импорты транспортов
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	webrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
)

const protocolID = "/p2p-chat/1.0.0"

func main() {
	rendezvous := flag.String("rendezvous", "my-super-secret-rendezvous-point", "Уникальная строка для поиска пиров")
	discoverMode := flag.Bool("discover", false, "Запустить в режиме поиска пиров")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		log.Fatalf("Ошибка генерации ключа: %v", err)
	}

	// --- Сборка списка Relay-узлов ---
	// Наши "спасательные круги" - статические релеи, работающие на стандартных портах.
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

	// ИСПОЛЬЗУЕМ ЛИБУ, ЧТОБЫ ПОЛУЧИТЬ СПИСОК BOOTSTRAP-УЗЛОВ.
	// Они также могут выступать в роли релеев.
	bootstrapPeers := dht.GetDefaultBootstrapPeerAddrInfos()
	allRelays = append(allRelays, bootstrapPeers...)

	log.Printf("🔗 Всего relay-кандидатов: %d (статические: %d + bootstrap: %d)",
		len(allRelays), len(staticRelaysStrings), len(bootstrapPeers))

	var kademliaDHT *dht.IpfsDHT

	// --- "Ультимативная" конфигурация хоста ---
	node, err := libp2p.New(
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
			autorelay.WithBootDelay(2*time.Second), // Уменьшаем задержку для быстрого старта
			autorelay.WithMaxCandidates(10),        // Увеличиваем количество кандидатов
		),
	)
	if err != nil {
		log.Fatalf("Ошибка создания хоста: %v", err)
	}
	defer node.Close()

	log.Printf("[*] ID нашего узла: %s", node.ID())
	log.Println("[*] Наши адреса:")
	for _, addr := range node.Addrs() {
		fmt.Printf("    - %s\n", addr)
	}
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

	// Подключаемся к bootstrap-пирам для наполнения таблицы маршрутизации
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

	// --- Основная логика: "Слушатель" или "Искатель" ---
	node.SetStreamHandler(protocolID, handleStream)
	routingDiscovery := routing.NewRoutingDiscovery(kademliaDHT)

	if !*discoverMode {
		// Режим "Слушателя": анонсируем себя в DHT
		log.Printf("Анонсируем себя по rendezvous-строке: %s", *rendezvous)

		// Агрессивное анонсирование для стабильности relay-адресов
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					dutil.Advertise(ctx, routingDiscovery, *rendezvous)
					log.Printf("🔄 Анонсировано в DHT: %s", *rendezvous)
					time.Sleep(15 * time.Second) // Повторяем каждые 15 секунд
				}
			}
		}()

		log.Println("Успешно анонсировано. Ожидание подключений...")
	} else {
		// Режим "Искателя": ищем и подключаемся к пирам
		log.Printf("Ищем пиров по rendezvous-строке: %s", *rendezvous)

		// Агрессивный поиск пиров с повторениями
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					peerChan, err := routingDiscovery.FindPeers(ctx, *rendezvous)
					if err != nil {
						log.Printf("Ошибка поиска пиров: %v", err)
						time.Sleep(10 * time.Second)
						continue
					}

					for p := range peerChan {
						if p.ID == node.ID() {
							continue
						}
						log.Printf("Найден пир: %s. Адреса: %v", p.ID, p.Addrs)

						// --- УПРОЩЕННАЯ И НАДЕЖНАЯ ЛОГИКА ПОДКЛЮЧЕНИЯ ---
						// Мы просто вызываем Connect и доверяем libp2p сделать всю магию:
						// он сам попробует прямые адреса, сам сделает hole punching,
						// сам использует relay-адрес (если "Слушатель" смог его анонсировать).
						// Даем ему достаточно времени, т.к. relay может быть медленным.
						ctxConnect, cancelConnect := context.WithTimeout(ctx, 90*time.Second)
						if err := node.Connect(ctxConnect, p); err != nil {
							log.Printf("❌ Не удалось подключиться к %s: %v", p.ID.ShortString(), err)
							cancelConnect()
							continue
						}
						cancelConnect()

						log.Printf("✅ УСПЕШНОЕ СОЕДИНЕНИЕ с %s!", p.ID.ShortString())
						log.Printf("   🔍 Создаем стрим с протоколом: %s", protocolID)

						// --- ВОТ ОНО, РЕШЕНИЕ ---
						// Создаем новый контекст с увеличенным таймаутом специально для открытия стрима.
						// 30 секунд - это хороший, надежный таймаут для медленных мобильных сетей.
						streamCtx, streamCancel := context.WithTimeout(ctx, 60*time.Second)
						stream, err := node.NewStream(streamCtx, p.ID, protocolID)
						streamCancel() // Не забываем отменять контекст

						if err != nil {
							log.Printf("   ❌ Не удалось открыть стрим: %v", err)
							continue
						}

						log.Printf("   ✅ Стрим успешно создан! Соединение через: %s", stream.Conn().RemoteMultiaddr())
						log.Println("🎉 Начинаем чат!")
						runChat(stream, node)
						// Успешно подключились и запустили чат, выходим из программы.
						// В реальном приложении здесь будет другая логика.
						return
					}

					time.Sleep(15 * time.Second) // Повторяем поиск каждые 15 секунд
				}
			}
		}()

		log.Println("Поиск пиров запущен. Ожидание...")
	}

	// Ожидаем сигнала завершения (Ctrl+C)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\nПолучен сигнал завершения, закрываем узел...")
}

func handleStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	log.Printf("📡 Получен новый стрим от %s", remotePeer.ShortString())
	runChat(stream, nil)
}

func runChat(stream network.Stream, node host.Host) {
	// Создаем буферы для чтения и записи
	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)

	// Создаем каналы для координации горутин
	// out: для чтения из stdin и отправки в сеть
	// done: для завершения работы
	out := make(chan string)
	done := make(chan struct{})

	// --- Горутина для чтения из сети и вывода в stdout ---
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

	// --- Горутина для записи в сеть ---
	go func() {
		for {
			select {
			case <-done:
				return
			case msg := <-out:
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
		}
	}()

	// --- Основной цикл: чтение из stdin в текущей горутине ---
	stdReader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-done:
			log.Println("Собеседник отключился.")
			return
		default:
			fmt.Print("> ")
			sendData, err := stdReader.ReadString('\n')
			if err != nil {
				log.Printf("Ошибка чтения stdin: %v", err)
				return
			}
			out <- sendData
		}
	}
}
