# NodeConfig - Конфигурация узла

## Обзор

`NodeConfig` - это структура конфигурации, которая позволяет полностью настраивать поведение libp2p узла. Все параметры из рабочего `poc.go` вынесены в конфигурируемую структуру.

## Структура NodeConfig

```go
type NodeConfig struct {
    // Транспорты
    EnableTCP              bool
    EnableQUIC             bool
    EnableWebSocket        bool
    EnableWebRTC           bool
    
    // Шифрование и безопасность
    EnableNoise            bool
    EnableTLS              bool
    
    // NAT и сетевая доступность
    EnableNATPortMap       bool
    EnableHolePunching     bool
    EnableAutoNATv2        bool
    ForceReachabilityPublic bool
    ForceReachabilityPrivate bool
    
    // Relay и авторелей
    EnableRelay            bool
    EnableAutoRelay        bool
    UseBootstrapAsRelay    bool
    EnableAutoRelayWithStaticRelays bool
    EnableAutoRelayWithPeerSource   bool
    AutoRelayBootDelay     time.Duration
    AutoRelayMaxCandidates int
    
    // Discovery
    EnableMDNS             bool
    EnableDHT              bool
    
    // Адреса для прослушивания
    ListenAddresses        []string
    
    // Статические релеи
    StaticRelays           []string
    
    // Таймауты для стримов
    StreamCreationTimeout  time.Duration
    StreamReadTimeout      time.Duration
    StreamWriteTimeout     time.Duration
    
    // Протокол
    ProtocolID             string
}
```

## API функции для работы с конфигурацией

### C-API

```c
// Запуск с дефолтным конфигом
int StartOwlWhisperWithDefaultConfig();

// Запуск с кастомным конфигом
int StartOwlWhisperWithCustomConfig(char* configJSON);

// Получение текущего конфига
char* GetCurrentNodeConfig();

// Обновление конфига
int UpdateNodeConfig(char* configJSON);
```

### Python примеры

```python
import ctypes
import json

# Загружаем библиотеку
lib = ctypes.CDLL('./libowlwhisper.so')

# Запуск с дефолтным конфигом
result = lib.StartOwlWhisperWithDefaultConfig()
if result == 0:
    print("✅ Узел запущен с дефолтным конфигом")

# Запуск с кастомным конфигом
custom_config = {
    "enableTCP": True,
    "enableQUIC": True,
    "enableWebSocket": False,  # Отключаем WebSocket
    "enableWebRTC": False,     # Отключаем WebRTC
    "enableNoise": True,
    "enableTLS": True,
    "enableNATPortMap": True,
    "enableHolePunching": True,
    "enableAutoNATv2": True,
    "enableRelay": True,
    "enableAutoRelay": True,
    "useBootstrapAsRelay": True,
    "autoRelayBootDelay": 2,
    "autoRelayMaxCandidates": 10,
    "enableMDNS": True,
    "enableDHT": True,
    "listenAddresses": [
        "/ip4/0.0.0.0/tcp/0",
        "/ip4/0.0.0.0/udp/0/quic-v1"
    ],
    "staticRelays": [
        "/dns4/relay.dev.svcs.d.foundation/tcp/443/wss/p2p/12D3KooWCKd2fU1g4k15u3J5i6pGk26h3g68d3amEa2S71G5v1jS"
    ],
    "forceReachabilityPublic": True,
    "forceReachabilityPrivate": False,
    "streamCreationTimeout": 60,
    "streamReadTimeout": 30,
    "streamWriteTimeout": 10,
    "enableAutoRelayWithStaticRelays": True,
    "enableAutoRelayWithPeerSource": True,
    "protocolID": "/p2p-chat/1.0.0"
}

config_json = json.dumps(custom_config)
result = lib.StartOwlWhisperWithCustomConfig(config_json.encode('utf-8'))
if result == 0:
    print("✅ Узел запущен с кастомным конфигом")

# Получение текущего конфига
config_ptr = lib.GetCurrentNodeConfig()
if config_ptr:
    current_config = ctypes.string_at(config_ptr).decode('utf-8')
    print(f"📋 Текущий конфиг: {current_config}")
    lib.FreeString(config_ptr)

# Обновление конфига на лету
updated_config = {
    "enableWebSocket": True,  # Включаем WebSocket
    "streamCreationTimeout": 120  # Увеличиваем таймаут
}
config_json = json.dumps(updated_config)
result = lib.UpdateNodeConfig(config_json.encode('utf-8'))
if result == 0:
    print("✅ Конфиг обновлен")
```

## Дефолтная конфигурация

