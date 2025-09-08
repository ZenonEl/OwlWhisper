// Путь: cmd/fyne-gui/ui/contact_request_dialog.go

package ui

import (
	"fmt"

	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"

	"fyne.io/fyne/v2/dialog"
)

// ShowContactRequestDialog показывает диалог с запросом на добавление в контакты.
func (ui *AppUI) ShowContactRequestDialog(senderID string, profile *protocol.ProfileInfo) {
	fullAddress := fmt.Sprintf("%s#%s", profile.Nickname, profile.Discriminator)

	dialog.ShowConfirm(
		"Запрос на добавление в контакты", // Заголовок
		fmt.Sprintf("Пользователь %s (%s) хочет добавить вас в свой список контактов.", fullAddress, senderID[:8]), // Сообщение
		func(confirm bool) {
			if !confirm {
				// TODO: Отправить сообщение об отклонении
				ui.statusLabelText.Set(fmt.Sprintf("Запрос от %s отклонен.", fullAddress))
				return
			}

			// Пользователь нажал "Принять"
			ui.statusLabelText.Set(fmt.Sprintf("Подтверждение дружбы с %s...", fullAddress))
			go ui.contactService.AcceptContactRequest(senderID, profile)
		},
		ui.mainWindow,
	)
}
