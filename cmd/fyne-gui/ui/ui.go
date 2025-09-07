// Путь: cmd/fyne-gui/ui/ui.go

package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding" // <-- НОВЫЙ ИМПОРТ
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	services "OwlWhisper/cmd/fyne-gui/ui/service"
)

type AppUI struct {
	peerIDLabel    *widget.Label
	statusLabel    *widget.Label
	coreController newcore.ICoreController
	contactService *services.ContactService
	app            fyne.App
	mainWindow     fyne.Window

	// --- ИЗМЕНЕНО: Переходим на Data Binding ---
	peerIDLabelText binding.String
	statusLabelText binding.String
	messages        binding.StringList  // <-- Связанный список строк для чата
	contacts        binding.UntypedList // <-- Связанный список контактов

	// Состояние
	currentChatPeerID string
}

func NewAppUI(core newcore.ICoreController) *AppUI {
	a := app.New()
	win := a.NewWindow("Owl Whisper - Fyne GUI Test")

	ui := &AppUI{
		coreController: core,
		app:            a,
		mainWindow:     win,

		// Инициализируем связанные данные
		peerIDLabelText: binding.NewString(),
		statusLabelText: binding.NewString(),
		messages:        binding.NewStringList(),
		contacts:        binding.NewUntypedList(),
	}
	ui.peerIDLabelText.Set("PeerID: загрузка...")
	ui.statusLabelText.Set("Статус: инициализация...")

	ui.contactService = services.NewContactService(core, func() {
		// Callback от ContactService теперь тоже работает через Data Binding
		ui.refreshContacts()
	})

	win.SetContent(ui.buildUI())
	win.Resize(fyne.NewSize(800, 600))
	return ui
}

func (ui *AppUI) Start() {
	ui.mainWindow.Show()
	go ui.eventLoop()
	ui.app.Run() // Блокирующая функция

	if err := ui.coreController.Stop(); err != nil {
		log.Printf("ERROR: Ошибка при остановке ядра: %v", err)
	}
	log.Println("INFO: Приложение завершило работу.")
}

func (ui *AppUI) buildUI() fyne.CanvasObject {
	// ИСПОЛЬЗУЕМ WIDGET.NEW...WITHDATA для привязки
	ui.peerIDLabel = widget.NewLabelWithData(ui.peerIDLabelText)
	ui.statusLabel = widget.NewLabelWithData(ui.statusLabelText)

	contactsList := widget.NewListWithData(
		ui.contacts, // <-- Привязываем к списку контактов
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewIcon(theme.RadioButtonIcon()), widget.NewLabel("template"))
		},
		func(item binding.DataItem, o fyne.CanvasObject) {
			untyped, _ := item.(binding.Untyped).Get()
			contact := untyped.(*services.Contact) // <-- Извлекаем наш объект

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

	contactsList.OnSelected = func(id widget.ListItemID) {
		item, _ := ui.contacts.GetValue(id)
		contact := item.(*services.Contact)
		ui.currentChatPeerID = contact.PeerID
		ui.messages.Set([]string{}) // Очищаем чат
		log.Printf("INFO: Выбран чат с %s", contact.FullAddress())
		contactsList.UnselectAll()
	}

	addContactButton := widget.NewButtonWithIcon("Добавить контакт", theme.ContentAddIcon(), func() {
		ui.ShowSearchDialog()
	})
	leftPanel := container.NewBorder(container.NewVBox(widget.NewLabel("Контакты:"), addContactButton), nil, nil, nil, contactsList)

	chatMessages := widget.NewListWithData(ui.messages,
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(item binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(item.(binding.String))
		},
	)

	messageEntry := widget.NewEntry()
	messageEntry.SetPlaceHolder("Выберите контакт...")

	sendButton := widget.NewButton("Отправить", func() {
		text := messageEntry.Text
		if text != "" && ui.currentChatPeerID != "" {
			ui.coreController.SendDataToPeer(ui.currentChatPeerID, []byte(text))
			myProfile := ui.contactService.GetMyProfile()
			ui.messages.Append(fmt.Sprintf("[%s]: %s", myProfile.Nickname, text))
			messageEntry.SetText("")
		}
	})

	bottomPanel := container.NewBorder(nil, nil, nil, sendButton, messageEntry)
	rightPanel := container.NewBorder(nil, bottomPanel, nil, nil, chatMessages)
	split := container.NewHSplit(leftPanel, rightPanel)
	split.Offset = 0.3
	return container.NewBorder(container.NewVBox(ui.peerIDLabel, ui.statusLabel, widget.NewSeparator()), nil, nil, nil, split)
}

func (ui *AppUI) eventLoop() {
	for event := range ui.coreController.Events() {
		switch event.Type {
		case "CoreReady":
			if payload, ok := event.Payload.(newcore.CoreReadyPayload); ok {
				// БЕЗОПАСНОЕ ОБНОВЛЕНИЕ: Просто меняем данные, Fyne обновит виджет сам.
				ui.peerIDLabelText.Set("PeerID: " + payload.PeerID)
				ui.contactService.UpdateMyProfile(payload.PeerID)
			}
		case "NewMessage":
			if payload, ok := event.Payload.(newcore.NewMessagePayload); ok {
				msgText := string(payload.Data)
				senderShort := payload.SenderID[:8]
				fullMessage := fmt.Sprintf("[%s]: %s", senderShort, msgText)
				// БЕЗОПАСНОЕ ОБНОВЛЕНИЕ
				ui.messages.Append(fullMessage)
			}
			// case "PeerConnected", "PeerDisconnected":
			// 	// Обновляем статусы контактов и затем UI
			// 	//ui.contactService.UpdateContactStatuses(ui.coreController.GetConnectedPeers())
			// }
		}
	}
}

// refreshContacts безопасно обновляет список контактов в UI.
func (ui *AppUI) refreshContacts() {
	contacts := ui.contactService.GetContacts()
	// Преобразуем наш слайс в формат, понятный Data Binding
	items := make([]interface{}, len(contacts))
	for i, v := range contacts {
		items[i] = v
	}
	// Устанавливаем новые данные, Fyne сам обновит список.
	ui.contacts.Set(items)
}
