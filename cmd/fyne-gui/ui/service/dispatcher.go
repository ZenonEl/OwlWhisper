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

func NewMessageDispatcher(contactService *ContactService, chatService *ChatService) *MessageDispatcher {
	return &MessageDispatcher{
		contactService: contactService,
		chatService:    chatService,
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

	// 2. Проверяем, какой тип полезной нагрузки (payload) внутри конверта.
	// Это и есть наша "сортировка".
	switch pld := envelope.Payload.(type) {
	case *protocol.Envelope_Content:
		// Это сообщение с контентом (текст, файл и т.д.)
		content := pld.Content
		if textMsg := content.GetText(); textMsg != nil {
			// Если внутри текст, передаем его в ChatService
			d.chatService.ProcessTextMessage(senderID, textMsg)
		}
		// TODO: В будущем здесь будет обработка файлов (content.GetFile())

	case *protocol.Envelope_ProfileRequest:
		// Нас "пингуют" с запросом профиля.
		log.Printf("INFO: [Dispatcher] Получен ProfileRequest от %s", senderID)
		d.contactService.RespondToProfileRequest(senderID)

	case *protocol.Envelope_ProfileResponse:
		// Нам прислали ответ с профилем.
		log.Printf("INFO: [Dispatcher] Получен ProfileResponse от %s", senderID)
		d.contactService.HandleProfileResponse(senderID, pld.ProfileResponse)

	case *protocol.Envelope_ReadReceipts:
		// TODO: Обработка уведомлений о прочтении

	default:
		log.Printf("WARN: [Dispatcher] Получен Envelope с неизвестным типом payload от %s", senderID)
	}
}
