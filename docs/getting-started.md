# 🚀 **Быстрый старт с Owl Whisper Core**

**Последнее обновление:** 28 августа 2025

## 📦 **Установка**

### **Загрузка библиотеки**

```python
import ctypes
import os

# Путь к библиотеке
lib_path = "./dist/libowlwhisper.so"  # Linux
# lib_path = "./dist/owlwhisper.dll"   # Windows  
# lib_path = "./dist/libowlwhisper.dylib"  # macOS

# Загружаем библиотеку
owlwhisper = ctypes.CDLL(lib_path)
```

### **Настройка типов для функций**

```python
# Основные функции
owlwhisper.StartOwlWhisper.restype = ctypes.c_int
owlwhisper.StartOwlWhisperWithKey.argtypes = [ctypes.c_char_p, ctypes.c_int]
owlwhisper.StartOwlWhisperWithKey.restype = ctypes.c_int
owlwhisper.StopOwlWhisper.restype = ctypes.c_int

# Функции, возвращающие строки
owlwhisper.GetMyPeerID.restype = ctypes.c_char_p
owlwhisper.GetConnectedPeers.restype = ctypes.c_char_p
owlwhisper.GetNetworkStats.restype = ctypes.c_char_p
owlwhisper.FindPeer.restype = ctypes.c_char_p
owlwhisper.FindProvidersForContent.restype = ctypes.c_char_p
owlwhisper.GetNextEvent.restype = ctypes.c_char_p

# Функции управления памятью
owlwhisper.FreeString.argtypes = [ctypes.c_char_p]

# Новые функции v1.5
owlwhisper.Connect.argtypes = [ctypes.c_char_p, ctypes.c_char_p]
owlwhisper.Connect.restype = ctypes.c_int
owlwhisper.SetupAutoRelayWithDHT.restype = ctypes.c_int
owlwhisper.StartAggressiveDiscovery.argtypes = [ctypes.c_char_p]
owlwhisper.StartAggressiveDiscovery.restype = ctypes.c_int
owlwhisper.StartAggressiveAdvertising.argtypes = [ctypes.c_char_p]
owlwhisper.StartAggressiveAdvertising.restype = ctypes.c_int
owlwhisper.FindPeersOnce.argtypes = [ctypes.c_char_p]
owlwhisper.FindPeersOnce.restype = ctypes.c_char_p
owlwhisper.AdvertiseOnce.argtypes = [ctypes.c_char_p]
owlwhisper.AdvertiseOnce.restype = ctypes.c_int
```

## 🚀 **Базовое использование**

### **1. Запуск Owl Whisper**

```python
# Запуск с автоматически сгенерированным ключом
result = owlwhisper.StartOwlWhisper()
if result == 0:
    print("✅ Owl Whisper запущен")
else:
    print("❌ Ошибка запуска Owl Whisper")
    exit(1)
```

### **2. Получение Peer ID**

```python
# Получение нашего Peer ID
peer_id_ptr = owlwhisper.GetMyPeerID()
if peer_id_ptr:
    peer_id = ctypes.string_at(peer_id_ptr).decode()
    print(f"👤 Мой Peer ID: {peer_id}")
    
    # ВАЖНО: Освобождаем память
    owlwhisper.FreeString(peer_id_ptr)
else:
    print("❌ Не удалось получить Peer ID")
```

### **3. Мониторинг сети**

```python
import json

# Получение статистики сети
stats_ptr = owlwhisper.GetNetworkStats()
if stats_ptr:
    stats_json = ctypes.string_at(stats_ptr).decode()
    stats = json.loads(stats_json)
    
    print(f"🌐 Статистика сети:")
    print(f"   Подключенных пиров: {stats.get('connected_peers', 0)}")
    print(f"   Статус: {stats.get('status', 'unknown')}")
    
    owlwhisper.FreeString(stats_ptr)

# Получение списка подключенных пиров
peers_ptr = owlwhisper.GetConnectedPeers()
if peers_ptr:
    peers_json = ctypes.string_at(peers_ptr).decode()
    peers = json.loads(peers_json)
    
    print(f"🔗 Подключенные пиры: {len(peers)}")
    for peer in peers:
        print(f"   - {peer['id']}")
    
    owlwhisper.FreeString(peers_ptr)
```

### **4. Отправка сообщений**

```python
# Отправка сообщения всем подключенным пирам
message = "Привет, мир!".encode('utf-8')
result = owlwhisper.SendMessage(message)
if result == 0:
    print("✅ Сообщение отправлено всем пирам")
else:
    print("❌ Ошибка отправки сообщения")

# Отправка сообщения конкретному пиру
peer_id = "12D3KooW...".encode('utf-8')
message = "Привет, конкретный пир!".encode('utf-8')
result = owlwhisper.SendMessageToPeer(peer_id, message)
if result == 0:
    print("✅ Сообщение отправлено конкретному пиру")
else:
    print("❌ Ошибка отправки сообщения")
```

### **5. Остановка**

```python
# Остановка Owl Whisper
result = owlwhisper.StopOwlWhisper()
if result == 0:
    print("✅ Owl Whisper остановлен")
else:
    print("❌ Ошибка остановки")
```

## 🔑 **Создание нового профиля**

### **Генерация ключей**

