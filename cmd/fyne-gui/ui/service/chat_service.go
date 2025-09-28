// Путь: cmd/fyne-gui/services/chat_service.go
package services

import (
	"fmt"
	"log"

	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
	encryption "OwlWhisper/cmd/fyne-gui/ui/service/encryption"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// ChatService управляет бизнес-логикой, связанной с чатами.
type ChatService struct {
	sender          IMessageSender
	protocolService IProtocolService
	identityService IIdentityService
	sessionService  ISessionService
	contactProvider ContactProvider
	onNewMessageUI  func(widget fyne.CanvasObject)
}

// ИЗМЕНЕН КОНСТРУКТОР
func NewChatService(sender IMessageSender, protoSvc IProtocolService, idSvc IIdentityService, sessionSvc ISessionService, contactProvider ContactProvider, onNewMessageUI func(fyne.CanvasObject)) *ChatService {
	return &ChatService{
		sender:          sender,
		protocolService: protoSvc,
		identityService: idSvc,
		sessionService:  sessionSvc,
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
// SendTextMessage шифрует и отправляет текстовое сообщение.
func (cs *ChatService) SendTextMessage(recipientID string, text string) error {
	authorIdentity := cs.identityService.GetMyIdentityPublicKeyProto()
	contextID := CreateContextIDForPeers(cs.identityService.GetMyPeerID(), recipientID)

	// 1. Создаем и сериализуем полезную нагрузку (ChatContent)
	plaintextBytes, err := cs.protocolService.CreateChatContent_TextMessage(text)
	if err != nil {
		return fmt.Errorf("ошибка создания ChatContent: %w", err)
	}

	// ================================================================= //
	//                      ЛОГ №1: ОТКРЫТЫЙ ТЕКСТ (ОТПРАВКА)            //
	// ================================================================= //
	log.Printf("DEBUG [ENCRYPT]: Открытый текст (сериализованный): %x", plaintextBytes)

	// 2. Шифруем сериализованный payload
	encryptedMsg, err := cs.sessionService.EncryptForSession(contextID, plaintextBytes)
	if err != nil {
		return fmt.Errorf("ошибка шифрования: %w", err)
	}

	// ================================================================= //
	//                      ЛОГ №2: ЗАШИФРОВАННЫЙ ТЕКСТ (ОТПРАВКА)       //
	// ================================================================= //
	log.Printf("DEBUG [ENCRYPT]: Зашифрованный текст (ciphertext): %x", encryptedMsg.Ciphertext)

	// 3. Упаковываем в SecureEnvelope
	payloadType := "encrypted/chat-v1" // Явный тип для зашифрованного чата
	envelopeBytes, err := cs.protocolService.CreateSecureEnvelope(authorIdentity, payloadType, encryptedMsg.Ciphertext, encryptedMsg.Nonce)
	if err != nil {
		return fmt.Errorf("ошибка создания SecureEnvelope: %w", err)
	}

	// 4. Отправляем
	return cs.sender.SendSecureEnvelope(recipientID, envelopeBytes)
}

// HandleEncryptedMessage вызывается из Dispatcher'а.
func (cs *ChatService) HandleEncryptedMessage(senderID string, envelope *protocol.SecureEnvelope) {
	contextID := CreateContextIDForPeers(cs.identityService.GetMyPeerID(), senderID)

	encryptedMsg := &encryption.EncryptedMessage{
		Ciphertext: envelope.Ciphertext,
		Nonce:      envelope.Nonce,
	}

	// ================================================================= //
	//                      ЛОГ №3: ЗАШИФРОВАННЫЙ ТЕКСТ (ПОЛУЧЕНИЕ)      //
	// ================================================================= //
	log.Printf("DEBUG [DECRYPT]: Получен зашифрованный текст (ciphertext): %x", encryptedMsg.Ciphertext)

	// 1. Расшифровываем
	plaintextBytes, err := cs.sessionService.DecryptForSession(contextID, encryptedMsg)
	if err != nil {
		log.Printf("WARN: [ChatService] Не удалось расшифровать сообщение от %s: %v", senderID, err)
		return
	}
	if plaintextBytes == nil {
		return // Сообщение поставлено в очередь
	}

	// ================================================================= //
	//                      ЛОГ №4: ОТКРЫТЫЙ ТЕКСТ (ПОЛУЧЕНИЕ)           //
	// ================================================================= //
	log.Printf("DEBUG [DECRYPT]: Открытый текст (сериализованный): %x", plaintextBytes)

	// 2. Распаковываем ChatContent
	chatContent, err := cs.protocolService.ParseChatContent(plaintextBytes)
	if err != nil {
		log.Printf("WARN: [ChatService] Не удалось распарсить ChatContent после расшифровки: %v", err)
		return
	}

	if textMsg := chatContent.GetText(); textMsg != nil {
		cs.ProcessTextMessage(senderID, textMsg)
	}
}
