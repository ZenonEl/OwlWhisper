// Путь: cmd/fyne-gui/services/dispatcher.go
package services

import (
	"log"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
)

// MessageDispatcher десериализует и маршрутизирует входящие сообщения.
type MessageDispatcher struct {
	protocolService IProtocolService
	contactService  *ContactService
	chatService     *ChatService
	fileService     *FileService
	callService     *CallService
}

// Конструктор остается прежним
func NewMessageDispatcher(protoSvc IProtocolService, cs *ContactService, chs *ChatService, fs *FileService, cls *CallService) *MessageDispatcher {
	return &MessageDispatcher{
		protocolService: protoSvc,
		contactService:  cs,
		chatService:     chs,
		fileService:     fs,
		callService:     cls,
	}
}

// HandleIncomingData определяет тип входящего сообщения и передает его соответствующему обработчику.
func (d *MessageDispatcher) HandleIncomingData(senderID string, msgType newcore.MessageType, data []byte) {
	log.Printf("DEBUG [RECEIVER]: Получено сообщение типа %d от %s. Длина: %d байт.", msgType, senderID, len(data))

	switch msgType {
	case newcore.MsgTypeSecureEnvelope:
		if envelope, err := d.protocolService.ParseSecureEnvelope(data); err == nil {
			d.handleSecureEnvelope(senderID, envelope)
		} else {
			log.Printf("WARN: [Dispatcher] Ошибка парсинга SecureEnvelope: %v", err)
		}

	case newcore.MsgTypeSignedCommand:
		if signedCmd, err := d.protocolService.ParseSignedCommand(data); err == nil {
			d.handleSignedCommand(senderID, signedCmd)
		} else {
			log.Printf("WARN: [Dispatcher] Ошибка парсинга SignedCommand: %v", err)
		}

	case newcore.MsgTypePingEnvelope:
		if ping, err := d.protocolService.ParsePingEnvelope(data); err == nil {
			d.handlePingEnvelope(senderID, ping)
		} else {
			log.Printf("WARN: [Dispatcher] Ошибка парсинга PingEnvelope: %v", err)
		}

	case newcore.MsgTypeSignaling:
		if signalingMsg, err := d.protocolService.ParseSignalingMessage(data); err == nil {
			d.handleSignalingMessage(senderID, signalingMsg)
		} else {
			log.Printf("WARN: [Dispatcher] Ошибка парсинга SignalingMessage: %v", err)
		}

	default:
		log.Printf("WARN: [Dispatcher] Получено сообщение неизвестного типа от %s", senderID)
	}
}

// handleSignedCommand распаковывает и маршрутизирует криптографически подписанные команды.
func (d *MessageDispatcher) handleSignedCommand(senderID string, signedCmd *protocol.SignedCommand) {
	innerCmd, err := d.protocolService.ParseCommand(signedCmd.CommandData)
	if err != nil {
		log.Printf("WARN: [Dispatcher] Не удалось распарсить CommandData от %s: %v", senderID, err)
		return
	}

	switch payload := innerCmd.Payload.(type) {
	// --- Команды для ContactService ---

	// ИСПРАВЛЕНИЕ: Раскомментируем этот блок
	case *protocol.Command_DiscloseProfile:
		log.Printf("INFO: [Dispatcher] Получена команда DiscloseProfile от %s", senderID)
		d.contactService.HandleDiscloseProfile(senderID, signedCmd, payload.DiscloseProfile)

	case *protocol.Command_InitiateContext:
		log.Printf("INFO: [Dispatcher] Получена команда InitiateContext от %s", senderID)
		d.contactService.HandleInitiateContext(senderID, signedCmd, payload.InitiateContext)

	case *protocol.Command_AcknowledgeContext:
		log.Printf("INFO: [Dispatcher] Получена команда AcknowledgeContext от %s", senderID)
		d.contactService.HandleAcknowledgeContext(senderID, signedCmd, payload.AcknowledgeContext)

	// --- Команды для будущего GroupService ---
	case *protocol.Command_AddMembers:
		log.Printf("INFO: [Dispatcher] Получена команда AddMembers от %s", senderID)
	case *protocol.Command_RemoveMembers:
		log.Printf("INFO: [Dispatcher] Получена команда RemoveMembers от %s", senderID)

	default:
		log.Printf("WARN: [Dispatcher] Получена SignedCommand с неизвестным типом payload от %s, %s", senderID, innerCmd.Payload)
	}
}

