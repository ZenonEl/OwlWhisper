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

	// EventManager для отправки событий статуса сети
	eventManager *EventManager
}

// NewDiscoveryManager создает новый менеджер обнаружения
func NewDiscoveryManager(ctx context.Context, host host.Host, onPeer func(peer.AddrInfo), eventManager *EventManager) (*DiscoveryManager, error) {
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
		eventManager:     eventManager,
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

// SaveDHTRoutingTable сохраняет DHT routing table в кэш
func (dm *DiscoveryManager) SaveDHTRoutingTable(persistence *PersistenceManager) error {
	if dm.dht == nil {
		return fmt.Errorf("DHT недоступен")
	}

	// Получаем все пиры из DHT
	peers := dm.dht.RoutingTable().ListPeers()

	Info("💾 Сохраняем DHT routing table: %d пиров", len(peers))

	for _, peerID := range peers {
		// Получаем адреса пира
		addrs := dm.host.Peerstore().Addrs(peerID)
		var addrStrings []string
		for _, addr := range addrs {
			addrStrings = append(addrStrings, addr.String())
		}

		// Определяем, является ли пир "здоровым" (есть адреса)
		healthy := len(addrStrings) > 0

		// Сохраняем в кэш
		if err := persistence.SavePeerToCache(peerID, addrStrings, healthy); err != nil {
			Warn("⚠️ Не удалось сохранить пира %s в кэш: %v", peerID.ShortString(), err)
		}
	}

	Info("✅ DHT routing table сохранена в кэш")
	return nil
}

// LoadDHTRoutingTableFromCache загружает DHT routing table из кэша
func (dm *DiscoveryManager) LoadDHTRoutingTableFromCache(persistence *PersistenceManager) error {
	if dm.dht == nil {
		return fmt.Errorf("DHT недоступен")
	}

	// Загружаем кэшированных пиров
	cachedPeers, err := persistence.GetHealthyCachedPeers()
	if err != nil {
		return fmt.Errorf("не удалось загрузить кэшированных пиров: %w", err)
	}

	if len(cachedPeers) == 0 {
		Info("💾 Кэш пиров пуст, используем только bootstrap узлы")
		return nil
	}

	Info("💾 Загружаем DHT routing table из кэша: %d пиров", len(cachedPeers))

	// Добавляем кэшированных пиров в DHT routing table
	for _, cachedPeer := range cachedPeers {
		peerID, err := peer.Decode(cachedPeer.PeerID)
		if err != nil {
			Warn("⚠️ Неверный Peer ID в кэше: %s", cachedPeer.PeerID)
			continue
		}

		// Пытаемся подключиться к кэшированному пиру
		if err := dm.host.Connect(dm.ctx, peer.AddrInfo{ID: peerID}); err != nil {
			Warn("⚠️ Не удалось подключиться к кэшированному пиру %s: %v", peerID.ShortString(), err)
		} else {
			Info("✅ Подключились к кэшированному пиру %s", peerID.ShortString())
		}
	}

	Info("✅ DHT routing table загружена из кэша")
	return nil
}

// GetRoutingTableStats возвращает статистику DHT routing table
func (dm *DiscoveryManager) GetRoutingTableStats() map[string]interface{} {
	if dm.dht == nil {
		return map[string]interface{}{
			"status": "dht_unavailable",
		}
	}

	rt := dm.dht.RoutingTable()
	peers := rt.ListPeers()

	stats := map[string]interface{}{
		"total_peers": len(peers),
		"size":        rt.Size(),
	}

	return stats
}

// fallbackToCachedPeers пытается подключиться к кэшированным пирам при недоступности bootstrap
func (dm *DiscoveryManager) fallbackToCachedPeers() error {
	// Создаем временный PersistenceManager для доступа к кэшу
	persistence, err := NewPersistenceManager()
	if err != nil {
		return fmt.Errorf("не удалось создать PersistenceManager: %w", err)
	}

	// Загружаем здоровых кэшированных пиров
	cachedPeers, err := persistence.GetHealthyCachedPeers()
	if err != nil {
		return fmt.Errorf("не удалось загрузить кэшированных пиров: %w", err)
	}

	if len(cachedPeers) == 0 {
		return fmt.Errorf("кэш пиров пуст")
	}

	Info("🔄 Пытаемся подключиться к %d кэшированным пирам...", len(cachedPeers))

	// Пытаемся подключиться к кэшированным пирам
	connectedCount := 0
	for _, cachedPeer := range cachedPeers {
		peerID, err := peer.Decode(cachedPeer.PeerID)
		if err != nil {
			Warn("⚠️ Неверный Peer ID в кэше: %s", cachedPeer.PeerID)
			continue
		}

		// Пытаемся подключиться
		if err := dm.host.Connect(dm.ctx, peer.AddrInfo{ID: peerID}); err != nil {
			Warn("⚠️ Не удалось подключиться к кэшированному пиру %s: %v", peerID.ShortString(), err)
		} else {
			Info("✅ Подключились к кэшированному пиру %s", peerID.ShortString())
			connectedCount++
		}
	}

	if connectedCount > 0 {
		Info("✅ Успешно подключились к %d кэшированным пирам", connectedCount)
		return nil
	}

	return fmt.Errorf("не удалось подключиться ни к одному кэшированному пиру")
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
	// Отправляем событие о начале подключения к DHT
	if dm.eventManager != nil {
		event := NetworkStatusEvent("CONNECTING_TO_DHT", "Подключение к bootstrap-узлам...")
		dm.eventManager.PushEvent(event)
	}

	Info("🌐 Подключение к bootstrap узлам...")

	// ИЗМЕНЕНИЕ: Ждем, пока мы подключимся хотя бы к одному bootstrap-пиру.
	// Это гарантирует, что наша таблица не пуста перед анонсом.
	var wg sync.WaitGroup
	bootstrapConnected := false

	for _, p := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(p)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dm.host.Connect(dm.ctx, *peerinfo); err != nil {
				// Info("Не удалось подключиться к bootstrap-пиру: %s", err)
			} else {
				Info("✅ Установлено соединение с bootstrap-пиром: %s", peerinfo.ID.ShortString())
				bootstrapConnected = true
			}
		}()
	}
	wg.Wait()

	// Если не удалось подключиться к bootstrap узлам, пробуем кэшированные пиры
	if !bootstrapConnected {
		Info("⚠️ Не удалось подключиться к bootstrap узлам, пробуем кэшированные пиры...")
		if err := dm.fallbackToCachedPeers(); err != nil {
			Warn("⚠️ Не удалось подключиться к кэшированным пирам: %v", err)
		}
	}

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

	// Отправляем событие о готовности сети
	if dm.eventManager != nil {
		event := NetworkStatusEvent("NETWORK_READY", "Готов к работе в сети")
		dm.eventManager.PushEvent(event)
	}
}

