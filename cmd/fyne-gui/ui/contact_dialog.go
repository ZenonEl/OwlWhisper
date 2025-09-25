// Путь: cmd/fyne-gui/ui/contact_dialog.go
package ui

import (
	"fmt"

	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ОБНОВЛЕННАЯ СИГНАТУРА
func (ui *AppUI) ShowConfirmContactDialog(peerID string, profile *protocol.ProfilePayload, fingerprint string) {
	fullAddress := fmt.Sprintf("%s#%s", profile.Nickname, profile.Discriminator)

	addressLabel := widget.NewLabel("Адрес: " + fullAddress)
	peerIDLabel := widget.NewLabel("PeerID: " + peerID[:12] + "...")

	// НОВОЕ ПОЛЕ: Отображаем отпечаток безопасности
	fingerprintLabel := widget.NewLabel("Отпечаток:")
	fingerprintEntry := widget.NewEntry()
	fingerprintEntry.SetText(fingerprint)
	fingerprintEntry.Disable() // Чтобы его нельзя было редактировать

	confirmDialog := dialog.NewCustomConfirm(
		"Найден контакт",
		"Добавить",
		"Отмена",
		container.NewVBox(
			widget.NewLabel("Найден следующий пользователь. Хотите добавить его в контакты?"),
			widget.NewSeparator(),
			addressLabel,
			peerIDLabel,
			fingerprintLabel,
			fingerprintEntry,
			widget.NewSeparator(),
			widget.NewLabel("Сверьте отпечаток с владельцем контакта для 100% гарантии безопасности."),
		),
		func(confirm bool) {
			if !confirm {
				ui.statusLabelText.Set("Статус: Добавление контакта отменено.")
				return
			}

			// ИЗМЕНЕНО: Вызов SendContactRequest теперь требует больше данных.
			// Мы пока не можем его полностью реализовать, т.к. нам нужен публичный ключ,
			// который UI не знает. ContactService должен будет его где-то сохранить.
			// Пока что просто инициируем процесс.
			ui.statusLabelText.Set(fmt.Sprintf("Статус: Отправка запроса на добавление %s...", fullAddress))
			go ui.contactService.InitiateNewChatFromProfile(peerID) // Нужен новый метод-хелпер в ContactService
		},
		ui.mainWindow,
	)
	confirmDialog.Show()
}
