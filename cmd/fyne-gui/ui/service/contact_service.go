// Путь: cmd/fyne-gui/services/contact_service.go
package services

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/protobuf/proto"
)

// ================================================================= //
//                      ЛОКАЛЬНЫЕ ТИПЫ И ИНТЕРФЕЙСЫ                   //
// ================================================================= //

// ВОЗВРАЩЕНО: VerificationStatus описывает уровень доверия к криптографической личности.
type VerificationStatus int

const (
	StatusUnverified VerificationStatus = iota
	StatusVerified
	StatusBlocked
)

// ВОЗВРАЩЕНО: ContactStatus описывает онлайн-статус контакта.
type ContactStatus int

const (
	StatusOffline ContactStatus = iota
	StatusOnline
	StatusConnecting
	StatusAwaitingApproval
)

// ВОЗВРАЩЕНО: Contact представляет собой контакт в адресной книге.
type Contact struct {
	PeerID        string
	Nickname      string
	Discriminator string // Последние 6 символов PeerID
	Status        ContactStatus
	IsSelf        bool
}

func (c *Contact) FullAddress() string {
	return fmt.Sprintf("%s#%s", c.Nickname, c.Discriminator)
}

// ВОЗВРАЩЕНО: ContactProvider - это интерфейс для хранилища контактов.
type ContactProvider interface {
	GetContacts() []*Contact
	GetContactByPeerID(peerID string) (*Contact, bool)
	AddContact(contact *Contact)
}

// ВОЗВРАЩЕНО: InMemoryContactProvider - простая реализация хранилища в памяти.
type InMemoryContactProvider struct {
	contacts map[string]*Contact
	mu       sync.RWMutex
}

