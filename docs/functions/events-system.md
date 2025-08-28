# 📡 **Система событий**

**Последнее обновление:** 23 августа 2025

Система асинхронных событий - единственный канал связи от Core к клиенту.

## 🎯 **Обзор**

**Система событий** - это механизм, позволяющий Core асинхронно уведомлять клиента о важных сетевых изменениях. Это **единственный канал асинхронной связи** от Core к клиенту для сетевых событий.

> **Важно:** `GetNextEvent()` - это ЕДИНСТВЕННЫЙ способ получения ВСЕХ событий от Core, включая входящие сообщения, подключение/отключение пиров и статус сети.

### **Ключевые особенности:**
- **Блокирующий вызов** - клиент ждет события, не тратя ресурсы на polling
- **JSON формат** - универсальная сериализация для любого клиента
- **Автоматическая генерация** - Core автоматически создает события для всех важных изменений
- **Потокобезопасность** - можно вызывать из нескольких потоков

## 📋 **Функции**

### **`GetNextEvent()`**

**Описание:** Блокирующая функция для получения следующего события из очереди.

**Сигнатура:**
```c
char* GetNextEvent();
```

**Параметры:** Нет

**Возвращает:**
- `char*` - JSON строка с событием (требует `FreeString()`)
- `NULL` - нет событий или ошибка

**Пример использования:**
```python
import ctypes
import json
import threading

# Настройка типа возвращаемого значения
owlwhisper.GetNextEvent.restype = ctypes.c_char_p

def event_listener():
    """Слушатель событий в отдельном потоке"""
    while True:
        try:
            event_ptr = owlwhisper.GetNextEvent()
            if event_ptr:
                event_json = ctypes.string_at(event_ptr).decode()
                owlwhisper.FreeString(event_ptr)
                
                event = json.loads(event_json)
                handle_event(event)
            else:
                # Нет событий, небольшая пауза
                time.sleep(0.1)
                
        except Exception as e:
            print(f"❌ Ошибка в цикле событий: {e}")
            time.sleep(1)

def handle_event(event):
    """Обработка события по типу"""
    event_type = event['type']
    
    if event_type == 'PeerConnected':
        peer_id = event['payload']['peerID']
        print(f"🔗 Подключился пир: {peer_id}")
        
    elif event_type == 'PeerDisconnected':
        peer_id = event['payload']['peerID']
        print(f"🔌 Отключился пир: {peer_id}")
        
    elif event_type == 'NetworkStatus':
        status = event['payload']['status']
        message = event['payload']['message']
        print(f"🌐 Статус сети: {status} - {message}")

# Запускаем слушатель в отдельном потоке
thread = threading.Thread(target=event_listener, daemon=True)
thread.start()

print("🚀 Слушатель событий запущен")
```

**Что происходит при вызове:**
1. Функция блокирует выполнение до появления события
2. При появлении события возвращает JSON строку
3. Если нет событий, возвращает `NULL`
4. Таймаут 30 секунд для предотвращения бесконечного ожидания

**Важные замечания:**
- ⚠️ **ВСЕГДА вызывать `FreeString()`** после использования
- ⚠️ **Блокирующий вызов** - не вызывать в основном потоке UI
- 💡 **Используйте в отдельном потоке** для асинхронной обработки
- 🔄 **Автоматическое управление очередью** - Core сам добавляет события

## 📨 **Типы событий**

> **Примечание:** `GetNextEvent()` возвращает ВСЕ типы событий, включая входящие сообщения, подключение/отключение пиров и статус сети.

### **1. NewMessage**

**Описание:** Новое входящее сообщение от пира.

**Структура:**
```json
{
    "type": "NewMessage",
    "payload": {
        "senderID": "12D3KooW...",
        "data": [1, 2, 3, 4, 5]
    },
    "timestamp": 1755969188
}
```

**Поля:**
- `senderID` - Peer ID отправителя
- `data` - массив байт сообщения ([]byte данные)
- `timestamp` - Unix timestamp получения

