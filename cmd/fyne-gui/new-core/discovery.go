// cmd/fyne-gui/new-core/discovery.go

package newcore

import (
	"context"
	"log"
	"sync"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
)

// mdnsNotifee реализует интерфейс для получения уведомлений от сервиса mDNS.
type mdnsNotifee struct {
	host        host.Host
	onPeerFound func(peer.AddrInfo)
}

// HandlePeerFound вызывается, когда mDNS находит нового участника в локальной сети.
func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	log.Printf("INFO: [mDNS] Обнаружен пир: %s", pi.ID.ShortString())
	n.onPeerFound(pi)
}

// DiscoveryManager управляет всеми механизмами обнаружения (mDNS, DHT).
type DiscoveryManager struct {
	host        host.Host
	cfg         Config
	ctx         context.Context
	dht         *dht.IpfsDHT
	onPeerFound func(peer.AddrInfo)
}

// NewDiscoveryManager создает и настраивает новый менеджер обнаружения.
func NewDiscoveryManager(ctx context.Context, h host.Host, cfg Config, onPeerFound func(peer.AddrInfo)) (*DiscoveryManager, error) {
	dm := &DiscoveryManager{
		host:        h,
		cfg:         cfg,
		ctx:         ctx,
		onPeerFound: onPeerFound,
	}
	if cfg.EnableDHT {
		kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
		if err != nil {
			return nil, err
		}
		dm.dht = kadDHT
	}
	return dm, nil
}

// Start запускает все включенные в конфигурации сервисы обнаружения в фоновом режиме.
func (dm *DiscoveryManager) Start() {
	if dm.cfg.EnableMDNS {
		go dm.startXxxMDNS()
	}

	if dm.cfg.EnableDHT {
		go dm.startXxxDHT()
	}
}

// DHT возвращает экземпляр KadDHT для выполнения прямых запросов (Provide/FindProviders).
func (dm *DiscoveryManager) DHT() *dht.IpfsDHT {
	return dm.dht
}

// startXxxMDNS настраивает и запускает mDNS для обнаружения в локальной сети.
func (dm *DiscoveryManager) startXxxMDNS() {
	notifee := &mdnsNotifee{
		host:        dm.host,
		onPeerFound: dm.onPeerFound,
	}
	// mDNS будет использовать тот же RendezvousString, что и DHT для единообразия.
	service := mdns.NewMdnsService(dm.host, dm.cfg.RendezvousString, notifee)
	if err := service.Start(); err != nil {
		log.Printf("ERROR: [mDNS] Не удалось запустить сервис: %v", err)
	}
	log.Println("INFO: [mDNS] Сервис обнаружения в локальной сети запущен.")
}

// startXxxDHT подключается к bootstrap-узлам и запускает обнаружение в глобальной сети.
func (dm *DiscoveryManager) startXxxDHT() {
	Info("[DHT] Подключение к bootstrap-узлам...")
	if err := dm.dht.Bootstrap(dm.ctx); err != nil {
		Error("[DHT] Ошибка bootstrap: %v", err)
		return
	}
	Info("[DHT] Bootstrap завершен.")

	// Принудительное подключение, как в PoC
	var wg sync.WaitGroup
	for _, maddr := range dht.DefaultBootstrapPeers {
		wg.Add(1)
		go func(peerMaddr multiaddr.Multiaddr) {
			defer wg.Done()
			peerInfo, _ := peer.AddrInfoFromP2pAddr(peerMaddr)
			if err := dm.host.Connect(dm.ctx, *peerInfo); err != nil {
				// Warn("[DHT] Не удалось подключиться к bootstrap-пиру %s: %v", peerInfo.ID.ShortString(), err)
			} else {
				Info("[DHT] Установлено соединение с bootstrap-пиром: %s", peerInfo.ID.ShortString())
			}
		}(maddr)
	}
	wg.Wait()

	// --- ИНТЕГРАЦИЯ ЛОГИКИ RENDEZVOUS ИЗ PoC ---
	routingDiscovery := routing.NewRoutingDiscovery(dm.dht)

	// 1. Анонсируем себя в "общей комнате"
	Info("[DHT] Начинаем анонсирование в rendezvous-точке: %s", dm.cfg.RendezvousString)
	dutil.Advertise(dm.ctx, routingDiscovery, dm.cfg.RendezvousString)
	Info("[DHT] Анонсирование запущено.")

	// 2. Ищем других в "общей комнате"
	Info("[DHT] Начинаем поиск пиров в rendezvous-точке: %s", dm.cfg.RendezvousString)
	peerChan, err := routingDiscovery.FindPeers(dm.ctx, dm.cfg.RendezvousString)
	if err != nil {
		Error("[DHT] Ошибка поиска пиров по rendezvous: %v", err)
		return
	}

	// Обрабатываем найденных "случайных" пиров
	go func() {
		for p := range peerChan {
			if p.ID == dm.host.ID() {
				continue
			}
			Info("[DHT] Rendezvous: Найден пир: %s", p.ID.ShortString())
			dm.onPeerFound(p) // Сообщаем наверх о находке
		}
	}()
}
