package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// GenerateKeyBytes генерирует новые байты ключа Ed25519
func GenerateKeyBytes() ([]byte, error) {
	// Генерируем новую пару ключей Ed25519
	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации ключа: %w", err)
	}

	// Сериализуем ключ в libp2p формат (сырые байты)
	keyBytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации ключа: %w", err)
	}

	Info("🔑 Сгенерированы сырые байты ключа длиной %d байт", len(keyBytes))
	return keyBytes, nil
}

// GenerateKeyPair генерирует новую пару ключей и возвращает PeerID
func GenerateKeyPair() (string, error) {
	keyBytes, err := GenerateKeyBytes()
	if err != nil {
		return "", err
	}

	// Создаем контроллер для получения PeerID
	ctx := context.Background()
	controller, err := NewCoreControllerWithKeyBytes(ctx, keyBytes)
	if err != nil {
		return "", fmt.Errorf("ошибка создания контроллера: %w", err)
	}

	// Получаем PeerID
	peerID := controller.GetMyID()

	// Останавливаем контроллер
	controller.Stop()

	return peerID, nil
}

// ComputeContentID вычисляет ContentID из строки
func ComputeContentID(input string) string {
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)
}

// EncodeKeyToBase64 кодирует ключ в base64
func EncodeKeyToBase64(keyBytes []byte) string {
	return base64.StdEncoding.EncodeToString(keyBytes)
}

// DecodeKeyFromBase64 декодирует ключ из base64
func DecodeKeyFromBase64(encodedKey string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encodedKey)
}

// CreateContentID создает правильный CIDv1 из строки
func CreateContentID(data string) (string, error) {
	// 1. Хэшируем данные
	hash := sha256.Sum256([]byte(data))

	// 2. Создаем multihash
	mh, err := multihash.Encode(hash[:], multihash.SHA2_256)
	if err != nil {
		return "", fmt.Errorf("ошибка создания multihash: %w", err)
	}

	// 3. Создаем CIDv1 с кодеком raw
	// CID.Raw - это стандарт для указания на сырые бинарные данные
	cidV1 := cid.NewCidV1(cid.Raw, mh)

	return cidV1.String(), nil
}
