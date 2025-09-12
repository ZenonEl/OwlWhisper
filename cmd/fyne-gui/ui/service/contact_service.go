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
	StatusAwaitingApproval
)

// Contact представляет собой контакт в адресной книге.
type Contact struct {
	PeerID        string
	Nickname      string
	Discriminator string
	Status        ContactStatus
	IsSelf        bool
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

type ContactUIManager interface {
	OnProfileReceived(senderID string, profile *protocol.ProfileInfo)
	OnContactRequestReceived(senderID string, profile *protocol.ProfileInfo) // <-- НОВЫЙ МЕТОД
}

// ContactService управляет всей бизнес-логикой, связанной с контактами.
type ContactService struct {
	core      newcore.ICoreController
	Provider  ContactProvider
	onUpdate  func()   // Callback для обновления UI
	myProfile *Contact // Профиль текущего пользователя
	uiManager ContactUIManager
}

func NewContactService(core newcore.ICoreController, onUpdate func(), uiManager ContactUIManager) *ContactService {
	// Создаем наш профиль на основе данных из Core
	myProfile := &Contact{
		PeerID:        "загрузка...",
		Nickname:      "Me",     // Временное имя
		Discriminator: "xxxxxx", // Временный дискриминатор
		Status:        StatusOnline,
		IsSelf:        true,
	}

	cs := &ContactService{
		core:      core,
		Provider:  NewInMemoryContactProvider(),
		onUpdate:  onUpdate,
		myProfile: myProfile,
		uiManager: uiManager,
	}

	return cs
}

// GetContacts возвращает список всех контактов.
func (cs *ContactService) GetContacts() []*Contact {
	return cs.Provider.GetContacts()
}

// HandleProfileResponse обрабатывает ответ на наш "пинг".
func (cs *ContactService) HandleProfileResponse(senderID string, res *protocol.ProfileResponse) {
	if res.Profile == nil {
		log.Printf("WARN: [ContactService] Получен пустой ProfileResponse от %s", senderID)
		return
	}
	log.Printf("INFO: [ContactService] Получен профиль от %s. Передаем в UI для подтверждения.", senderID[:8])

	// Вызываем метод интерфейса, реализованный в UI, чтобы показать диалог.
	// Этот вызов передаст управление в AppUI.OnProfileReceived.
	cs.uiManager.OnProfileReceived(senderID, res.Profile)
}

// HandleContactRequest обрабатывает входящий запрос на добавление в контакты.
func (cs *ContactService) HandleContactRequest(senderID string, req *protocol.ContactRequest) {
	if req.SenderProfile == nil {
		return
	}
	log.Printf("INFO: [ContactService] Получен ContactRequest от %s. Передаем в UI для подтверждения.", senderID[:8])

	// ИСПОЛЬЗУЕМ ИНТЕРФЕЙС, чтобы уведомить UI
	cs.uiManager.OnContactRequestReceived(senderID, req.SenderProfile)
}

// НОВЫЙ МЕТОД: AcceptContactRequest - вызывается из UI, когда пользователь нажимает "Принять".
func (cs *ContactService) AcceptContactRequest(peerID string, profile *protocol.ProfileInfo) {
	log.Printf("INFO: [ContactService] Запрос от %s принят. Добавляем контакт и отправляем подтверждение.", peerID[:8])

	// 1. Добавляем контакт в нашу БД
	contact := &Contact{
		PeerID:        peerID,
		Nickname:      profile.Nickname,
		Discriminator: profile.Discriminator,
		Status:        StatusOnline,
	}
	cs.Provider.AddContact(contact)
	cs.onUpdate()

	// 2. Отправляем в ответ подтверждение
	cs.SendContactAccept(peerID, cs.myProfile)
}

// HandleContactAccept обрабатывает подтверждение дружбы.
func (cs *ContactService) HandleContactAccept(senderID string, acc *protocol.ContactAccept) {
	if acc.SenderProfile == nil {
		return
	}
	log.Printf("INFO: [ContactService] Получен ContactAccept от %s. Дружба подтверждена!", senderID[:8])

	// Обновляем статус контакта в нашей БД с "ожидает" на "подтвержден"
	profile := acc.SenderProfile
	contact := &Contact{
		PeerID:        senderID,
		Nickname:      profile.Nickname,
		Discriminator: profile.Discriminator,
		Status:        StatusOnline,
	}
	cs.Provider.AddContact(contact) // Метод AddContact должен уметь обновлять существующие
	cs.onUpdate()
}

// SendContactAccept отправляет подтверждение дружбы.
func (cs *ContactService) SendContactAccept(recipientID string, myProfile *Contact) {
	// Логика очень похожа на SendContactRequest
	acc := &protocol.ContactAccept{
		SenderProfile: &protocol.ProfileInfo{
			Nickname:      myProfile.Nickname,
			Discriminator: myProfile.Discriminator,
		},
	}
	contactMsg := &protocol.ContactMessage{
		Type: &protocol.ContactMessage_ContactAccept{ContactAccept: acc},
	}
	envelope := &protocol.Envelope{
		// ... заполняем поля ...
		Payload: &protocol.Envelope_ContactMessage{ContactMessage: contactMsg},
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		return
	}

	log.Printf("INFO: [ContactService] Отправка ContactAccept пиру %s", recipientID[:8])
	cs.core.SendDataToPeer(recipientID, data)
}

// UpdateContactStatus обновляет онлайн-статус контакта.
func (cs *ContactService) UpdateContactStatus(peerID string, status ContactStatus) {
	// Ищем контакт в нашем хранилище
	contact, ok := cs.Provider.GetContactByPeerID(peerID)
	if !ok {
		// Если это не наш контакт, нам нет дела до его статуса. Просто выходим.
		return
	}

	// Обновляем статус и сохраняем обратно в хранилище
	contact.Status = status
	cs.Provider.AddContact(contact) // В нашей in-memory реализации это работает как "update"

	log.Printf("INFO: [ContactService] Статус для %s обновлен.", contact.FullAddress())

	// Уведомляем UI, что нужно перерисовать список контактов
	cs.onUpdate()
}

// SendContactRequest (Фаза 3)
func (cs *ContactService) SendContactRequest(recipientID string, myProfile *Contact) {
	// Создаем Protobuf-запрос с нашим профилем
	req := &protocol.ContactRequest{
		SenderProfile: &protocol.ProfileInfo{
			Nickname:      myProfile.Nickname,
			Discriminator: myProfile.Discriminator,
		},
	}

	contactMsg := &protocol.ContactMessage{
		Type: &protocol.ContactMessage_ContactRequest{ContactRequest: req},
	}

	envelope := &protocol.Envelope{
		MessageId:     uuid.New().String(),
		SenderId:      cs.myProfile.PeerID,
		TimestampUnix: time.Now().Unix(),
		Payload:       &protocol.Envelope_ContactMessage{ContactMessage: contactMsg},
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		log.Printf("ERROR: [ContactService] Ошибка Marshal при создании ContactRequest: %v", err)
		return
	}

	// Добавляем контакт в нашу БД со статусом "ожидает"
	pendingContact := &Contact{
		PeerID: recipientID,
		// Мы пока не знаем его ник, поэтому используем временные данные
		Nickname:      "Pending...",
		Discriminator: recipientID[len(recipientID)-6:],
		// ИСПРАВЛЕНО: Убираем префикс `services.`
		Status: StatusAwaitingApproval,
	}
	// ИСПРАВЛЕНО: Используем публичное поле `Provider`
	cs.Provider.AddContact(pendingContact)
	cs.onUpdate() // Обновляем UI, чтобы показать "pending" контакт

	log.Printf("INFO: [ContactService] Отправка ContactRequest пиру %s", recipientID[:8])
	if err := cs.core.SendDataToPeer(recipientID, data); err != nil {
		log.Printf("ERROR: [ContactService] Не удалось отправить ContactRequest: %v", err)
	}
}

// RespondToProfileRequest отправляет наш профиль в ответ на "пинг".
func (cs *ContactService) RespondToProfileRequest(recipientID string) {
	// 1. Создаем самый внутренний payload - наш профиль
	profileRes := &protocol.ProfileResponse{
		Profile: &protocol.ProfileInfo{
			Nickname:      cs.myProfile.Nickname,
			Discriminator: cs.myProfile.Discriminator,
			// TODO: Добавить другие поля профиля, когда они появятся
		},
	}

	// 2. Оборачиваем его в ContactMessage
	contactMsg := &protocol.ContactMessage{
		Type: &protocol.ContactMessage_ProfileResponse{ProfileResponse: profileRes},
	}

	// 3. Оборачиваем ContactMessage в главный Envelope
	envelope := &protocol.Envelope{
		MessageId:     uuid.New().String(),
		SenderId:      cs.myProfile.PeerID,
		TimestampUnix: time.Now().Unix(),
		// ВАЖНО: Поля ChatType и ChatId здесь больше не нужны,
		// так как мы отправляем ContactMessage, а не ChatMessage.
		Payload: &protocol.Envelope_ContactMessage{ContactMessage: contactMsg},
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		log.Printf("ERROR: [ContactService] Ошибка Marshal при ответе на ProfileRequest: %v", err)
		return
	}

	log.Printf("INFO: [ContactService] Отправка ProfileResponse пиру %s", recipientID)
	if err := cs.core.SendDataToPeer(recipientID, data); err != nil {
		log.Printf("ERROR: [ContactService] Не удалось отправить ProfileResponse: %v", err)
	}
}

// Этот метод будет вызываться из UI, когда пользователь введет свой ник.
func (cs *ContactService) SetMyNickname(nickname string) {
	cs.myProfile.Nickname = nickname
	log.Printf("INFO: [ContactService] Никнейм установлен: %s", nickname)

	// Проверяем, готов ли наш профиль для анонса
	cs.checkAndFinalizeProfile()
}

func (cs *ContactService) UpdateMyProfile(peerID string) {
	cs.myProfile.PeerID = peerID
	cs.myProfile.Discriminator = peerID[len(peerID)-6:]

	// Теперь, когда у нас есть правильный PeerID, добавляем себя в хранилище.
	cs.Provider.AddContact(cs.myProfile)

	log.Printf("INFO: [ContactService] Профиль инициализирован: %s", cs.myProfile.FullAddress())

	// Проверяем, готов ли наш профиль для анонса
	cs.checkAndFinalizeProfile()
}

// Внутренний метод, который проверяет, есть ли у нас и PeerID, и никнейм.
// Если да, то он финализирует профиль и запускает анонс.
func (cs *ContactService) checkAndFinalizeProfile() {
	if cs.myProfile.PeerID != "загрузка..." && cs.myProfile.Nickname != "..." {
		// Все данные на месте!
		// Добавляем себя в список контактов и запускаем анонс.
		cs.Provider.AddContact(cs.myProfile)
		cs.onUpdate()
		go cs.announceLoop() // Запускаем анонс в фоне
		log.Printf("INFO: [ContactService] Профиль финализирован: %s", cs.myProfile.FullAddress())
	}
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
