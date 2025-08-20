package core

import (
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
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
	mdnsService mdns.Service
	notifee     *DiscoveryNotifee
	ctx         context.Context
}

// NewDiscoveryManager создает новый менеджер обнаружения
func NewDiscoveryManager(ctx context.Context, node host.Host) *DiscoveryManager {
	notifee := &DiscoveryNotifee{
		node: node,
		ctx:  ctx,
	}

	// Создаем mDNS сервис
	mdnsService := mdns.NewMdnsService(node, DISCOVERY_TAG, notifee)

	return &DiscoveryManager{
		mdnsService: mdnsService,
		notifee:     notifee,
		ctx:         ctx,
	}
}

// Start запускает все механизмы обнаружения
func (dm *DiscoveryManager) Start() error {
	// Запускаем mDNS discovery
	if err := dm.mdnsService.Start(); err != nil {
		return fmt.Errorf("не удалось запустить mDNS: %w", err)
	}
	log.Println("📡 Сервис mDNS запущен. Идет поиск других участников...")

	// TODO: Здесь будет запуск DHT discovery для глобальной сети
	// go dm.startDHTDiscovery()

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

// TODO: Методы для DHT discovery будут добавлены позже
// func (dm *DiscoveryManager) startDHTDiscovery() {
//     // Логика подключения к bootstrap узлам и поиска через DHT
// }
