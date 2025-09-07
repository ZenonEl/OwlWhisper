// Путь: cmd/fyne-gui/ui/ui.go

package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	newcore "OwlWhisper/cmd/fyne-gui/new-core" // <-- Убедитесь, что путь импорта верный
)

// AppUI инкапсулирует все Fyne-виджеты и состояние UI.
type AppUI struct {
	coreController newcore.ICoreController

	// Виджеты
	peerIDLabel  *widget.Label
	statusLabel  *widget.Label
	peersList    *widget.List
	chatMessages *widget.List
	messageEntry *widget.Entry
	sendButton   *widget.Button
	app          fyne.App // <-- Нам нужна ссылка на приложение для CallOnMain
	mainWindow   fyne.Window

	// Состояние UI (должно изменяться только в основном потоке)
	connectedPeers []string
	messages       []string
}

// NewAppUI - конструктор для нашего UI.
func NewAppUI(core newcore.ICoreController) *AppUI {
	a := app.New()
	win := a.NewWindow("Owl Whisper - Fyne GUI Test")

	ui := &AppUI{
		coreController: core,
		app:            a, // Сохраняем экземпляр приложения
		mainWindow:     win,
		connectedPeers: []string{},
		messages:       []string{},
	}

	win.SetContent(ui.buildUI())
	win.Resize(fyne.NewSize(800, 600))

	return ui
}

// Start запускает UI и все фоновые процессы.
func (ui *AppUI) Start() {
	// Запускаем фоновый цикл прослушивания событий от Core.
	go ui.StartEventLoop()

	ui.mainWindow.SetOnClosed(func() {
		if err := ui.coreController.Stop(); err != nil {
			log.Printf("ERROR: Ошибка при остановке ядра: %v", err)
		}
		log.Println("INFO: Приложение завершило работу.")
	})

	// Эта функция блокирует и запускает главный цикл Fyne.
	ui.mainWindow.ShowAndRun()
}

// buildUI создает и компонует все виджеты Fyne.
func (ui *AppUI) buildUI() fyne.CanvasObject {
	ui.peerIDLabel = widget.NewLabel("PeerID: загрузка...")
	ui.statusLabel = widget.NewLabel("Статус: инициализация...")

	ui.peersList = widget.NewList(
		func() int { return len(ui.connectedPeers) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			shortID := ui.connectedPeers[i]
			if len(shortID) > 12 {
				shortID = shortID[:6] + "..." + shortID[len(shortID)-6:]
			}
			o.(*widget.Label).SetText(shortID)
		},
	)
	leftPanel := container.NewBorder(widget.NewLabel("Подключенные пиры:"), nil, nil, nil, ui.peersList)

	ui.chatMessages = widget.NewList(
		func() int { return len(ui.messages) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(ui.messages[i])
		},
	)

	ui.messageEntry = widget.NewEntry()
	ui.messageEntry.SetPlaceHolder("Введите сообщение...")

	ui.sendButton = widget.NewButton("Отправить", func() {
		text := ui.messageEntry.Text
		if text != "" {
			ui.coreController.BroadcastData([]byte(text))
			ui.messageEntry.SetText("")
		}
	})

	bottomPanel := container.NewBorder(nil, nil, nil, ui.sendButton, ui.messageEntry)
	rightPanel := container.NewBorder(nil, bottomPanel, nil, nil, ui.chatMessages)
	split := container.NewHSplit(leftPanel, rightPanel)
	split.Offset = 0.3

	return container.NewBorder(
		container.NewVBox(ui.peerIDLabel, ui.statusLabel, widget.NewSeparator()),
		nil, nil, nil,
		split,
	)
}

func (ui *AppUI) StartEventLoop() {
	// Your event loop logic
	for event := range ui.coreController.Events() {
		fyne.Do(func() {
			ui.handleEventInMainThread(event) // Schedule UI updates safely
		})
	}
}

// eventLoop - это фоновая горутина, которая слушает Core и ПЛАНИРУЕТ обновления UI.
func (ui *AppUI) eventLoop() {
	myID := ui.coreController.GetMyPeerID()

	// Use a proper mechanism to schedule on the main thread
	// For example, use a channel or handle in the main loop
	go func() {
		// Queue the update safely
		ui.app.Run() // If this is in the main context, ensure it's not called multiple times
		// Directly update if in main thread, or use bindings
		ui.peerIDLabel.SetText("PeerID: " + myID) // Ensure this is called on main thread
	}()

	for event := range ui.coreController.Events() {
		capturedEvent := event     // Avoid closure issues
		go func(e newcore.Event) { // Pass event explicitly
			// Schedule the handler on the main thread
			// Fyne recommends using app lifecycle or bindings; here's a safe pattern
			ui.app.Run() // Ensure this is contextual
			ui.handleEventInMainThread(e)
		}(capturedEvent) // Immediate invocation
	}
}

// handleEventInMainThread ВЫПОЛНЯЕТСЯ ГАРАНТИРОВАННО В ОСНОВНОМ ПОТОКЕ.
// Здесь безопасно изменять любые виджеты.
func (ui *AppUI) handleEventInMainThread(event newcore.Event) {
	switch event.Type {
	case "PeerConnected":
		if payload, ok := event.Payload.(newcore.PeerStatusPayload); ok {
			ui.statusLabel.SetText(fmt.Sprintf("Статус: Подключен пир %s", payload.PeerID[:8]))
			// Обновляем список пиров
			ui.connectedPeers = ui.coreController.GetConnectedPeers()
			ui.peersList.Refresh()
		}
	case "PeerDisconnected":
		if payload, ok := event.Payload.(newcore.PeerStatusPayload); ok {
			ui.statusLabel.SetText(fmt.Sprintf("Статус: Отключен пир %s", payload.PeerID[:8]))
			// Обновляем список пиров
			ui.connectedPeers = ui.coreController.GetConnectedPeers()
			ui.peersList.Refresh()
		}
	case "NewMessage":
		if payload, ok := event.Payload.(newcore.NewMessagePayload); ok {
			msgText := string(payload.Data)
			senderShort := payload.SenderID[:8]
			fullMessage := fmt.Sprintf("[%s]: %s", senderShort, msgText)

			// Обновляем UI
			ui.messages = append(ui.messages, fullMessage)
			ui.chatMessages.Refresh()
			ui.chatMessages.ScrollToBottom()
		}
	}
}