**Когда генерируется:**
- При получении данных из входящего потока
- Обрабатывается в `handleStream` функции Node

**Пример обработки:**
```python
def handle_new_message(event):
    """Обработка нового сообщения"""
    sender_id = event['payload']['senderID']
    data = event['payload']['data']
    timestamp = event['timestamp']
    
    print(f"📨 Новое сообщение от {sender_id}")
    print(f"   Данные: {data}")
    print(f"   Время: {timestamp}")
    
    # Здесь можно:
    # - Сохранить сообщение в базу данных
    # - Обновить UI чата
    # - Уведомить пользователя
    # - Обработать данные (текст, файлы, JSON и т.д.)
```

---

### **2. PeerConnected**

**Описание:** Подключение нового пира к сети.

**Структура:**
```json
{
    "type": "PeerConnected",
    "payload": {
        "peerID": "12D3KooW..."
    },
    "timestamp": 1755969189
}
```

**Поля:**
- `peerID` - Peer ID подключившегося пира
- `timestamp` - Unix timestamp подключения

**Когда генерируется:**
- При успешной установке соединения с пиром
- Обрабатывается в `NetworkEventLogger.Connected`

**Пример обработки:**
```python
def handle_peer_connected(event):
    """Обработка подключения пира"""
    peer_id = event['payload']['peerID']
    timestamp = event['timestamp']
    
    print(f"🔗 Подключился пир: {peer_id}")
    
    # Здесь можно:
    # - Обновить список пиров в UI
    # - Показать уведомление о новом участнике
    # - Обновить статус "онлайн" для контакта
    # - Запустить обмен информацией с новым пиром
```

---

### **3. PeerDisconnected**

**Описание:** Отключение пира от сети.

**Структура:**
```json
{
    "type": "PeerDisconnected",
    "payload": {
        "peerID": "12D3KooW..."
    },
    "timestamp": 1755969190
}
```

**Поля:**
- `peerID` - Peer ID отключившегося пира
- `timestamp` - Unix timestamp отключения

**Когда генерируется:**
- При разрыве соединения с пиром
- Обрабатывается в `NetworkEventLogger.Disconnected`

**Пример обработки:**
```python
def handle_peer_disconnected(event):
    """Обработка отключения пира"""
    peer_id = event['payload']['peerID']
    timestamp = event['timestamp']
    
    print(f"🔌 Отключился пир: {peer_id}")
    
    # Здесь можно:
    # - Обновить статус "офлайн" для контакта
    # - Показать уведомление об отключении
    # - Обновить список пиров в UI
    # - Запустить логику переподключения
```

---

### **4. NetworkStatus**

**Описание:** Изменение статуса сети.

**Структура:**
```json
{
    "type": "NetworkStatus",
    "payload": {
        "status": "CONNECTING_TO_DHT",
        "message": "Подключение к bootstrap-узлам..."
    },
    "timestamp": 1755969188
}
```

**Поля:**
- `status` - код статуса сети
- `message` - человекочитаемое описание статуса
- `timestamp` - Unix timestamp изменения статуса

**Возможные статусы:**
- `CONNECTING_TO_DHT` - подключение к bootstrap узлам
- `NETWORK_READY` - сеть готова к работе
- `DHT_BOOTSTRAP_FAILED` - не удалось подключиться к bootstrap
- `FALLBACK_TO_CACHE` - использование кэшированных пиров

**Когда генерируется:**
- При изменении состояния DHT discovery
- При подключении/отключении к bootstrap узлам
- При изменении доступности сети

**Пример обработки:**
```python
def handle_network_status(event):
    """Обработка изменения статуса сети"""
    status = event['payload']['status']
    message = event['payload']['message']
    timestamp = event['timestamp']
    
    print(f"🌐 Статус сети: {status}")
    print(f"   Сообщение: {message}")
    
    # Здесь можно:
    # - Обновить индикатор статуса сети в UI
    # - Показать прогресс подключения
    # - Уведомить пользователя о проблемах
    # - Запустить диагностику при ошибках
```

