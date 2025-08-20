package api

import (
	"fmt"
	"log"
	"time"
)

// ExampleUsage демонстрирует как использовать OwlWhisper API
func ExampleUsage() {
	// Создаем конфигурацию
	config := DefaultAPIConfig()
	config.EnableTUI = false // Отключаем TUI для примера
	config.DatabasePath = "example.db"

	// Создаем API
	api, err := NewOwlWhisperAPI(config)
	if err != nil {
		log.Fatalf("Ошибка создания API: %v", err)
	}

	// Запускаем API
	if err := api.Start(); err != nil {
		log.Fatalf("Ошибка запуска API: %v", err)
	}
	defer api.Stop()

	fmt.Printf("🚀 OwlWhisper запущен! PeerID: %s\n", api.GetMyPeerID())

	// Запускаем горутину для обработки сообщений
	go func() {
		for msg := range api.MessageChannel() {
			if msg.IsOutgoing {
				fmt.Printf("📤 Вы -> %s: %s\n", msg.RecipientID, msg.Text)
			} else {
				fmt.Printf("📥 %s: %s\n", msg.Sender, msg.Text)
			}
		}
	}()

	// Запускаем горутину для обработки пиров
	go func() {
		for peers := range api.PeerChannel() {
			fmt.Printf("🔌 Подключенные пиры: %d\n", len(peers))
			for _, peer := range peers {
				fmt.Printf("  - %s (%s)\n", peer.Nickname, peer.Status)
			}
		}
	}()

	// Ждем подключения пиров
	fmt.Println("⏳ Ожидание подключения пиров...")
	for {
		status := api.GetConnectionStatus()
		if status.IsConnected {
			fmt.Printf("✅ Подключено к %d пирам\n", status.PeerCount)
			break
		}
		time.Sleep(2 * time.Second)
	}

	// Отправляем тестовое сообщение
	request := SendMessageRequest{
		Text:     "Привет из OwlWhisper API!",
		ChatType: "broadcast",
	}

	if err := api.SendMessage(request); err != nil {
		log.Printf("❌ Ошибка отправки сообщения: %v", err)
	} else {
		fmt.Println("✅ Сообщение отправлено")
	}

	// Получаем историю сообщений
	history, err := api.GetHistory(10)
	if err != nil {
		log.Printf("❌ Ошибка получения истории: %v", err)
	} else {
		fmt.Printf("📚 История сообщений (%d из %d):\n", len(history.Messages), history.TotalCount)
		for _, msg := range history.Messages {
			fmt.Printf("  %s - %s: %s\n", msg.Timestamp.Format("15:04:05"), msg.Sender, msg.Text)
		}
	}

	// Работаем 30 секунд
	fmt.Println("⏰ Работаем 30 секунд...")
	time.Sleep(30 * time.Second)

	fmt.Println("👋 Завершение работы")
}

// SimpleExample показывает минимальный пример использования
func SimpleExample() error {
	// Создаем API с настройками по умолчанию
	api, err := NewOwlWhisperAPI(nil)
	if err != nil {
		return fmt.Errorf("не удалось создать API: %w", err)
	}

	// Запускаем
	if err := api.Start(); err != nil {
		return fmt.Errorf("не удалось запустить API: %w", err)
	}
	defer api.Stop()

	// Отправляем сообщение
	return api.SendMessage(SendMessageRequest{
		Text:     "Hello, World!",
		ChatType: "broadcast",
	})
}
