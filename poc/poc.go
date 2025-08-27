package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	mrand "math/rand" // Алиас для избежания конфликта с crypto/rand
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
	pstore "github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"

	// Актуальные импорты транспортов
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	webrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"

	"github.com/multiformats/go-multiaddr"
)

const protocolID = "/p2p-chat/1.0.0"

func dhtPeerSource(ctx context.Context, numPeers int) <-chan peer.AddrInfo {

	ch := make(chan peer.AddrInfo)

	go func() {

		defer close(ch)

		// Инициализируйте DHT и получите пиры (упрощенный пример)

		// Возвращаем статические релеи для авторелеев
		staticRelays := []string{
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		}

		for _, relayStr := range staticRelays {
			if numPeers <= 0 {
				break
			}

			pi, err := peer.AddrInfoFromString(relayStr)
			if err != nil {
				log.Printf("Ошибка парсинга relay адреса %s: %v", relayStr, err)
				continue
			}

			ch <- *pi
			numPeers--
		}

	}()

	return ch

}
func main() {
	// Инициализируем генератор случайных чисел для jitter
	mrand.Seed(time.Now().UnixNano())

	rendezvous := flag.String("rendezvous", "my-super-secret-rendezvous-point", "Уникальная строка для поиска пиров")
	discoverMode := flag.Bool("discover", false, "Запустить в режиме поиска пиров")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Генерация ключей
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		log.Fatalf("Ошибка генерации ключа: %v", err)
	}

	relayAddrsStrings := []string{
		"/ip4/139.178.68.125/tcp/4001/p2p/12D3KooWL1V2Wp155eQtKork2S51RNCyX55K2iA6Ln52a83f23tt",
		"/dns4/relay.dev.svcs.d.foundation/tcp/443/wss/p2p/12D3KooWCKd2fU1g4k15u3J5i6pGk26h3g68d3amEa2S71G5v1jS",
	}
	var staticRelays []peer.AddrInfo
	for _, addrStr := range relayAddrsStrings {
		pi, err := peer.AddrInfoFromString(addrStr)
		if err != nil {
			log.Printf("Не удалось распарсить статический relay-адрес: %v", err)
			continue
		}
		staticRelays = append(staticRelays, *pi)
	}

	// Создание хоста с улучшенными настройками для обхода NAT и маскировки трафика
	node, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			// НЕпривилегированные порты для маскировки под веб-трафик
			"/ip4/0.0.0.0/tcp/8080/ws", // Альтернативный HTTP (не требует root)
			"/ip4/0.0.0.0/tcp/8443/ws", // Альтернативный HTTPS (не требует root)
			"/ip4/0.0.0.0/tcp/8888/ws", // Еще один веб-порт
			"/ip4/0.0.0.0/tcp/9000/ws", // И еще один
			// Динамические порты для гибкости
			"/ip4/0.0.0.0/tcp/0/ws",            // WebSocket на случайном порту
			"/ip4/0.0.0.0/tcp/0",               // TCP на случайном порту
			"/ip4/0.0.0.0/udp/0/quic-v1",       // QUIC на случайном порту
			"/ip4/0.0.0.0/udp/0/webrtc-direct", // WebRTC для совместимости
		),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(webrtc.New), // Добавляем WebRTC обратно для совместимости
		libp2p.Transport(ws.New),
		libp2p.Transport(quic.NewTransport),
		// Двойное шифрование: Noise (лучше для NAT) + TLS
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(tls.ID, tls.New),
		// Улучшенные настройки для обхода NAT
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
		libp2p.EnableAutoNATv2(),
		libp2p.EnableNATService(),
		// Relay настройки - ПРИНУДИТЕЛЬНО включаем для межсетевых соединений
		libp2p.EnableRelay(),        // allow using relays
		libp2p.EnableRelayService(), // only if you want to act as relay (requires extra perms)
		libp2p.EnableAutoRelayWithPeerSource(dhtPeerSource, autorelay.WithBootDelay(5*time.Second)), // Уменьшил задержку
		//libp2p.EnableAutoRelayWithStaticRelays(staticRelays),
		// Принудительно включаем relay для всех соединений
		libp2p.ForceReachabilityPublic(),
	)
	if err != nil {
		log.Fatalf("Ошибка создания хоста: %v", err)
	}
	defer node.Close()

	fmt.Printf("[*] ID нашего узла: %s\n", node.ID())
	fmt.Println("[*] Наши адреса:")
	for _, addr := range node.Addrs() {
		fmt.Printf("    - %s/p2p/%s\n", addr, node.ID())
	}
	fmt.Println()

	// Детальная информация об адресах
	fmt.Println("[*] Детальная информация об адресах:")
	for _, addr := range node.Addrs() {
		protocols := addr.Protocols()
		fmt.Printf("    %s:\n", addr)
		fmt.Printf("      - Протоколы: %v\n", protocols)
		fmt.Printf("      - IP: %s\n", addr.String())
	}
	fmt.Println()

	// Stream handler
	log.Printf("🔧 Регистрируем stream handler для протокола: %s", protocolID)
	node.SetStreamHandler(protocolID, handleStream)

	// Проверяем, что handler зарегистрирован
	handlers := node.Mux().Protocols()
	log.Printf("🔧 Зарегистрированные протоколы: %v", handlers)

	// 🔧 Регистрируем обработчик подключений для двустороннего hole punching
	// 🔧 Регистрируем обработчик подключений для двустороннего hole punching
	var globalRoutingDiscovery *drouting.RoutingDiscovery

	setupConnectionNotifier := func() {
		node.Network().Notify(&network.NotifyBundle{
			ConnectedF: func(n network.Network, conn network.Conn) {

				// 🔄 АКТИВНО анонсируем себя когда видим попытку подключения
				go func() {
					for i := 0; i < 5; i++ { // 5 попыток
						select {
						case <-ctx.Done():
							return
						default:
							// Используем глобальную переменную routingDiscovery
							if globalRoutingDiscovery != nil {
								dutil.Advertise(ctx, globalRoutingDiscovery, *rendezvous)
							}
							time.Sleep(2 * time.Second)
						}
					}
				}()

				// Запускаем двусторонний hole punching
				go bidirectionalHolePunch(ctx, node, conn.RemotePeer())
			},
			DisconnectedF: func(n network.Network, conn network.Conn) {
			},
		})
	}

	// Запускаем setup после создания DHT
	setupConnectionNotifier()

	// DHT
	log.Println("Подключение к DHT...")
	kademliaDHT, err := dht.New(ctx, node, dht.Mode(dht.ModeServer))
	if err != nil {
		log.Fatalf("Ошибка создания DHT: %v", err)
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		log.Fatalf("Ошибка bootstrap DHT: %v", err)
	}

	// Connect to bootstrap peers
	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		pi, err := peer.AddrInfoFromString(peerAddr.String())
		if err != nil {
			log.Printf("Неверный формат bootstrap-адреса %s: %v", peerAddr, err)
			continue // Пропускаем этот пир
		}
		wg.Add(1)
		go func(pi peer.AddrInfo) {
			defer wg.Done()
			ctxConnect, cancelConnect := context.WithTimeout(ctx, 10*time.Second)
			defer cancelConnect()
			if err := node.Connect(ctxConnect, pi); err != nil {
				log.Printf("Не удалось подключиться к bootstrap-пиру %s: %s", pi.ID, err)
			} else {
				log.Printf("Успешное подключение к bootstrap-пиру: %s", pi.ID)
			}
		}(*pi)
	}
	wg.Wait()

	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	globalRoutingDiscovery = routingDiscovery // Обновляем глобальную переменную

	if !*discoverMode {
		log.Printf("Анонсируем себя по rendezvous-строке: %s", *rendezvous)

		// Агрессивное анонсирование с повторениями
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					dutil.Advertise(ctx, routingDiscovery, *rendezvous)
					log.Printf("Анонсировано в DHT: %s", *rendezvous)
					time.Sleep(30 * time.Second) // Повторяем каждые 30 секунд
				}
			}
		}()

		log.Println("Успешно анонсировано. Ожидание подключений...")
	} else {
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

					// Обрабатываем найденных пиров
					for p := range peerChan {
						if p.ID == node.ID() {
							continue
						}
						log.Printf("Найден пир: %s. Адреса: %v", p.ID, p.Addrs)

						if err := tryConnect(ctx, node, kademliaDHT, p, relayAddrsStrings); err == nil {
							log.Printf("✅ УСПЕШНОЕ СОЕДИНЕНИЕ с %s! Открываем стрим...", p.ID)
							log.Printf("   🔍 Создаем стрим с протоколом: %s", protocolID)

							// Логируем информацию о соединении для диагностики
							conns := node.Network().ConnsToPeer(p.ID)
							if len(conns) > 0 {
								workingConn := conns[0]
								log.Printf("   🔗 Активное соединение через: %s", workingConn.RemoteMultiaddr())
							}

							// Создаем стрим через конкретное соединение
							var stream network.Stream
							var err error

							// Создаем стрим С ПРОТОКОЛОМ через основной узел
							log.Printf("   🔍 Создаем стрим с протоколом: %s", protocolID)

							// Сначала пробуем прямое соединение
							log.Printf("   🔍 Создаем стрим напрямую...")
							stream, err = node.NewStream(ctx, p.ID, protocol.ID(protocolID))
							if err != nil {
								log.Printf("❌ Прямое соединение не удалось: %v", err)
								log.Printf("   🔍 Детали ошибки: %T: %v", err, err)

								// Принудительно используем relay для межсетевых соединений
								log.Printf("   🔄 Принудительно используем relay для межсетевого соединения...")

								// Принудительно используем relay для межсетевых соединений
								stream, err = createStreamViaRelay(ctx, node, p.ID, protocolID, relayAddrsStrings)
								if err != nil {
									log.Printf("   ❌ Relay соединение тоже не удалось: %v", err)

									// Последняя попытка - принудительный relay через bootstrap
									log.Printf("   🔄 Последняя попытка - принудительный relay через bootstrap...")
									stream, err = forceRelayConnection(ctx, node, p.ID, protocolID)
									if err != nil {
										log.Printf("   ❌ Принудительный relay тоже не удался: %v", err)
										continue
									}
									log.Printf("   ✅ Стрим создан через принудительный relay!")
								} else {
									log.Printf("   ✅ Стрим создан через relay!")
								}
							}

							log.Printf("✅ Стрим успешно создан!")
							log.Printf("   📍 Локальный адрес: %s", stream.Conn().LocalMultiaddr())
							log.Printf("   🌐 Удаленный адрес: %s", stream.Conn().RemoteMultiaddr())
							log.Printf("   🔒 Протокол: %s", stream.Protocol())

							// Явно устанавливаем протокол для multistream negotiation
							log.Printf("   🔧 Устанавливаем протокол: %s", protocolID)
							stream.SetProtocol(protocolID)

							// Проверяем, что протокол установлен
							if stream.Protocol() == protocolID {
								log.Printf("   ✅ Протокол успешно установлен: %s", stream.Protocol())
							} else {
								log.Printf("   ⚠️ Протокол не установлен, текущий: %s", stream.Protocol())
							}

							log.Println("🎉 Начинаем чат!")
							runChat(stream)
							return
						}
					}

					time.Sleep(15 * time.Second) // Повторяем поиск каждые 15 секунд
				}
			}
		}()

		// Ждем завершения поиска
		select {
		case <-ctx.Done():
			return
		}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\nПолучен сигнал завершения, закрываем узел...")
}

func handleStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	remoteAddr := stream.Conn().RemoteMultiaddr()
	localAddr := stream.Conn().LocalMultiaddr()

	log.Printf("📡 НОВЫЙ СТРИМ: от %s", remotePeer)
	log.Printf("   📍 Локальный адрес: %s", localAddr)
	log.Printf("   🌐 Удаленный адрес: %s", remoteAddr)
	log.Printf("   🔒 Протокол: %s", stream.Protocol())
	log.Printf("   🔧 Ожидаемый протокол: %s", protocolID)

	// Проверяем, что протокол совпадает
	if stream.Protocol() == protocolID {
		log.Printf("   ✅ Протокол совпадает, начинаем чат")
		runChat(stream)
	} else {
		log.Printf("   ❌ Протокол не совпадает! Ожидали: %s, получили: %s", protocolID, stream.Protocol())
		log.Printf("   🔧 Пытаемся установить правильный протокол...")

		// Пытаемся установить правильный протокол
		stream.SetProtocol(protocolID)
		if stream.Protocol() == protocolID {
			log.Printf("   ✅ Протокол исправлен, начинаем чат")
			runChat(stream)
		} else {
			log.Printf("   ❌ Не удалось исправить протокол, закрываем стрим")
			stream.Close()
		}
	}
}

