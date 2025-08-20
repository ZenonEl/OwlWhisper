# OwlWhisper API

Публичный API для P2P мессенджера OwlWhisper.

## Быстрый старт

```go
package main

import (
    "log"
    "OwlWhisper/api"
)

func main() {
    // Создаем API
    owlAPI, err := api.NewOwlWhisperAPI(nil)
    if err != nil {
        log.Fatal(err)
    }

    // Запускаем
    if err := owlAPI.Start(); err != nil {
        log.Fatal(err)
    }
    defer owlAPI.Stop()

    // Отправляем сообщение
    err = owlAPI.SendMessage(api.SendMessageRequest{
        Text:     "Привет, мир!",
        ChatType: "broadcast",
    })
    if err != nil {
        log.Printf("Ошибка: %v", err)
    }
}
```

## Основные компоненты

### OwlWhisperAPI

Главный интерфейс платформы:

```go
type OwlWhisperAPI interface {
    Start() error
    Stop() error
    SendMessage(request SendMessageRequest) error
    GetPeers() []Peer
    GetConnectionStatus() ConnectionStatus
    GetHistory(limit int) (ChatHistory, error)
    MessageChannel() <-chan Message
    PeerChannel() <-chan []Peer
    GetMyPeerID() string
}
```

### Основные структуры

**Message** - сообщение:
```go
type Message struct {
    ID          string
    Sender      string
    Text        string
    Timestamp   time.Time
    ChatType    string
    RecipientID string
    IsOutgoing  bool
}
```

**Peer** - участник сети:
```go
type Peer struct {
    ID       string
    Nickname string
    Status   string
    LastSeen time.Time
}
```

**ConnectionStatus** - статус подключения:
```go
type ConnectionStatus struct {
    IsConnected bool
    PeerCount   int
    MyPeerID    string
    LastUpdate  time.Time
    NetworkType string
}
```

## Примеры использования

### Обработка сообщений

```go
go func() {
    for msg := range owlAPI.MessageChannel() {
        if msg.IsOutgoing {
            fmt.Printf("Отправлено: %s\n", msg.Text)
        } else {
            fmt.Printf("Получено от %s: %s\n", msg.Sender, msg.Text)
        }
    }
}()
```

### Мониторинг пиров

```go
go func() {
    for peers := range owlAPI.PeerChannel() {
        fmt.Printf("Подключено пиров: %d\n", len(peers))
        for _, peer := range peers {
            fmt.Printf("- %s (%s)\n", peer.Nickname, peer.Status)
        }
    }
}()
```

### Отправка сообщений

```go
// Broadcast сообщение
err := owlAPI.SendMessage(api.SendMessageRequest{
    Text:     "Сообщение для всех",
    ChatType: "broadcast",
})

// Приватное сообщение
err := owlAPI.SendMessage(api.SendMessageRequest{
    Text:        "Приватное сообщение",
    ChatType:    "private",
    RecipientID: "12D3KooW...",
})
```

### Получение истории

```go
history, err := owlAPI.GetHistory(50)
if err == nil {
    for _, msg := range history.Messages {
        fmt.Printf("%s - %s: %s\n", 
            msg.Timestamp.Format("15:04:05"), 
            msg.Sender, 
            msg.Text)
    }
}
```

## Конфигурация

```go
config := &api.APIConfig{
    EnableTUI:      false,           // Отключить TUI
    DatabasePath:   "custom.db",     // Путь к БД
    LogLevel:       "debug",         // Уровень логов
    MaxMessageSize: 8192,            // Макс. размер сообщения
    HistoryLimit:   200,             // Лимит истории
}

owlAPI, err := api.NewOwlWhisperAPI(config)
```

## Архитектура

```
┌─────────────────┐
│   GUI Clients   │ ← Ваши приложения
└─────────────────┘
         │
┌─────────────────┐
│   OwlWhisper    │ ← Этот API
│      API        │
└─────────────────┘
         │
┌─────────────────┐
│   Core Engine   │ ← P2P движок
│   (libp2p)      │
└─────────────────┘
```

API предоставляет стабильный интерфейс для любых UI клиентов, скрывая сложность P2P сети.