## 🔄 **Сценарии использования**

### **Мониторинг состояния сети:**

```python
def monitor_network_status():
    """Мониторинг состояния сети через события"""
    def event_handler(event):
        if event['type'] == 'NetworkStatus':
            status = event['payload']['status']
            message = event['payload']['message']
            
            if status == 'NETWORK_READY':
                print("✅ Сеть готова к работе")
                # Показать зеленый индикатор в UI
            elif status == 'CONNECTING_TO_DHT':
                print("🔄 Подключение к сети...")
                # Показать желтый индикатор в UI
            elif status == 'DHT_BOOTSTRAP_FAILED':
                print("❌ Проблемы с сетью")
                # Показать красный индикатор в UI
    
    return event_handler
```

### **Автоматическое обновление списка пиров:**

```python
def auto_update_peer_list():
    """Автоматическое обновление списка пиров"""
    def event_handler(event):
        if event['type'] == 'PeerConnected':
            peer_id = event['payload']['peerID']
            # Добавить пира в список онлайн
            add_peer_to_online_list(peer_id)
            
        elif event['type'] == 'PeerDisconnected':
            peer_id = event['payload']['peerID']
            # Убрать пира из списка онлайн
            remove_peer_from_online_list(peer_id)
    
    return event_handler
```

### **Обработка входящих сообщений:**

```python
def handle_incoming_messages():
    """Обработка входящих сообщений"""
    def event_handler(event):
        # Обработка входящих сообщений через события
        # См. NewMessage событие выше
        pass
    
    return event_handler
```

### **Комбинированный обработчик:**

```python
def create_event_processor():
    """Создание комбинированного обработчика событий"""
    def process_event(event):
        event_type = event['type']
        
        # Маршрутизация событий по типам
        if event_type in ['PeerConnected', 'PeerDisconnected']:
            auto_update_peer_list()(event)
        elif event_type == 'NetworkStatus':
            monitor_network_status()(event)
        else:
            print(f"⚠️ Неизвестный тип события: {event_type}")
    
    return process_event

# Использование
event_processor = create_event_processor()

def event_listener():
    while True:
        event_ptr = owlwhisper.GetNextEvent()
        if event_ptr:
            event_json = ctypes.string_at(event_ptr).decode()
            owlwhisper.FreeString(event_ptr)
            
            event = json.loads(event_json)
            event_processor(event)
```

## ⚠️ **Важные замечания**

### **Архитектурные принципы:**
- **Единственный канал связи** - `GetNextEvent()` как единственный способ получить события от Core
- **Автоматическая генерация** - Core автоматически создает события для всех важных изменений
- **Блокирующий API** - клиент ждет события, не тратя ресурсы на polling
- **JSON формат** - универсальная сериализация для любого клиента

### **Производительность:**
- **События генерируются в реальном времени** - минимальная задержка
- **Очередь событий** - буферизация для предотвращения потери
- **Автоматическая очистка** - старые события не накапливаются
- **Потокобезопасность** - можно вызывать из нескольких потоков

### **Надежность:**
- **Таймаут 30 секунд** - предотвращение бесконечного ожидания
- **Обработка ошибок** - graceful degradation при проблемах
- **Автоматическое восстановление** - система продолжает работать после ошибок
- **Логирование** - все события логируются для отладки

### **Ограничения:**
- **Только входящие события** - нет возможности отправлять события в Core
- **Только JSON формат** - нет бинарных событий
- **Только сетевые события** - нет системных или пользовательских событий
- **Только один слушатель** - несколько потоков могут конкурировать за события

## 🔗 **Связанные функции**

- [Управление Core](../functions/core-management.md) - запуск/остановка Core для получения событий
- [Универсальный обмен данными](../functions/messaging.md) - отправка данных и получение через события
- [Поиск пиров](../functions/peer-discovery.md) - поиск пиров, которые могут генерировать события
- [Утилиты](../functions/utilities.md) - управление памятью для строк событий

---

**Последнее обновление:** 23 августа 2025  
**Автор:** Core Development Team 