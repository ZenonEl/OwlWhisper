// Путь: cmd/fyne-gui/services/chat_service.go
package services

import (
	"fmt"
	"log"

	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	// "github.com/google/uuid" // Больше не нужен здесь
	// "google.golang.org/protobuf/proto" // Больше не нужен здесь!
)

// ChatService управляет бизнес-логикой, связанной с чатами.
type ChatService struct {
	sender          IMessageSender
	protocolService IProtocolService
	identityService IIdentityService
	contactProvider ContactProvider
	onNewMessageUI  func(widget fyne.CanvasObject)
}

// ИЗМЕНЕН КОНСТРУКТОР
func NewChatService(sender IMessageSender, protoSvc IProtocolService, idSvc IIdentityService, contactProvider ContactProvider, onNewMessageUI func(fyne.CanvasObject)) *ChatService {
	return &ChatService{
		sender:          sender,
		protocolService: protoSvc,
		identityService: idSvc,
		contactProvider: contactProvider,
		onNewMessageUI:  onNewMessageUI,
	}
}

// ProcessTextMessage не изменился, так как он уже работает с распарсенным сообщением.
// Изменится то, КАК он будет вызываться из главного обработчика событий.
func (cs *ChatService) ProcessTextMessage(senderID string, msg *protocol.TextMessage) {
	sender, ok := cs.contactProvider.GetContactByPeerID(senderID)
	senderName := senderID[:8]
	if ok {
		senderName = sender.Nickname
	}
	fullMessage := fmt.Sprintf("[%s]: %s", senderName, msg.Body)
	textWidget := widget.NewLabel(fullMessage)
	textWidget.Wrapping = fyne.TextWrapWord
	cs.onNewMessageUI(textWidget)
}

func (cs *ChatService) ProcessWidgetMessage(widget fyne.CanvasObject) {
	cs.onNewMessageUI(widget)
}

// ПОЛНОСТЬЮ ПЕРЕПИСАННЫЙ МЕТОД
func (cs *ChatService) SendTextMessage(recipientID string, text string) error {
	// 1. Получаем автора из IdentityService. Больше никаких "поисков себя".
	authorIdentity := cs.identityService.GetMyIdentityPublicKeyProto()
	if authorIdentity == nil {
		return fmt.Errorf("не удалось получить профиль отправителя")
	}

	// 2. Создаем "содержимое" сообщения
	plaintext, err := cs.protocolService.CreateChatContent_TextMessage(text)
	if err != nil {
		return fmt.Errorf("ошибка создания ChatContent: %w", err)
	}

	// 3. Шифруем (заглушка)
	ciphertext := plaintext
	nonce := []byte("dummy-nonce-chat")

	// 4. Собираем финальный конверт
	payloadType := cs.protocolService.GetPayloadType(&protocol.ChatContent{})
	envelopeBytes, err := cs.protocolService.CreateSecureEnvelope(authorIdentity, payloadType, ciphertext, nonce)
	if err != nil {
		return fmt.Errorf("ошибка создания SecureEnvelope: %w", err)
	}

	// 5. Отправляем
	log.Printf("INFO: [ChatService] Отправка TextMessage пиру %s", recipientID[:8])
	return cs.sender.SendSecureEnvelope(recipientID, envelopeBytes)
}
