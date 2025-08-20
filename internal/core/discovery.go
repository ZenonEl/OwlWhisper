package core

import (
	"context"
	"fmt"
	"log"
	"sync" // ИЗМЕНЕНИЕ: Добавляем пакет sync
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

// DISCOVERY_TAG_MDNS - "секретное слово" для поиска участников через mDNS
const DISCOVERY_TAG_MDNS = "owl-whisper-mdns"

// DISCOVERY_TAG_DHT - "секретное слово" для поиска участников через DHT
const DISCOVERY_TAG_DHT = "owl-whisper-global-rendezvous"

// DiscoveryNotifee обрабатывает события обнаружения новых участников сети
type DiscoveryNotifee struct {
	node host.Host
	ctx  context.Context
}

// HandlePeerFound вызывается, когда mDNS находит нового участника
func (n *DiscoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	// Пропускаем, если нашли самого себя
	if pi.ID == n.node.ID() {
		return
	}
	log.Printf("📢 Обнаружен новый участник: %s", pi.ID.String())

	// Пытаемся подключиться к найденному участнику
	err := n.node.Connect(n.ctx, pi)
	if err != nil {
		log.Printf("❌ Не удалось подключиться к %s: %v", pi.ID.String(), err)
	} else {
		log.Printf("✅ Успешное подключение к %s", pi.ID.String())
	}
}

// DiscoveryManager управляет всеми механизмами обнаружения
type DiscoveryManager struct {
	mdnsService      mdns.Service
	dht              *dht.IpfsDHT
	routingDiscovery *routing.RoutingDiscovery
	notifee          *DiscoveryNotifee
	host             host.Host // ИЗМЕНЕНИЕ: Добавляем host для доступа
	ctx              context.Context
}

// NewDiscoveryManager создает новый менеджер обнаружения
func NewDiscoveryManager(ctx context.Context, node host.Host) *DiscoveryManager {
	notifee := &DiscoveryNotifee{
		node: node,
		ctx:  ctx,
	}

	mdnsService := mdns.NewMdnsService(node, DISCOVERY_TAG_MDNS, notifee)

	// ИЗМЕНЕНИЕ: Переключаем DHT в режим сервера. Это критически важно!
	kadDHT, err := dht.New(ctx, node, dht.Mode(dht.ModeServer))
	if err != nil {
		log.Printf("⚠️ Не удалось создать DHT: %v", err)
	} else {
		log.Printf("✅ DHT создан в режиме сервера")
	}

	routingDiscovery := routing.NewRoutingDiscovery(kadDHT)
	log.Printf("✅ Routing discovery создан")

	return &DiscoveryManager{
		mdnsService:      mdnsService,
		dht:              kadDHT,
		routingDiscovery: routingDiscovery,
		notifee:          notifee,
		host:             node, // ИЗМЕНЕНИЕ: Сохраняем host
		ctx:              ctx,
	}
}

// Start запускает все механизмы обнаружения
func (dm *DiscoveryManager) Start() error {
	if err := dm.mdnsService.Start(); err != nil {
		return fmt.Errorf("не удалось запустить mDNS: %w", err)
	}
	log.Println("📡 Сервис mDNS запущен.")

	go dm.startDHTDiscovery()
	log.Println("🌐 DHT discovery запущен.")

	return nil
}

// startDHTDiscovery запускает поиск через DHT
func (dm *DiscoveryManager) startDHTDiscovery() {
	log.Println("🌐 Подключение к bootstrap узлам...")
	if err := dm.dht.Bootstrap(dm.ctx); err != nil {
		log.Printf("⚠️ Не удалось подключиться к bootstrap узлам: %v", err)
		return
	}
	log.Println("✅ Bootstrap DHT завершен")

	// ИЗМЕНЕНИЕ: Ждем, пока мы подключимся хотя бы к одному bootstrap-пиру.
	// Это гарантирует, что наша таблица не пуста перед анонсом.
	var wg sync.WaitGroup
	for _, p := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(p)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dm.host.Connect(dm.ctx, *peerinfo); err != nil {
				// log.Printf("Не удалось подключиться к bootstrap-пиру: %s", err)
			} else {
				log.Printf("✅ Установлено соединение с bootstrap-пиром: %s", peerinfo.ID.ShortString())
			}
		}()
	}
	wg.Wait()

	log.Println("📢 Анонсируемся в глобальной сети...")
	routingDiscovery := routing.NewRoutingDiscovery(dm.dht)
	// Используем Ticker для периодического анонсирования, чтобы оставаться видимыми
	ticker := time.NewTicker(time.Minute * 1)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-dm.ctx.Done():
				return
			case <-ticker.C:
				log.Println("📢 Повторно анонсируемся в сети...")
				_, err := routingDiscovery.Advertise(dm.ctx, DISCOVERY_TAG_DHT)
				if err != nil {
					log.Printf("⚠️ Ошибка повторного анонса: %v", err)
				}
			}
		}
	}()

	// Первоначальный анонс
	_, err := routingDiscovery.Advertise(dm.ctx, DISCOVERY_TAG_DHT)
	if err != nil {
		log.Printf("⚠️ Ошибка первоначального анонса: %v", err)
	}

	log.Println("🔍 Поиск участников в глобальной сети...")
	peerChan, err := routingDiscovery.FindPeers(dm.ctx, DISCOVERY_TAG_DHT)
	if err != nil {
		log.Printf("⚠️ Ошибка поиска в глобальной сети: %v", err)
		return
	}

	for p := range peerChan {
		if p.ID == dm.host.ID() {
			continue
		}
		log.Printf("🌐 Найден участник в глобальной сети: %s", p.ID.ShortString())
		dm.notifee.HandlePeerFound(p)
	}
}

// Stop останавливает все механизмы обнаружения
func (dm *DiscoveryManager) Stop() error {
	// Останавливаем mDNS
	if dm.mdnsService != nil {
		dm.mdnsService.Close()
	}

	// TODO: Здесь будет остановка DHT discovery

	return nil
}
