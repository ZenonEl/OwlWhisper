// Путь: cmd/fyne-gui/services/encryption/engine.go
package encryption

// EncryptedMessage представляет собой результат операции шифрования.
// Это наш стандартный "конверт" для зашифрованных данных.
type EncryptedMessage struct {
	Ciphertext []byte
	Nonce      []byte
}

// ICryptoEngine определяет модульный интерфейс для криптографического движка.
// Он полностью инкапсулирует работу с эфемерными ключами.
type ICryptoEngine interface {
	// InitiateHandshake начинает "рукопожатие".
	// Возвращает ТОЛЬКО публичную часть ключа и непрозрачный "хэндл" состояния.
	// Приватный ключ остается внутри движка и никогда его не покидает.
	InitiateHandshake() (handshakeState []byte, ephemeralPublicKey []byte, err error)

	// FinalizeHandshake завершает "рукопожатие", используя сохраненное состояние
	// и публичный ключ собеседника. Возвращает финальный сессионный ключ.
	FinalizeHandshake(handshakeState []byte, peerPublicKey []byte) (sessionKey []byte, err error)

	// Encrypt шифрует сообщение с использованием установленного ключа сессии.
	Encrypt(sessionKey, plaintext []byte) (*EncryptedMessage, error)

	// Decrypt расшифровывает сообщение.
	Decrypt(sessionKey []byte, encryptedMsg *EncryptedMessage) ([]byte, error)
}
