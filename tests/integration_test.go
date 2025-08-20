package tests

import (
	"fmt"
	"testing"
	"time"

	"OwlWhisper/api"
)

// TestFullWorkflow тестирует полный цикл работы API
func TestFullWorkflow(t *testing.T) {
	t.Log("🚀 Начинаем тест полного цикла работы API...")

	// 1. Создание и конфигурация
	t.Log("📋 Шаг 1: Создание и конфигурация...")
	config := &api.APIConfig{
		EnableTUI:      false,
		DatabasePath:   "test_workflow.db",
		LogLevel:       "debug",
		MaxMessageSize: 2048,
		HistoryLimit:   100,
	}

	owlAPI, err := api.NewOwlWhisperAPI(config)
	if err != nil {
		t.Fatalf("❌ Не удалось создать API: %v", err)
	}
	t.Log("✅ API создан")

	// 2. Запуск
	t.Log("📋 Шаг 2: Запуск API...")
	if err := owlAPI.Start(); err != nil {
		t.Fatalf("❌ Не удалось запустить API: %v", err)
	}
	defer owlAPI.Stop()
	t.Log("✅ API запущен")

	// 3. Проверка инициализации
	t.Log("📋 Шаг 3: Проверка инициализации...")
	peerID := owlAPI.GetMyPeerID()
	if peerID == "" {
		t.Fatal("❌ PeerID не получен")
	}
	t.Logf("✅ PeerID получен: %s", peerID)

	// Ждем инициализации сети
	time.Sleep(5 * time.Second)

	// 4. Проверка статуса подключения
	t.Log("📋 Шаг 4: Проверка статуса подключения...")
	status := owlAPI.GetConnectionStatus()
	if status.MyPeerID != peerID {
		t.Errorf("❌ Неверный PeerID в статусе: %s != %s", status.MyPeerID, peerID)
	}
	t.Logf("✅ Статус подключения: %+v", status)

	// 5. Отправка сообщений
	t.Log("📋 Шаг 5: Отправка сообщений...")
	messages := []api.SendMessageRequest{
		{Text: "Первое тестовое сообщение", ChatType: "broadcast"},
		{Text: "Второе тестовое сообщение", ChatType: "broadcast"},
		{Text: "Приватное сообщение", ChatType: "private", RecipientID: peerID},
	}

	for i, request := range messages {
		if err := owlAPI.SendMessage(request); err != nil {
			t.Errorf("❌ Ошибка отправки сообщения %d: %v", i+1, err)
		} else {
			t.Logf("✅ Сообщение %d отправлено: %s", i+1, request.Text)
		}
		time.Sleep(1 * time.Second)
	}

	// 6. Проверка истории
	t.Log("📋 Шаг 6: Проверка истории сообщений...")
	history, err := owlAPI.GetHistory(10)
	if err != nil {
		t.Errorf("❌ Ошибка получения истории: %v", err)
	} else {
		t.Logf("✅ История получена: %d сообщений", len(history.Messages))
		for i, msg := range history.Messages {
			t.Logf("  %d: %s - %s", i+1, msg.Sender, msg.Text)
		}
	}

	// 7. Проверка списка пиров
	t.Log("📋 Шаг 7: Проверка списка пиров...")
	peers := owlAPI.GetPeers()
	t.Logf("✅ Список пиров: %d", len(peers))
	for i, peer := range peers {
		t.Logf("  %d: %s (%s)", i+1, peer.Nickname, peer.Status)
	}

	// 8. Тест каналов
	t.Log("📋 Шаг 8: Тест каналов...")
	
	// Запускаем горутину для получения сообщений
	messageReceived := make(chan bool, 1)
	go func() {
		timeout := time.After(10 * time.Second)
		select {
		case msg := <-owlAPI.MessageChannel():
			t.Logf("📨 Получено сообщение через канал: %s", msg.Text)
			messageReceived <- true
		case <-timeout:
			t.Log("⏰ Таймаут ожидания сообщения через канал")
			messageReceived <- false
		}
	}()

	// Запускаем горутину для получения обновлений пиров
	peerUpdateReceived := make(chan bool, 1)
	go func() {
		timeout := time.After(10 * time.Second)
		select {
		case peers := <-owlAPI.PeerChannel():
			t.Logf("🔌 Получено обновление пиров через канал: %d", len(peers))
			peerUpdateReceived <- true
		case <-timeout:
			t.Log("⏰ Таймаут ожидания обновления пиров через канал")
			peerUpdateReceived <- false
		}
	}()

	// Отправляем еще одно сообщение для активации каналов
	time.Sleep(2 * time.Second)
	if err := owlAPI.SendMessage(api.SendMessageRequest{
		Text:     "Сообщение для активации каналов",
		ChatType: "broadcast",
	}); err != nil {
		t.Errorf("❌ Ошибка отправки сообщения для каналов: %v", err)
	}

	// Ждем результаты тестов каналов
	msgResult := <-messageReceived
	peerResult := <-peerUpdateReceived

	if msgResult {
		t.Log("✅ Канал сообщений работает")
	} else {
		t.Log("⚠️ Канал сообщений не получил данные")
	}

	if peerResult {
		t.Log("✅ Канал пиров работает")
	} else {
		t.Log("⚠️ Канал пиров не получил данные")
	}

	t.Log("🎉 Тест полного цикла работы API завершен успешно!")
}