func runChat(stream network.Stream) {
	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)

	// Горутина для чтения входящих сообщений
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

	// Горутина для отправки сообщений из stdin
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

// ---------- Функции для многоступенчатой стратегии подключения ----------

func dialWithTimeout(ctx context.Context, node host.Host, pi peer.AddrInfo, t time.Duration) error {
	ctxd, cancel := context.WithTimeout(ctx, t)
	defer cancel()
	return node.Connect(ctxd, pi)
}

func refreshPeerAddrs(ctx context.Context, kademliaDHT *dht.IpfsDHT, p peer.AddrInfo) (peer.AddrInfo, error) {
	if kademliaDHT == nil {
		return p, errors.New("dht is nil")
	}
	return kademliaDHT.FindPeer(ctx, p.ID)
}

func tryConnect(ctx context.Context, node host.Host, kademliaDHT *dht.IpfsDHT, p peer.AddrInfo, relayAddrs []string) error {
	log.Printf("[connect] 🔍 Попытка подключения к пиру %s", p.ID)
	log.Printf("[connect] 📍 Адреса пира: %v", p.Addrs)

	// 0: Быстрое обновление адресов через DHT
	ctxDHT, cancelDHT := context.WithTimeout(ctx, 5*time.Second)
	if pi, err := refreshPeerAddrs(ctxDHT, kademliaDHT, p); err == nil {
		log.Printf("[connect] ✅ DHT обновил адреса: %v", pi.Addrs)
		p = pi
	} else {
		log.Printf("[connect] ⚠️ DHT не смог обновить адреса: %v", err)
	}
	cancelDHT()

	// 1: Прямые попытки подключения
	for _, t := range []time.Duration{2 * time.Second, 5 * time.Second} {
		if err := dialWithTimeout(ctx, node, p, t); err == nil {
			log.Println("[connect] Успешное прямое подключение!")
			return nil
		} else {
			log.Printf("[connect] direct dial (%s) -> %v", t, err)
		}
	}

	// 2: Агрессивный Hole Punching с множественными попытками
	if len(p.Addrs) > 0 {
		node.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.TempAddrTTL)

		// Множественные попытки hole punching с разными таймаутами
		for _, timeout := range []time.Duration{5 * time.Second, 10 * time.Second, 15 * time.Second} {
			hpCtx, hpCancel := context.WithTimeout(ctx, timeout)

			// Попытка подключения
			if err := node.Connect(hpCtx, p); err == nil {
				hpCancel()
				log.Printf("[connect] Успешный Hole Punch с таймаутом %v!", timeout)
				return nil
			} else {
				log.Printf("[connect] holepunch attempt (%v) -> %v", timeout, err)
			}
			hpCancel()

			// Небольшая пауза между попытками
			time.Sleep(1 * time.Second)
		}
	}

	// 3: Даем AutoRelay немного времени на работу
	time.Sleep(1 * time.Second)
	arCtx, arCancel := context.WithTimeout(ctx, 5*time.Second)
	if err := node.Connect(arCtx, p); err == nil {
		arCancel()
		log.Println("[connect] Успешное подключение через AutoRelay!")
		return nil
	} else {
		log.Printf("[connect] post-auto-relay connect -> %v", err)
	}
	arCancel()

	// 4: Явные попытки через Relay
	for _, r := range relayAddrs {
		ma, err := multiaddr.NewMultiaddr(r + "/p2p-circuit/p2p/" + p.ID.String())
		if err != nil {
			log.Printf("[relay] bad relay addr %s: %v", r, err)
			continue
		}
		pi := peer.AddrInfo{ID: p.ID, Addrs: []multiaddr.Multiaddr{ma}}
		if err := dialWithTimeout(ctx, node, pi, 12*time.Second); err == nil {
			log.Printf("[relay] Успешное подключение через явный Relay: %s", r)
			return nil
		} else {
			log.Printf("[relay] dial via %s -> %v", r, err)
		}
	}

	// 5: Цикл повторных попыток с экспоненциальной задержкой
	for i := 1; i <= 3; i++ {
		wait := time.Duration(math.Pow(2.0, float64(i))) * time.Second
		wait += time.Duration(mrand.Intn(1000)) * time.Millisecond // Jitter
		log.Printf("[retry] Попытка #%d, ждем %v", i, wait)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}

		if pi, err := refreshPeerAddrs(ctx, kademliaDHT, p); err == nil {
			p = pi
		}

		if err := dialWithTimeout(ctx, node, p, 8*time.Second); err == nil {
			log.Println("[retry] Успешное прямое подключение при повторе!")
			return nil
		}
	}

	return errors.New("все попытки подключения провалились")
}

