// cmd/fyne-gui/new-core/config.go

package newcore

import "time"

type Reachability string

const (
	// ReachabilityPublic - узел считает, что у него "белый" IP.
	ReachabilityPublic Reachability = "public"
	// ReachabilityPrivate - узел считает, что он за NAT.
	ReachabilityPrivate Reachability = "private"
	// ReachabilityUnknown - libp2p сам определит достижимость (рекомендуется).
	ReachabilityUnknown Reachability = "unknown"
)

// Config определяет все настраиваемые параметры для запуска узла OwlWhisper.
// Эта структура позволит нам гибко управлять поведением узла без изменения кода.
type Config struct {
	// --- Основные Модули ---

	// EnableMDNS включает обнаружение пиров в локальной сети (LAN).
	// Очень быстрый и эффективный способ найти "соседей".
	EnableMDNS bool

	// EnableDHT включает обнаружение пиров в глобальной сети (WAN).
	// Необходимо для связи через интернет.
	EnableDHT bool

	// --- Настройки NAT и Relay ---

	// EnableAutoRelay включает автоматическое использование Relay-серверов,
	// если узел определяет, что он находится за строгим NAT. Критически важно для мобильных сетей.
	EnableAutoRelay bool

	// EnableHolePunching включает попытки "пробивания" NAT напрямую.
	EnableHolePunching bool

	// EnableNATPortMap запрашивает у роутера "пробросить" порт (используя UPnP/NAT-PMP).
	EnableNATPortMap bool

	EnableAutoNATv2 bool

	// --- Настройки Транспортов ---
	// Позволяют выборочно отключать протоколы для отладки или специфичных окружений.
	EnableTCP       bool
	EnableQUIC      bool
	EnableWebSocket bool
	EnableWebRTC    bool

	// --- Адреса и Точки Встречи ---

	// ListenAddresses - список адресов, которые узел будет пытаться слушать.
	// "0.0.0.0" означает прослушивание на всех сетевых интерфейсах. "0" для порта означает "выбери случайный".
	ListenAddresses []string

	// CustomBootstrapNodes - список пользовательских bootstrap-узлов, которые будут
	// добавлены к стандартному списку libp2p.
	CustomBootstrapNodes []string

	// StaticRelays - список статических, доверенных Relay-серверов.
	// AutoRelay будет пытаться использовать их в первую очередь.
	StaticRelays []string

	// RendezvousString - это "общая комната" в DHT, где клиенты Owl Whisper могут
	// найти друг друга для первоначального знакомства.
	// ВАЖНО: Это НЕ используется для поиска по конкретному никнейму.
	RendezvousString string

	// --- Политика Достижимости ---
	ForceReachability Reachability // Принудительная установка статуса "за NAT" или "публичный".

	// --- Тонкая Настройка (для продвинутых пользователей) ---

	// AutoRelayBootDelay - задержка перед тем, как AutoRelay начнет свою работу.
	AutoRelayBootDelay time.Duration
	// AutoRelayMaxCandidates - сколько кандидатов в релеи будет проверять AutoRelay.
	AutoRelayMaxCandidates int

	// Период, с которым мы повторно анонсируем себя в сети.
	AnnounceInterval time.Duration
}

// DefaultConfig возвращает рекомендуемую конфигурацию "по умолчанию",
// которая в точности повторяет нашу рабочую "ультимативную" конфигурацию из PoC.
func DefaultConfig() Config {
	return Config{
		EnableMDNS: true, // Включаем по умолчанию, это очень полезно.
		EnableDHT:  true,

		EnableAutoRelay:    true,
		EnableHolePunching: true,
		EnableNATPortMap:   true,
		EnableAutoNATv2:    true,

		EnableTCP:       true,
		EnableQUIC:      true,
		EnableWebSocket: true,
		EnableWebRTC:    true,

		ListenAddresses: []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/tcp/0/ws",
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip4/0.0.0.0/udp/0/webrtc-direct",
		},

		StaticRelays: []string{
			"/dns4/relay.dev.svcs.d.foundation/tcp/443/wss/p2p/12D3KooWCKd2fU1g4k15u3J5i6pGk26h3g68d3amEa2S71G5v1jS",
		},

		RendezvousString: "owl-whisper-rendezvous-v1",

		// По умолчанию, позволяем libp2p самому определять, за NAT мы или нет.
		ForceReachability: ReachabilityUnknown,

		// Параметры из PoC
		AutoRelayBootDelay:     2 * time.Second,
		AutoRelayMaxCandidates: 10,
		AnnounceInterval:       15 * time.Second, // Агрессивное анонсирование для быстрой отладки
	}
}