```python
import base64
import json

# Генерируем новую пару ключей (JSON с информацией)
key_data = owlwhisper.GenerateNewKeyPair()
if key_data:
    # Декодируем данные
    json_str = ctypes.string_at(key_data).decode()
    key_info = json.loads(json_str)
    
    print(f"🔑 Новый профиль создан:")
    print(f"   Peer ID: {key_info['peer_id']}")
    print(f"   Тип ключа: {key_info['key_type']}")
    
    # Получаем приватный ключ (он уже в base64)
    private_key = base64.b64decode(key_info['private_key'])
    
    # Освобождаем память
    owlwhisper.FreeString(key_data)
    
    # Теперь можно зашифровать и сохранить ключ
    # или сразу запустить с ним
    result = owlwhisper.StartOwlWhisperWithKey(private_key, len(private_key))
else:
    print("❌ Ошибка генерации ключей")
```

### **Запуск с существующим ключом**

```python
# Запуск с переданным ключом
with open("private_key.bin", "rb") as f:
    private_key = f.read()

result = owlwhisper.StartOwlWhisperWithKey(private_key, len(private_key))
if result == 0:
    print("✅ Owl Whisper запущен с существующим ключом")
else:
    print("❌ Ошибка запуска с ключом")
```

## 🔍 **Поиск и анонсирование**

### **Поиск пира по Peer ID**

```python
# Поиск пира в сети
peer_id = "12D3KooW...".encode('utf-8')
peer_info_ptr = owlwhisper.FindPeer(peer_id)
if peer_info_ptr:
    peer_info_json = ctypes.string_at(peer_info_ptr).decode()
    if not peer_info_json.startswith("ERROR"):
        peer_info = json.loads(peer_info_json)
        print(f"✅ Пир найден: {peer_info['id']}")
        print(f"   Адреса: {peer_info['addrs']}")
    else:
        print(f"❌ Ошибка поиска: {peer_info_json}")
    
    owlwhisper.FreeString(peer_info_ptr)
else:
    print("❌ Пир не найден")
```

### **Анонсирование контента**

```python
# Анонсируем себя как провайдера контента
content_id = "my-content-123".encode('utf-8')
result = owlwhisper.ProvideContent(content_id)
if result == 0:
    print("✅ Успешно анонсировали контент в сети")
else:
    print("❌ Не удалось анонсировать контент")
```

### **Поиск провайдеров контента**

```python
# Ищем других провайдеров контента
content_id = "my-content-123".encode('utf-8')
providers_ptr = owlwhisper.FindProvidersForContent(content_id)
if providers_ptr:
    providers_json = ctypes.string_at(providers_ptr).decode()
    if not providers_json.startswith("ERROR"):
        providers = json.loads(providers_json)
        print(f"🔍 Найдено провайдеров: {len(providers)}")
        for provider in providers:
            print(f"   - {provider['id']} ({provider['health']})")
    else:
        print(f"❌ Ошибка поиска: {providers_json}")
    
    owlwhisper.FreeString(providers_ptr)
else:
    print("❌ Провайдеры не найдены")
```

## 📡 **Система событий**

### **Слушатель событий**

```python
import threading
import time

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
    
    if event_type == 'NewMessage':
        sender_id = event['payload']['senderID']
        data = event['payload']['data']
        print(f"📨 Новое сообщение от {sender_id}")
        
    elif event_type == 'PeerConnected':
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

## ⚠️ **Важные замечания**

### **Управление памятью:**
- **ВСЕГДА вызывать `FreeString()`** после использования строк от Core
- **Проверять возвращаемые значения** от всех функций
- **Использовать try-catch** для всех Core API вызовов

### **Типы возвращаемых значений:**
- **`int` функции**: 0 = успех, -1 = ошибка
- **`char*` функции**: JSON строки, требующие `FreeString()` после использования

### **Обработка ошибок:**
```python
try:
    result = owlwhisper.SomeFunction()
    if result == 0:
        print("✅ Успешно")
    else:
        print(f"❌ Ошибка: {result}")
except Exception as e:
    print(f"❌ Исключение: {e}")
```

## 🆕 **Новые функции v1.5**

### **Агрессивное Discovery и Advertising**

```python
# Запуск агрессивного поиска пиров
result = owlwhisper.StartAggressiveDiscovery("my-rendezvous")
if result == 0:
    print("✅ Агрессивный поиск запущен")

# Запуск агрессивного анонсирования
result = owlwhisper.StartAggressiveAdvertising("my-rendezvous")
if result == 0:
    print("✅ Агрессивное анонсирование запущено")

# Однократный поиск пиров
peers_ptr = owlwhisper.FindPeersOnce("my-rendezvous")
if peers_ptr:
    peers_json = ctypes.string_at(peers_ptr).decode()
    peers = json.loads(peers_json)
    print(f"Найдено пиров: {len(peers)}")
    owlwhisper.FreeString(peers_ptr)
```

### **Улучшенное подключение к пирам**

```python
# Настройка AutoRelay с DHT
result = owlwhisper.SetupAutoRelayWithDHT()
if result == 0:
    print("✅ AutoRelay с DHT настроен")

# Подключение к пиру по адресам
addrs_json = '["/ip4/192.168.1.100/tcp/1234"]'
result = owlwhisper.Connect("12D3KooW...", addrs_json)
if result == 0:
    print("✅ Успешно подключились к пиру")
```

## 🔗 **Следующие шаги**

1. **Изучите справочник функций** - см. [functions/](./functions/) папку
2. **Настройте логирование** - см. [Утилиты](./functions/utilities.md)
3. **Управляйте соединениями** - см. [Управление соединениями](./functions/connection-management.md)
4. **Интегрируйте события** - см. [Система событий](./functions/events-system.md)
5. **Используйте агрессивное discovery** - см. [Агрессивное Discovery](./functions/aggressive-discovery.md)
6. **Подключайтесь к пирам** - см. [Подключение к пирам](./functions/peer-connection.md)

---

**Последнее обновление:** 28 августа 2025  
**Автор:** Core Development Team 