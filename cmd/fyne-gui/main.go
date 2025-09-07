// Создайте этот файл по пути: cmd/fyne-gui/main.go

package main

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	// ВАЖНО: Импортируем наш пакет new-core.
	// Замените на ваш реальный путь к модулю Go.
	newcore "OwlWhisper/cmd/fyne-gui/new-core"

	"github.com/libp2p/go-libp2p/core/crypto"
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
	app          fyne.App
	mainWindow   fyne.Window

	// Состояние UI
	connectedPeers []string
	messages       []string
}

func main() {
	// --- Шаг 1: Инициализация Core ---
	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		log.Fatalf("Ошибка генерации ключа: %v", err)
	}
	cfg := newcore.DefaultConfig()
	core, err := newcore.NewCoreController(privKey, cfg)
	if err != nil {
		log.Fatalf("Ошибка создания Core Controller: %v", err)
	}

	// --- Шаг 2: Создание и настройка Fyne GUI ---
	a := app.New()
	win := a.NewWindow("Owl Whisper - Fyne GUI Test")
	ui := &AppUI{
		coreController: core,
		app:            a,
		mainWindow:     win,
		connectedPeers: []string{},
		messages:       []string{},
	}

	// Устанавливаем содержимое окна ДО запуска Core.
	// Это важно, чтобы виджеты уже существовали.
	win.SetContent(ui.buildUI())
	win.Resize(fyne.NewSize(800, 600))

	// --- Шаг 3: Запуск Core и цикла событий ---

	// Запускаем Core в отдельной горутине.
	go func() {
		log.Println("INFO: Запуск Core в фоновом режиме...")
		if err := ui.coreController.Start(); err != nil {
			log.Printf("ERROR: Не удалось запустить ядро: %v", err)
			// Безопасно обновляем UI из горутины
			ui.statusLabel.SetText(fmt.Sprintf("Ошибка запуска: %v", err))
		}
	}()

	// Запускаем цикл прослушивания событий от Core в другой горутине.
	win.SetOnClosed(func() {
		if err := ui.coreController.Stop(); err != nil {
			log.Printf("ERROR: Ошибка при остановке ядра: %v", err)
		}
		log.Println("INFO: Приложение завершило работу.")
	})

	// Запускаем цикл событий ПОСЛЕ того, как окно готово.
	go ui.eventLoop()

	win.ShowAndRun()

	// --- Шаг 4: Сборка и запуск UI ---

	// Устанавливаем содержимое окна и запускаем главный цикл Fyne.
	win.SetContent(ui.buildUI())
	win.Resize(fyne.NewSize(800, 600))
	win.ShowAndRun()

	// Корректно останавливаем Core при закрытии окна.
	if err := ui.coreController.Stop(); err != nil {
		log.Printf("ERROR: Ошибка при остановке ядра: %v", err)
	}
	log.Println("INFO: Приложение завершило работу.")
}

// buildUI создает и компонует все виджеты Fyne.
func (ui *AppUI) buildUI() fyne.CanvasObject {
	ui.peerIDLabel = widget.NewLabel("PeerID: загрузка...")
	ui.statusLabel = widget.NewLabel("Статус: инициализация...")

	// --- Левая панель: Пиры ---
	ui.peersList = widget.NewList(
		func() int {
			return len(ui.connectedPeers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			shortID := ui.connectedPeers[i]
			if len(shortID) > 12 {
				shortID = shortID[:6] + "..." + shortID[len(shortID)-6:]
			}
			o.(*widget.Label).SetText(shortID)
		},
	)

	leftPanel := container.NewBorder(widget.NewLabel("Подключенные пиры:"), nil, nil, nil, ui.peersList)

	// --- Правая панель: Чат ---
	ui.chatMessages = widget.NewList(
		func() int {
			return len(ui.messages)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(ui.messages[i])
		},
	)

	ui.messageEntry = widget.NewEntry()
	ui.messageEntry.SetPlaceHolder("Введите сообщение...")

	ui.sendButton = widget.NewButton("Отправить", func() {
		text := ui.messageEntry.Text
		if text != "" {
			// Пока отправляем всем (broadcast) для простоты теста.
			ui.coreController.BroadcastData([]byte(text))
			ui.messageEntry.SetText("")
		}
	})

	bottomPanel := container.NewBorder(nil, nil, nil, ui.sendButton, ui.messageEntry)
	rightPanel := container.NewBorder(nil, bottomPanel, nil, nil, ui.chatMessages)

	// --- Сборка всего вместе ---
	split := container.NewHSplit(leftPanel, rightPanel)
	split.Offset = 0.3 // Левая панель занимает 30% ширины

	return container.NewBorder(
		container.NewVBox(ui.peerIDLabel, ui.statusLabel, widget.NewSeparator()),
		nil, nil, nil,
		split,
	)
}

// eventLoop - это сердце нашего UI, слушающее события от Core.
func (ui *AppUI) eventLoop() {
	// Сразу после запуска обновляем наш PeerID
	time.Sleep(1 * time.Second)
	myID := ui.coreController.GetMyPeerID()
	if myID != "" {
		ui.peerIDLabel.SetText("PeerID: " + myID)
	}
	ui.peerIDLabel.SetText("PeerID: " + myID)

	// Бесконечный цикл чтения из канала событий
	for event := range ui.coreController.Events() {
		switch event.Type {
		case "PeerConnected":
			payload, ok := event.Payload.(newcore.PeerStatusPayload)
			if ok {
				ui.statusLabel.SetText(fmt.Sprintf("Статус: Подключен пир %s", payload.PeerID[:8]))
				ui.updatePeersList()
			}
		case "PeerDisconnected":
			payload, ok := event.Payload.(newcore.PeerStatusPayload)
			if ok {
				ui.statusLabel.SetText(fmt.Sprintf("Статус: Отключен пир %s", payload.PeerID[:8]))
				ui.updatePeersList()
			}
		case "NewMessage":
			payload, ok := event.Payload.(newcore.NewMessagePayload)
			if ok {
				// ВАЖНО: Мы еще не реализовали Protobuf, поэтому пока просто
				// отображаем сырые байты как строку.
				msgText := string(payload.Data)
				senderShort := payload.SenderID[:8]

				ui.messages = append(ui.messages, fmt.Sprintf("[%s]: %s", senderShort, msgText))
				ui.chatMessages.Refresh()
				// Прокручиваем список вниз к новому сообщению
				ui.chatMessages.ScrollToBottom()
			}
		}
	}
}

// updatePeersList безопасно обновляет список пиров в UI.
func (ui *AppUI) updatePeersList() {
	ui.connectedPeers = ui.coreController.GetConnectedPeers()
	ui.peersList.Refresh()
}
