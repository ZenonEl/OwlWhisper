// Путь: cmd/fyne-gui/services/protocol_service.go
package services

import (
	"fmt"

	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// IProtocolService определяет интерфейс для работы с Protobuf-сообщениями.
// Это "фабрика" и "парсер", скрывающая всю сложность сериализации.
type IProtocolService interface {
	// --- Методы для создания сообщений ---

	// CreateSecureEnvelope собирает внешний конверт для конфиденциальных данных.
	// Принимает уже зашифрованное тело сообщения.
	CreateSecureEnvelope(author *protocol.IdentityPublicKey, payloadType string, ciphertext, nonce []byte) ([]byte, error)

	// CreateChatContent_TextMessage создает внутренний, незашифрованный контент текстового сообщения.
	// GUI должен будет зашифровать результат перед передачей в CreateSecureEnvelope.
	CreateChatContent_TextMessage(text string) ([]byte, error)

	// --- Методы для парсинга сообщений ---

	// ParseSecureEnvelope разбирает внешний конверт.
	// Не расшифровывает данные, просто извлекает зашифрованный ciphertext.
	ParseSecureEnvelope(data []byte) (*protocol.SecureEnvelope, error)

	// ParseChatContent разбирает расшифрованное тело сообщения.
	ParseChatContent(data []byte) (*protocol.ChatContent, error)

	// GetPayloadType возвращает строковое имя типа для Protobuf сообщения.
	GetPayloadType(msg protoreflect.ProtoMessage) string

	ParseFileControl(data []byte) (*protocol.FileControl, error)
	ParseSignalingMessage(data []byte) (*protocol.SignalingMessage, error)

	// CreateSignedCommand собирает полностью готовую к отправке команду.
	CreateSignedCommand(author *protocol.IdentityPublicKey, commandData []byte, signature []byte) ([]byte, error)

	// CreateCommand_InitiateContext создает внутреннюю команду для начала нового чата.
	CreateCommand_InitiateContext(contextID string, seqNum uint64, initialMembers []*protocol.IdentityPublicKey, senderProfile *protocol.ProfilePayload, ephemeralKey []byte, chosenSuite string) ([]byte, error)

	// ParseSignedCommand разбирает внешний конверт команды.
	// Не проверяет подпись, это задача CryptoService.
	ParseSignedCommand(data []byte) (*protocol.SignedCommand, error)

	// ParseCommand разбирает внутреннюю, подписанную часть команды.
	ParseCommand(data []byte) (*protocol.Command, error)

	CreatePing_ProfileRequest(senderIdentity *protocol.IdentityPublicKey, senderPeerID string) ([]byte, error)
	CreateCommand_DiscloseProfile(contextID string, seqNum uint64, profile *protocol.ProfilePayload) ([]byte, error)
	ParsePingEnvelope(data []byte) (*protocol.PingEnvelope, error)
	// НОВЫЙ МЕТОД
	CreateCommand_AcknowledgeContext(contextID string, seqNum uint64, senderProfile *protocol.ProfilePayload, ephemeralKey []byte) ([]byte, error)

	// --- НОВЫЕ МЕТОДЫ ДЛЯ FILE TRANSFER ---
	CreateChatContent_FileMetadata(metadata *protocol.FileMetadata) ([]byte, error)
	CreateFileControl_DownloadRequest(transferID string) ([]byte, error)

	CreateSignaling_Offer(callID, sdp string) ([]byte, error)
	CreateSignaling_Answer(callID, sdp string) ([]byte, error)
	CreateSignaling_Candidate(callID, candidate string) ([]byte, error)
	CreateSignaling_Hangup(callID string, reason protocol.CallHangup_Reason) ([]byte, error)

	CreateFileControl_ChunkAck(transferID string, offset int64) ([]byte, error)
	ParseFileData(data []byte) (*protocol.FileData, error)
}

// protocolService - конкретная реализация IProtocolService.
type protocolService struct{}

// NewProtocolService - конструктор нашего сервиса.
func NewProtocolService() *protocolService {
	return &protocolService{}
}

// --- Реализация методов ---

func (ps *protocolService) CreateSecureEnvelope(author *protocol.IdentityPublicKey, payloadType string, ciphertext, nonce []byte) ([]byte, error) {
	envelope := &protocol.SecureEnvelope{
		AuthorIdentity: author,
		PayloadType:    payloadType,
		Ciphertext:     ciphertext,
		Nonce:          nonce,
	}

	return proto.Marshal(envelope)
}

func (ps *protocolService) CreateChatContent_TextMessage(text string) ([]byte, error) {
	textMsg := &protocol.TextMessage{
		Body: text,
	}
	content := &protocol.ChatContent{
		Payload: &protocol.ChatContent_Text{Text: textMsg},
	}
	return proto.Marshal(content)
}

func (ps *protocolService) ParseSecureEnvelope(data []byte) (*protocol.SecureEnvelope, error) {
	envelope := &protocol.SecureEnvelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		return nil, fmt.Errorf("ошибка десериализации SecureEnvelope: %w", err)
	}
	return envelope, nil
}

