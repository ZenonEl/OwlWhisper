package transport

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
)

const (
	PROTOCOL_ID    = "/owl-whisper/1.0.0"
	DISCOVERY_TAG  = "owl-whisper-rendezvous-point"
	STREAM_TIMEOUT = 30 * time.Second
)

// Libp2pTransport реализует ITransport интерфейс используя libp2p
type Libp2pTransport struct {
	host             host.Host
	dht              *dht.IpfsDHT
	routingDiscovery *routing.RoutingDiscovery
	listenPort       int
	enableTLS        bool
	enableNoise      bool
	enableNAT        bool
	enableHolePunch  bool
	enableRelay      bool
	messageHandler   func(peer.ID, []byte)
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
}

// NewLibp2pTransport создает новый экземпляр Libp2pTransport
func NewLibp2pTransport(listenPort int, enableTLS, enableNoise, enableNAT, enableHolePunch, enableRelay bool) (*Libp2pTransport, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем опции для libp2p
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
	}

	// Добавляем опции безопасности
	if enableTLS {
		opts = append(opts, libp2p.Security(libp2ptls.ID, libp2ptls.New))
	}
	if enableNoise {
		opts = append(opts, libp2p.Security(noise.ID, noise.New))
	}

	// Добавляем опции для NAT и relay
	if enableNAT {
		opts = append(opts, libp2p.EnableNATService())
	}
	if enableHolePunch {
		opts = append(opts, libp2p.EnableHolePunching())
	}
	if enableRelay {
		opts = append(opts, libp2p.EnableRelay())
	}

	// Создаем узел
	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	transport := &Libp2pTransport{
		host:            h,
		listenPort:      listenPort,
		enableTLS:       enableTLS,
		enableNoise:     enableNoise,
		enableNAT:       enableNAT,
		enableHolePunch: enableHolePunch,
		enableRelay:     enableRelay,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Устанавливаем обработчик потоков
	h.SetStreamHandler(PROTOCOL_ID, transport.handleStream)

	return transport, nil
}

// Start запускает транспортный слой
func (t *Libp2pTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Создаем DHT
	var err error
	t.dht, err = dht.New(ctx, t.host)
	if err != nil {
		return fmt.Errorf("failed to create DHT: %w", err)
	}

	// Подключаемся к bootstrap узлам
	if err = t.dht.Bootstrap(ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	// Ждем немного, чтобы DHT инициализировался
	time.Sleep(2 * time.Second)

	// Создаем routing discovery
	t.routingDiscovery = routing.NewRoutingDiscovery(t.dht)

	// Пытаемся анонсироваться в сети (может не получиться сразу)
	go func() {
		// Пробуем несколько раз с задержкой
		for i := 0; i < 3; i++ {
			time.Sleep(time.Duration(i+1) * time.Second)
			_, err := t.routingDiscovery.Advertise(ctx, DISCOVERY_TAG)
			if err == nil {
				log.Printf("✅ Успешно анонсировались в сети")
				break
			}
			log.Printf("⚠️ Попытка анонса %d не удалась: %v", i+1, err)
		}
	}()

	// Запускаем поиск пиров в фоне
	go t.discoverPeers(ctx)

	log.Printf("✅ Транспорт запущен. PeerID: %s", t.host.ID().String())
	for _, addr := range t.host.Addrs() {
		log.Printf("  %s/p2p/%s", addr, t.host.ID().String())
	}

	return nil
}

// Stop останавливает транспортный слой
func (t *Libp2pTransport) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cancel != nil {
		t.cancel()
	}

	if t.dht != nil {
		if err := t.dht.Close(); err != nil {
			log.Printf("Ошибка при закрытии DHT: %v", err)
		}
	}

	if t.host != nil {
		if err := t.host.Close(); err != nil {
			log.Printf("Ошибка при закрытии хоста: %v", err)
		}
	}

	return nil
}

// Connect подключается к указанному пиру
func (t *Libp2pTransport) Connect(ctx context.Context, peerID peer.ID) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.host == nil {
		return fmt.Errorf("транспорт не запущен")
	}

	// Проверяем, не подключены ли уже
	if t.host.Network().Connectedness(peerID) == network.Connected {
		return nil
	}

	// Ищем пира через DHT
	peerChan, err := t.routingDiscovery.FindPeers(ctx, DISCOVERY_TAG)
	if err != nil {
		return fmt.Errorf("failed to find peers: %w", err)
	}

	for p := range peerChan {
		if p.ID == peerID {
			if err := t.host.Connect(ctx, p); err != nil {
				return fmt.Errorf("failed to connect to peer %s: %w", peerID, err)
			}
			log.Printf("✅ Подключились к %s", peerID.ShortString())
			return nil
		}
	}

	return fmt.Errorf("peer %s not found in network", peerID)
}

