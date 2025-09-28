// Путь: cmd/fyne-gui/services/session_service.go
package services

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"sync"

	encryption "OwlWhisper/cmd/fyne-gui/ui/service/encryption"

	"golang.org/x/crypto/hkdf"
)

// SessionStateEnum определяет возможное состояние сессии.
type SessionStateEnum string

const (
	StatePending SessionStateEnum = "pending" // Рукопожатие начато, но не завершено
	StateActive  SessionStateEnum = "active"  // Сессия активна, можно шифровать
)

// SessionState хранит все данные, необходимые для одной защищенной сессии.
type SessionState struct {
	State           SessionStateEnum
	HandshakeState  []byte // Непрозрачный хэндл от CryptoEngine (эфемерный приватный ключ)
	SessionKey      []byte // Финальный ключ AES-256-GCM
	PendingMessages []*encryption.EncryptedMessage
}

// ISessionService определяет интерфейс для управления E2EE-сессиями.
type ISessionService interface {
	// --- Управление Сессиями ---
	PrepareNewSession(contextID string) (ephemeralPublicKey []byte, err error)
	ActivateSessionFromInitiator(contextID string, peerEphemeralKey []byte) error
	ActivateSessionFromRecipient(contextID string, peerEphemeralKey []byte) (ephemeralPublicKey []byte, err error)

	// --- Работа с Основным Ключом Сессии ---
	EncryptForSession(contextID string, plaintext []byte) (*encryption.EncryptedMessage, error)
	DecryptForSession(contextID string, encryptedMsg *encryption.EncryptedMessage) ([]byte, error)

	// --- Генерация Производных Ключей ---
	GetFileTransferKey(contextID, transferID string) ([]byte, error)

	// --- Общие Крипто-Операции ---
	EncryptWithKey(key, plaintext []byte) (*encryption.EncryptedMessage, error)
	DecryptWithKey(key []byte, encryptedMsg *encryption.EncryptedMessage) ([]byte, error)
}

// sessionService - конкретная реализация ISessionService.
type sessionService struct {
	engine   encryption.ICryptoEngine
	sessions map[string]*SessionState
	mu       sync.RWMutex
}

// NewSessionService - конструктор.
func NewSessionService(engine encryption.ICryptoEngine) ISessionService {
	return &sessionService{
		engine:   engine,
		sessions: make(map[string]*SessionState),
	}
}

// PrepareNewSession генерирует эфемерные ключи и сохраняет состояние ожидания.
func (s *sessionService) PrepareNewSession(contextID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Начинаем рукопожатие
	handshakeState, ephemeralPublicKey, err := s.engine.InitiateHandshake()
	if err != nil {
		return nil, err
	}

	// Сохраняем состояние ожидания
	s.sessions[contextID] = &SessionState{
		State:           StatePending,
		HandshakeState:  handshakeState,
		PendingMessages: make([]*encryption.EncryptedMessage, 0),
	}

	return ephemeralPublicKey, nil
}

// ActivateSessionFromInitiator завершает рукопожатие со стороны инициатора.
func (s *sessionService) ActivateSessionFromInitiator(contextID string, peerEphemeralKey []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.sessions[contextID]
	if !ok || state.State != StatePending {
		return fmt.Errorf("нет ожидающей сессии для активации с contextID: %s", contextID)
	}

	// Вычисляем финальный ключ
	sessionKey, err := s.engine.FinalizeHandshake(state.HandshakeState, peerEphemeralKey)
	if err != nil {
		return fmt.Errorf("не удалось завершить рукопожатие: %w", err)
	}

	// Сессия активна!
	state.SessionKey = sessionKey
	state.State = StateActive
	state.HandshakeState = nil // Больше не нужен
	log.Printf("INFO: [SessionService] Сессия для contextID %s АКТИВИРОВАНА (инициатор).", contextID)

	// Обрабатываем сообщения, которые могли прийти раньше
	// TODO: Эта логика потребует вызова callback'а в ChatService
	s.processPendingMessages(contextID, state)
	return nil
}

