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
	CreateCommand_InitiateContext(contextID string, seqNum uint64, initialMembers []*protocol.IdentityPublicKey) ([]byte, error)

	// ParseSignedCommand разбирает внешний конверт команды.
	// Не проверяет подпись, это задача CryptoService.
	ParseSignedCommand(data []byte) (*protocol.SignedCommand, error)

	// ParseCommand разбирает внутреннюю, подписанную часть команды.
	ParseCommand(data []byte) (*protocol.Command, error)

	CreatePing_ProfileRequest(senderIdentity *protocol.IdentityPublicKey, senderPeerID string) ([]byte, error)
	CreateCommand_DiscloseProfile(contextID string, seqNum uint64, profile *protocol.ProfilePayload) ([]byte, error)
	ParsePingEnvelope(data []byte) (*protocol.PingEnvelope, error)
}

// protocolService - конкретная реализация IProtocolService.
type protocolService struct{}

// NewProtocolService - конструктор нашего сервиса.
func NewProtocolService() IProtocolService {
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

func (ps *protocolService) CreateCommand_InitiateContext(contextID string, seqNum uint64, initialMembers []*protocol.IdentityPublicKey) ([]byte, error) {
	initiate := &protocol.InitiateContext{
		InitialMembers: initialMembers,
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
