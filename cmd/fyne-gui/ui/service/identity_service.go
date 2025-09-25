// Путь: cmd/fyne-gui/services/identity_service.go
package services

import (
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
)

// IIdentityService - это единый источник правды о текущем пользователе.
type IIdentityService interface {
	GetMyPeerID() string
	GetMyNickname() string
	GetMyPublicKeyBytes() []byte
	GetMyIdentityPublicKeyProto() *protocol.IdentityPublicKey
	// В будущем здесь будут методы для установки аватара, статуса и т.д.
}

// В main.go мы создадим конкретную реализацию этого сервиса.