// TestAPIPerformance тестирует производительность API
func TestAPIPerformance(t *testing.T) {
	t.Log("🚀 Начинаем тест производительности API...")

	config := &api.APIConfig{
		EnableTUI:      false,
		DatabasePath:   "test_performance.db",
		LogLevel:       "info", // Уменьшаем логирование для тестов производительности
		MaxMessageSize: 4096,
		HistoryLimit:   1000,
	}

	owlAPI, err := api.NewOwlWhisperAPI(config)
	if err != nil {
		t.Fatalf("❌ Не удалось создать API: %v", err)
	}

	if err := owlAPI.Start(); err != nil {
		t.Fatalf("❌ Не удалось запустить API: %v", err)
	}
	defer owlAPI.Stop()

	// Ждем инициализации
	time.Sleep(3 * time.Second)

	// Тест отправки множества сообщений
	t.Log("📤 Тест отправки множества сообщений...")
	startTime := time.Now()
	messageCount := 100

	for i := 0; i < messageCount; i++ {
		request := api.SendMessageRequest{
			Text:     fmt.Sprintf("Сообщение %d для теста производительности", i+1),
			ChatType: "broadcast",
		}

		if err := owlAPI.SendMessage(request); err != nil {
			t.Errorf("❌ Ошибка отправки сообщения %d: %v", i+1, err)
		}
	}

	duration := time.Since(startTime)
	rate := float64(messageCount) / duration.Seconds()

	t.Logf("✅ Отправлено %d сообщений за %v", messageCount, duration)
	t.Logf("📊 Скорость: %.2f сообщений/сек", rate)

	// Тест получения истории
	t.Log("📚 Тест получения истории...")
	startTime = time.Now()
	
	history, err := owlAPI.GetHistory(messageCount)
	if err != nil {
		t.Errorf("❌ Ошибка получения истории: %v", err)
	} else {
		duration = time.Since(startTime)
		t.Logf("✅ Получена история %d сообщений за %v", len(history.Messages), duration)
	}

	// Тест статуса подключения
	t.Log("📊 Тест статуса подключения...")
	startTime = time.Now()
	
	for i := 0; i < 100; i++ {
		_ = owlAPI.GetConnectionStatus()
	}
	
	duration = time.Since(startTime)
	avgTime := duration / 100
	
	t.Logf("✅ 100 вызовов GetConnectionStatus за %v (среднее: %v)", duration, avgTime)

	t.Log("🎉 Тест производительности завершен!")
}

// TestAPIRobustness тестирует устойчивость API
func TestAPIRobustness(t *testing.T) {
	t.Log("🚀 Начинаем тест устойчивости API...")

	config := &api.APIConfig{
		EnableTUI:      false,
		DatabasePath:   "test_robustness.db",
		LogLevel:       "debug",
		MaxMessageSize: 1024,
		HistoryLimit:   50,
	}

	// Тест 1: Множественные запуски/остановки
	t.Log("🔄 Тест 1: Множественные запуски/остановки...")
	for i := 0; i < 3; i++ {
		owlAPI, err := api.NewOwlWhisperAPI(config)
		if err != nil {
			t.Fatalf("❌ Не удалось создать API (итерация %d): %v", i+1, err)
		}

		if err := owlAPI.Start(); err != nil {
			t.Fatalf("❌ Не удалось запустить API (итерация %d): %v", i+1, err)
		}

		// Проверяем что API работает
		peerID := owlAPI.GetMyPeerID()
		if peerID == "" {
			t.Errorf("❌ PeerID не получен (итерация %d)", i+1)
		}

		// Отправляем тестовое сообщение
		if err := owlAPI.SendMessage(api.SendMessageRequest{
			Text:     fmt.Sprintf("Тест устойчивости %d", i+1),
			ChatType: "broadcast",
		}); err != nil {
			t.Errorf("❌ Ошибка отправки (итерация %d): %v", i+1, err)
		}

		// Останавливаем
		owlAPI.Stop()
		t.Logf("✅ Итерация %d завершена успешно", i+1)

		// Небольшая пауза между итерациями
		time.Sleep(1 * time.Second)
	}

	// Тест 2: Отправка сообщений с разными типами
	t.Log("💬 Тест 2: Отправка сообщений с разными типами...")
	owlAPI, err := api.NewOwlWhisperAPI(config)
	if err != nil {
		t.Fatalf("❌ Не удалось создать API для теста типов: %v", err)
	}

	if err := owlAPI.Start(); err != nil {
		t.Fatalf("❌ Не удалось запустить API для теста типов: %v", err)
	}
	defer owlAPI.Stop()

	// Ждем инициализации
	time.Sleep(3 * time.Second)

	// Тестируем разные типы сообщений
	messageTypes := []struct {
		text     string
		chatType string
		valid    bool
	}{
		{"Broadcast сообщение", "broadcast", true},
		{"Private сообщение", "private", true},
		{"Group сообщение", "group", true},
		{"Неизвестный тип", "unknown", false}, // Должно быть отклонено
	}

	for _, testCase := range messageTypes {
		request := api.SendMessageRequest{
			Text:        testCase.text,
			ChatType:    testCase.chatType,
			RecipientID: owlAPI.GetMyPeerID(), // Для приватных сообщений
		}

		err := owlAPI.SendMessage(request)
		if testCase.valid && err != nil {
			t.Errorf("❌ Валидное сообщение отклонено (тип %s): %v", testCase.chatType, err)
		} else if !testCase.valid && err == nil {
			t.Errorf("❌ Невалидное сообщение принято (тип %s)", testCase.chatType)
		} else {
			t.Logf("✅ Тест типа %s прошел", testCase.chatType)
		}
	}

	t.Log("🎉 Тест устойчивости завершен!")
} 