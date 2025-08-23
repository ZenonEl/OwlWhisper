package tests

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"OwlWhisper/api"
)

// TestTwoClientsCommunication тестирует общение двух клиентов через API
func TestTwoClientsCommunication(t *testing.T) {
	// Создаем первого клиента
	client1Config := &api.APIConfig{
		EnableTUI:      false,
		DatabasePath:   "test_client1.db",
		LogLevel:       "debug",
		MaxMessageSize: 1024,
		HistoryLimit:   50,
	}

	client1, err := api.NewOwlWhisperAPI(client1Config)
	if err != nil {
		t.Fatalf("Не удалось создать клиент 1: %v", err)
	}

	// Создаем второго клиента
	client2Config := &api.APIConfig{
		EnableTUI:      false,
		DatabasePath:   "test_client2.db",
		LogLevel:       "debug",
		MaxMessageSize: 1024,
		HistoryLimit:   50,
	}

	client2, err := api.NewOwlWhisperAPI(client2Config)
	if err != nil {
		t.Fatalf("Не удалось создать клиент 2: %v", err)
	}

	// Запускаем оба клиента
	if err := client1.Start(); err != nil {
		t.Fatalf("Не удалось запустить клиент 1: %v", err)
	}
	defer client1.Stop()

	if err := client2.Start(); err != nil {
		t.Fatalf("Не удалось запустить клиент 2: %v", err)
	}
	defer client2.Stop()

	// Ждем инициализации
	time.Sleep(3 * time.Second)

	// Получаем PeerID клиентов
	client1ID := client1.GetMyPeerID()
	client2ID := client2.GetMyPeerID()

	t.Logf("🔌 Клиент 1: %s", client1ID)
	t.Logf("🔌 Клиент 2: %s", client2ID)

	// Проверяем статус подключения
	status1 := client1.GetConnectionStatus()
	status2 := client2.GetConnectionStatus()

	t.Logf("📊 Статус клиента 1: %+v", status1)
	t.Logf("📊 Статус клиента 2: %+v", status2)

	// Ждем пока клиенты найдут друг друга (через DHT)
	t.Log("⏳ Ожидание обнаружения пиров...")
	timeout := time.After(30 * time.Second)
	peersFound := false

	for !peersFound {
		select {
		case <-timeout:
			t.Log("⏰ Таймаут ожидания обнаружения пиров")
			break
		default:
			peers1 := client1.GetConnectedPeers()
			peers2 := client2.GetConnectedPeers()

			if len(peers1) > 0 || len(peers2) > 0 {
				peersFound = true
				t.Logf("✅ Пиры обнаружены! Клиент 1: %d, Клиент 2: %d", len(peers1), len(peers2))
			} else {
				time.Sleep(2 * time.Second)
			}
		}
	}

	// Тестируем обмен сообщениями
	t.Log("💬 Начинаем тест обмена сообщениями...")

	var wg sync.WaitGroup
	wg.Add(2)

	// Клиент 1 слушает сообщения
	go func() {
		defer wg.Done()
		messageCount := 0
		timeout := time.After(20 * time.Second)

		for {
			select {
			case msg := <-client1.MessageChannel():
				messageCount++
				t.Logf("📨 Клиент 1 получил сообщение %d: %s от %s", messageCount, msg.Text, msg.Sender)
				if messageCount >= 3 {
					return
				}
			case <-timeout:
				t.Log("⏰ Таймаут ожидания сообщений для клиента 1")
				return
			}
		}
	}()

	// Клиент 2 слушает сообщения
	go func() {
		defer wg.Done()
		messageCount := 0
		timeout := time.After(20 * time.Second)

		for {
			select {
			case msg := <-client2.MessageChannel():
				messageCount++
				t.Logf("📨 Клиент 2 получил сообщение %d: %s от %s", messageCount, msg.Text, msg.Sender)
				if messageCount >= 3 {
					return
				}
			case <-timeout:
				t.Log("⏰ Таймаут ожидания сообщений для клиента 2")
				return
			}
		}
	}()

	// Ждем немного для запуска горутин
	time.Sleep(1 * time.Second)

	// Клиент 1 отправляет broadcast сообщение
	t.Log("📤 Клиент 1 отправляет broadcast сообщение...")
	broadcastRequest := api.SendMessageRequest{
		Text:     "Привет всем от клиента 1!",
		ChatType: "broadcast",
	}

	if err := client1.SendMessage(broadcastRequest); err != nil {
		t.Errorf("Ошибка отправки broadcast от клиента 1: %v", err)
	} else {
		t.Log("✅ Broadcast сообщение от клиента 1 отправлено")
	}

	time.Sleep(2 * time.Second)

	// Клиент 2 отправляет broadcast сообщение
	t.Log("📤 Клиент 2 отправляет broadcast сообщение...")
	broadcastRequest = api.SendMessageRequest{
		Text:     "Привет всем от клиента 2!",
		ChatType: "broadcast",
	}

	if err := client2.SendMessage(broadcastRequest); err != nil {
		t.Errorf("Ошибка отправки broadcast от клиента 2: %v", err)
	} else {
		t.Log("✅ Broadcast сообщение от клиента 2 отправлено")
	}

	time.Sleep(2 * time.Second)

	// Клиент 1 отправляет приватное сообщение клиенту 2
	t.Log("📤 Клиент 1 отправляет приватное сообщение клиенту 2...")
	privateRequest := api.SendMessageRequest{
		Text:        "Приватное сообщение от клиента 1",
		ChatType:    "private",
		RecipientID: client2ID,
	}

	if err := client1.SendMessage(privateRequest); err != nil {
		t.Errorf("Ошибка отправки приватного сообщения от клиента 1: %v", err)
	} else {
		t.Log("✅ Приватное сообщение от клиента 1 отправлено")
	}

	// Ждем завершения тестов
	wg.Wait()
	t.Log("✅ Тест общения двух клиентов завершен")

	// Проверяем историю сообщений
	t.Log("📚 Проверяем историю сообщений...")

	history1, err := client1.GetHistory(10)
	if err != nil {
		t.Errorf("Ошибка получения истории клиента 1: %v", err)
	} else {
		t.Logf("📚 История клиента 1: %d сообщений", len(history1.Messages))
		for i, msg := range history1.Messages {
			t.Logf("  %d: %s - %s", i+1, msg.Sender, msg.Text)
		}
	}

	history2, err := client2.GetHistory(10)
	if err != nil {
		t.Errorf("Ошибка получения истории клиента 2: %v", err)
	} else {
		t.Logf("📚 История клиента 2: %d сообщений", len(history2.Messages))
		for i, msg := range history2.Messages {
			t.Logf("  %d: %s - %s", i+1, msg.Sender, msg.Text)
		}
	}
}

