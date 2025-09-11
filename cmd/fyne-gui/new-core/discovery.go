// cmd/fyne-gui/new-core/discovery.go

package newcore

import (
	"context"
	"log"
	"sync"
	"time"

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
	// 1. Подключаемся к bootstrap-узлам.
	Info("[DHT] Подключение к bootstrap-узлам...")
	if err := dm.dht.Bootstrap(dm.ctx); err != nil {
		Error("[DHT] Ошибка bootstrap: %v", err)
		return
	}
	Info("[DHT] Bootstrap DHT завершен.")

	// 2. Принудительно подключаемся к нескольким bootstrap-пирам, как в PoC.
	var wg sync.WaitGroup
	// Объединяем стандартные и пользовательские bootstrap-узлы
	allBootstrapAddrs := append(dht.DefaultBootstrapPeers, parseMultiaddrs(dm.cfg.CustomBootstrapNodes)...)

	for _, maddr := range allBootstrapAddrs {
		wg.Add(1)
		go func(peerMaddr multiaddr.Multiaddr) {
			defer wg.Done()
			peerInfo, err := peer.AddrInfoFromP2pAddr(peerMaddr)
			if err != nil {
				Warn("[DHT] Не удалось распарсить bootstrap-адрес: %v", err)
				return
			}
			if err := dm.host.Connect(dm.ctx, *peerInfo); err != nil {
				Warn("[DHT] Не удалось подключиться к bootstrap-пиру %s: %v", peerInfo.ID.ShortString(), err)
			} else {
				Info("[DHT] Установлено соединение с bootstrap-пиром: %s", peerInfo.ID.ShortString())
			}
		}(maddr)
	}
	wg.Wait()

	// 3. Используем RoutingDiscovery для Rendezvous-механизма из PoC.
	routingDiscovery := routing.NewRoutingDiscovery(dm.dht)

	// 4. Постоянно анонсируем свое присутствие в "общей комнате".
	go func() {
		ticker := time.NewTicker(dm.cfg.AnnounceInterval)
		defer ticker.Stop()
		for {
			Info("[DHT] Анонсируем себя в rendezvous-точке: %s", dm.cfg.RendezvousString)
			dutil.Advertise(dm.ctx, routingDiscovery, dm.cfg.RendezvousString)
			select {
			case <-dm.ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	// 5. Постоянно ищем других пиров в той же "комнате".
	go func() {
		for {
			Info("[DHT] Ищем других пиров в rendezvous-точке: %s", dm.cfg.RendezvousString)
			peerChan, err := routingDiscovery.FindPeers(dm.ctx, dm.cfg.RendezvousString)
			if err != nil {
				Error("[DHT] Ошибка поиска пиров по rendezvous: %v", err)
			} else {
				for p := range peerChan {
					if p.ID == dm.host.ID() {
						continue // Пропускаем себя
					}
					Info("[DHT] Rendezvous: Найден пир: %s", p.ID.ShortString())
					dm.onPeerFound(p) // Сообщаем наверх о находке
				}
			}
			select {
			case <-dm.ctx.Done():
				return
			// Пауза между циклами поиска, чтобы не перегружать сеть.
			case <-time.After(dm.cfg.AnnounceInterval * 2):
			}
		}
	}()
}

// parseMultiaddrs - helper-функция для преобразования строк в multi-адреса.
func parseMultiaddrs(addrs []string) []multiaddr.Multiaddr {
	var multiaddrs []multiaddr.Multiaddr
	for _, addrStr := range addrs {
		maddr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			Warn("Не удалось распарсить multi-адрес '%s': %v", addrStr, err)
			continue
		}
		multiaddrs = append(multiaddrs, maddr)
	}
	return multiaddrs
}
