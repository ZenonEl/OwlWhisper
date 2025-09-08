// Путь: cmd/fyne-gui/ui/contact_dialog.go

package ui

import (
	"fmt"

	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ShowConfirmContactDialog показывает диалог с информацией о найденном профиле
// и кнопками для добавления в контакты.
func (ui *AppUI) ShowConfirmContactDialog(profile *protocol.ProfileInfo, peerID string) {
	// Формируем красивое представление профиля
	fullAddress := fmt.Sprintf("%s#%s", profile.Nickname, profile.Discriminator)

	// Создаем виджеты для отображения информации
	addressLabel := widget.NewLabel("Адрес: " + fullAddress)
	peerIDLabel := widget.NewLabel("PeerID: " + peerID[:12] + "...") // Показываем сокращенный PeerID

	// Создаем диалоговое окно
	confirmDialog := dialog.NewCustomConfirm(
		"Найден контакт",
		"Добавить", // Текст кнопки подтверждения
		"Отмена",   // Текст кнопки отмены
		container.NewVBox(
			widget.NewLabel("Найден следующий пользователь. Хотите добавить его в контакты?"),
			widget.NewSeparator(),
			addressLabel,
			peerIDLabel,
		),
		func(confirm bool) {
			if !confirm {
				// Пользователь нажал "Отмена"
				ui.statusLabelText.Set("Статус: Добавление контакта отменено.")
				return
			}

			// Пользователь нажал "Добавить".
			// Запускаем Фазу 3: отправку ContactRequest.
			ui.statusLabelText.Set(fmt.Sprintf("Статус: Отправка запроса на добавление %s...", fullAddress))
			go ui.contactService.SendContactRequest(peerID, ui.contactService.GetMyProfile())
		},
		ui.mainWindow,
	)

	confirmDialog.Show()
}
