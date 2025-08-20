package main

import (
	"log"
	"os"

	"OwlWhisper/internal/app"
)

func main() {
	// Создаем приложение
	application, err := app.NewApp()
	if err != nil {
		log.Fatalf("❌ Не удалось создать приложение: %v", err)
	}

	// Запускаем приложение
	if err := application.Run(); err != nil {
		log.Printf("❌ Ошибка выполнения приложения: %v", err)
		os.Exit(1)
	}
}
