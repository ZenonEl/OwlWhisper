// Путь: cmd/fyne-gui/services/trust_service.go
package services

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ITrustService определяет интерфейс для проверки криптографической подлинности
// и управления доверием к публичным ключам.
type ITrustService interface {
	// --- Методы криптографической верификации ---

	// VerifySignature проверяет, что подпись в команде соответствует ее автору.
	VerifySignature(cmd *protocol.SignedCommand) (bool, error)

	// VerifyPeerID делает то же, что и VerifySignature, но дополнительно сверяет,
	// что PeerID, вычисленный из ключа в команде, совпадает с ожидаемым PeerID.
	// Это ключевая защита от MITM-атаки при первом контакте.
	VerifyPeerID(cmd *protocol.SignedCommand, expectedPeerID string) (bool, error)

	// --- Методы управления доверием ---

	// GetVerificationStatus возвращает текущий уровень доверия к публичному ключу.
	GetVerificationStatus(publicKey []byte) VerificationStatus

	// SetVerificationStatus устанавливает новый уровень доверия для публичного ключа.
	SetVerificationStatus(publicKey []byte, status VerificationStatus)

	// --- Методы-хелперы ---

	// GenerateFingerprint создает легко сравниваемый человеко-читаемый отпечаток ключа.
	GenerateFingerprint(publicKey []byte) string
}

// trustService - конкретная реализация ITrustService.
type trustService struct {
	cryptoModule newcore.ICryptoModule
	// Хранилище статусов доверия. Ключ - это hex-представление публичного ключа.
	verificationStatuses map[string]VerificationStatus
	mu                   sync.RWMutex
}

// NewTrustService - конструктор для нашего сервиса.
func NewTrustService(cryptoModule newcore.ICryptoModule) ITrustService {
	return &trustService{
		cryptoModule:         cryptoModule,
		verificationStatuses: make(map[string]VerificationStatus),
	}
}

// --- Реализация методов верификации ---

func (ts *trustService) VerifySignature(cmd *protocol.SignedCommand) (bool, error) {
	if cmd == nil || cmd.AuthorIdentity == nil {
		return false, fmt.Errorf("команда или ее автор не могут быть nil")
	}
	return ts.cryptoModule.Verify(cmd.AuthorIdentity.PublicKey, cmd.CommandData, cmd.Signature)
}

func (ts *trustService) VerifyPeerID(cmd *protocol.SignedCommand, expectedPeerID string) (bool, error) {
	// Шаг 1: Проверяем базовую валидность подписи.
	// Если подпись неверна, дальнейшие проверки не имеют смысла.
	isValid, err := ts.VerifySignature(cmd)
	if !isValid || err != nil {
		return false, err // Возвращаем ошибку, если она была, или просто false
	}

	// Шаг 2: Вычисляем PeerID из ключа в команде.
	actualPeerID, err := ts.cryptoModule.GetPeerIDFromPublicKey(cmd.AuthorIdentity.PublicKey)
	if err != nil {
		return false, fmt.Errorf("не удалось получить PeerID из ключа в команде: %w", err)
	}

	// Шаг 3: Сравниваем вычисленный PeerID с тем, который мы ожидали (например, из DHT).
	if actualPeerID != expectedPeerID {
		// Это может быть MITM-атака!
		return false, nil
	}

	// Если все проверки пройдены, возвращаем true.
	return true, nil
}

// --- Реализация методов управления доверием ---

func (ts *trustService) GetVerificationStatus(publicKey []byte) VerificationStatus {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	keyHex := hex.EncodeToString(publicKey)
	status, ok := ts.verificationStatuses[keyHex]
	if !ok {
		// По умолчанию все ключи, которые мы еще не видели, - непроверенные.
		return StatusUnverified
	}
	return status
}

func (ts *trustService) SetVerificationStatus(publicKey []byte, status VerificationStatus) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	keyHex := hex.EncodeToString(publicKey)
	ts.verificationStatuses[keyHex] = status
	log.Printf("INFO: [TrustService] Статус для ключа %s... установлен на %d", keyHex[:8], status)
}

// --- Реализация хелперов ---

func (ts *trustService) GenerateFingerprint(publicKey []byte) string {
	// Используем SHA-256 хеш от ключа для создания отпечатка.
	// Это стандартная практика, чтобы не показывать сам ключ.
	hash := ts.cryptoModule.Hash(publicKey)
	hexHash := hex.EncodeToString(hash)

	// Для простоты и читаемости, мы можем его отформатировать.
	// Пример: "ABCD 1234 EFGH 5678 ..."
	var formatted strings.Builder
	for i, r := range hexHash {
		if i > 0 && i%4 == 0 {
			formatted.WriteRune(' ')
		}
		formatted.WriteRune(r)
	}

	// Преобразуем в верхний регистр для единообразия
	return cases.Upper(language.English).String(formatted.String())
}

// Этот метод не является частью интерфейса, но может быть полезен внутри
// для сравнения ключей, хранящихся в виде байтов.
func areKeysEqual(key1, key2 []byte) bool {
	return bytes.Equal(key1, key2)
}
