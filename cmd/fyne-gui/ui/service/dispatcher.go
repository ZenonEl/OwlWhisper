// Путь: cmd/fyne-gui/services/dispatcher.go
package services

import (
	"log"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
	encryption "OwlWhisper/cmd/fyne-gui/ui/service/encryption"
)

// MessageDispatcher десериализует и маршрутизирует входящие сообщения.
type MessageDispatcher struct {
	protocolService IProtocolService
	contactService  *ContactService
	sessionService  ISessionService
	chatService     *ChatService
	fileService     *FileService
	callService     *CallService
}

// Конструктор остается прежним
func NewMessageDispatcher(protoSvc IProtocolService, sessionSvc ISessionService, cs *ContactService, chs *ChatService, fs *FileService, cls *CallService) *MessageDispatcher {
	return &MessageDispatcher{
		protocolService: protoSvc,
		sessionService:  sessionSvc,
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

// handleSecureEnvelope расшифровывает и маршрутизирует конфиденциальные данные.
func (d *MessageDispatcher) handleSecureEnvelope(senderID string, envelope *protocol.SecureEnvelope) {
	log.Printf("DEBUG [Dispatcher]: Обработка SecureEnvelope. PayloadType: '%s'", envelope.PayloadType)
	contextID := CreateContextIDForPeers(d.contactService.identityService.GetMyPeerID(), senderID)

	encryptedMsg := &encryption.EncryptedMessage{
		Ciphertext: envelope.Ciphertext,
		Nonce:      envelope.Nonce,
	}

	// 1. Централизованно расшифровываем сообщение
	plaintextBytes, err := d.sessionService.DecryptForSession(contextID, encryptedMsg)
	if err != nil {
		log.Printf("WARN: [Dispatcher] Не удалось расшифровать SecureEnvelope от %s: %v", senderID, err)
		return
	}
	if plaintextBytes == nil {
		return
	} // Сообщение в очереди

	// 2. Используем PayloadType для однозначной маршрутизации
	switch envelope.PayloadType {
	case "encrypted/chat-v1":
		// Это точно содержимое чата (текст или анонс файла)
		if chatContent, err := d.protocolService.ParseChatContent(plaintextBytes); err == nil {
			d.handleChatContent(senderID, chatContent)
		} else {
			log.Printf("WARN: [Dispatcher] Ошибка парсинга ChatContent: %v", err)
		}

	case "encrypted/file-control-v1":
		// Это точно управляющая команда для файлов (запрос или ACK)
		if fileControl, err := d.protocolService.ParseFileControl(plaintextBytes); err == nil {
			d.handleFileControl(senderID, fileControl)
		} else {
			log.Printf("WARN: [Dispatcher] Ошибка парсинга FileControl: %v", err)
		}

	default:
		log.Printf("WARN: [Dispatcher] Получен SecureEnvelope с неизвестным PayloadType '%s'", envelope.PayloadType)
	}
}

// handleChatContent маршрутизирует сообщения, относящиеся к содержимому чата.
func (d *MessageDispatcher) handleChatContent(senderID string, content *protocol.ChatContent) {
	switch payload := content.Payload.(type) {
	case *protocol.ChatContent_Text:
		d.chatService.ProcessTextMessage(senderID, payload.Text)

	case *protocol.ChatContent_File:
		// Вызываем FileService, чтобы он создал виджет, и передаем виджет в ChatService
		card, err := d.fileService.HandleFileAnnouncement(senderID, payload.File)
		if err == nil && card != nil {
			d.chatService.ProcessWidgetMessage(card)
		}
	}
}

// handleFileControl маршрутизирует команды для управления передачей файлов.
func (d *MessageDispatcher) handleFileControl(senderID string, content *protocol.FileControl) {
	log.Printf("DEBUG [Dispatcher]: Обработка FileControl. Тип: %T", content.Payload)
	switch payload := content.Payload.(type) {
	case *protocol.FileControl_Request:
		d.fileService.HandleDownloadRequest(payload.Request, senderID)

	case *protocol.FileControl_Ack:
		d.fileService.HandleChunkAck(payload.Ack)
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
