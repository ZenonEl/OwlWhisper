// Путь: cmd/fyne-gui/ui/ui.go
package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/libp2p/go-libp2p/core/crypto"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
	services "OwlWhisper/cmd/fyne-gui/ui/service"
)

// AppUI является корневым компонентом UI и реализует интерфейсы для сервисов.
type AppUI struct {
	app        fyne.App
	mainWindow fyne.Window

	// --- Источники данных для Fyne ---
	peerIDLabelText      binding.String
	statusLabelText      binding.String
	fingerprintLabelText binding.String
	messages             binding.UntypedList
	contacts             binding.UntypedList

	// --- Зависимости (сервисы) ---
	coreController  newcore.ICoreController
	dispatcher      *services.MessageDispatcher
	identityService services.IIdentityService
	contactService  *services.ContactService
	chatService     *services.ChatService
	fileService     *services.FileService
	callService     *services.CallService

	// --- Внутреннее состояние UI ---
	currentChatPeerID string
}

// NewAppUI - ЕДИНСТВЕННЫЙ КОНСТРУКТОР. Он собирает всё приложение.
func NewAppUI(core newcore.ICoreController, privKey crypto.PrivKey) *AppUI {
	a := app.NewWithID("com.owlwhisper.desktop")
	win := a.NewWindow("Owl Whisper")

	ui := &AppUI{
		app:                  a,
		mainWindow:           win,
		coreController:       core,
		messages:             binding.NewUntypedList(),
		contacts:             binding.NewUntypedList(),
		peerIDLabelText:      binding.NewString(),
		statusLabelText:      binding.NewString(),
		fingerprintLabelText: binding.NewString(),
	}
	ui.peerIDLabelText.Set("PeerID: загрузка...")
	ui.statusLabelText.Set("Статус: инициализация сервисов...")
	ui.fingerprintLabelText.Set("Отпечаток: ...")

	// --- СБОРКА СЕРВИСНОГО СЛОЯ ВНУТРИ UI ---

	// Базовые сервисы
	coreCryptoModule := newcore.NewCryptoModule()
	protocolService := services.NewProtocolService() // Локальная переменная, не нужна в AppUI
	messageSender := services.NewMessageSender(core)

	privKeyBytes, _ := crypto.MarshalPrivateKey(privKey)
	var err error
	cryptoService, err := services.NewCryptoService(privKeyBytes)
	if err != nil {
		log.Fatalf("Ошибка CryptoService: %v", err)
	}

	trustService := services.NewTrustService(coreCryptoModule)
	ui.identityService = services.NewIdentityService(cryptoService, trustService)

	// Бизнес-сервисы
	ui.contactService = services.NewContactService(core, messageSender, protocolService, cryptoService, ui.identityService, trustService, ui, ui.refreshContacts)
	ui.chatService = services.NewChatService(messageSender, protocolService, ui.identityService, ui.contactService.Provider, ui.onNewChatMessage)
	ui.fileService = services.NewFileService(core, messageSender, protocolService, ui.identityService, ui)
	ui.callService, err = services.NewCallService(messageSender, protocolService, ui.OnIncomingCall)

	if err != nil {
		log.Fatalf("Ошибка CallService: %v", err)
	}

	// Диспетчер
	ui.dispatcher = services.NewMessageDispatcher(protocolService, ui.contactService, ui.chatService, ui.fileService, ui.callService)

	win.SetContent(ui.buildMainLayout())
	win.Resize(fyne.NewSize(800, 600))
	return ui
}

// ================================================================= //
//                      РЕАЛИЗАЦИЯ ИНТЕРФЕЙСОВ ДЛЯ СЕРВИСОВ            //
// ================================================================= //

// --- Реализация services.ContactUIManager ---

func (ui *AppUI) OnProfileReceived(senderID string, profile *protocol.ProfilePayload, fingerprint string) {
	ui.ShowConfirmContactDialog(senderID, profile, fingerprint)
}

func (ui *AppUI) OnContactRequestReceived(senderID string, profile *protocol.ProfilePayload, fingerprint string, status services.VerificationStatus) {
	ui.ShowContactRequestDialog(senderID, profile, fingerprint, status)
}

// --- Реализация services.FileCardGenerator ---

func (ui *AppUI) NewFileCard(metadata *protocol.FileMetadata, onDownload func(*protocol.FileMetadata)) fyne.CanvasObject {
	return NewFileCardWidget(metadata, onDownload)
}

// --- Реализация callback'ов для сервисов ---

