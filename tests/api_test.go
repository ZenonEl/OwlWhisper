package tests

import (
	"sync"
	"testing"
	"time"

	"OwlWhisper/api"
)

// TestAPIBasicFunctionality тестирует базовую функциональность API
func TestAPIBasicFunctionality(t *testing.T) {
	// Создаем API с тестовой конфигурацией
	config := &api.APIConfig{
		EnableTUI:      false,
		DatabasePath:   "test_api.db",
		LogLevel:       "debug",
		MaxMessageSize: 1024,
		HistoryLimit:   50,
	}

	owlAPI, err := api.NewOwlWhisperAPI(config)
	if err != nil {
		t.Fatalf("Не удалось создать API: %v", err)
	}

	// Запускаем API
	if err := owlAPI.Start(); err != nil {
		t.Fatalf("Не удалось запустить API: %v", err)
	}
	defer owlAPI.Stop()

	// Проверяем что API запущен
	peerID := owlAPI.GetMyPeerID()
	if peerID == "" {
		t.Error("PeerID не получен")
	}
	t.Logf("✅ API запущен с PeerID: %s", peerID)

	// Проверяем статус подключения
	status := owlAPI.GetConnectionStatus()
	if status.MyPeerID != peerID {
		t.Errorf("Неверный PeerID в статусе: %s != %s", status.MyPeerID, peerID)
	}
	t.Logf("✅ Статус подключения: %+v", status)

	// Проверяем список пиров
	peers := owlAPI.GetConnectedPeers()
	t.Logf("✅ Список пиров: %d", len(peers))

	// Проверяем историю сообщений
	history, err := owlAPI.GetHistory(10)
	if err != nil {
		t.Errorf("Ошибка получения истории: %v", err)
	} else {
		t.Logf("✅ История сообщений: %d", len(history.Messages))
	}
}

// TestAPIMessageSending тестирует отправку сообщений
func TestAPIMessageSending(t *testing.T) {
	config := &api.APIConfig{
		EnableTUI:      false,
		DatabasePath:   "test_messages.db",
		LogLevel:       "debug",
		MaxMessageSize: 1024,
		HistoryLimit:   50,
	}

	owlAPI, err := api.NewOwlWhisperAPI(config)
	if err != nil {
		t.Fatalf("Не удалось создать API: %v", err)
	}

	if err := owlAPI.Start(); err != nil {
		t.Fatalf("Не удалось запустить API: %v", err)
	}
	defer owlAPI.Stop()

	// Ждем немного для инициализации
	time.Sleep(2 * time.Second)

	// Отправляем broadcast сообщение
	request := api.SendMessageRequest{
		Text:     "Тестовое сообщение из API теста",
		ChatType: "broadcast",
	}

	if err := owlAPI.SendMessage(request); err != nil {
		t.Errorf("Ошибка отправки broadcast сообщения: %v", err)
	} else {
		t.Log("✅ Broadcast сообщение отправлено")
	}

	// Отправляем приватное сообщение (себе)
	request = api.SendMessageRequest{
		Text:        "Приватное тестовое сообщение",
		ChatType:    "private",
		RecipientID: owlAPI.GetMyPeerID(),
	}

	if err := owlAPI.SendMessage(request); err != nil {
		t.Errorf("Ошибка отправки приватного сообщения: %v", err)
	} else {
		t.Log("✅ Приватное сообщение отправлено")
	}

	// Ждем обработки сообщений
	time.Sleep(1 * time.Second)

	// Проверяем историю
	history, err := owlAPI.GetHistory(10)
	if err != nil {
		t.Errorf("Ошибка получения истории: %v", err)
	} else {
		t.Logf("✅ В истории %d сообщений", len(history.Messages))
		for i, msg := range history.Messages {
			t.Logf("  %d: %s - %s", i+1, msg.Sender, msg.Text)
		}
	}
}

