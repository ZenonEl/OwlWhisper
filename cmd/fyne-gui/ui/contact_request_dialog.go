// Путь: cmd/fyne-gui/ui/contact_request_dialog.go
package ui

import (
	"fmt"

	protocol "OwlWhisper/cmd/fyne-gui/new-core/protocol"
	"OwlWhisper/cmd/fyne-gui/services"

	"fyne.io/fyne/v2/dialog"
)

// ОБНОВЛЕННАЯ СИГНАТУРА
func (ui *AppUI) ShowContactRequestDialog(senderID string, profile *protocol.ProfilePayload, fingerprint string, status services.VerificationStatus) {
	fullAddress := fmt.Sprintf("%s#%s", profile.Nickname, profile.Discriminator)

	var statusText string
	switch status {
	case services.StatusVerified:
		statusText = " (ПРОВЕРЕННЫЙ КОНТАКТ)"
	case services.StatusBlocked:
		statusText = " (ЗАБЛОКИРОВАН)"
	default:
		statusText = " (НЕ ПРОВЕРЕН)"
	}

	dialog.ShowConfirm(
		"Запрос на добавление в контакты",
		fmt.Sprintf("Пользователь %s%s (%s) хочет добавить вас. Отпечаток: %s.", fullAddress, statusText, senderID[:8], fingerprint[:19]),
		func(confirm bool) {
			if !confirm {
				// TODO: Отправить сообщение об отклонении
				ui.statusLabelText.Set(fmt.Sprintf("Запрос от %s отклонен.", fullAddress))
				return
			}
			ui.statusLabelText.Set(fmt.Sprintf("Подтверждение дружбы с %s...", fullAddress))
			// ИЗМЕНЕНО: Вызов AcceptContactRequest тоже изменится.
			// Пока создадим заглушку для нового метода.
			go ui.contactService.AcceptChatInvitation(senderID)
		},
		ui.mainWindow,
	)
}