// TestMultipleClients тестирует работу с несколькими клиентами
func TestMultipleClients(t *testing.T) {
	// Создаем несколько клиентов
	clients := make([]api.OwlWhisperAPI, 3)
	configs := []*api.APIConfig{
		{
			EnableTUI:      false,
			DatabasePath:   "test_multi1.db",
			LogLevel:       "debug",
			MaxMessageSize: 1024,
			HistoryLimit:   50,
		},
		{
			EnableTUI:      false,
			DatabasePath:   "test_multi2.db",
			LogLevel:       "debug",
			MaxMessageSize: 1024,
			HistoryLimit:   50,
		},
		{
			EnableTUI:      false,
			DatabasePath:   "test_multi3.db",
			LogLevel:       "debug",
			MaxMessageSize: 1024,
			HistoryLimit:   50,
		},
	}

	// Создаем и запускаем клиентов
	for i := 0; i < 3; i++ {
		client, err := api.NewOwlWhisperAPI(configs[i])
		if err != nil {
			t.Fatalf("Не удалось создать клиент %d: %v", i+1, err)
		}

		if err := client.Start(); err != nil {
			t.Fatalf("Не удалось запустить клиент %d: %v", i+1, err)
		}
		defer client.Stop()

		clients[i] = client
		t.Logf("✅ Клиент %d запущен: %s", i+1, client.GetMyPeerID())
	}

	// Ждем инициализации
	time.Sleep(5 * time.Second)

	// Проверяем что все клиенты видят друг друга
	t.Log("🔍 Проверяем видимость клиентов...")
	for i, client := range clients {
		peers := client.GetConnectedPeers()
		t.Logf("📊 Клиент %d видит %d пиров", i+1, len(peers))
	}

	// Отправляем сообщения от каждого клиента
	t.Log("💬 Отправляем сообщения от всех клиентов...")
	for i, client := range clients {
		request := api.SendMessageRequest{
			Text:     fmt.Sprintf("Сообщение от клиента %d", i+1),
			ChatType: "broadcast",
		}

		if err := client.SendMessage(request); err != nil {
			t.Errorf("Ошибка отправки от клиента %d: %v", i+1, err)
		} else {
			t.Logf("✅ Сообщение от клиента %d отправлено", i+1)
		}

		time.Sleep(1 * time.Second)
	}

	// Ждем обработки сообщений
	time.Sleep(3 * time.Second)

	// Проверяем историю у каждого клиента
	t.Log("📚 Проверяем историю у всех клиентов...")
	for i, client := range clients {
		history, err := client.GetHistory(10)
		if err != nil {
			t.Errorf("Ошибка получения истории клиента %d: %v", i+1, err)
		} else {
			t.Logf("📚 Клиент %d: %d сообщений в истории", i+1, len(history.Messages))
		}
	}

	t.Log("✅ Тест множественных клиентов завершен")
}

