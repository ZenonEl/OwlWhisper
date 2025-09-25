// Путь: cmd/fyne-gui/ui/search_dialog.go
package ui

import (
	"fmt"
	// Убираем лишние импорты: time, newcore, protocol, uuid, proto, peer
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
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
			// ИЗМЕНЕНО: Просто вызываем метод сервиса, передавая ему колбэки
			// для обновления UI. UI больше не занимается сетевой логикой.
			go ui.contactService.SearchAndVerifyContact(
				searchEntry.Text,
				func(status string) { ui.statusLabelText.Set(status) },                     // onStatusUpdate
				func(err error) { ui.statusLabelText.Set(fmt.Sprintf("Ошибка: %v", err)) }, // onError
			)
		},
		ui.mainWindow,
	)

	searchDialog.Resize(fyne.NewSize(400, 150))
	searchDialog.Show()
}

// Метод performSearchAndPing УДАЛЕН. Вся его логика переезжает в ContactService.
