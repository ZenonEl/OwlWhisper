// Путь: cmd/fyne-gui/services/sender_service.go
package services

import (
	newcore "OwlWhisper/cmd/fyne-gui/new-core"
)

// IMessageSender определяет интерфейс для отправки типизированных сообщений.
type IMessageSender interface {
	SendSecureEnvelope(peerID string, data []byte) error
	SendSignedCommand(peerID string, data []byte) error
	SendPingEnvelope(peerID string, data []byte) error
	SendSignaling(peerID string, data []byte) error
}

// messageSender - конкретная реализация IMessageSender.
type messageSender struct {
	core newcore.ICoreController
}

// NewMessageSender - конструктор для нашего сервиса.
func NewMessageSender(core newcore.ICoreController) IMessageSender {
	return &messageSender{core: core}
}

func (s *messageSender) send(peerID string, msgType newcore.MessageType, data []byte) error {
	payload := make([]byte, 1+len(data))
	payload[0] = byte(msgType)
	copy(payload[1:], data)
	return s.core.SendDataToPeer(peerID, payload)
}

func (s *messageSender) SendSecureEnvelope(peerID string, data []byte) error {
	// ИСПОЛЬЗУЕМ КОНСТАНТУ ИЗ newcore
	return s.send(peerID, newcore.MsgTypeSecureEnvelope, data)
}

func (s *messageSender) SendSignedCommand(peerID string, data []byte) error {
	// ИСПОЛЬЗУЕМ КОНСТАНТУ ИЗ newcore
	return s.send(peerID, newcore.MsgTypeSignedCommand, data)
}

func (s *messageSender) SendPingEnvelope(peerID string, data []byte) error {
	// ИСПОЛЬЗУЕМ КОНСТАНТУ ИЗ newcore
	return s.send(peerID, newcore.MsgTypePingEnvelope, data)
}

func (s *messageSender) SendSignaling(peerID string, data []byte) error {
	// ИСПОЛЬЗУЕМ КОНСТАНТУ ИЗ newcore
	return s.send(peerID, newcore.MsgTypeSignaling, data)
}