// TestClientReconnection тестирует переподключение клиента
func TestClientReconnection(t *testing.T) {
	// Создаем клиент
	config := &api.APIConfig{
		EnableTUI:      false,
		DatabasePath:   "test_reconnect.db",
		LogLevel:       "debug",
		MaxMessageSize: 1024,
		HistoryLimit:   50,
	}

	client, err := api.NewOwlWhisperAPI(config)
	if err != nil {
		t.Fatalf("Не удалось создать клиент: %v", err)
	}

	// Запускаем
	if err := client.Start(); err != nil {
		t.Fatalf("Не удалось запустить клиент: %v", err)
	}

	// Ждем инициализации
	time.Sleep(3 * time.Second)

	// Отправляем сообщение
	request := api.SendMessageRequest{
		Text:     "Сообщение до переподключения",
		ChatType: "broadcast",
	}

	if err := client.SendMessage(request); err != nil {
		t.Errorf("Ошибка отправки до переподключения: %v", err)
	}

	// Останавливаем
	t.Log("🛑 Останавливаем клиент...")
	client.Stop()

	// Ждем
	time.Sleep(2 * time.Second)

	// Запускаем снова
	t.Log("🚀 Перезапускаем клиент...")
	if err := client.Start(); err != nil {
		t.Fatalf("Не удалось перезапустить клиент: %v", err)
	}
	defer client.Stop()

	// Ждем инициализации
	time.Sleep(3 * time.Second)

	// Отправляем сообщение после переподключения
	request = api.SendMessageRequest{
		Text:     "Сообщение после переподключения",
		ChatType: "broadcast",
	}

	if err := client.SendMessage(request); err != nil {
		t.Errorf("Ошибка отправки после переподключения: %v", err)
	} else {
		t.Log("✅ Сообщение после переподключения отправлено")
	}

	// Проверяем историю
	history, err := client.GetHistory(10)
	if err != nil {
		t.Errorf("Ошибка получения истории после переподключения: %v", err)
	} else {
		t.Logf("📚 История после переподключения: %d сообщений", len(history.Messages))
	}

	t.Log("✅ Тест переподключения завершен")
}
