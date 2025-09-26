// Путь: cmd/fyne-gui/main.go
package main

import (
	"log"

	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	"OwlWhisper/cmd/fyne-gui/ui"

	"github.com/libp2p/go-libp2p/core/crypto"
)

func main() {
	log.Println("INFO: Запуск Owl Whisper...")

	// 1. Создаем криптографическую основу
	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		log.Fatalf("Ошибка генерации ключа: %v", err)
	}

	// 2. Создаем Core
	cfg := newcore.DefaultConfig()
	core, err := newcore.NewCoreController(privKey, cfg)
	if err != nil {
		log.Fatalf("Ошибка создания Core Controller: %v", err)
	}

	// 3. Создаем и запускаем UI. Вся остальная инициализация будет внутри AppUI.
	appUI := ui.NewAppUI(core, privKey)

	log.Println("INFO: Инициализация ядра...")
	go func() {
		if err := core.Start(); err != nil {
			log.Fatalf("КРИТИЧЕСКАЯ ОШИБКА: Не удалось запустить ядро: %v", err)
		}
	}()

	log.Println("INFO: Запуск UI...")
	appUI.Start()

	log.Println("INFO: Приложение полностью остановлено.")
}
