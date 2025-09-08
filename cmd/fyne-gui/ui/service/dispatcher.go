// Путь: cmd/fyne-gui/services/dispatcher.go

package services

import (
	"log"

	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"google.golang.org/protobuf/proto"
)

// MessageDispatcher десериализует и маршрутизирует входящие сообщения.
type MessageDispatcher struct {
	contactService *ContactService
	chatService    *ChatService
}

type AppUIProvider interface {
	OnProfileReceived(senderID string, profile *protocol.ProfileInfo)
	// В будущем здесь будут другие методы, например, OnContactRequestReceived
}

func NewMessageDispatcher(cs *ContactService, chs *ChatService) *MessageDispatcher {
	return &MessageDispatcher{
		contactService: cs,
		chatService:    chs,
	}
}

// HandleIncomingData - это главный метод, который принимает сырые байты от Core.
func (d *MessageDispatcher) HandleIncomingData(senderID string, data []byte) {
	// 1. Пытаемся десериализовать данные в наш главный "конверт".
	envelope := &protocol.Envelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		log.Printf("WARN: [Dispatcher] Не удалось распознать Envelope от %s: %v", senderID, err)
		return
	}

	// 2. ИЗМЕНЕНО: Проверяем, какой тип полезной нагрузки внутри:
	// сообщение для чата или для управления контактами.
	switch payload := envelope.Payload.(type) {
	case *protocol.Envelope_ChatMessage:
		// Это сообщение для чата, обрабатываем его контент.
		d.handleChatMessage(senderID, payload.ChatMessage)

	case *protocol.Envelope_ContactMessage:
		// Это сообщение для управления контактами.
		d.handleContactMessage(senderID, payload.ContactMessage)

	default:
		log.Printf("WARN: [Dispatcher] Получен Envelope с неизвестным типом payload от %s", senderID)
	}
}

// handleChatMessage обрабатывает сообщения, относящиеся к чатам.
func (d *MessageDispatcher) handleChatMessage(senderID string, msg *protocol.ChatMessage) {
	switch content := msg.Content.(type) {
	case *protocol.ChatMessage_Text:
		d.chatService.ProcessTextMessage(senderID, content.Text)

	case *protocol.ChatMessage_File:
		// TODO: Логика обработки файлов

	case *protocol.ChatMessage_ReadReceipts:
		// TODO: Логика обработки статусов прочтения
	}
}

// handleContactMessage обрабатывает сообщения для управления контактами.
func (d *MessageDispatcher) handleContactMessage(senderID string, msg *protocol.ContactMessage) {
	switch typ := msg.Type.(type) {
	case *protocol.ContactMessage_ProfileRequest:
		// Нас "пингуют" с запросом профиля. Нужно ответить.
		log.Printf("INFO: [Dispatcher] Получен ProfileRequest от %s", senderID)
		d.contactService.RespondToProfileRequest(senderID)

	case *protocol.ContactMessage_ProfileResponse:
		log.Printf("INFO: [Dispatcher] Получен ProfileResponse от %s", senderID)
		d.contactService.HandleProfileResponse(senderID, typ.ProfileResponse)

	case *protocol.ContactMessage_ContactRequest:
		d.contactService.HandleContactRequest(senderID, typ.ContactRequest)

	case *protocol.ContactMessage_ContactAccept:
		// НОВОЕ: Обрабатываем подтверждение дружбы
		d.contactService.HandleContactAccept(senderID, typ.ContactAccept)
	}
}
