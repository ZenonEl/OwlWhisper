// Путь: cmd/fyne-gui/services/identity_service.go
package services

import (
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
	"log"
	"sync"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// IIdentityService - это единый источник правды о текущем пользователе.
type IIdentityService interface {
	GetMyPeerID() string
	GetMyNickname() string
	GetMyPublicKeyBytes() []byte
	GetMyIdentityPublicKeyProto() *protocol.IdentityPublicKey
	GetMyProfileContact() *Contact
	GetMyProfilePayload() *protocol.ProfilePayload
	SetMyNickname(nickname string)
	UpdateMyPeerID(peerID string)
}

// identityService - stateful-реализация, хранящая данные о профиле в памяти.
type identityService struct {
	cryptoService ICryptoService // Нужен для получения публичного ключа
	myProfile     *Contact
	mu            sync.RWMutex
}

// NewIdentityService - конструктор для нашего сервиса.
func NewIdentityService(cryptoService ICryptoService) IIdentityService {
	return &identityService{
		cryptoService: cryptoService,
		myProfile: &Contact{
			// Изначально PeerID неизвестен, он придет из Core
			PeerID:   "загрузка...",
			Nickname: "Избранное",
			IsSelf:   true,
		},
	}
}

func (s *identityService) GetMyPeerID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.myProfile.PeerID
}

func (s *identityService) GetMyNickname() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.myProfile.Nickname
}

func (s *identityService) GetMyPublicKeyBytes() []byte {
	pubKey := s.cryptoService.GetPublicKey()
	marshaledBytes, err := crypto.MarshalPublicKey(pubKey)
	if err != nil {
		log.Printf("FATAL: Не удалось сериализовать собственный публичный ключ: %v", err)
		return nil
	}
	return marshaledBytes
}

func (s *identityService) GetMyIdentityPublicKeyProto() *protocol.IdentityPublicKey {
	return &protocol.IdentityPublicKey{
		KeyType:   protocol.KeyType_ED25519,
		PublicKey: s.GetMyPublicKeyBytes(),
	}
}

func (s *identityService) GetMyProfileContact() *Contact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Возвращаем копию, чтобы избежать гонок данных
	profileCopy := *s.myProfile
	return &profileCopy
}

func (s *identityService) GetMyProfilePayload() *protocol.ProfilePayload {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &protocol.ProfilePayload{
		Nickname:      s.myProfile.Nickname,
		Discriminator: s.myProfile.Discriminator,
	}
}

func (s *identityService) SetMyNickname(nickname string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.myProfile.Nickname = nickname
}

func (s *identityService) UpdateMyPeerID(peerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.myProfile.PeerID = peerID
	if len(peerID) >= 6 {
		s.myProfile.Discriminator = peerID[len(peerID)-6:]
	}
}
