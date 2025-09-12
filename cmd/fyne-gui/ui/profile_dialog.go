// Путь: cmd/fyne-gui/ui/profile_dialog.go

package ui

import (
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ShowNicknameDialog показывает диалог для ввода никнейма.
// Он вызывает callback `onSubmit` с введенным ником.
func (ui *AppUI) ShowNicknameDialog(onSubmit func(nickname string)) {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Например, alice")

	dialog.ShowForm(
		"Выберите ваш никнейм", // Заголовок
		"Готово",               // Текст кнопки подтверждения
		"Отмена",               // Текст кнопки отмены
		[]*widget.FormItem{
			widget.NewFormItem("Никнейм", entry),
		},
		func(confirmed bool) {
			if !confirmed || entry.Text == "" {
				// Если пользователь нажал "Отмена" или ничего не ввел,
				// мы можем либо выйти, либо сгенерировать случайный ник.
				// Пока что просто будем использовать "Me".
				onSubmit("Me")
				return
			}
			onSubmit(entry.Text)
		},
		ui.mainWindow,
	)
}