func (ui *AppUI) refreshContacts() {
	contacts := ui.contactService.GetContacts()
	items := make([]interface{}, len(contacts))
	for i, v := range contacts {
		items[i] = v
	}
	ui.contacts.Set(items)
}

func (ui *AppUI) onNewChatMessage(widget fyne.CanvasObject) {
	ui.messages.Append(widget)
}

func (ui *AppUI) OnIncomingCall(senderID, callID string) {
	contact, _ := ui.contactService.Provider.GetContactByPeerID(senderID)
	senderInfo := senderID[:12] + "..."
	if contact != nil {
		senderInfo = contact.FullAddress()
	}

	dialog.ShowConfirm("Входящий звонок", fmt.Sprintf("Вам звонит %s. Принять?", senderInfo),
		func(confirm bool) {
			go func() {
				if !confirm {
					ui.callService.HangupCall()
					return
				}
				if err := ui.callService.AcceptCall(); err != nil {
					log.Printf("ERROR: [UI] Ошибка принятия звонка: %v", err)
				}
			}()
		},
		ui.mainWindow,
	)
}

// ================================================================= //
//                         ОСНОВНЫЕ МЕТОДЫ                           //
// ================================================================= //

// Start запускает главный цикл приложения.
func (ui *AppUI) Start() {
	ui.mainWindow.Show()
	go ui.eventLoop()
	ui.app.Run()

	if err := ui.coreController.Stop(); err != nil {
		log.Printf("ERROR: Ошибка при остановке ядра: %v", err)
	}
	log.Println("INFO: Приложение завершило работу.")
}

// eventLoop обрабатывает события от Core.
func (ui *AppUI) eventLoop() {
	for event := range ui.coreController.Events() {
		switch event.Type {
		case "CoreReady":
			payload := event.Payload.(newcore.CoreReadyPayload)
			ui.peerIDLabelText.Set("PeerID: " + payload.PeerID)
			ui.identityService.UpdateMyPeerID(payload.PeerID)

			ui.ShowNicknameDialog(func(nickname string) {
				ui.identityService.SetMyNickname(nickname)
				profile := ui.identityService.GetMyProfileContact()
				fingerprint := ui.identityService.GetMyFingerprint()

				ui.mainWindow.SetTitle(fmt.Sprintf("Owl Whisper - %s", profile.FullAddress()))
				ui.fingerprintLabelText.Set("Отпечаток: " + fingerprint)
				ui.refreshContacts()
				ui.contactService.StartAnnouncing()
			})

		case "NewMessage":
			payload := event.Payload.(newcore.NewMessagePayload)
			ui.dispatcher.HandleIncomingData(payload.SenderID, payload.MessageType, payload.Data)

		case "PeerConnected":
			payload := event.Payload.(newcore.PeerStatusPayload)
			ui.statusLabelText.Set(fmt.Sprintf("Статус: Подключен пир %s", payload.PeerID[:8]))

		case "PeerDisconnected":
			payload := event.Payload.(newcore.PeerStatusPayload)
			ui.statusLabelText.Set(fmt.Sprintf("Статус: Отключен пир %s", payload.PeerID[:8]))

		case "NewIncomingStream":
			ui.fileService.HandleIncomingStream(event.Payload.(newcore.NewIncomingStreamPayload))
		case "StreamDataReceived":
			ui.fileService.HandleStreamData(event.Payload.(newcore.StreamDataReceivedPayload))
		case "StreamClosed":
			ui.fileService.HandleStreamClosed(event.Payload.(newcore.StreamClosedPayload))
		}
	}
}

// ================================================================= //
//                         СБОРКА МАКЕТА UI                          //
// ================================================================= //

func (ui *AppUI) buildMainLayout() fyne.CanvasObject {
	// --- Статус-бар ---
	peerIdLabel := widget.NewLabelWithData(ui.peerIDLabelText)
	fingerprintLabel := widget.NewLabelWithData(ui.fingerprintLabelText)
	peerIdLabel.Selectable = true
	fingerprintLabel.Selectable = true
	statusPanel := container.NewVBox(
		peerIdLabel,
		fingerprintLabel,
		widget.NewLabelWithData(ui.statusLabelText),
		widget.NewSeparator(),
	)

	// --- Левая панель (Контакты) ---
	contactsList := ui.buildContactsList()
	addContactButton := widget.NewButtonWithIcon("Добавить", theme.ContentAddIcon(), ui.ShowSearchDialog)
	leftPanel := container.NewBorder(container.NewVBox(widget.NewLabel("Контакты"), addContactButton), nil, nil, nil, contactsList)

	// --- Правая панель (Чат) ---
	chatMessages := ui.buildChatList()
	messageInput, sendButton := ui.buildMessageInput()
	fileButton := ui.buildFileButton()
	callButton := ui.buildCallButton()

	chatHeader := container.NewBorder(nil, nil, nil, callButton, widget.NewLabel("Чат"))
	bottomPanel := container.NewBorder(nil, nil, fileButton, sendButton, messageInput)
	rightPanel := container.NewBorder(chatHeader, bottomPanel, nil, nil, chatMessages)

	// --- Сборка ---
	split := container.NewHSplit(leftPanel, rightPanel)
	split.Offset = 0.3

	return container.NewBorder(statusPanel, nil, nil, nil, split)
}

