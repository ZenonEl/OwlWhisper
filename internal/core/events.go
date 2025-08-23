package core

import (
	"fmt"
	"sync"
	"time"
)

// EventType определяет тип события
type EventType string

const (
	// События сообщений
	EventTypeNewMessage EventType = "NewMessage"
	
	// События подключения пиров
	EventTypePeerConnected    EventType = "PeerConnected"
	EventTypePeerDisconnected EventType = "PeerDisconnected"
	
	// События статуса сети
	EventTypeNetworkStatus EventType = "NetworkStatus"
)

// Event представляет событие, которое будет отправлено клиенту
type Event struct {
	Type      EventType   `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp int64       `json:"timestamp"` // Unix timestamp
}

// NewMessagePayload содержит данные о новом сообщении
type NewMessagePayload struct {
	SenderID string `json:"senderID"` // PeerID отправителя
	Data     []byte `json:"data"`     // Protobuf данные
}

// PeerEventPayload содержит данные о событии пира
type PeerEventPayload struct {
	PeerID string `json:"peerID"` // PeerID пира
}

// NetworkStatusPayload содержит статус сети
type NetworkStatusPayload struct {
	Status  string `json:"status"`  // Статус (CONNECTING_TO_DHT, NETWORK_READY, etc.)
	Message string `json:"message"` // Описание статуса
}

// EventManager управляет очередью событий
type EventManager struct {
	eventQueue chan Event
	mu         sync.RWMutex
	running    bool
}

// NewEventManager создает новый менеджер событий
func NewEventManager(queueSize int) *EventManager {
	return &EventManager{
		eventQueue: make(chan Event, queueSize),
		running:    true,
	}
}

// PushEvent добавляет событие в очередь
func (em *EventManager) PushEvent(event Event) error {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	if !em.running {
		return fmt.Errorf("EventManager остановлен")
	}
	
	// Устанавливаем timestamp если не задан
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}
	
	select {
	case em.eventQueue <- event:
		return nil
	default:
		return fmt.Errorf("очередь событий переполнена")
	}
}

// GetNextEvent блокирующе получает следующее событие из очереди
func (em *EventManager) GetNextEvent() (Event, error) {
	em.mu.RLock()
	if !em.running {
		em.mu.RUnlock()
		return Event{}, fmt.Errorf("EventManager остановлен")
	}
	em.mu.RUnlock()
	
	select {
	case event := <-em.eventQueue:
		return event, nil
	case <-time.After(30 * time.Second): // Таймаут для предотвращения бесконечного ожидания
		return Event{}, fmt.Errorf("таймаут ожидания события")
	}
}

// Stop останавливает EventManager
func (em *EventManager) Stop() {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	if em.running {
		em.running = false
		close(em.eventQueue)
	}
}

// IsRunning проверяет, работает ли EventManager
func (em *EventManager) IsRunning() bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.running
}

// QueueSize возвращает текущий размер очереди
func (em *EventManager) QueueSize() int {
	return len(em.eventQueue)
}

// Вспомогательные функции для создания событий

// NewMessageEvent создает событие нового сообщения
func NewMessageEvent(senderID string, data []byte) Event {
	return Event{
		Type: EventTypeNewMessage,
		Payload: NewMessagePayload{
			SenderID: senderID,
			Data:     data,
		},
		Timestamp: time.Now().Unix(),
	}
}

// PeerConnectedEvent создает событие подключения пира
func PeerConnectedEvent(peerID string) Event {
	return Event{
		Type: EventTypePeerConnected,
		Payload: PeerEventPayload{
			PeerID: peerID,
		},
		Timestamp: time.Now().Unix(),
	}
}

// PeerDisconnectedEvent создает событие отключения пира
func PeerDisconnectedEvent(peerID string) Event {
	return Event{
		Type: EventTypePeerDisconnected,
		Payload: PeerEventPayload{
			PeerID: peerID,
		},
		Timestamp: time.Now().Unix(),
	}
}

// NetworkStatusEvent создает событие статуса сети
func NetworkStatusEvent(status, message string) Event {
	return Event{
		Type: EventTypeNetworkStatus,
		Payload: NetworkStatusPayload{
			Status:  status,
			Message: message,
		},
		Timestamp: time.Now().Unix(),
	}
} 