// Путь: cmd/fyne-gui/main.go

package main

import (
	"log"

	"github.com/libp2p/go-libp2p/core/crypto"
	// Импортируем наши новые пакеты
	newcore "OwlWhisper/cmd/fyne-gui/new-core" // <-- Убедитесь, что путь верный

	ui "OwlWhisper/cmd/fyne-gui/ui" // <-- Убедитесь, что путь верный
)

func main() {
	// --- Шаг 1: Инициализация Core ---
	log.Println("INFO: Инициализация ядра...")

	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		log.Fatalf("Ошибка генерации ключа: %v", err)
	}

	cfg := newcore.DefaultConfig()
	core, err := newcore.NewCoreController(privKey, cfg)
	if err != nil {
		log.Fatalf("Ошибка создания Core Controller: %v", err)
	}

	// Запускаем Core в фоновом режиме.
	go func() {
		if err := core.Start(); err != nil {
			log.Fatalf("КРИТИЧЕСКАЯ ОШИБКА: Не удалось запустить ядро: %v", err)
		}
	}()

	// --- Шаг 2: Создание и запуск UI ---
	log.Println("INFO: Инициализация и запуск UI...")

	// Создаем UI, передавая ему уже готовый контроллер (Dependency Injection).
	appUI := ui.NewAppUI(core)

	// Запускаем UI. Эта функция блокирует основной поток до закрытия окна.
	appUI.Start()

	log.Println("INFO: UI завершил работу. Приложение закрывается.")
}
