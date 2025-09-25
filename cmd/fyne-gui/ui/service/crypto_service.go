// Путь: cmd/fyne-gui/services/crypto_service.go
package services

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// ICryptoService - это stateful-сервис, который хранит приватный ключ
// и предоставляет операции, требующие его наличия (подпись).
type ICryptoService interface {
	Sign(data []byte) ([]byte, error)
	GetPublicKey() crypto.PubKey
	GetPrivateKey() crypto.PrivKey // Может понадобиться для шифрования
}

// cryptoService - реализация, хранящая ключи в памяти.
type cryptoService struct {
	privKey crypto.PrivKey
	pubKey  crypto.PubKey
}

// NewCryptoService создает сервис из сырых байт приватного ключа.
func NewCryptoService(privateKeyBytes []byte) (ICryptoService, error) {
	privKey, err := crypto.UnmarshalPrivateKey(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("не удалось распаковать приватный ключ: %w", err)
	}
	return &cryptoService{
		privKey: privKey,
		pubKey:  privKey.GetPublic(),
	}, nil
}

func (s *cryptoService) Sign(data []byte) ([]byte, error) {
	return s.privKey.Sign(data)
}

func (s *cryptoService) GetPublicKey() crypto.PubKey {
	return s.pubKey
}

func (s *cryptoService) GetPrivateKey() crypto.PrivKey {
	return s.privKey
}
