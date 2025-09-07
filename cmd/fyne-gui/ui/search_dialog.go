// Путь: cmd/fyne-gui/ui/search_dialog.go

package ui

import (
	"fmt"
	"log"
	"strings"
	"time"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/protobuf/proto"
)

func (ui *AppUI) ShowSearchDialog() {
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Введите PeerID или nickname#discriminator")

	searchDialog := dialog.NewCustomConfirm(
		"Найти и добавить контакт",
		"Найти",
		"Отмена",
		container.NewVBox(
			widget.NewLabel("Введите адрес контакта для поиска в сети."),
			searchEntry,
		),
		func(confirm bool) {
			if !confirm {
				return
			}
			// Запускаем всю логику поиска в фоновой горутине, чтобы не блокировать UI
			go ui.performSearchAndPing(searchEntry.Text)
		},
		ui.mainWindow,
	)

	searchDialog.Resize(fyne.NewSize(400, 150))
	searchDialog.Show()
}

// performSearchAndPing реализует полный сценарий поиска и верификации.
func (ui *AppUI) performSearchAndPing(address string) {
	ui.statusLabelText.Set("Статус: Поиск...") // Используем Data Binding для обновления

	var targetPeer peer.AddrInfo
	var err error

	// --- ШАГ 1: ОПРЕДЕЛЯЕМ ТИП ПОИСКА И НАХОДИМ ПИРА ---
	// Проверяем, является ли введенный адрес PeerID
	if strings.HasPrefix(address, "12D3KooW") {
		log.Printf("INFO: [Search] Выполняется поиск по PeerID: %s", address)

		// Напрямую ищем пира по его ID
		addrInfo, findErr := ui.coreController.FindPeer(address)
		if findErr != nil {
			ui.statusLabelText.Set(fmt.Sprintf("Ошибка: %v", findErr))
			return
		}
		targetPeer = *addrInfo

	} else {
		// Иначе, считаем, что это nickname#discriminator и ищем по контенту
		log.Printf("INFO: [Search] Выполняется поиск по никнейму: %s", address)

		// 1. Вычисляем ContentID
		contentID, createErr := newcore.CreateContentID(address)
		if createErr != nil {
			ui.statusLabelText.Set(fmt.Sprintf("Ошибка: Неверный формат адреса: %v", createErr))
			return
		}

		// 2. Ищем в сети
		providers, findErr := ui.coreController.FindProvidersForContent(contentID)
		if findErr != nil {
			ui.statusLabelText.Set(fmt.Sprintf("Ошибка поиска: %v", findErr))
			return
		}
		if len(providers) == 0 {
			ui.statusLabelText.Set("Контакт не найден (офлайн или не существует).")
			return
		}
		targetPeer = providers[0] // Берем первого найденного
	}

	// --- ШАГ 2: "ПИНГУЕМ" НАЙДЕННОГО ПИРА ДЛЯ ПОЛУЧЕНИЯ ПРОФИЛЯ ---
	statusMsg := fmt.Sprintf("Контакт найден! PeerID: %s. Запрос профиля...", targetPeer.ID.ShortString())
	ui.statusLabelText.Set(statusMsg)

	// Создаем Protobuf-запрос ("пинг")
	req := &protocol.ProfileRequest{}
	payload := &protocol.Envelope_ProfileRequest{ProfileRequest: req}

	envelope := &protocol.Envelope{
		MessageId:     uuid.New().String(),
		SenderId:      ui.coreController.GetMyPeerID(),
		TimestampUnix: time.Now().Unix(),
		ChatType:      protocol.Envelope_PRIVATE,
		ChatId:        targetPeer.ID.String(),
		Payload:       payload,
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		ui.statusLabelText.Set(fmt.Sprintf("Ошибка: не удалось создать запрос: %v", err))
		return
	}

	// Отправляем "пинг"
	if err := ui.coreController.SendDataToPeer(targetPeer.ID.String(), data); err != nil {
		ui.statusLabelText.Set(fmt.Sprintf("Ошибка: не удалось отправить запрос (контакт может быть офлайн): %v", err))
		return
	}

	ui.statusLabelText.Set(fmt.Sprintf("Запрос профиля отправлен %s. Ожидание ответа...", targetPeer.ID.ShortString()))

	// Дальше происходит магия:
	// 1. Клиент на другой стороне получит наш ProfileRequest.
	// 2. Его ContactService автоматически отправит нам в ответ ProfileResponse.
	// 3. Наш eventLoop получит это событие.
	// 4. Наш handleEvent вызовет наш ContactService, который добавит контакт в хранилище.
	// 5. ContactService вызовет callback, который обновит список контактов в UI.
	// Вся эта логика уже реализована, и теперь мы ее задействуем.
}