// GetRoutingDiscovery возвращает routing discovery для внутреннего использования
func (dm *DiscoveryManager) GetRoutingDiscovery() *routing.RoutingDiscovery {
	return dm.routingDiscovery
}

// StartAggressiveDiscovery запускает агрессивный поиск пиров (как в poc.go)
func (dm *DiscoveryManager) StartAggressiveDiscovery(rendezvous string) {
	Info("🚀 Запуск агрессивного поиска пиров по rendezvous: %s", rendezvous)

	go func() {
		for {
			select {
			case <-dm.ctx.Done():
				return
			default:
				peerChan, err := dm.routingDiscovery.FindPeers(dm.ctx, rendezvous)
				if err != nil {
					Warn("Ошибка поиска пиров: %v", err)
					time.Sleep(10 * time.Second)
					continue
				}

				for p := range peerChan {
					if p.ID == dm.host.ID() {
						continue
					}
					Info("Найден пир: %s. Адреса: %v", p.ID, p.Addrs)
					dm.notifee.HandlePeerFound(p)
				}

				time.Sleep(15 * time.Second) // Повторяем поиск каждые 15 секунд
			}
		}
	}()

	Info("Поиск пиров запущен. Ожидание...")
}

// StartAggressiveAdvertising запускает агрессивное анонсирование (как в poc.go)
func (dm *DiscoveryManager) StartAggressiveAdvertising(rendezvous string) {
	Info("🚀 Запуск агрессивного анонсирования по rendezvous: %s", rendezvous)

	go func() {
		for {
			select {
			case <-dm.ctx.Done():
				return
			default:
				_, err := dm.routingDiscovery.Advertise(dm.ctx, rendezvous)
				if err != nil {
					Warn("Ошибка анонсирования: %v", err)
				} else {
					Info("🔄 Анонсировано в DHT: %s", rendezvous)
				}
				time.Sleep(15 * time.Second) // Повторяем каждые 15 секунд
			}
		}
	}()

	Info("Анонсирование запущено. Ожидание...")
}

// FindPeersOnce выполняет однократный поиск пиров
func (dm *DiscoveryManager) FindPeersOnce(rendezvous string) ([]peer.AddrInfo, error) {
	Info("🔍 Однократный поиск пиров по rendezvous: %s", rendezvous)

	peerChan, err := dm.routingDiscovery.FindPeers(dm.ctx, rendezvous)
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска пиров: %w", err)
	}

	var peers []peer.AddrInfo
	for p := range peerChan {
		if p.ID == dm.host.ID() {
			continue
		}
		peers = append(peers, p)
		Info("Найден пир: %s. Адреса: %v", p.ID, p.Addrs)
	}

	return peers, nil
}

// AdvertiseOnce выполняет однократное анонсирование
func (dm *DiscoveryManager) AdvertiseOnce(rendezvous string) error {
	Info("📢 Однократное анонсирование по rendezvous: %s", rendezvous)

	_, err := dm.routingDiscovery.Advertise(dm.ctx, rendezvous)
	if err != nil {
		return fmt.Errorf("ошибка анонсирования: %w", err)
	}

	Info("📢 Анонсировано в DHT: %s", rendezvous)
	return nil
}
