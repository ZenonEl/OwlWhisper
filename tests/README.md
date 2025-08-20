# Тесты OwlWhisper API

Этот пакет содержит комплексные тесты для проверки функциональности OwlWhisper API.

## Структура тестов

```
tests/
├── api_test.go          # Базовые тесты API функциональности
├── network_test.go      # Сетевые тесты (общение клиентов)
├── integration_test.go  # Интеграционные тесты
└── README.md            # Эта документация
```

## Запуск тестов

### Запуск всех тестов
```bash
go test ./tests/ -v
```

### Запуск конкретного теста
```bash
# Только базовые тесты API
go test ./tests/ -v -run TestAPIBasicFunctionality

# Только сетевые тесты
go test ./tests/ -v -run TestTwoClientsCommunication

# Только интеграционные тесты
go test ./tests/ -v -run TestFullWorkflow
```

### Запуск с подробным выводом
```bash
go test ./tests/ -v -timeout 5m
```

## Описание тестов

### 1. Базовые тесты API (`api_test.go`)

#### `TestAPIBasicFunctionality`
- ✅ Создание и запуск API
- ✅ Получение PeerID
- ✅ Проверка статуса подключения
- ✅ Получение списка пиров
- ✅ Получение истории сообщений

#### `TestAPIMessageSending`
- ✅ Отправка broadcast сообщений
- ✅ Отправка приватных сообщений
- ✅ Проверка сохранения в истории

#### `TestAPIMessageChannels`
- ✅ Тестирование канала сообщений
- ✅ Тестирование канала пиров
- ✅ Асинхронная обработка событий

#### `TestAPIConfiguration`
- ✅ Проверка конфигурации по умолчанию
- ✅ Проверка пользовательской конфигурации

#### `TestAPIMessageSizeLimit`
- ✅ Проверка лимита размера сообщений
- ✅ Отклонение слишком длинных сообщений

### 2. Сетевые тесты (`network_test.go`)

#### `TestTwoClientsCommunication`
- 🔌 Создание двух клиентов API
- 🌐 Ожидание обнаружения через DHT
- 💬 Обмен broadcast сообщениями
- 🔒 Отправка приватных сообщений
- 📚 Проверка истории у обоих клиентов

#### `TestMultipleClients`
- 🔌 Создание трех клиентов
- 🌐 Проверка взаимной видимости
- 💬 Отправка сообщений от всех клиентов
- 📚 Проверка истории у всех клиентов

#### `TestClientReconnection`
- 🔄 Тест остановки и перезапуска клиента
- 💾 Сохранение сообщений между сессиями
- 📚 Восстановление истории после переподключения

### 3. Интеграционные тесты (`integration_test.go`)

#### `TestFullWorkflow`
- 📋 Полный цикл работы API
- 🚀 Создание → Запуск → Использование → Остановка
- 💬 Отправка разных типов сообщений
- 📚 Проверка всех функций
- 🔌 Тестирование каналов

#### `TestAPIPerformance`
- 📊 Тест производительности отправки сообщений
- ⚡ Измерение скорости работы
- 📚 Тест получения истории
- 🔍 Тест статуса подключения

#### `TestAPIRobustness`
- 🔄 Множественные запуски/остановки
- 💬 Тест разных типов сообщений
- 🛡️ Проверка устойчивости к ошибкам

## Особенности тестирования

### Временные ожидания
- **Инициализация сети**: 3-5 секунд
- **Обнаружение пиров**: до 30 секунд
- **Обработка сообщений**: 1-2 секунды

### Базы данных
Каждый тест использует **отдельную базу данных**:
- `test_api.db`
- `test_messages.db`
- `test_channels.db`
- `test_client1.db`
- `test_client2.db`
- `test_workflow.db`
- `test_performance.db`
- `test_robustness.db`

### Сетевая изоляция
- Тесты **не влияют** на основное приложение
- Каждый клиент работает **независимо**
- **DHT discovery** работает в реальной сети

## Примеры использования

### Простой тест
```go
func TestSimple(t *testing.T) {
    // Создаем API
    config := &api.APIConfig{
        EnableTUI:      false,
        DatabasePath:   "test_simple.db",
        LogLevel:       "debug",
        MaxMessageSize: 1024,
        HistoryLimit:   50,
    }

    owlAPI, err := api.NewOwlWhisperAPI(config)
    if err != nil {
        t.Fatalf("Ошибка создания: %v", err)
    }

    // Запускаем
    if err := owlAPI.Start(); err != nil {
        t.Fatalf("Ошибка запуска: %v", err)
    }
    defer owlAPI.Stop()

    // Тестируем функциональность
    peerID := owlAPI.GetMyPeerID()
    if peerID == "" {
        t.Error("PeerID не получен")
    }

    // Отправляем сообщение
    err = owlAPI.SendMessage(api.SendMessageRequest{
        Text:     "Тестовое сообщение",
        ChatType: "broadcast",
    })
    if err != nil {
        t.Errorf("Ошибка отправки: %v", err)
    }
}
```

### Тест с ожиданием событий
```go
func TestWithChannels(t *testing.T) {
    // ... создание и запуск API ...

    // Ждем сообщение
    select {
    case msg := <-owlAPI.MessageChannel():
        t.Logf("Получено: %s", msg.Text)
    case <-time.After(10 * time.Second):
        t.Error("Таймаут ожидания сообщения")
    }
}
```

## Отладка тестов

### Включение подробных логов
```bash
go test ./tests/ -v -timeout 5m -logtostderr
```

### Запуск одного теста с повторами
```bash
go test ./tests/ -v -run TestTwoClientsCommunication -count=3
```

### Параллельное выполнение
```bash
go test ./tests/ -v -parallel 4
```

## Требования

- **Go 1.21+** для тестов
- **SQLite3** для баз данных
- **Сетевое подключение** для DHT discovery
- **Время выполнения**: 2-5 минут для полного набора

## Примечания

1. **Сетевые тесты** могут занять время из-за DHT discovery
2. **Базы данных** создаются автоматически и удаляются после тестов
3. **Пиры** могут не обнаружиться в изолированной сети
4. **Таймауты** настроены для стабильной работы

## Устранение проблем

### Тест зависает
```bash
# Увеличиваем таймаут
go test ./tests/ -v -timeout 10m
```

### Ошибки сети
```bash
# Проверяем подключение к интернету
ping 8.8.8.8

# Проверяем DHT порты
netstat -tuln | grep 4001
```

### Ошибки базы данных
```bash
# Удаляем тестовые базы
rm -f test_*.db

# Проверяем права доступа
ls -la test_*.db
``` 