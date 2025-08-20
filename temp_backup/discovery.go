package core

import (
	"context"
	"fmt"
	"log"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

// DISCOVERY_TAG - "секретное слово" для поиска участников через mDNS
const DISCOVERY_TAG = "owl-whisper-mdns"

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
	ctx              context.Context
}

// NewDiscoveryManager создает новый менеджер обнаружения
func NewDiscoveryManager(ctx context.Context, node host.Host) *DiscoveryManager {
	notifee := &DiscoveryNotifee{
		node: node,
		ctx:  ctx,
	}

	// Создаем mDNS сервис
	mdnsService := mdns.NewMdnsService(node, DISCOVERY_TAG, notifee)

	// Создаем DHT
	kadDHT, err := dht.New(ctx, node)
	if err != nil {
		log.Printf("⚠️ Не удалось создать DHT: %v", err)
	} else {
		log.Printf("✅ DHT создан")
	}

	// Создаем routing discovery
	var routingDiscovery *routing.RoutingDiscovery
	if kadDHT != nil {
		routingDiscovery = routing.NewRoutingDiscovery(kadDHT)
		log.Printf("✅ Routing discovery создан")
	}

	return &DiscoveryManager{
		mdnsService:      mdnsService,
		dht:              kadDHT,
		routingDiscovery: routingDiscovery,
		notifee:          notifee,
		ctx:              ctx,
	}
}

// Start запускает все механизмы обнаружения
func (dm *DiscoveryManager) Start() error {
	// Запускаем mDNS discovery
	if err := dm.mdnsService.Start(); err != nil {
		return fmt.Errorf("не удалось запустить mDNS: %w", err)
	}
	log.Println("📡 Сервис mDNS запущен. Идет поиск других участников...")

	// Запускаем DHT discovery для глобальной сети
	if dm.dht != nil && dm.routingDiscovery != nil {
		go dm.startDHTDiscovery()
		log.Println("🌐 DHT discovery запущен для глобальной сети")
	}

	return nil
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

// startDHTDiscovery запускает поиск через DHT
func (dm *DiscoveryManager) startDHTDiscovery() {
	// Подключаемся к bootstrap узлам
	log.Println("🌐 Подключение к bootstrap узлам...")
	if err := dm.dht.Bootstrap(dm.ctx); err != nil {
		log.Printf("⚠️ Не удалось подключиться к bootstrap узлам: %v", err)
		return
	}
	log.Println("✅ Bootstrap завершен")

	// Ждем немного для стабилизации
	time.Sleep(2 * time.Second)

	// Анонсируемся в глобальной сети
	ttl, err := dm.routingDiscovery.Advertise(dm.ctx, "owl-whisper-global-rendezvous")
	if err != nil {
		log.Printf("⚠️ Не удалось анонсироваться в глобальной сети: %v", err)
	} else {
		log.Printf("📢 Анонсировались в глобальной сети, TTL: %v", ttl)
	}

	// Начинаем поиск других участников
	log.Println("🔍 Поиск участников в глобальной сети...")
	peerChan, err := dm.routingDiscovery.FindPeers(dm.ctx, "owl-whisper-global-rendezvous")
	if err != nil {
		log.Printf("⚠️ Ошибка поиска в глобальной сети: %v", err)
		return
	}

	// Обрабатываем найденных пиров
	for p := range peerChan {
		if p.ID == dm.notifee.node.ID() {
			continue // Пропускаем себя
		}
		log.Printf("🌐 Найден участник в глобальной сети: %s", p.ID.ShortString())

		// Передаем найденного пира в notifee для подключения
		dm.notifee.HandlePeerFound(p)
	}
}
