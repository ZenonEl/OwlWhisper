// Путь: cmd/fyne-gui/new-core/crypto.go

package newcore

import (
	"crypto/sha256"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ICryptoModule определяет интерфейс для нашего stateless-крипто-модуля.
type ICryptoModule interface {
	GenerateIdentityKeyPair() (publicKeyBytes []byte, privateKeyBytes []byte, err error)
	Verify(publicKeyBytes []byte, message []byte, signature []byte) (bool, error)
	Hash(data []byte) []byte
	GetPeerIDFromPublicKey(publicKeyBytes []byte) (string, error)
}

// cryptoModule - это конкретная реализация ICryptoModule.
// Она не имеет состояния (stateless).
type cryptoModule struct{}

// NewCryptoModule - конструктор для нашего крипто-модуля.
func NewCryptoModule() ICryptoModule {
	return &cryptoModule{}
}

// GenerateIdentityKeyPair генерирует новую криптографическую пару Ed25519.
func (cm *cryptoModule) GenerateIdentityKeyPair() ([]byte, []byte, error) {
	// 1. Генерируем ключ. Ed25519 - отличный выбор для подписей.
	priv, pub, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка генерации ключей: %w", err)
	}

	// 2. ИСПРАВЛЕНО: Используем общую функцию MarshalPublicKey.
	// Она сама определяет тип ключа (Ed25519, RSA и т.д.) и правильно
	// его сериализует.
	pubBytes, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка сериализации публичного ключа: %w", err)
	}

	// 3. ИСПРАВЛЕНО: Используем общую функцию MarshalPrivateKey.
	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка сериализации приватного ключа: %w", err)
	}

	return pubBytes, privBytes, nil
}

// Verify проверяет, что подпись является валидной для данного сообщения и публичного ключа.
func (cm *cryptoModule) Verify(publicKeyBytes []byte, message []byte, signature []byte) (bool, error) {
	// 1. Используем ОБЩИЙ десериализатор, который понимает формат libp2p.
	pubKey, err := crypto.UnmarshalPublicKey(publicKeyBytes)
	if err != nil {
		// Эта ошибка будет более информативной, если что-то не так с форматом.
		return false, fmt.Errorf("не удалось распаковать публичный ключ libp2p: %w", err)
	}

	// 2. Вызываем метод Verify() у полученного интерфейса.
	// Он сам разберется, какой алгоритм подписи использовать (Ed25519, RSA и т.д.).
	return pubKey.Verify(message, signature)
}

// Hash возвращает детерминированный хеш SHA-256 от входных данных.
func (cm *cryptoModule) Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:] // Возвращаем срез, а не массив
}

// GetPeerIDFromPublicKey преобразует публичный ключ libp2p в его строковое представление PeerID.
func (cm *cryptoModule) GetPeerIDFromPublicKey(publicKeyBytes []byte) (string, error) {
	pubKey, err := crypto.UnmarshalPublicKey(publicKeyBytes)
	if err != nil {
		return "", fmt.Errorf("не удалось распаковать публичный ключ libp2p: %w", err)
	}

	peerID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return "", fmt.Errorf("не удалось получить PeerID из публичного ключа: %w", err)
	}

	return peerID.String(), nil
}