// ---------- Функция для создания стрима через relay ----------

func createStreamViaRelay(ctx context.Context, node host.Host, peerID peer.ID, protocolID string, relayAddrs []string) (network.Stream, error) {
	log.Printf("[relay] 🔄 Создаем стрим через relay для пира %s", peerID)

	// Пробуем каждый relay
	for _, relayAddr := range relayAddrs {
		log.Printf("[relay] 🔍 Пробуем relay: %s", relayAddr)

		// Парсим relay адрес
		relayInfo, err := peer.AddrInfoFromString(relayAddr)
		if err != nil {
			log.Printf("[relay] ❌ Ошибка парсинга relay адреса: %v", err)
			continue
		}

		// Подключаемся к relay
		if err := node.Connect(ctx, *relayInfo); err != nil {
			log.Printf("[relay] ❌ Не удалось подключиться к relay %s: %v", relayInfo.ID, err)
			continue
		}

		log.Printf("[relay] ✅ Подключились к relay: %s", relayInfo.ID)

		// Создаем стрим через relay
		relayStream, err := node.NewStream(ctx, peerID, protocol.ID(protocolID))
		if err != nil {
			log.Printf("[relay] ❌ Не удалось создать стрим через relay %s: %v", relayInfo.ID, err)
			continue
		}

		log.Printf("[relay] ✅ Стрим создан через relay %s!", relayInfo.ID)
		return relayStream, nil
	}

	return nil, errors.New("не удалось создать стрим ни через один relay")
}

