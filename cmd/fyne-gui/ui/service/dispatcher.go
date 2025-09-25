// Путь: cmd/fyne-gui/services/dispatcher.go
package services

import (
	"log"

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

// ПОЛНОСТЬЮ ОБНОВЛЕННЫЙ МЕТОД
func (d *MessageDispatcher) HandleIncomingData(senderID string, data []byte) {
	// Попытка №1: SecureEnvelope (конфиденциальные данные чата)
	envelope, err := d.protocolService.ParseSecureEnvelope(data)
	if err == nil {
		d.handleSecureEnvelope(senderID, envelope)
		return
	}

	// Попытка №2: SignedCommand (управляющие команды)
	signedCmd, err := d.protocolService.ParseSignedCommand(data)
	if err == nil {
		d.handleSignedCommand(senderID, signedCmd)
		return
	}

	// Попытка №3: SignalingMessage (WebRTC сигнализация)
	signalingMsg, err := d.protocolService.ParseSignalingMessage(data)
	if err == nil {
		log.Printf("INFO: [Dispatcher] Получено SignalingMessage от %s", senderID)
		d.callService.HandleSignalingMessage(senderID, signalingMsg)
		return
	}

	// Если ничего не подошло - логируем ошибку.
	log.Printf("WARN: [Dispatcher] Не удалось распознать входящее сообщение от %s", senderID)
}

// НОВЫЙ МЕТОД: Обработка и маршрутизация SignedCommand
func (d *MessageDispatcher) handleSignedCommand(senderID string, signedCmd *protocol.SignedCommand) {
	innerCmd, err := d.protocolService.ParseCommand(signedCmd.CommandData)
	if err != nil {
		log.Printf("WARN: [Dispatcher] Не удалось распарсить CommandData от %s: %v", senderID, err)
		return
	}

	// Маршрутизируем команду, передавая конкретный payload в сервис
	switch payload := innerCmd.Payload.(type) {
	case *protocol.Command_InitiateContext:
		log.Printf("INFO: [Dispatcher] Получена команда InitiateContext от %s", senderID)
		// ИСПРАВЛЕНО: Передаем конкретный payload `payload.InitiateContext`
		d.contactService.HandleInitiateContext(senderID, signedCmd, payload.InitiateContext)

	// --- МЕСТО ДЛЯ БУДУЩИХ КОМАНД ---
	case *protocol.Command_AddMembers:
		log.Printf("INFO: [Dispatcher] Получена команда AddMembers от %s", senderID)
		// d.groupService.HandleAddMembers(senderID, signedCmd, payload.AddMembers)
	case *protocol.Command_RemoveMembers:
		log.Printf("INFO: [Dispatcher] Получена команда RemoveMembers от %s", senderID)
		// d.groupService.HandleRemoveMembers(senderID, signedCmd, payload.RemoveMembers)

	// TODO: Добавить case для DiscloseProfile, когда мы добавим его в .proto
	// case *protocol.Command_DiscloseProfile:
	// 	 log.Printf("INFO: [Dispatcher] Получена команда DiscloseProfile от %s", senderID)
	// 	 d.contactService.HandleDiscloseProfile(senderID, signedCmd, payload.DiscloseProfile)

	default:
		log.Printf("WARN: [Dispatcher] Получена SignedCommand с неизвестным типом payload от %s", senderID)
	}
}

// Метод handleSecureEnvelope и его помощники остаются без изменений.
// ... (handleSecureEnvelope, handleChatContent, handleFileControl) ...
// Путь: cmd/fyne-gui/services/dispatcher.go
func (d *MessageDispatcher) handleSecureEnvelope(senderID string, envelope *protocol.SecureEnvelope) {
	// !!! ЗАГЛУШКА ДЛЯ РАСШИФРОВКИ !!!
	// В будущем здесь будет вызов cryptoService.Decrypt(envelope.Ciphertext)
	// А пока считаем, что ciphertext - это и есть plaintext.
	plaintext := envelope.Ciphertext

	// Маршрутизируем в зависимости от типа полезной нагрузки
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

func (d *MessageDispatcher) handleChatContent(senderID string, content *protocol.ChatContent) {
	switch payload := content.Payload.(type) {
	case *protocol.ChatContent_Text:
		d.chatService.ProcessTextMessage(senderID, payload.Text)

	case *protocol.ChatContent_File:
		log.Printf("INFO: [Dispatcher] Получен анонс файла (FileMetadata) от %s", senderID)
		card, err := d.fileService.HandleFileAnnouncement(senderID, payload.File)
		if err == nil && card != nil {
			d.chatService.ProcessWidgetMessage(card)
		}

	case *protocol.ChatContent_Receipts:
		// TODO: Обработка статусов прочтения
	case *protocol.ChatContent_Edit:
		// TODO: Обработка редактирования
	case *protocol.ChatContent_Delete:
		// TODO: Обработка удаления
	}
}

func (d *MessageDispatcher) handleFileControl(senderID string, content *protocol.FileControl) {
	switch payload := content.Payload.(type) {
	case *protocol.FileControl_Request:
		log.Printf("INFO: [Dispatcher] Получен запрос на скачивание файла от %s", senderID)
		d.fileService.HandleDownloadRequest(payload.Request, senderID)

	case *protocol.FileControl_Status:
		// TODO: Обработка статусов (файл недоступен и т.д.)

	case *protocol.FileControl_SeederUpdate:
		// TODO: Обработка появления нового сида
	}
}
