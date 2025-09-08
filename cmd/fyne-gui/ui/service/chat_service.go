// Путь: cmd/fyne-gui/services/chat_service.go

package services

import (
	"fmt"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
)

// ChatService управляет бизнес-логикой, связанной с чатами.
type ChatService struct {
	core            newcore.ICoreController
	contactProvider ContactProvider               // Нужен для получения никнеймов
	onNewMessageUI  func(formattedMessage string) // Callback для обновления UI
}

func NewChatService(core newcore.ICoreController, contactProvider ContactProvider, onNewMessageUI func(string)) *ChatService {
	return &ChatService{
		core:            core,
		contactProvider: contactProvider,
		onNewMessageUI:  onNewMessageUI,
	}
}

// ProcessTextMessage обрабатывает входящее текстовое сообщение.
func (cs *ChatService) ProcessTextMessage(senderID string, msg *protocol.TextMessage) {
	sender, ok := cs.contactProvider.GetContactByPeerID(senderID)
	senderName := senderID[:8] // Имя по умолчанию - короткий PeerID
	if ok {
		senderName = sender.Nickname // Если контакт известен, используем никнейм
	}

	formattedMessage := fmt.Sprintf("[%s]: %s", senderName, msg.Body)

	// Вызываем callback, чтобы передать готовую строку в UI
	cs.onNewMessageUI(formattedMessage)

	// TODO: В будущем здесь будет логика сохранения сообщения в БД.
}

// SendTextMessage отправляет текстовое сообщение.
func (cs *ChatService) SendTextMessage(recipientID string, text string) error {
	// TODO: Реализовать логику отправки
	// 1. Создать Envelope с TextMessage
	// 2. Marshal
	// 3. core.SendDataToPeer
	return nil
}