func NewInMemoryContactProvider() ContactProvider {
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

// ContactUIManager определяет контракт между ContactService и UI.
type ContactUIManager interface {
	OnProfileReceived(senderID string, profile *protocol.ProfilePayload, fingerprint string)
	OnContactRequestReceived(senderID string, profile *protocol.ProfilePayload, fingerprint string, status VerificationStatus)
}

// ContactService управляет бизнес-логикой, связанной с контактами,
// реализуя "Протокол Безопасного Знакомства".
type ContactService struct {
	// --- Зависимости ---
	core            newcore.ICoreController
	protocolService IProtocolService
	cryptoService   ICryptoService
	identityService IIdentityService
	trustService    ITrustService
	uiManager       ContactUIManager

	// --- Внутреннее состояние ---
	Provider             ContactProvider
	onUpdate             func()                             // Callback для обновления списка контактов в UI
	pendingVerifications map[string]*pendingVerification    // Ключ: PeerID
	pendingInvitations   map[string]*protocol.SignedCommand // Ключ: PeerID отправителя
	mu                   sync.RWMutex
}

// pendingVerification временно хранит проверенные данные о контакте
// между верификацией профиля и отправкой запроса на дружбу.
type pendingVerification struct {
	publicKey []byte
	profile   *protocol.ProfilePayload
}

// NewContactService - конструктор для ContactService.
func NewContactService(
	core newcore.ICoreController,
	protoSvc IProtocolService,
	cryptoSvc ICryptoService,
	idSvc IIdentityService,
	trustSvc ITrustService,
	uiManager ContactUIManager,
	onUpdate func(),
) *ContactService {
	cs := &ContactService{
		core:                 core,
		protocolService:      protoSvc,
		cryptoService:        cryptoSvc,
		identityService:      idSvc,
		trustService:         trustSvc,
		uiManager:            uiManager,
		Provider:             NewInMemoryContactProvider(), // ИСПРАВЛЕНО: Вызов функции
		onUpdate:             onUpdate,
		pendingVerifications: make(map[string]*pendingVerification),
		pendingInvitations:   make(map[string]*protocol.SignedCommand),
	}

	// ИСПРАВЛЕНО: Используем методы, которые мы определим в IIdentityService
	myProfile := idSvc.GetMyProfileContact()
	cs.Provider.AddContact(myProfile)
	return cs
}

// ================================================================= //
//                      ПУБЛИЧНЫЕ МЕТОДЫ (API для UI)                  //
// ================================================================= //

// SearchAndVerifyContact запускает полный процесс поиска и верификации контакта.
func (cs *ContactService) SearchAndVerifyContact(address string, onStatusUpdate func(string), onError func(error)) {
	go func() {
		onStatusUpdate("Поиск...")
		targetPeer, err := cs.findPeer(address)
		if err !=
			nil {
			onError(err)
			return
		}
		onStatusUpdate(fmt.Sprintf("Контакт найден! PeerID: %s...", targetPeer.ID.ShortString()))

		if err := cs.sendProfileRequest(targetPeer.ID); err != nil {
			onError(err)
			return
		}
		onStatusUpdate("Запрос профиля отправлен. Ожидание ответа...")
	}()
}

// InitiateNewChatFromProfile вызывается UI после того, как пользователь подтвердил
// проверенный профиль и нажал "Добавить".
func (cs *ContactService) InitiateNewChatFromProfile(recipientPeerID string) {
	cs.mu.RLock()
	pending, ok := cs.pendingVerifications[recipientPeerID]
	cs.mu.RUnlock()

	if !ok {
		log.Printf("ERROR: [ContactService] Не найдены данные для инициации чата с %s", recipientPeerID)
		return
	}

	log.Printf("INFO: [ContactService] Инициация чата с проверенным контактом %s", recipientPeerID)
	go cs.sendInitiateContext(recipientPeerID, pending.publicKey, pending.profile)
}

// AcceptChatInvitation вызывается UI, когда пользователь принимает запрос на дружбу.
func (cs *ContactService) AcceptChatInvitation(senderPeerID string) {
	cs.mu.Lock()
	originalCmd, ok := cs.pendingInvitations[senderPeerID]
	if !ok {
		cs.mu.Unlock()
		log.Printf("ERROR: [ContactService] Не найдено исходное приглашение от %s", senderPeerID)
		return
	}
	delete(cs.pendingInvitations, senderPeerID)
	cs.mu.Unlock()

	// 1. Отправляем ответную команду `AcknowledgeContext`
	go cs.sendAcknowledgeContext(senderPeerID, originalCmd)

	// 2. Добавляем контакт в нашу базу немедленно
	senderProfile, err := cs.extractProfileFromCmd(originalCmd)
	if err != nil {
		log.Printf("ERROR: [ContactService] Не удалось извлечь профиль из команды: %v", err)
		return
	}

	contact := &Contact{
		PeerID:        senderPeerID,
		Nickname:      senderProfile.Nickname,
		Discriminator: senderPeerID[len(senderPeerID)-6:],
		Status:        StatusOnline,
	}
	cs.Provider.AddContact(contact)
	cs.onUpdate()
	log.Printf("INFO: [ContactService] Контакт %s#%s добавлен.", senderProfile.Nickname, contact.Discriminator)
}

// GetContacts возвращает текущий список контактов для отображения.
func (cs *ContactService) GetContacts() []*Contact {
	return cs.Provider.GetContacts()
}

// ================================================================= //
//               ПУБЛИЧНЫЕ МЕТОДЫ (ОБРАБОТЧИКИ от DISPATCHER)         //
// ================================================================= //

// HandlePingRequest обрабатывает входящий "пинг" и отвечает подписанным профилем.
func (cs *ContactService) HandlePingRequest(senderID string, req *protocol.ProfileRequest) {
	log.Printf("INFO: [ContactService] Получен ProfileRequest от %s", senderID)
	go cs.sendDiscloseProfileResponse(senderID)
}

// HandleDiscloseProfile обрабатывает ответ на "пинг", верифицирует его и показывает UI.
func (cs *ContactService) HandleDiscloseProfile(senderID string, cmd *protocol.SignedCommand, payload *protocol.DiscloseProfile) {
	log.Printf("INFO: [ContactService] Получен DiscloseProfile от %s", senderID)

	isValid, err := cs.trustService.VerifyPeerID(cmd, senderID)
	if err != nil || !isValid {
		log.Printf("WARN: [ContactService] ПРОВАЛ ВЕРИФИКАЦИИ DiscloseProfile от %s. Error: %v", senderID, err)
		return
	}
	log.Printf("INFO: [ContactService] Верификация DiscloseProfile от %s УСПЕШНА!", senderID)

	cs.mu.Lock()
	cs.pendingVerifications[senderID] = &pendingVerification{
		publicKey: cmd.AuthorIdentity.PublicKey,
		profile:   payload.Profile,
	}
	cs.mu.Unlock()

	fingerprint := cs.trustService.GenerateFingerprint(cmd.AuthorIdentity.PublicKey)
	cs.uiManager.OnProfileReceived(senderID, payload.Profile, fingerprint)
}

// HandleInitiateContext обрабатывает входящий запрос на добавление в контакты.
func (cs *ContactService) HandleInitiateContext(senderID string, cmd *protocol.SignedCommand, payload *protocol.InitiateContext) {
	log.Printf("INFO: [ContactService] Получен InitiateContext от %s", senderID)

	isValid, err := cs.trustService.VerifySignature(cmd)
	if err != nil || !isValid {
		log.Printf("WARN: [ContactService] ПРОВАЛ ВЕРИФИКАЦИИ InitiateContext от %s. Error: %v", senderID, err)
		return
	}

	status := cs.trustService.GetVerificationStatus(cmd.AuthorIdentity.PublicKey)
	fingerprint := cs.trustService.GenerateFingerprint(cmd.AuthorIdentity.PublicKey)

	// ИСПРАВЛЕНО: Теперь профиль находится прямо в payload
	senderProfile := payload.SenderProfile
	if senderProfile == nil {
		log.Printf("ERROR: [ContactService] InitiateContext от %s пришел без профиля.", senderID)
		return
	}

	cs.mu.Lock()
	cs.pendingInvitations[senderID] = cmd
	cs.mu.Unlock()

	cs.uiManager.OnContactRequestReceived(senderID, senderProfile, fingerprint, status)
}

func (cs *ContactService) HandleAcknowledgeContext(senderID string, cmd *protocol.SignedCommand, payload *protocol.AcknowledgeContext) {
	log.Printf("INFO: [ContactService] Получен AcknowledgeContext от %s. Рукопожатие завершено.", senderID)

	isValid, err := cs.trustService.VerifySignature(cmd)
	if err != nil || !isValid {
		log.Printf("WARN: [ContactService] ПРОВАЛ ВЕРИФИКАЦИИ AcknowledgeContext от %s. Error: %v", senderID, err)
		return
	}

	// Просто добавляем контакт в базу, если его там еще нет (например, из-за гонки состояний)
	if _, ok := cs.Provider.GetContactByPeerID(senderID); !ok {
		contact := &Contact{
			PeerID:        senderID,
			Nickname:      payload.SenderProfile.Nickname,
			Discriminator: senderID[len(senderID)-6:],
			Status:        StatusOnline,
		}
		cs.Provider.AddContact(contact)
		cs.onUpdate()
	}

	// TODO: Уведомить UI о том, что контакт подтвержден, и можно открывать чат.
}

// ================================================================= //
//                    ВНУТРЕННИЕ МЕТОДЫ (ЛОГИКА)                     //
// ================================================================= //

// findPeer ищет пира в сети по адресу (PeerID или nickname#disc).
func (cs *ContactService) findPeer(address string) (peer.AddrInfo, error) {
	if strings.HasPrefix(address, "12D3KooW") {
		addrInfo, err := cs.core.FindPeer(address)
		if err != nil {
			return peer.AddrInfo{}, err
		}
		return *addrInfo, nil
	}

	contentID, err := newcore.CreateContentID(address)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	providers, err := cs.core.FindProvidersForContent(contentID)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	if len(providers) == 0 {
		return peer.AddrInfo{}, fmt.Errorf("контакт не найден (офлайн или не существует)")
	}
	return providers[0], nil
}

// sendProfileRequest отправляет "пинг" для запроса профиля.
func (cs *ContactService) sendProfileRequest(recipientID peer.ID) error {
	pingMsg, err := cs.protocolService.CreatePing_ProfileRequest(
		cs.identityService.GetMyIdentityPublicKeyProto(),
		cs.identityService.GetMyPeerID(),
	)
	log.Printf("DEBUG [SENDER]: Отправка PingEnvelope (длина: %d байт) пиру %s", len(pingMsg), recipientID.ShortString())
	if err != nil {
		return fmt.Errorf("ошибка создания пинга: %w", err)
	}
	return cs.core.SendDataToPeer(recipientID.String(), pingMsg)
}

// sendDiscloseProfileResponse отправляет подписанный профиль в ответ на "пинг".
func (cs *ContactService) sendDiscloseProfileResponse(recipientID string) error {
	// ИСПРАВЛЕНО: Используем новый метод из IdentityService
	myProfilePayload := cs.identityService.GetMyProfilePayload()

	// Контекст для DiscloseProfile - это PeerID получателя. SequenceNumber всегда 1.
	commandData, err := cs.protocolService.CreateCommand_DiscloseProfile(recipientID, 1, myProfilePayload)
	if err != nil {
		return err
	}

	signature, err := cs.cryptoService.Sign(commandData)
	if err != nil {
		return err
	}

	signedCmd, err := cs.protocolService.CreateSignedCommand(
		cs.identityService.GetMyIdentityPublicKeyProto(),
		commandData,
		signature,
	)
	if err != nil {
		return err
	}

	signedCmdBytes, err := cs.protocolService.CreateSignedCommand(
		cs.identityService.GetMyIdentityPublicKeyProto(),
		commandData,
		signature,
	)
	if err != nil {
		return err
	}

	testCmd := &protocol.SignedCommand{}
	if err := proto.Unmarshal(signedCmdBytes, testCmd); err != nil {
		log.Printf("FATAL DEBUG [SENDER]: Не удалось распарсить собственное сообщение: %v", err)
	} else {
		testInnerCmd, _ := cs.protocolService.ParseCommand(testCmd.CommandData)
		log.Printf("DEBUG [SENDER]: Готовим к отправке. Длина: %d байт. Payload: %T", len(signedCmdBytes), testInnerCmd.GetPayload())
	}

	log.Printf("INFO: [ContactService] Отправка DiscloseProfile пиру %s", recipientID)
	return cs.core.SendDataToPeer(recipientID, signedCmd)
}

func (cs *ContactService) sendAcknowledgeContext(recipientID string, originalCmd *protocol.SignedCommand) error {
	innerOriginalCmd, _ := cs.protocolService.ParseCommand(originalCmd.CommandData)
	contextID := innerOriginalCmd.ContextId

	myProfilePayload := cs.identityService.GetMyProfilePayload()

	// Наша ответная команда должна иметь тот же sequence_number, что и у инициатора,
	// чтобы показать, что мы отвечаем на его первое действие.
	commandData, err := cs.protocolService.CreateCommand_AcknowledgeContext(contextID, 1, myProfilePayload)
	if err != nil {
		return err
	}

	signature, err := cs.cryptoService.Sign(commandData)
	if err != nil {
		return err
	}

	signedCmd, err := cs.protocolService.CreateSignedCommand(
		cs.identityService.GetMyIdentityPublicKeyProto(),
		commandData,
		signature,
	)
	if err != nil {
		return err
	}

	log.Printf("INFO: [ContactService] Отправка AcknowledgeContext пиру %s", recipientID)
	return cs.core.SendDataToPeer(recipientID, signedCmd)
}

// sendInitiateContext отправляет формальный запрос на добавление в контакты.
func (cs *ContactService) sendInitiateContext(recipientPeerID string, recipientPublicKey []byte, recipientProfile *protocol.ProfilePayload) error {
	myIdentityProto := cs.identityService.GetMyIdentityPublicKeyProto()
	recipientIdentityProto := &protocol.IdentityPublicKey{
		KeyType:   protocol.KeyType_ED25519,
		PublicKey: recipientPublicKey,
	}

	// TODO: Реализовать детерминированный contextID
	contextID := fmt.Sprintf("chat-%s", time.Now().UnixNano())

	// ИСПРАВЛЕНО: Добавляем свой ProfilePayload
	myProfilePayload := cs.identityService.GetMyProfilePayload()
	commandData, err := cs.protocolService.CreateCommand_InitiateContext(contextID, 1, []*protocol.IdentityPublicKey{myIdentityProto, recipientIdentityProto}, myProfilePayload)
	if err != nil {
		return err
	}

	signature, err := cs.cryptoService.Sign(commandData)
	if err != nil {
		return err
	}

	signedCmd, err := cs.protocolService.CreateSignedCommand(myIdentityProto, commandData, signature)
	if err != nil {
		return err
	}

	log.Printf("INFO: [ContactService] Отправка InitiateContext пиру %s", recipientPeerID)
	return cs.core.SendDataToPeer(recipientPeerID, signedCmd)
}

// extractProfileFromCmd - хелпер для извлечения профиля из разных типов команд.
func (cs *ContactService) extractProfileFromCmd(cmd *protocol.SignedCommand) (*protocol.ProfilePayload, error) {
	innerCmd, err := cs.protocolService.ParseCommand(cmd.CommandData)
	if err != nil {
		return nil, err
	}

	// ИСПРАВЛЕНО: Добавлена логика для разных команд
	if initCtx := innerCmd.GetInitiateContext(); initCtx != nil {
		return initCtx.SenderProfile, nil
	}
	if disclose := innerCmd.GetDiscloseProfile(); disclose != nil {
		return disclose.Profile, nil
	}
	if ack := innerCmd.GetAcknowledgeContext(); ack != nil {
		return ack.SenderProfile, nil
	}

	return nil, fmt.Errorf("не удалось найти профиль в команде")
}