// ---------- Принудительный relay через bootstrap ----------

func forceRelayConnection(ctx context.Context, node host.Host, peerID peer.ID, protocolID string) (network.Stream, error) {
	log.Printf("[force-relay] 🚀 Принудительный relay через bootstrap для пира %s", peerID)

	// Используем bootstrap пиры как relay
	bootstrapPeers := []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	}

	for _, bootstrapAddr := range bootstrapPeers {
		log.Printf("[force-relay] 🔍 Пробуем bootstrap: %s", bootstrapAddr)

		// Парсим bootstrap адрес
		bootstrapInfo, err := peer.AddrInfoFromString(bootstrapAddr)
		if err != nil {
			log.Printf("[force-relay] ❌ Ошибка парсинга bootstrap адреса: %v", err)
			continue
		}

		// Подключаемся к bootstrap
		if err := node.Connect(ctx, *bootstrapInfo); err != nil {
			log.Printf("[force-relay] ❌ Не удалось подключиться к bootstrap %s: %v", bootstrapInfo.ID, err)
			continue
		}

		log.Printf("[force-relay] ✅ Подключились к bootstrap: %s", bootstrapInfo.ID)

		// Создаем стрим через bootstrap как relay
		relayStream, err := node.NewStream(ctx, peerID, protocol.ID(protocolID))
		if err != nil {
			log.Printf("[force-relay] ❌ Не удалось создать стрим через bootstrap %s: %v", bootstrapInfo.ID, err)
			continue
		}

		log.Printf("[force-relay] ✅ Стрим создан через bootstrap %s!", bootstrapInfo.ID)
		return relayStream, nil
	}

	return nil, errors.New("не удалось создать стрим ни через один bootstrap")
}

// ---------- Двусторонний hole punching ----------

func bidirectionalHolePunch(ctx context.Context, node host.Host, remotePeer peer.ID) {

	// Получаем адреса удаленного пира
	remoteAddrs := node.Peerstore().Addrs(remotePeer)
	if len(remoteAddrs) == 0 {
		return
	}

	// Агрессивный двусторонний hole punching
	for _, timeout := range []time.Duration{3 * time.Second, 5 * time.Second, 8 * time.Second} {
		select {
		case <-ctx.Done():
			return
		default:
			// Пробуем подключиться с разными таймаутами
			hpCtx, hpCancel := context.WithTimeout(ctx, timeout)

			// Создаем AddrInfo для подключения
			addrInfo := peer.AddrInfo{
				ID:    remotePeer,
				Addrs: remoteAddrs,
			}

			if err := node.Connect(hpCtx, addrInfo); err == nil {
				hpCancel()

				// Пытаемся создать стрим для подтверждения
				go tryCreateStreamAfterHolePunch(ctx, node, remotePeer)
				return
			} else {
			}

			hpCancel()
			time.Sleep(500 * time.Millisecond)
		}
	}

}

// ---------- Создание стрима после успешного hole punching ----------

func tryCreateStreamAfterHolePunch(ctx context.Context, node host.Host, remotePeer peer.ID) {

	// Ждем немного для стабилизации соединения
	time.Sleep(1 * time.Second)

	// Пытаемся создать стрим
	stream, err := node.NewStream(ctx, remotePeer, protocol.ID(protocolID))
	if err != nil {
		return
	}

	log.Printf("[post-holepunch] ✅ Стрим создан после hole punching!")
	log.Printf("[post-holepunch] 📍 Локальный адрес: %s", stream.Conn().LocalMultiaddr())
	log.Printf("[post-holepunch] 🌐 Удаленный адрес: %s", stream.Conn().RemoteMultiaddr())
	log.Printf("[post-holepunch] 🔒 Протокол: %s", stream.Protocol())

	// Запускаем чат
	log.Println("[post-holepunch] 🎉 Начинаем чат после hole punching!")
	runChat(stream)
}
