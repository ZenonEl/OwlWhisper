// Путь: cmd/fyne-gui/services/contact_service.go

package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

// ContactStatus описывает состояние контакта.
type ContactStatus int

const (
	StatusOffline ContactStatus = iota
	StatusOnline
	StatusConnecting
	StatusUnknown
)

// Contact представляет собой контакт в адресной книге.
type Contact struct {
	PeerID        string
	Nickname      string
	Discriminator string
	Status        ContactStatus
}

func (c *Contact) FullAddress() string {
	return fmt.Sprintf("%s#%s", c.Nickname, c.Discriminator)
}

// ContactProvider - это интерфейс для хранилища контактов (в будущем - SQLite).
type ContactProvider interface {
	GetContacts() []*Contact
	GetContactByPeerID(peerID string) (*Contact, bool)
	AddContact(contact *Contact)
}

// InMemoryContactProvider - простая реализация хранилища в памяти для тестов.
type InMemoryContactProvider struct {
	contacts map[string]*Contact
	mu       sync.RWMutex
}

func NewInMemoryContactProvider() *InMemoryContactProvider {
	return &InMemoryContactProvider{
		contacts: make(map[string]*Contact),
	}
}
func (p *InMemoryContactProvider) GetContacts() []*Contact {
	p.mu.RLock()
	defer p.mu.RUnlock()
	list := make([]*Contact, 0, len(p.contacts))
	for _, c := range p.contacts {
		list = append(list, c)
	}
	return list
}
func (p *InMemoryContactProvider) GetContactByPeerID(peerID string) (*Contact, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	c, ok := p.contacts[peerID]
	return c, ok
}
func (p *InMemoryContactProvider) AddContact(contact *Contact) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.contacts[contact.PeerID] = contact
}

// ContactService управляет всей бизнес-логикой, связанной с контактами.
type ContactService struct {
	core      newcore.ICoreController
	provider  ContactProvider
	onUpdate  func()   // Callback для обновления UI
	myProfile *Contact // Профиль текущего пользователя
}

func NewContactService(core newcore.ICoreController, onUpdate func()) *ContactService {
	// Создаем наш профиль на основе данных из Core
	myProfile := &Contact{
		PeerID:        "загрузка...",
		Nickname:      "Me",     // Временное имя
		Discriminator: "xxxxxx", // Временный дискриминатор
		Status:        StatusOnline,
	}

	cs := &ContactService{
		core:      core,
		provider:  NewInMemoryContactProvider(),
		onUpdate:  onUpdate,
		myProfile: myProfile,
	}

	// Начинаем анонсировать свой профиль в DHT
	go cs.announceLoop()

	return cs
}

// GetContacts возвращает список всех контактов.
func (cs *ContactService) GetContacts() []*Contact {
	return cs.provider.GetContacts()
}

// HandleProfileResponse обрабатывает ответ на наш "пинг".
func (cs *ContactService) HandleProfileResponse(senderID string, res *protocol.ProfileResponse) {
	if res.Profile == nil {
		return
	}

	contact := &Contact{
		PeerID:        senderID,
		Nickname:      res.Profile.Nickname,
		Discriminator: res.Profile.Discriminator,
		Status:        StatusOnline,
	}

	cs.provider.AddContact(contact)
	log.Printf("INFO: [ContactService] Профиль для %s обновлен: %s", senderID[:8], contact.FullAddress())
	cs.onUpdate() // Уведомляем UI, что список контактов изменился
}

// RespondToProfileRequest отправляет наш профиль в ответ на "пинг".
func (cs *ContactService) RespondToProfileRequest(recipientID string) {
	// Создаем Protobuf-ответ с нашим профилем
	res := &protocol.ProfileResponse{
		Profile: &protocol.ProfileInfo{
			Nickname:      cs.myProfile.Nickname,
			Discriminator: cs.myProfile.Discriminator,
		},
	}

	payload := &protocol.Envelope_ProfileResponse{ProfileResponse: res}

	envelope := &protocol.Envelope{
		MessageId:     uuid.New().String(),
		SenderId:      cs.myProfile.PeerID,
		TimestampUnix: time.Now().Unix(),
		ChatType:      protocol.Envelope_PRIVATE,
		ChatId:        recipientID,
		Payload:       payload,
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		log.Printf("ERROR: [ContactService] Ошибка Marshal при ответе на ProfileRequest: %v", err)
		return
	}

	cs.core.SendDataToPeer(recipientID, data)
}

func (cs *ContactService) UpdateMyProfile(peerID string) {
	cs.myProfile.PeerID = peerID
	cs.myProfile.Discriminator = peerID[len(peerID)-6:]

	// Теперь, когда у нас есть правильный PeerID, добавляем себя в хранилище.
	cs.provider.AddContact(cs.myProfile)

	log.Printf("INFO: [ContactService] Профиль инициализирован: %s", cs.myProfile.FullAddress())
}

func (cs *ContactService) GetMyProfile() *Contact {
	return cs.myProfile
}

// announceLoop периодически анонсирует наш профиль в DHT.
func (cs *ContactService) announceLoop() {
	ticker := time.NewTicker(5 * time.Minute) // Анонсируем каждые 5 минут
	defer ticker.Stop()

	doAnnounce := func() {
		if cs.myProfile.PeerID == "загрузка..." {
			return // Еще не готовы
		}
		contentID, err := newcore.CreateContentID(cs.myProfile.FullAddress())
		if err != nil {
			log.Printf("ERROR: [ContactService] Не удалось создать ContentID для анонса: %v", err)
			return
		}
		if err := cs.core.ProvideContent(contentID); err != nil {
			// Это может происходить, пока DHT не "разогрелся", это нормально
			// log.Printf("WARN: [ContactService] Ошибка анонсирования профиля: %v", err)
		} else {
			log.Printf("INFO: [ContactService] Профиль %s успешно анонсирован в DHT.", cs.myProfile.FullAddress())
		}
	}

	time.Sleep(10 * time.Second) // Даем DHT время на первоначальный разогрев
	doAnnounce()                 // Первый анонс

	for {
		select {
		case <-cs.core.Events(): // TODO: это неверно, нужно использовать контекст
			return
		case <-ticker.C:
			doAnnounce()
		}
	}
}
