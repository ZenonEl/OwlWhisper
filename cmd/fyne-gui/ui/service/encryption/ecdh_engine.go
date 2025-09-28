// Путь: cmd/fyne-gui/services/encryption/ecdh_engine.go
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// simpleEcdhEngine - это реализация ICryptoEngine на базе ECDH (кривая X25519)
// и AES-256-GCM для шифрования.
type simpleEcdhEngine struct {
	curve ecdh.Curve
}

// NewSimpleEcdhEngine - конструктор для нашего движка.
func NewSimpleEcdhEngine() (ICryptoEngine, error) {
	return &simpleEcdhEngine{
		curve: ecdh.X25519(),
	}, nil
}

// InitiateHandshake генерирует новую эфемерную пару ключей.
func (e *simpleEcdhEngine) InitiateHandshake() ([]byte, []byte, error) {
	// 1. Генерируем приватный ключ.
	privateKey, err := e.curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка генерации ECDH ключа: %w", err)
	}

	// 2. Получаем его публичную часть.
	publicKey := privateKey.PublicKey()

	// 3. Возвращаем приватный ключ как "handshakeState", а публичный - открыто.
	return privateKey.Bytes(), publicKey.Bytes(), nil
}

// FinalizeHandshake вычисляет общий секрет и генерирует из него сессионный ключ.
func (e *simpleEcdhEngine) FinalizeHandshake(handshakeState []byte, peerPublicKeyBytes []byte) ([]byte, error) {
	// 1. Восстанавливаем наш приватный ключ из state.
	privateKey, err := e.curve.NewPrivateKey(handshakeState)
	if err != nil {
		return nil, fmt.Errorf("неверный формат handshakeState: %w", err)
	}

	// 2. Восстанавливаем публичный ключ собеседника.
	peerPublicKey, err := e.curve.NewPublicKey(peerPublicKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("неверный формат публичного ключа собеседника: %w", err)
	}

	// 3. Выполняем операцию ECDH для получения общего секрета.
	sharedSecret, err := privateKey.ECDH(peerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("ошибка вычисления ECDH секрета: %w", err)
	}

	// 4. Используем HKDF для получения из секрета криптографически надежного ключа.
	// Это стандартная практика (Key Derivation Function).
	kdf := hkdf.New(sha256.New, sharedSecret, nil, []byte("owl-whisper-session-key"))
	sessionKey := make([]byte, 32) // 32 байта для AES-256
	if _, err := io.ReadFull(kdf, sessionKey); err != nil {
		return nil, fmt.Errorf("ошибка генерации ключа из секрета (KDF): %w", err)
	}

	return sessionKey, nil
}

// Encrypt шифрует данные с помощью AES-256-GCM.
func (e *simpleEcdhEngine) Encrypt(sessionKey, plaintext []byte) (*EncryptedMessage, error) {
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Nonce должен быть уникальным для каждой операции шифрования с одним и тем же ключом.
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedMessage{
		Ciphertext: ciphertext,
		Nonce:      nonce,
	}, nil
}

// Decrypt расшифровывает данные.
func (e *simpleEcdhEngine) Decrypt(sessionKey []byte, encryptedMsg *EncryptedMessage) ([]byte, error) {
	if encryptedMsg == nil {
		return nil, fmt.Errorf("сообщение для расшифровки не может быть nil")
	}

	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(encryptedMsg.Nonce) != aesgcm.NonceSize() {
		return nil, fmt.Errorf("неверная длина nonce")
	}

	plaintext, err := aesgcm.Open(nil, encryptedMsg.Nonce, encryptedMsg.Ciphertext, nil)
	if err != nil {
		// Эта ошибка возникает, если ключ не тот или сообщение было изменено в пути (целостность GCM).
		return nil, fmt.Errorf("ошибка расшифровки (возможно, неверный ключ или поврежденные данные): %w", err)
	}

	return plaintext, nil
}