// ConnectDirectly подключается к пиру по multiaddr (как в poc.go)
func (t *Libp2pTransport) ConnectDirectly(ctx context.Context, multiaddrStr string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.host == nil {
		return fmt.Errorf("транспорт не запущен")
	}

	log.Printf("🔗 Попытка прямого подключения к %s", multiaddrStr)

	// Парсим multiaddr
	maddr, err := multiaddr.NewMultiaddr(multiaddrStr)
	if err != nil {
		return fmt.Errorf("неверный формат multiaddr: %w", err)
	}

	// Извлекаем AddrInfo
	pinfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("не удалось извлечь AddrInfo: %w", err)
	}

	// Проверяем, не подключены ли уже
	if t.host.Network().Connectedness(pinfo.ID) == network.Connected {
		log.Printf("ℹ️ Уже подключены к %s", pinfo.ID.ShortString())
		return nil
	}

	// Пытаемся подключиться напрямую
	if err := t.host.Connect(ctx, *pinfo); err != nil {
		return fmt.Errorf("не удалось подключиться к %s: %w", multiaddrStr, err)
	}

	log.Printf("✅ Успешное прямое подключение к %s", pinfo.ID.ShortString())
	return nil
}

// Disconnect отключается от указанного пира
func (t *Libp2pTransport) Disconnect(ctx context.Context, peerID peer.ID) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.host == nil {
		return fmt.Errorf("транспорт не запущен")
	}

	if err := t.host.Network().ClosePeer(peerID); err != nil {
		return fmt.Errorf("failed to disconnect from peer %s: %w", peerID, err)
	}

	log.Printf("Отключились от %s", peerID.ShortString())
	return nil
}

// SendMessage отправляет сообщение указанному пиру
func (t *Libp2pTransport) SendMessage(ctx context.Context, peerID peer.ID, message []byte) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.host == nil {
		return fmt.Errorf("транспорт не запущен")
	}

	// Проверяем подключение
	if t.host.Network().Connectedness(peerID) != network.Connected {
		return fmt.Errorf("peer %s not connected", peerID)
	}

	// Создаем поток
	stream, err := t.host.NewStream(ctx, peerID, PROTOCOL_ID)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()

	// Устанавливаем таймаут
	stream.SetDeadline(time.Now().Add(STREAM_TIMEOUT))

	// Отправляем сообщение
	_, err = stream.Write(append(message, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	log.Printf("📤 Сообщение отправлено к %s", peerID.ShortString())
	return nil
}

// GetConnectedPeers возвращает список подключенных пиров
func (t *Libp2pTransport) GetConnectedPeers() []peer.ID {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.host == nil {
		return nil
	}

	return t.host.Network().Peers()
}

// GetPeerID возвращает ID текущего узла
func (t *Libp2pTransport) GetPeerID() peer.ID {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.host == nil {
		return ""
	}

	return t.host.ID()
}

// GetMultiaddrs возвращает multiaddr текущего узла
func (t *Libp2pTransport) GetMultiaddrs() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.host == nil {
		return nil
	}

	var addrs []string
	for _, addr := range t.host.Addrs() {
		addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", addr, t.host.ID()))
	}

	return addrs
}

// SetMessageHandler устанавливает обработчик входящих сообщений
func (t *Libp2pTransport) SetMessageHandler(handler func(peer.ID, []byte)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.messageHandler = handler
}

// handleStream обрабатывает входящие потоки
func (t *Libp2pTransport) handleStream(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("📥 Получен поток от %s", remotePeer.ShortString())

	reader := bufio.NewReader(stream)
	for {
		// Устанавливаем таймаут на чтение
		stream.SetReadDeadline(time.Now().Add(STREAM_TIMEOUT))

		message, err := reader.ReadBytes('\n')
		if err != nil {
			log.Printf("Ошибка чтения из потока: %v", err)
			return
		}

		// Убираем символ новой строки
		if len(message) > 0 && message[len(message)-1] == '\n' {
			message = message[:len(message)-1]
		}

		// Вызываем обработчик сообщений
		if t.messageHandler != nil {
			t.messageHandler(remotePeer, message)
		}
	}
}

// discoverPeers ищет новых пиров в сети
func (t *Libp2pTransport) discoverPeers(ctx context.Context) {
	// Ждем немного перед первым поиском
	time.Sleep(5 * time.Second)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Проверяем, что DHT готов
			if t.routingDiscovery == nil {
				log.Printf("⚠️ DHT еще не готов, пропускаем поиск пиров")
				continue
			}

			// Ищем новых пиров
			peerChan, err := t.routingDiscovery.FindPeers(ctx, DISCOVERY_TAG)
			if err != nil {
				log.Printf("⚠️ Ошибка поиска пиров: %v", err)
				continue
			}

			peerCount := 0
			for p := range peerChan {
				if p.ID == t.host.ID() {
					continue // Пропускаем себя
				}

				peerCount++
				// Пытаемся подключиться
				if err := t.host.Connect(ctx, p); err != nil {
					log.Printf("⚠️ Не удалось подключиться к %s: %v", p.ID.ShortString(), err)
				} else {
					log.Printf("✅ Подключились к новому пиру: %s", p.ID.ShortString())
				}
			}

			if peerCount == 0 {
				log.Printf("ℹ️ Новых пиров не найдено")
			}
		}
	}
}
