// Путь: cmd/fyne-gui/services/chat_service.go

package services

import (
	"fmt"
	"log"
	"time"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
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

// SendTextMessage создает, сериализует и отправляет текстовое сообщение.
func (cs *ChatService) SendTextMessage(recipientID string, text string) error {
	// 1. Получаем ID отправителя. Нам нужен полный PeerID, а не "Me".
	sender, ok := cs.contactProvider.GetContactByPeerID(cs.core.GetMyPeerID())
	if !ok {
		// Этого никогда не должно произойти, так как "Me" всегда в контактах.
		return fmt.Errorf("не удалось найти собственный профиль")
	}

	// 2. Создаем Protobuf-сообщение
	textMsg := &protocol.TextMessage{
		Body: text,
	}
	chatMsg := &protocol.ChatMessage{
		ChatType: protocol.ChatMessage_PRIVATE,
		ChatId:   recipientID, // В приватном чате ID чата - это ID собеседника
		Content:  &protocol.ChatMessage_Text{Text: textMsg},
	}
	envelope := &protocol.Envelope{
		MessageId:     uuid.New().String(),
		SenderId:      sender.PeerID,
		TimestampUnix: time.Now().Unix(),
		Payload:       &protocol.Envelope_ChatMessage{ChatMessage: chatMsg},
	}

	// 3. Сериализуем в байты
	data, err := proto.Marshal(envelope)
	if err != nil {
		log.Printf("ERROR: [ChatService] Ошибка Marshal при создании TextMessage: %v", err)
		return err
	}

	// 4. Отправляем через Core
	log.Printf("INFO: [ChatService] Отправка TextMessage пиру %s", recipientID[:8])
	return cs.core.SendDataToPeer(recipientID, data)
}
