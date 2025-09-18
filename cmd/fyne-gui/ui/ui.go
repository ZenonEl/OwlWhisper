// Путь: cmd/fyne-gui/ui/ui.go

package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding" // <-- НОВЫЙ ИМПОРТ
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
	services "OwlWhisper/cmd/fyne-gui/ui/service"
)

type AppUI struct {
	peerIDLabel    *widget.Label
	statusLabel    *widget.Label
	coreController newcore.ICoreController
	contactService *services.ContactService
	chatService    *services.ChatService
	fileService    *services.FileService
	dispatcher     *services.MessageDispatcher // <-- НОВОЕ ПОЛЕ
	app            fyne.App
	mainWindow     fyne.Window

	// --- ИЗМЕНЕНО: Переходим на Data Binding ---
	peerIDLabelText binding.String
	statusLabelText binding.String
	messages        binding.UntypedList // <-- Связанный список строк для чата
	contacts        binding.UntypedList // <-- Связанный список контактов

	// Состояние
	currentChatPeerID string
}

func NewAppUI(core newcore.ICoreController) *AppUI {
	a := app.NewWithID("com.owlwhisper.desktop")
	win := a.NewWindow("Owl Whisper - Fyne GUI Test")

	// 1. Сначала создаем пустую структуру AppUI
	ui := &AppUI{
		coreController:  core,
		app:             a,
		mainWindow:      win,
		messages:        binding.NewUntypedList(),
		contacts:        binding.NewUntypedList(),
		peerIDLabelText: binding.NewString(),
		statusLabelText: binding.NewString(),
	}
	ui.peerIDLabelText.Set("PeerID: загрузка...")
	ui.statusLabelText.Set("Статус: инициализация...")

	// --- ИСПРАВЛЕННЫЙ ПОРЯДОК ---

	// 2. ТЕПЕРЬ создаем все сервисы. `ui` уже существует.
	ui.contactService = services.NewContactService(core, ui.refreshContacts, ui)
	ui.chatService = services.NewChatService(core, ui.contactService.Provider, func(newWidget fyne.CanvasObject) {
		// Callback для добавления нового сообщения в UI
		ui.messages.Append(newWidget)
	})
	// 3. Создаем FileService. Ему больше не нужен callback.
	ui.fileService = services.NewFileService(core, ui.contactService, ui.chatService)

	// 4. Создаем Диспетчер, передавая ему все три сервиса
	ui.dispatcher = services.NewMessageDispatcher(ui.contactService, ui.chatService, ui.fileService)

	// 3. И ТОЛЬКО ТЕПЕРЬ, когда все сервисы готовы, мы создаем UI,
	// который будет их использовать.
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
	ui.peerIDLabel = widget.NewLabelWithData(ui.peerIDLabelText)
	ui.statusLabel = widget.NewLabelWithData(ui.statusLabelText)
	ui.peerIDLabel.Selectable = true

	contactsList := widget.NewListWithData(
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

	contactsList.OnSelected = func(id widget.ListItemID) {
		item, _ := ui.contacts.GetValue(id)
		contact := item.(*services.Contact)
		ui.currentChatPeerID = contact.PeerID

		// TODO: Загрузка истории чата из БД

		ui.statusLabelText.Set(fmt.Sprintf("Открыт чат с %s", contact.Nickname))
		log.Printf("INFO: Выбран чат с %s", contact.FullAddress())
		contactsList.UnselectAll()
	}

	addContactButton := widget.NewButtonWithIcon("Добавить контакт", theme.ContentAddIcon(), func() {
		ui.ShowSearchDialog()
	})
	leftPanel := container.NewBorder(container.NewVBox(widget.NewLabel("Контакты:"), addContactButton), nil, nil, nil, contactsList)

	chatMessages := widget.NewListWithData(
		ui.messages,
		// Этот контейнер будет "хостом" для наших виджетов
		func() fyne.CanvasObject {
			return container.NewStack()
		},
		// Эта функция будет класть нужный виджет в контейнер
		func(item binding.DataItem, o fyne.CanvasObject) {
			untyped, _ := item.(binding.Untyped).Get()

			// ПРОВЕРЯЕМ ТИП: Если это виджет, используем его.
			if wid, ok := untyped.(fyne.CanvasObject); ok {
				o.(*fyne.Container).Objects = []fyne.CanvasObject{wid}
				o.(*fyne.Container).Refresh()
			}
		},
	)

	messageEntry := widget.NewEntry()
	messageEntry.SetPlaceHolder("Выберите контакт для начала общения...")

	sendButton := widget.NewButton("Отправить", func() {
		text := messageEntry.Text
		if text == "" || ui.currentChatPeerID == "" {
			return // Нечего отправлять или некому
		}

		// --- ИЗМЕНЕНО: Используем ChatService для отправки ---
		err := ui.chatService.SendTextMessage(ui.currentChatPeerID, text)
		if err != nil {
			log.Printf("ERROR: [UI] Не удалось отправить сообщение: %v", err)
			ui.statusLabelText.Set(fmt.Sprintf("Ошибка отправки: %v", err))
			return
		}

		// Оптимистичное обновление UI: добавляем свое сообщение сразу
		myProfile := ui.contactService.GetMyProfile()
		fullMessage := fmt.Sprintf("%s: %s", myProfile.FullAddress(), text)
		textWidget := widget.NewLabel(fullMessage)
		textWidget.Wrapping = fyne.TextWrapWord
		ui.messages.Append(textWidget)

		messageEntry.SetText("")
	})

	fileButton := widget.NewButtonWithIcon("", theme.FileIcon(), func() {
		if ui.currentChatPeerID == "" {
			ui.statusLabelText.Set("Сначала выберите контакт для отправки файла.")
			return
		}

		// 2. Открываем системный диалог выбора файла
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return // Пользователь отменил выбор или произошла ошибка
			}
			filePath := reader.URI().Path()
			log.Printf("INFO: [UI] Выбран файл для отправки: %s", filePath)

			// 3. Вызываем метод FileService для анонса в фоновой горутине
			go func() {
				// ИЗМЕНЕНО: AnnounceFile теперь возвращает виджет
				card, announceErr := ui.fileService.AnnounceFile(ui.currentChatPeerID, filePath)
				if announceErr != nil {
					// TODO: Показать ошибку в UI
					log.Printf("ERROR: [UI] Ошибка анонса файла: %v", announceErr)
					return
				}
				if card != nil {
					// Добавляем СВОЙ виджет в СВОЙ чат
					ui.messages.Append(card)
				}
			}()

		}, ui.mainWindow)
	})

	bottomPanel := container.NewBorder(nil, nil, fileButton, sendButton, messageEntry)
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
				ui.peerIDLabelText.Set("PeerID: " + payload.PeerID)
				ui.contactService.UpdateMyProfile(payload.PeerID)

				ui.ShowNicknameDialog(func(nickname string) {
					ui.contactService.SetMyNickname(nickname)
					profile := ui.contactService.GetMyProfile()
					ui.mainWindow.SetTitle(fmt.Sprintf("Owl Whisper - %s", profile.FullAddress()))
				})
			}

		case "NewMessage":
			if payload, ok := event.Payload.(newcore.NewMessagePayload); ok {
				// ВСЯ СЛОЖНАЯ ЛОГИКА ТЕПЕРЬ ЗДЕСЬ:
				// Просто передаем сырые данные в диспетчер.
				ui.dispatcher.HandleIncomingData(payload.SenderID, payload.Data)
			}

		case "PeerConnected":
			if payload, ok := event.Payload.(newcore.PeerStatusPayload); ok {
				ui.statusLabelText.Set(fmt.Sprintf("Статус: Подключен пир %s", payload.PeerID[:8]))
				ui.contactService.UpdateContactStatus(payload.PeerID, services.StatusOnline)
			}

		case "PeerDisconnected":
			if payload, ok := event.Payload.(newcore.PeerStatusPayload); ok {
				ui.statusLabelText.Set(fmt.Sprintf("Статус: Отключен пир %s", payload.PeerID[:8]))
				ui.contactService.UpdateContactStatus(payload.PeerID, services.StatusOffline)
			}
		case "NewIncomingStream":
			if payload, ok := event.Payload.(newcore.NewIncomingStreamPayload); ok {
				ui.fileService.HandleIncomingStream(payload)
			}
		case "StreamDataReceived":
			if payload, ok := event.Payload.(newcore.StreamDataReceivedPayload); ok {
				ui.fileService.HandleStreamData(payload)
			}
		case "StreamClosed":
			if payload, ok := event.Payload.(newcore.StreamClosedPayload); ok {
				// Передаем событие в FileService для завершения
				ui.fileService.HandleStreamClosed(payload)
			}
		}
	}
}

func (ui *AppUI) OnProfileReceived(senderID string, profile *protocol.ProfileInfo) {
	ui.ShowConfirmContactDialog(profile, senderID)
}

func (ui *AppUI) OnContactRequestReceived(senderID string, profile *protocol.ProfileInfo) {
	ui.ShowContactRequestDialog(senderID, profile)
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
