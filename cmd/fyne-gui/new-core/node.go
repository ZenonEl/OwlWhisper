// cmd/fyne-gui/new-core/node.go

package newcore

import (
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	webrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"

	// Импортируем все необходимые транспорты
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

// Node представляет собой полностью инкапсулированный узел libp2p.
type Node struct {
	host   host.Host
	cfg    Config // Храним копию конфигурации, с которой был запущен узел
	pubsub *pubsub.PubSub
}

// NewNode создает, конфигурирует и запускает новый узел OwlWhisper.
// Это главная "фабрика", которая собирает все вместе.
func NewNode(ctx context.Context, privKey crypto.PrivKey, cfg Config) (*Node, error) {
	// 1. Начинаем собирать опции для libp2p.New()
	opts := []libp2p.Option{
		// Идентичность: используем переданный ключ для постоянного PeerID
		libp2p.Identity(privKey),
		// Адреса прослушивания: берем из конфигурации
		libp2p.ListenAddrStrings(cfg.ListenAddresses...),
		// Безопасность: включаем и TLS, и Noise для максимальной совместимости
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(tls.ID, tls.New),
	}

	// 2. Динамически добавляем транспорты на основе конфигурации
	if cfg.EnableTCP {
		opts = append(opts, libp2p.Transport(tcp.NewTCPTransport))
	}
	if cfg.EnableQUIC {
		opts = append(opts, libp2p.Transport(quic.NewTransport))
	}
	if cfg.EnableWebSocket {
		opts = append(opts, libp2p.Transport(ws.New))
	}
	if cfg.EnableWebRTC {
		opts = append(opts, libp2p.Transport(webrtc.New))
	}
	// 3. Динамически добавляем механизмы обхода NAT
	if cfg.EnableNATPortMap {
		opts = append(opts, libp2p.NATPortMap())
	}
	if cfg.EnableHolePunching {
		opts = append(opts, libp2p.EnableHolePunching())
	}
	if cfg.EnableAutoNATv2 {
		opts = append(opts, libp2p.EnableAutoNATv2())
	}
	// Включаем Relay (он нужен для AutoRelay)
	if cfg.EnableAutoRelay {
		opts = append(opts, libp2p.EnableRelay())
	}

	// 4. Динамически настраиваем AutoRelay
	if cfg.EnableAutoRelay {
		// ИСПРАВЛЕНО: Возвращаем "ультимативную" и рабочую конфигурацию из PoC
		opts = append(opts, libp2p.EnableAutoRelayWithPeerSource(func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
			r := make(chan peer.AddrInfo)
			go func() {
				defer close(r)
				// Используем стандартные bootstrap-узлы как потенциальные ретрансляторы
				for _, pi := range dht.DefaultBootstrapPeers {
					addrInfo, err := peer.AddrInfoFromP2pAddr(pi)
					if err != nil {
						continue
					}
					select {
					case r <- *addrInfo:
					case <-ctx.Done():
						return
					}
				}
			}()
			return r
		}, autorelay.WithBootDelay(cfg.AutoRelayBootDelay),
			autorelay.WithMaxCandidates(cfg.AutoRelayMaxCandidates)))
	}

	// --- ШАГ 5: Устанавливаем политику достижимости ---
	switch cfg.ForceReachability {
	case ReachabilityPublic:
		opts = append(opts, libp2p.ForceReachabilityPublic())
	case ReachabilityPrivate:
		opts = append(opts, libp2p.ForceReachabilityPrivate())
	case ReachabilityUnknown:
		// Ничего не делаем, libp2p сам определит
	}

	// --- ФИНАЛ: Создаем узел libp2p со всеми собранными опциями ---
	host, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать libp2p узел: %w", err)
	}

	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать GossipSub: %w", err)
	}

	node := &Node{
		host:   host,
		cfg:    cfg,
		pubsub: ps, // Сохраняем экземпляр PubSub
	}

	return node, nil
}

// Host возвращает нижележащий libp2p хост.
// Это позволяет другим частям нашего core (например, DiscoveryManager)
// взаимодействовать с ним.
func (n *Node) Host() host.Host {
	return n.host
}

func (n *Node) PubSub() *pubsub.PubSub {
	return n.pubsub
}

// Close корректно останавливает узел.
func (n *Node) Close() error {
	return n.host.Close()
}

// SetStreamHandler регистрирует обработчик для нашего протокола.
func (n *Node) SetStreamHandler(protocolID protocol.ID, handler network.StreamHandler) {
	n.host.SetStreamHandler(protocolID, handler)
}

// --- Вспомогательные функции ---

// parseAddrInfo - это helper-функция для безопасного парсинга
// строковых multi-адресов в формат AddrInfo.
func parseAddrInfo(addrs []string) []peer.AddrInfo {
	var addrInfos []peer.AddrInfo
	for _, addrStr := range addrs {
		addrInfo, err := peer.AddrInfoFromString(addrStr)
		if err != nil {
			log.Printf("Предупреждение: не удалось распарсить адрес '%s': %v\n", addrStr, err)
			continue
		}
		addrInfos = append(addrInfos, *addrInfo)
	}
	return addrInfos
}
