// Путь: cmd/fyne-gui/main.go

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/libp2p/go-libp2p/core/crypto"
	newcore "OwlWhisper/cmd/fyne-gui/new-core"
	"OwlWhisper/cmd/fyne-gui/ui"
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("INFO: Получен сигнал завершения от ОС, инициируем остановку...")
	}()

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

	go func() {
		if err := core.Start(); err != nil {
			log.Fatalf("КРИТИЧЕСКАЯ ОШИБКА: Не удалось запустить ядро: %v", err)
		}
	}()

	log.Println("INFO: Инициализация и запуск UI...")

	appUI := ui.NewAppUI(core)

	appUI.Start()

	log.Println("INFO: Приложение полностью остановлено.")
}