// ActivateSessionFromRecipient завершает рукопожатие со стороны получателя.
func (s *sessionService) ActivateSessionFromRecipient(contextID string, peerEphemeralKey []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Получатель сразу инициирует и завершает рукопожатие
	handshakeState, ownEphemeralPublicKey, err := s.engine.InitiateHandshake()
	if err != nil {
		return nil, err
	}

	sessionKey, err := s.engine.FinalizeHandshake(handshakeState, peerEphemeralKey)
	if err != nil {
		return nil, err
	}

	s.sessions[contextID] = &SessionState{
		State:      StateActive,
		SessionKey: sessionKey,
	}
	log.Printf("INFO: [SessionService] Сессия для contextID %s АКТИВИРОВАНА (получатель).", contextID)

	return ownEphemeralPublicKey, nil
}

// EncryptForSession шифрует сообщение для активной сессии.
func (s *sessionService) EncryptForSession(contextID string, plaintext []byte) (*encryption.EncryptedMessage, error) {
	s.mu.RLock()
	state, ok := s.sessions[contextID]
	s.mu.RUnlock()
	if !ok || state.State != StateActive {
		return nil, fmt.Errorf("сессия для contextID %s не активна", contextID)
	}
	return s.EncryptWithKey(state.SessionKey, plaintext)
}

// DecryptForSession расшифровывает сообщение или ставит его в очередь.
func (s *sessionService) DecryptForSession(contextID string, encryptedMsg *encryption.EncryptedMessage) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.sessions[contextID]
	if !ok {
		// Сессии еще нет. Это может быть первое сообщение (InitiateContext),
		// которое обработает ContactService. Возвращаем ошибку, но это ожидаемо.
		return nil, fmt.Errorf("сессия для contextID %s еще не существует", contextID)
	}

	// РЕАЛИЗАЦИЯ УТОЧНЕНИЯ №3: Обработка гонки состояний
	if state.State == StatePending {
		log.Printf("INFO: [SessionService] Сообщение для pending-сессии %s поставлено в очередь.", contextID)
		state.PendingMessages = append(state.PendingMessages, encryptedMsg)
		return nil, nil // Возвращаем nil, nil, чтобы показать, что сообщение принято, но не расшифровано
	}

	if state.State == StateActive {
		return s.DecryptWithKey(state.SessionKey, encryptedMsg)
	}
	return nil, fmt.Errorf("сессия в неизвестном состоянии: %s", state.State)
}

func (s *sessionService) GetFileTransferKey(contextID, transferID string) ([]byte, error) {
	s.mu.RLock()
	state, ok := s.sessions[contextID]
	s.mu.RUnlock()
	if !ok || state.State != StateActive {
		return nil, fmt.Errorf("сессия для contextID %s не активна, невозможно создать ключ файла", contextID)
	}

	// Используем HKDF для создания детерминированного ключа
	salt := []byte("owl-whisper-file-transfer-salt")
	info := []byte(transferID)

	kdf := hkdf.New(sha256.New, state.SessionKey, salt, info)
	fileKey := make([]byte, 32) // 32 байта для AES-256
	if _, err := io.ReadFull(kdf, fileKey); err != nil {
		return nil, fmt.Errorf("ошибка генерации дочернего ключа (KDF): %w", err)
	}
	return fileKey, nil
}

func (s *sessionService) EncryptWithKey(key, plaintext []byte) (*encryption.EncryptedMessage, error) {
	return s.engine.Encrypt(key, plaintext)
}

func (s *sessionService) DecryptWithKey(key []byte, encryptedMsg *encryption.EncryptedMessage) ([]byte, error) {
	return s.engine.Decrypt(key, encryptedMsg)
}

// processPendingMessages обрабатывает очередь сообщений после активации сессии.
func (s *sessionService) processPendingMessages(contextID string, state *SessionState) {
	if len(state.PendingMessages) == 0 {
		return
	}

	log.Printf("INFO: [SessionService] Обработка %d отложенных сообщений для сессии %s...", len(state.PendingMessages), contextID)
	// TODO: Здесь нам нужен способ "вернуть" эти расшифрованные сообщения
	// в ChatService/FileService. Вероятно, через callback, переданный при инициализации SessionService.
	// for _, msg := range state.PendingMessages {
	// 	plaintext, err := s.engine.Decrypt(state.SessionKey, msg)
	// 	if err == nil {
	// 		// onDecrypted(contextID, plaintext)
	// 	}
	// }
	state.PendingMessages = nil // Очищаем очередь
}