// handlePingEnvelope обрабатывает простые "пинговые" сообщения.
func (d *MessageDispatcher) handlePingEnvelope(senderID string, ping *protocol.PingEnvelope) {
	switch payload := ping.Payload.(type) {
	case *protocol.PingEnvelope_ProfileRequest:
		log.Printf("INFO: [Dispatcher] Получен ProfileRequest от %s", senderID)
		// Передаем запрос в ContactService, чтобы он мог на него ответить
		d.contactService.HandlePingRequest(senderID, payload.ProfileRequest)
	default:
		log.Printf("WARN: [Dispatcher] Получен PingEnvelope с неизвестным типом payload от %s", senderID)
	}
}

// handleSecureEnvelope расшифровывает (пока заглушка) и маршрутизирует конфиденциальные данные.
func (d *MessageDispatcher) handleSecureEnvelope(senderID string, envelope *protocol.SecureEnvelope) {
	// !!! ЗАГЛУШКА ДЛЯ РАСШИФРОВКИ !!!
	plaintext := envelope.Ciphertext

	switch envelope.PayloadType {
	case "protocol.ChatContent":
		content, err := d.protocolService.ParseChatContent(plaintext)
		if err != nil {
			log.Printf("WARN: [Dispatcher] Ошибка парсинга ChatContent от %s: %v", senderID, err)
			return
		}
		d.handleChatContent(senderID, content)

	case "protocol.FileControl":
		content, err := d.protocolService.ParseFileControl(plaintext)
		if err != nil {
			log.Printf("WARN: [Dispatcher] Ошибка парсинга FileControl от %s: %v", senderID, err)
			return
		}
		d.handleFileControl(senderID, content)

	default:
		log.Printf("WARN: [Dispatcher] Получен SecureEnvelope с неизвестным PayloadType '%s' от %s", envelope.PayloadType, senderID)
	}
}

// handleChatContent маршрутизирует сообщения, относящиеся к содержимому чата.
func (d *MessageDispatcher) handleChatContent(senderID string, content *protocol.ChatContent) {
	switch payload := content.Payload.(type) {
	case *protocol.ChatContent_Text:
		d.chatService.ProcessTextMessage(senderID, payload.Text)
	case *protocol.ChatContent_File:
		card, err := d.fileService.HandleFileAnnouncement(senderID, payload.File)
		if err == nil && card != nil {
			d.chatService.ProcessWidgetMessage(card)
		}
		// ... другие типы содержимого чата ...
	}
}

// handleFileControl маршрутизирует команды для управления передачей файлов.
func (d *MessageDispatcher) handleFileControl(senderID string, content *protocol.FileControl) {
	switch payload := content.Payload.(type) {
	case *protocol.FileControl_Request:
		d.fileService.HandleDownloadRequest(payload.Request, senderID)
		// ... другие команды управления файлами ...
	}
}

func (d *MessageDispatcher) handleSignalingMessage(senderID string, msg *protocol.SignalingMessage) {
	log.Printf("INFO: [Dispatcher] Получено SignalingMessage (CallID: %s)", msg.CallId)

	switch payload := msg.Payload.(type) {
	case *protocol.SignalingMessage_Offer:
		d.callService.HandleIncomingOffer(senderID, msg.CallId, payload.Offer)
	case *protocol.SignalingMessage_Answer:
		d.callService.HandleIncomingAnswer(senderID, msg.CallId, payload.Answer)
	case *protocol.SignalingMessage_Candidate:
		d.callService.HandleIncomingICECandidate(senderID, msg.CallId, payload.Candidate)
	case *protocol.SignalingMessage_Hangup:
		d.callService.HandleIncomingHangup(senderID, msg.CallId, payload.Hangup)
	default:
		log.Printf("WARN: [Dispatcher] Получено SignalingMessage с неизвестным типом payload от %s", senderID)
	}
}
