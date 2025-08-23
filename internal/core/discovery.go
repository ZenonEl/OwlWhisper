package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

const (
	DISCOVERY_TAG     = "owl-whisper-mdns"
	GLOBAL_RENDEZVOUS = "owl-whisper-global"
)

// DiscoveryNotifee обрабатывает события обнаружения
type DiscoveryNotifee struct {
	node   host.Host
	ctx    context.Context
	onPeer func(peer.AddrInfo)
}

// HandlePeerFound вызывается когда найден новый пир
func (n *DiscoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	// Пропускаем себя
	if pi.ID == n.node.ID() {
		return
	}

	Info("📢 Обнаружен новый пир: %s", pi.ID.ShortString())

	// Пытаемся подключиться
	if err := n.node.Connect(n.ctx, pi); err != nil {
		Error("❌ Не удалось подключиться к %s: %v", pi.ID.ShortString(), err)
		return
	}

	Info("✅ Успешное подключение к %s", pi.ID.ShortString())

	// Уведомляем о новом пире
	if n.onPeer != nil {
		n.onPeer(pi)
	}
}

// DiscoveryManager управляет обнаружением пиров
type DiscoveryManager struct {
	host             host.Host
	dht              *dht.IpfsDHT
	routingDiscovery *routing.RoutingDiscovery
	mdnsService      mdns.Service
	notifee          *DiscoveryNotifee

	ctx    context.Context
	cancel context.CancelFunc

	// Канал для уведомлений о новых пирах
	peersChan chan peer.AddrInfo
}

// NewDiscoveryManager создает новый менеджер обнаружения
func NewDiscoveryManager(ctx context.Context, host host.Host, onPeer func(peer.AddrInfo)) (*DiscoveryManager, error) {
	ctx, cancel := context.WithCancel(ctx)

	notifee := &DiscoveryNotifee{
		node:   host,
		ctx:    ctx,
		onPeer: onPeer,
	}

	// Создаем mDNS сервис
	mdnsService := mdns.NewMdnsService(host, DISCOVERY_TAG, notifee)

	// Создаем DHT в режиме сервера
	kadDHT, err := dht.New(ctx, host, dht.Mode(dht.ModeServer))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать DHT: %w", err)
	}

	// Создаем routing discovery
	routingDiscovery := routing.NewRoutingDiscovery(kadDHT)

	dm := &DiscoveryManager{
		host:             host,
		dht:              kadDHT,
		routingDiscovery: routingDiscovery,
		mdnsService:      mdnsService,
		notifee:          notifee,
		ctx:              ctx,
		cancel:           cancel,
		peersChan:        make(chan peer.AddrInfo, 100),
	}

	return dm, nil
}

// Start запускает discovery сервисы
func (dm *DiscoveryManager) Start() error {
	// Запускаем mDNS
	if err := dm.mdnsService.Start(); err != nil {
		return fmt.Errorf("не удалось запустить mDNS: %w", err)
	}
	Info("📡 mDNS сервис запущен")

	// Подключаемся к bootstrap узлам
	if err := dm.dht.Bootstrap(dm.ctx); err != nil {
		Warn("⚠️ Не удалось подключиться к bootstrap узлам: %v", err)
	} else {
		Info("✅ Bootstrap завершен")
	}

	// Запускаем mDNS discovery в фоне
	go dm.startMDNSDiscovery()

	// Запускаем DHT discovery в фоне
	go dm.startDHTDiscovery()

	return nil
}

// Stop останавливает discovery сервисы
func (dm *DiscoveryManager) Stop() error {
	dm.cancel()

	if dm.mdnsService != nil {
		dm.mdnsService.Close()
	}

	if dm.dht != nil {
		return dm.dht.Close()
	}

	return nil
}

// GetDHT возвращает DHT для использования в других частях системы
func (dm *DiscoveryManager) GetDHT() *dht.IpfsDHT {
	return dm.dht
}

// startMDNSDiscovery запускает mDNS discovery
func (dm *DiscoveryManager) startMDNSDiscovery() {
	Info("🏠 Поиск локальных пиров через mDNS...")

	// mDNS работает автоматически через DiscoveryNotifee
	// Просто ждем завершения контекста
	<-dm.ctx.Done()
}

// startDHTDiscovery запускает поиск через DHT
func (dm *DiscoveryManager) startDHTDiscovery() {
	Info("🌐 Подключение к bootstrap узлам...")

	// ИЗМЕНЕНИЕ: Ждем, пока мы подключимся хотя бы к одному bootstrap-пиру.
	// Это гарантирует, что наша таблица не пуста перед анонсом.
	var wg sync.WaitGroup
	for _, p := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(p)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dm.host.Connect(dm.ctx, *peerinfo); err != nil {
				// Info("Не удалось подключиться к bootstrap-пиру: %s", err)
			} else {
				Info("✅ Установлено соединение с bootstrap-пиром: %s", peerinfo.ID.ShortString())
			}
		}()
	}
	wg.Wait()

	Info("📢 Анонсируемся в глобальной сети...")
	// Используем Ticker для периодического анонсирования, чтобы оставаться видимыми
	ticker := time.NewTicker(time.Minute * 1)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-dm.ctx.Done():
				return
			case <-ticker.C:
				Info("📢 Повторно анонсируемся в сети...")
				_, err := dm.routingDiscovery.Advertise(dm.ctx, GLOBAL_RENDEZVOUS)
				if err != nil {
					Warn("⚠️ Ошибка повторного анонса: %v", err)
				}
			}
		}
	}()

	// Первоначальный анонс
	_, err := dm.routingDiscovery.Advertise(dm.ctx, GLOBAL_RENDEZVOUS)
	if err != nil {
		Warn("⚠️ Ошибка первоначального анонса: %v", err)
	} else {
		Info("📢 Первоначальный анонс успешен")
	}

	Info("🔍 Поиск участников в глобальной сети...")
	peerChan, err := dm.routingDiscovery.FindPeers(dm.ctx, GLOBAL_RENDEZVOUS)
	if err != nil {
		Warn("⚠️ Ошибка поиска в глобальной сети: %v", err)
		return
	}

	for p := range peerChan {
		if p.ID == dm.host.ID() {
			continue
		}

		Info("🌐 Найден участник в глобальной сети: %s", p.ID.ShortString())
		dm.notifee.HandlePeerFound(p)
	}
}

// GetRoutingDiscovery возвращает routing discovery для внутреннего использования
func (dm *DiscoveryManager) GetRoutingDiscovery() *routing.RoutingDiscovery {
	return dm.routingDiscovery
}