func (ps *protocolService) ParseChatContent(data []byte) (*protocol.ChatContent, error) {
	content := &protocol.ChatContent{}
	if err := proto.Unmarshal(data, content); err != nil {
		return nil, fmt.Errorf("ошибка десериализации ChatContent: %w", err)
	}
	return content, nil
}

func (ps *protocolService) GetPayloadType(msg protoreflect.ProtoMessage) string {
	return string(msg.ProtoReflect().Descriptor().FullName())
}

func (ps *protocolService) ParseFileControl(data []byte) (*protocol.FileControl, error) {
	content := &protocol.FileControl{}
	if err := proto.Unmarshal(data, content); err != nil {
		return nil, fmt.Errorf("ошибка десериализации FileControl: %w", err)
	}
	return content, nil
}

func (ps *protocolService) ParseSignalingMessage(data []byte) (*protocol.SignalingMessage, error) {
	content := &protocol.SignalingMessage{}
	if err := proto.Unmarshal(data, content); err != nil {
		return nil, fmt.Errorf("ошибка десериализации SignalingMessage: %w", err)
	}
	return content, nil
}

func (ps *protocolService) CreateSignedCommand(author *protocol.IdentityPublicKey, commandData []byte, signature []byte) ([]byte, error) {
	signedCmd := &protocol.SignedCommand{
		AuthorIdentity: author,
		CommandData:    commandData,
		Signature:      signature,
	}
	return proto.Marshal(signedCmd)
}

func (ps *protocolService) CreateCommand_InitiateContext(contextID string, seqNum uint64, initialMembers []*protocol.IdentityPublicKey, senderProfile *protocol.ProfilePayload, ephemeralKey []byte, chosenSuite string) ([]byte, error) {
	initiate := &protocol.InitiateContext{
		InitialMembers:     initialMembers,
		SenderProfile:      senderProfile,
		EphemeralPublicKey: ephemeralKey,
		ChosenCryptoSuite:  chosenSuite,
	}
	cmd := &protocol.Command{
		ContextId:      contextID,
		SequenceNumber: seqNum,
		Payload: &protocol.Command_InitiateContext{
			InitiateContext: initiate,
		},
	}
	return proto.Marshal(cmd)
}

func (ps *protocolService) ParseSignedCommand(data []byte) (*protocol.SignedCommand, error) {
	signedCmd := &protocol.SignedCommand{}
	if err := proto.Unmarshal(data, signedCmd); err != nil {
		return nil, fmt.Errorf("ошибка десериализации SignedCommand: %w", err)
	}
	return signedCmd, nil
}

func (ps *protocolService) ParseCommand(data []byte) (*protocol.Command, error) {
	cmd := &protocol.Command{}
	if err := proto.Unmarshal(data, cmd); err != nil {
		return nil, fmt.Errorf("ошибка десериализации Command: %w", err)
	}
	return cmd, nil
}

func (ps *protocolService) CreatePing_ProfileRequest(senderIdentity *protocol.IdentityPublicKey, senderPeerID string) ([]byte, error) {
	req := &protocol.ProfileRequest{
		SenderIdentity: senderIdentity,
		SenderPeerId:   senderPeerID,
	}
	envelope := &protocol.PingEnvelope{
		Payload: &protocol.PingEnvelope_ProfileRequest{ProfileRequest: req},
	}
	return proto.Marshal(envelope)
}