func (ui *AppUI) buildContactsList() *widget.List {
	list := widget.NewListWithData(
		ui.contacts,
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewIcon(theme.RadioButtonIcon()), widget.NewLabel("template"))
		},
		func(item binding.DataItem, o fyne.CanvasObject) {
			untyped, _ := item.(binding.Untyped).Get()
			contact := untyped.(*services.Contact)
			hbox := o.(*fyne.Container)
			icon := hbox.Objects[0].(*widget.Icon)
			label := hbox.Objects[1].(*widget.Label)
			label.SetText(contact.FullAddress())
			if contact.Status == services.StatusOnline {
				icon.SetResource(theme.ConfirmIcon())
			} else {
				icon.SetResource(theme.RadioButtonIcon())
			}
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		item, _ := ui.contacts.GetValue(id)
		contact := item.(*services.Contact)

		// ИСПРАВЛЕНО: Убираем проверку, чтобы разрешить чат с собой ("Избранное")
		// if contact.IsSelf {
		// 	list.UnselectAll()
		// 	return
		// }

		ui.currentChatPeerID = contact.PeerID
		ui.statusLabelText.Set(fmt.Sprintf("Открыт чат с %s", contact.Nickname))
		log.Printf("INFO: Выбран чат с %s", contact.FullAddress())
		list.UnselectAll()
	}
	return list
}

func (ui *AppUI) buildChatList() *widget.List {
	return widget.NewListWithData(
		ui.messages,
		func() fyne.CanvasObject { return container.NewStack() },
		func(item binding.DataItem, o fyne.CanvasObject) {
			untyped, _ := item.(binding.Untyped).Get()
			if wid, ok := untyped.(fyne.CanvasObject); ok {
				o.(*fyne.Container).Objects = []fyne.CanvasObject{wid}
				o.(*fyne.Container).Refresh()
			}
		},
	)
}

func (ui *AppUI) buildMessageInput() (*widget.Entry, *widget.Button) {
	messageEntry := widget.NewEntry()
	messageEntry.SetPlaceHolder("Выберите контакт для начала общения...")
	sendButton := widget.NewButton("Отправить", func() {
		text := messageEntry.Text
		if text == "" || ui.currentChatPeerID == "" {
			return
		}
		if err := ui.chatService.SendTextMessage(ui.currentChatPeerID, text); err != nil {
			log.Printf("ERROR: [UI] Не удалось отправить сообщение: %v", err)
			return
		}
		myProfile := ui.identityService.GetMyProfileContact()
		fullMessage := fmt.Sprintf("%s: %s", myProfile.FullAddress(), text)
		textWidget := widget.NewLabel(fullMessage)
		textWidget.Wrapping = fyne.TextWrapWord
		ui.messages.Append(textWidget)
		messageEntry.SetText("")
	})
	return messageEntry, sendButton
}

func (ui *AppUI) buildFileButton() *widget.Button {
	return widget.NewButtonWithIcon("", theme.FileIcon(), func() {
		if ui.currentChatPeerID == "" {
			return
		}
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			filePath := reader.URI().Path()
			go func() {
				metadata, announceErr := ui.fileService.AnnounceFile(ui.currentChatPeerID, filePath)
				if announceErr != nil {
					log.Printf("ERROR: [UI] Ошибка анонса файла: %v", announceErr)
					return
				}
				// Отображаем карточку у себя в чате
				card := ui.NewFileCard(metadata, nil) // onDownload is nil for sender
				ui.messages.Append(card)
			}()
		}, ui.mainWindow)
	})
}

func (ui *AppUI) buildCallButton() *widget.Button {
	return widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		if ui.currentChatPeerID != "" {
			go func() {
				if err := ui.callService.InitiateCall(ui.currentChatPeerID); err != nil {
					log.Printf("ERROR: [UI] Ошибка инициации звонка: %v", err)
				}
			}()
		}
	})
}
