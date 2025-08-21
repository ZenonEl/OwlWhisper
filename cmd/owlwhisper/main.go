package main

import (
	"os"

	"OwlWhisper/internal/app"
	"OwlWhisper/internal/core"
)

func main() {
	// Создаем приложение
	application, err := app.NewApp()
	if err != nil {
		core.Error("❌ Не удалось создать приложение: %v", err)
		os.Exit(1)
	}

	// Запускаем приложение
	if err := application.Run(); err != nil {
		core.Error("❌ Ошибка выполнения приложения: %v", err)
		os.Exit(1)
	}
}