func (ps *protocolService) CreateCommand_DiscloseProfile(contextID string, seqNum uint64, profile *protocol.ProfilePayload) ([]byte, error) {
	disclose := &protocol.DiscloseProfile{
		Profile: profile,
	}
	cmd := &protocol.Command{
		ContextId:      contextID,
		SequenceNumber: seqNum,
		Payload: &protocol.Command_DiscloseProfile{
			DiscloseProfile: disclose,
		},
	}
	return proto.Marshal(cmd)
}

func (ps *protocolService) ParsePingEnvelope(data []byte) (*protocol.PingEnvelope, error) {
	envelope := &protocol.PingEnvelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		return nil, fmt.Errorf("ошибка десериализации PingEnvelope: %w", err)
	}
	return envelope, nil
}

func (ps *protocolService) CreateCommand_AcknowledgeContext(contextID string, seqNum uint64, senderProfile *protocol.ProfilePayload, ephemeralKey []byte) ([]byte, error) {
	ack := &protocol.AcknowledgeContext{
		SenderProfile:      senderProfile,
		EphemeralPublicKey: ephemeralKey,
	}
	cmd := &protocol.Command{
		ContextId:      contextID,
		SequenceNumber: seqNum,
		Payload: &protocol.Command_AcknowledgeContext{
			AcknowledgeContext: ack,
		},
	}
	return proto.Marshal(cmd)
}

func (ps *protocolService) CreateChatContent_FileMetadata(metadata *protocol.FileMetadata) ([]byte, error) {
	content := &protocol.ChatContent{
		Payload: &protocol.ChatContent_File{File: metadata},
	}
	return proto.Marshal(content)
}

func (ps *protocolService) CreateFileControl_DownloadRequest(transferID string) ([]byte, error) {
	req := &protocol.FileDownloadRequest{
		TransferId: transferID,
	}
	control := &protocol.FileControl{
		Payload: &protocol.FileControl_Request{Request: req},
	}
	return proto.Marshal(control)
}

func (ps *protocolService) CreateSignaling_Offer(callID, sdp string) ([]byte, error) {
	msg := &protocol.SignalingMessage{
		CallId:  callID,
		Payload: &protocol.SignalingMessage_Offer{Offer: &protocol.CallOffer{Sdp: sdp}},
	}
	return proto.Marshal(msg)
}

func (ps *protocolService) CreateSignaling_Answer(callID, sdp string) ([]byte, error) {
	msg := &protocol.SignalingMessage{
		CallId:  callID,
		Payload: &protocol.SignalingMessage_Answer{Answer: &protocol.CallAnswer{Sdp: sdp}},
	}
	return proto.Marshal(msg)
}

func (ps *protocolService) CreateSignaling_Candidate(callID, candidate string) ([]byte, error) {
	msg := &protocol.SignalingMessage{
		CallId:  callID,
		Payload: &protocol.SignalingMessage_Candidate{Candidate: &protocol.ICECandidate{Candidate: candidate}},
	}
	return proto.Marshal(msg)
}

func (ps *protocolService) CreateSignaling_Hangup(callID string, reason protocol.CallHangup_Reason) ([]byte, error) {
	msg := &protocol.SignalingMessage{
		CallId:  callID,
		Payload: &protocol.SignalingMessage_Hangup{Hangup: &protocol.CallHangup{Reason: reason}},
	}
	return proto.Marshal(msg)
}

func (ps *protocolService) CreateFileControl_ChunkAck(transferID string, offset int64) ([]byte, error) {
	ack := &protocol.FileChunkAck{
		TransferId:         transferID,
		AcknowledgedOffset: offset,
	}
	control := &protocol.FileControl{
		Payload: &protocol.FileControl_Ack{Ack: ack},
	}
	return proto.Marshal(control)
}

func (ps *protocolService) ParseFileData(data []byte) (*protocol.FileData, error) {
	fileData := &protocol.FileData{}
	if err := proto.Unmarshal(data, fileData); err != nil {
		return nil, fmt.Errorf("ошибка десериализации FileData: %w", err)
	}
	return fileData, nil
}