```go
func DefaultNodeConfig() *NodeConfig {
    return &NodeConfig{
        EnableTCP:              true,
        EnableQUIC:             true,
        EnableWebSocket:        true,
        EnableWebRTC:           true,
        EnableNoise:            true,
        EnableTLS:              true,
        EnableNATPortMap:       true,
        EnableHolePunching:     true,
        EnableAutoNATv2:        true,
        EnableRelay:            true,
        EnableAutoRelay:        true,
        UseBootstrapAsRelay:    true,
        AutoRelayBootDelay:     2 * time.Second,
        AutoRelayMaxCandidates: 10,
        EnableMDNS:             true,
        EnableDHT:              true,
        ListenAddresses: []string{
            "/ip4/0.0.0.0/tcp/0",
            "/ip4/0.0.0.0/tcp/0/ws",
            "/ip4/0.0.0.0/udp/0/quic-v1",
            "/ip4/0.0.0.0/udp/0/webrtc-direct",
        },
        StaticRelays: []string{
            "/dns4/relay.dev.svcs.d.foundation/tcp/443/wss/p2p/12D3KooWCKd2fU1g4k15u3J5i6pGk26h3g68d3amEa2S71G5v1jS",
        },
        ForceReachabilityPublic:         true,
        ForceReachabilityPrivate:        false,
        StreamCreationTimeout:           60 * time.Second,
        StreamReadTimeout:               30 * time.Second,
        StreamWriteTimeout:              10 * time.Second,
        EnableAutoRelayWithStaticRelays: true,
        EnableAutoRelayWithPeerSource:   true,
        ProtocolID:                      "/p2p-chat/1.0.0",
    }
}
```

## Сценарии использования

### 1. Минимальная конфигурация (только TCP)
```json
{
    "enableTCP": true,
    "enableQUIC": false,
    "enableWebSocket": false,
    "enableWebRTC": false,
    "enableNoise": true,
    "enableTLS": true,
    "enableDHT": true,
    "enableMDNS": true,
    "listenAddresses": ["/ip4/0.0.0.0/tcp/0"]
}
```

### 2. Web-совместимая конфигурация
```json
{
    "enableTCP": true,
    "enableWebSocket": true,
    "enableWebRTC": true,
    "enableQUIC": false,
    "enableNoise": true,
    "enableTLS": true,
    "listenAddresses": [
        "/ip4/0.0.0.0/tcp/0",
        "/ip4/0.0.0.0/tcp/0/ws",
        "/ip4/0.0.0.0/udp/0/webrtc-direct"
    ]
}
```

### 3. Высокопроизводительная конфигурация
```json
{
    "enableTCP": true,
    "enableQUIC": true,
    "enableWebSocket": true,
    "enableWebRTC": true,
    "enableNoise": true,
    "enableTLS": true,
    "enableHolePunching": true,
    "enableAutoNATv2": true,
    "enableRelay": true,
    "enableAutoRelay": true,
    "streamCreationTimeout": 120,
    "streamReadTimeout": 60,
    "streamWriteTimeout": 30
}
```

### 4. Конфигурация для NAT
```json
{
    "enableTCP": true,
    "enableQUIC": true,
    "enableHolePunching": true,
    "enableAutoNATv2": true,
    "enableRelay": true,
    "enableAutoRelay": true,
    "forceReachabilityPublic": true,
    "staticRelays": [
        "/dns4/relay1.example.com/tcp/443/wss/p2p/12D3KooW...",
        "/dns4/relay2.example.com/tcp/443/wss/p2p/12D3KooW..."
    ]
}
```

## Важные параметры

### Транспорты
- **TCP**: Базовый транспорт, работает везде
- **QUIC**: Быстрый UDP-транспорт, обходит некоторые NAT
- **WebSocket**: Для web-приложений и прокси
- **WebRTC**: Для браузеров и P2P соединений

### Безопасность
- **Noise**: Современный протокол шифрования
- **TLS**: Стандартное шифрование для совместимости

### NAT Traversal
- **Hole Punching**: Прямые соединения через NAT
- **AutoNATv2**: Автоматическое определение типа NAT
- **Relay**: Обход NAT через промежуточные узлы

### Discovery
- **mDNS**: Локальный поиск в одной сети
- **DHT**: Глобальный поиск через распределенную таблицу

## Ограничения

1. **Конфигурация применяется только при запуске** - для изменения транспортов нужен перезапуск
2. **Некоторые параметры взаимозависимы** - например, WebRTC требует WebSocket
3. **Таймауты влияют на производительность** - слишком короткие могут вызывать ошибки

## Рекомендации

1. **Для локальной сети**: включите mDNS, отключите relay
2. **Для интернета**: включите все NAT traversal механизмы
3. **Для web**: используйте WebSocket и WebRTC
4. **Для производительности**: используйте QUIC и увеличивайте таймауты
