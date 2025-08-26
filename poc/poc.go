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
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"

	// Актуальные импорты транспортов
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	webrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"

	"github.com/multiformats/go-multiaddr"
)

const protocolID = "/p2p-chat/1.0.0"

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

	// Создание хоста со всеми современными транспортами
	node, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip4/0.0.0.0/tcp/0/ws",
			"/ip4/0.0.0.0/udp/0/webrtc-direct",
		),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		libp2p.Transport(ws.New),
		libp2p.Transport(webrtc.New),
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
		libp2p.EnableAutoNATv2(),
		libp2p.EnableRelay(), // allow using relays
		libp2p.EnableAutoRelayWithStaticRelays(staticRelays),
		libp2p.EnableRelayService(), // only if you want to act as relay (requires extra perms)
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

	// Stream handler
	node.SetStreamHandler(protocolID, handleStream)

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

	if !*discoverMode {
		log.Printf("Анонсируем себя по rendezvous-строке: %s", *rendezvous)
		dutil.Advertise(ctx, routingDiscovery, *rendezvous)
		log.Println("Успешно анонсировано. Ожидание подключений...")
	} else {
		log.Printf("Ищем пиров по rendezvous-строке: %s", *rendezvous)
		peerChan, err := routingDiscovery.FindPeers(ctx, *rendezvous)
		if err != nil {
			log.Fatalf("Ошибка поиска пиров: %v", err)
		}

		relayAddrs := []string{
			"/ip4/139.178.68.125/tcp/4001/p2p/12D3KooWL1V2Wp155eQtKork2S51RNCyX55K2iA6Ln52a83f23tt",
			"/dns4/relay.dev.svcs.d.foundation/tcp/443/wss/p2p/12D3KooWCKd2fU1g4k15u3J5i6pGk26h3g68d3amEa2S71G5v1jS",
		}

		for p := range peerChan {
			if p.ID == node.ID() {
				continue
			}
			log.Printf("Найден пир: %s.", p.ID)

			if err := tryConnect(ctx, node, kademliaDHT, p, relayAddrs); err != nil {
				log.Printf("Полная неудача подключения к %s: %v", p.ID, err)
				continue
			}

			log.Printf("УСПЕШНОЕ СОЕДИНЕНИЕ с %s! Открываем стрим...", p.ID)
			stream, err := node.NewStream(ctx, p.ID, protocolID)
			if err != nil {
				log.Printf("Не удалось открыть стрим после успешного соединения: %v", err)
				continue
			}
			log.Println("Начинаем чат.")
			runChat(stream)
			goto End
		}
		log.Println("Не найдено активных пиров, к которым удалось бы подключиться.")
	}

End:
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("\nПолучен сигнал завершения, закрываем узел...")
}

func handleStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	log.Printf("Получено новое входящее соединение от %s", remotePeer)
	runChat(stream)
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
	// 0: Быстрое обновление адресов через DHT
	ctxDHT, cancelDHT := context.WithTimeout(ctx, 5*time.Second)
	if pi, err := refreshPeerAddrs(ctxDHT, kademliaDHT, p); err == nil {
		p = pi
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

	// 2: Попытка Hole Punching
	if len(p.Addrs) > 0 {
		node.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.TempAddrTTL)

		hpCtx, hpCancel := context.WithTimeout(ctx, 10*time.Second)
		if err := node.Connect(hpCtx, p); err == nil {
			hpCancel()
			log.Println("[connect] Успешный Hole Punch!")
			return nil
		} else {
			log.Printf("[connect] holepunch attempt -> %v", err)
		}
		hpCancel()
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