// TestAPIMessageChannels тестирует каналы сообщений и пиров
func TestAPIMessageChannels(t *testing.T) {
	config := &api.APIConfig{
		EnableTUI:      false,
		DatabasePath:   "test_channels.db",
		LogLevel:       "debug",
		MaxMessageSize: 1024,
		HistoryLimit:   50,
	}

	owlAPI, err := api.NewOwlWhisperAPI(config)
	if err != nil {
		t.Fatalf("Не удалось создать API: %v", err)
	}

	if err := owlAPI.Start(); err != nil {
		t.Fatalf("Не удалось запустить API: %v", err)
	}
	defer owlAPI.Stop()

	// Ждем инициализации
	time.Sleep(2 * time.Second)

	var wg sync.WaitGroup
	wg.Add(2)

	// Тестируем канал сообщений
	go func() {
		defer wg.Done()
		messageCount := 0
		timeout := time.After(5 * time.Second)

		for {
			select {
			case msg := <-owlAPI.MessageChannel():
				messageCount++
				t.Logf("📨 Получено сообщение %d: %s", messageCount, msg.Text)
				if messageCount >= 2 {
					return
				}
			case <-timeout:
				t.Log("⏰ Таймаут ожидания сообщений")
				return
			}
		}
	}()

	// Тестируем канал пиров
	go func() {
		defer wg.Done()
		peerCount := 0
		timeout := time.After(5 * time.Second)

		for {
			select {
			case peers := <-owlAPI.PeerChannel():
				peerCount++
				t.Logf("🔌 Обновление пиров %d: %d пиров", peerCount, len(peers))
				if peerCount >= 3 {
					return
				}
			case <-timeout:
				t.Log("⏰ Таймаут ожидания обновлений пиров")
				return
			}
		}
	}()

	// Отправляем тестовые сообщения
	time.Sleep(1 * time.Second)

	requests := []api.SendMessageRequest{
		{Text: "Сообщение 1 для теста каналов", ChatType: "broadcast"},
		{Text: "Сообщение 2 для теста каналов", ChatType: "broadcast"},
	}

	for i, request := range requests {
		if err := owlAPI.SendMessage(request); err != nil {
			t.Errorf("Ошибка отправки сообщения %d: %v", i+1, err)
		} else {
			t.Logf("✅ Сообщение %d отправлено", i+1)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Ждем завершения тестов
	wg.Wait()
	t.Log("✅ Тест каналов завершен")
}

// TestAPIConfiguration тестирует конфигурацию API
func TestAPIConfiguration(t *testing.T) {
	// Тест конфигурации по умолчанию
	defaultConfig := api.DefaultAPIConfig()
	if defaultConfig.EnableTUI != true {
		t.Error("EnableTUI по умолчанию должен быть true")
	}
	if defaultConfig.DatabasePath != "owlwhisper.db" {
		t.Error("DatabasePath по умолчанию неверный")
	}
	if defaultConfig.MaxMessageSize != 4096 {
		t.Error("MaxMessageSize по умолчанию неверный")
	}
	t.Log("✅ Конфигурация по умолчанию корректна")

	// Тест пользовательской конфигурации
	customConfig := &api.APIConfig{
		EnableTUI:      false,
		DatabasePath:   "custom_test.db",
		LogLevel:       "debug",
		MaxMessageSize: 2048,
		HistoryLimit:   25,
	}

	if customConfig.EnableTUI != false {
		t.Error("Пользовательская конфигурация не применена")
	}
	if customConfig.MaxMessageSize != 2048 {
		t.Error("Пользовательский MaxMessageSize не применен")
	}
	t.Log("✅ Пользовательская конфигурация корректна")
}

// TestAPIMessageSizeLimit тестирует лимит размера сообщений
func TestAPIMessageSizeLimit(t *testing.T) {
	config := &api.APIConfig{
		EnableTUI:      false,
		DatabasePath:   "test_size_limit.db",
		LogLevel:       "debug",
		MaxMessageSize: 100, // Очень маленький лимит
		HistoryLimit:   50,
	}

	owlAPI, err := api.NewOwlWhisperAPI(config)
	if err != nil {
		t.Fatalf("Не удалось создать API: %v", err)
	}

	if err := owlAPI.Start(); err != nil {
		t.Fatalf("Не удалось запустить API: %v", err)
	}
	defer owlAPI.Stop()

	// Ждем инициализации
	time.Sleep(2 * time.Second)

	// Тест сообщения в пределах лимита
	shortRequest := api.SendMessageRequest{
		Text:     "Короткое сообщение",
		ChatType: "broadcast",
	}

	if err := owlAPI.SendMessage(shortRequest); err != nil {
		t.Errorf("Короткое сообщение должно пройти: %v", err)
	} else {
		t.Log("✅ Короткое сообщение отправлено")
	}

	// Тест сообщения превышающего лимит
	longText := ""
	for i := 0; i < 150; i++ {
		longText += "a"
	}
	longRequest := api.SendMessageRequest{
		Text:     longText,
		ChatType: "broadcast",
	}

	if err := owlAPI.SendMessage(longRequest); err == nil {
		t.Error("Длинное сообщение должно быть отклонено")
	} else {
		t.Log("✅ Длинное сообщение корректно отклонено")
	}
}
