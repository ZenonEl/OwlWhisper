// Путь: cmd/fyne-gui/services/chat_service.go
package services

import (
	"fmt"
	"log"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	// "github.com/google/uuid" // Больше не нужен здесь
	// "google.golang.org/protobuf/proto" // Больше не нужен здесь!
)

// ChatService управляет бизнес-логикой, связанной с чатами.
type ChatService struct {
	core            newcore.ICoreController
	protocolService IProtocolService // <-- НОВАЯ ЗАВИСИМОСТЬ
	// cryptoService   ICryptoService   // <-- ДОБАВИМ В БУДУЩЕМ
	contactProvider ContactProvider
	onNewMessageUI  func(widget fyne.CanvasObject)
}

// ИЗМЕНЕН КОНСТРУКТОР
func NewChatService(core newcore.ICoreController, protoSvc IProtocolService, contactProvider ContactProvider, onNewMessageUI func(fyne.CanvasObject)) *ChatService {
	return &ChatService{
		core:            core,
		protocolService: protoSvc,
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
	// 1. Получаем профиль отправителя ("Me") для получения публичного ключа.
	sender, ok := cs.contactProvider.GetContactByPeerID(cs.core.GetMyPeerID())
	if !ok {
		return fmt.Errorf("не удалось найти собственный профиль")
	}
	authorIdentity := &protocol.IdentityPublicKey{
		KeyType:   protocol.KeyType_ED25519,
		PublicKey: []byte(sender.PeerID), // создать IdentityService и использовать реальный паблик кей
	}

	// 2. Создаем "содержимое" сообщения с помощью нашего нового сервиса.
	// Результат - это сериализованный `ChatContent`.
	plaintext, err := cs.protocolService.CreateChatContent_TextMessage(text)
	if err != nil {
		log.Printf("ERROR: [ChatService] Ошибка создания ChatContent: %v", err)
		return err
	}

	// 3. Шифруем содержимое.
	// !!! ВАЖНО: Сейчас здесь будет заглушка. Полноценное шифрование
	// мы добавим, когда будем реализовывать CryptoService и управление сессионными ключами.
	// Пока что мы просто передаем plaintext как "зашифрованный" текст.
	ciphertext := plaintext
	nonce := []byte("dummy-nonce-123") // Заглушка для nonce

	// 4. Получаем строковый тип для нашего `ChatContent`
	payloadType := cs.protocolService.GetPayloadType(&protocol.ChatContent{})

	// 5. Собираем финальный "конверт" (`SecureEnvelope`) с помощью сервиса.
	envelopeBytes, err := cs.protocolService.CreateSecureEnvelope(authorIdentity, payloadType, ciphertext, nonce)
	if err != nil {
		log.Printf("ERROR: [ChatService] Ошибка создания SecureEnvelope: %v", err)
		return err
	}

	// 6. Отправляем готовые байты через Core.
	log.Printf("INFO: [ChatService] Отправка TextMessage пиру %s", recipientID[:8])
	return cs.core.SendDataToPeer(recipientID, envelopeBytes)
}
