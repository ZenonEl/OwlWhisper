# 🦉 Owl Whisper API Documentation

## 📋 Обзор

**Owl Whisper** - это децентрализованный P2P мессенджер, который предоставляет Go shared library (`.so`, `.dll`, `.dylib`) для интеграции с любыми языками программирования через FFI/ctypes.

**Ключевые особенности:**
- ✅ **Автоматическое обнаружение пиров** через mDNS (локально) и DHT (глобально)
- ✅ **NAT traversal** с автоматическим hole punching
- ✅ **Поддержка множественных протоколов** (TCP, QUIC, WebRTC)
- ✅ **Прямой вызов функций** без HTTP/WebSocket
- ✅ **Постоянные PeerID** с сохранением ключей
- ✅ **Система никнеймов** с автоматическим обменом профилями
- ✅ **Приватные сообщения 1-на-1** с адресной отправкой


---

## 🚀 Быстрый старт

### 1. Загрузка библиотеки

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

### 2. Базовое использование

```python
# Запуск
result = owlwhisper.StartOwlWhisper()
if result == 0:
    print("✅ Owl Whisper запущен")
    
    # Получение Peer ID
    peer_id = owlwhisper.GetMyPeerID()
    print(f"👤 Мой Peer ID: {ctypes.string_at(peer_id).decode()}")
    owlwhisper.FreeString(peer_id)
    
    # Отправка сообщения
    message = "Привет, мир!".encode('utf-8')
    result = owlwhisper.SendMessage(message)
    
    # Остановка
    owlwhisper.StopOwlWhisper()
```

---

## 📚 API Reference

### 🔧 Основные функции

#### `StartOwlWhisper() -> int`
Запускает Owl Whisper core.
- **Возвращает:** `0` при успехе, `-1` при ошибке
- **Пример:**
```python
result = owlwhisper.StartOwlWhisper()
if result == 0:
    print("✅ Запущен")
else:
    print("❌ Ошибка запуска")
```

#### `StopOwlWhisper() -> int`
Останавливает Owl Whisper core.
- **Возвращает:** `0` при успехе, `-1` при ошибке
- **Пример:**
```python
result = owlwhisper.StopOwlWhisper()
if result == 0:
    print("✅ Остановлен")
```

### 💬 Отправка сообщений

#### `SendMessage(text: str) -> int`
Отправляет сообщение всем подключенным пирам (broadcast).
- **Параметры:** `text` - текст сообщения
- **Возвращает:** `0` при успехе, `-1` при ошибке
- **Пример:**
```python
message = "Привет всем!".encode('utf-8')
result = owlwhisper.SendMessage(message)
```

#### `SendMessageToPeer(peer_id: str, text: str) -> int`
Отправляет сообщение конкретному пиру.
- **Параметры:** 
  - `peer_id` - Peer ID получателя
  - `text` - текст сообщения
- **Возвращает:** `0` при успехе, `-1` при ошибке
- **Пример:**
```python
peer_id = "12D3KooW...".encode('utf-8')
message = "Привет!".encode('utf-8')
result = owlwhisper.SendMessageToPeer(peer_id, message)
```

### 📊 Информация о сети

#### `GetMyPeerID() -> str*`
Получает Peer ID текущего узла.
- **Возвращает:** C-строку с Peer ID
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **Пример:**
```python
peer_id = owlwhisper.GetMyPeerID()
my_id = ctypes.string_at(peer_id).decode()
print(f"Мой ID: {my_id}")
owlwhisper.FreeString(peer_id)  # Освобождаем память
```

#### `GetPeers() -> str*`
Получает список всех подключенных пиров.
- **Возвращает:** JSON-массив с Peer ID'ами
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **Пример:**
```python
import json

peers_data = owlwhisper.GetPeers()
peers_json = ctypes.string_at(peers_data).decode()
peers = json.loads(peers_json)
print(f"Подключено пиров: {len(peers)}")
for peer in peers:
    print(f"  - {peer}")
owlwhisper.FreeString(peers_data)  # Освобождаем память
```

#### `GetConnectionStatus() -> str*`
Получает общий статус подключения.
- **Возвращает:** JSON-объект с информацией о статусе
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **Пример:**
```python
import json

status_data = owlwhisper.GetConnectionStatus()
status_json = ctypes.string_at(status_data).decode()
status = json.loads(status_json)
print(f"Подключен: {status['connected']}")
print(f"Количество пиров: {status['peers']}")
print(f"Мой ID: {status['my_id']}")
owlwhisper.FreeString(status_data)  # Освобождаем память
```

### 👤 Профили и идентификация

#### `GetMyProfile() -> str*`
Получает профиль текущего узла.
- **Возвращает:** JSON-объект с информацией о профиле
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **Пример:**
```python
import json

profile_data = owlwhisper.GetMyProfile()
profile_json = ctypes.string_at(profile_data).decode()
profile = json.loads(profile_json)
print(f"Никнейм: {profile['nickname']}")
print(f"Дискриминатор: {profile['discriminator']}")
print(f"Отображаемое имя: {profile['display_name']}")
owlwhisper.FreeString(profile_data)  # Освобождаем память
```

#### `UpdateMyProfile(nickname: str) -> int`
Обновляет никнейм текущего узла.
- **Параметры:** `nickname` - новый никнейм
- **Возвращает:** `0` при успехе, `-1` при ошибке
- **Пример:**
```python
nickname = "МойНик".encode('utf-8')
result = owlwhisper.UpdateMyProfile(nickname)
if result == 0:
    print("✅ Профиль обновлен")
```

#### `GetPeerProfile(peer_id: str) -> str*`
Получает профиль указанного пира.
- **Параметры:** `peer_id` - Peer ID пира
- **Возвращает:** JSON-объект с информацией о профиле пира
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **Пример:**
```python
import json

peer_id = "12D3KooW...".encode('utf-8')
profile_data = owlwhisper.GetPeerProfile(peer_id)
profile_json = ctypes.string_at(profile_data).decode()
profile = json.loads(profile_json)
print(f"Пир: {profile['nickname']}{profile['discriminator']}")
owlwhisper.FreeString(profile_data)  # Освобождаем память
```

### 🔗 Управление подключениями

#### `ConnectToPeer(peer_id: str) -> int`
Подключается к указанному пиру.
- **Параметры:** `peer_id` - Peer ID для подключения
- **Возвращает:** `0` при успехе, `-1` при ошибке
- **Пример:**
```python
peer_id = "12D3KooW...".encode('utf-8')
result = owlwhisper.ConnectToPeer(peer_id)
if result == 0:
    print("✅ Подключение установлено")
```

### 📜 История сообщений

#### `GetChatHistory(peer_id: str) -> str*`
Получает историю сообщений с указанным пиром.
- **Параметры:** `peer_id` - Peer ID собеседника
- **Возвращает:** JSON-массив с сообщениями
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования

#### `GetChatHistoryLimit(peer_id: str, limit: int) -> str*`
Получает ограниченную историю сообщений.
- **Параметры:** 
  - `peer_id` - Peer ID собеседника
  - `limit` - максимальное количество сообщений
- **Возвращает:** JSON-массив с сообщениями
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования

### 🧹 Управление памятью

#### `FreeString(str_ptr: str*) -> void`
Освобождает память, выделенную для строки.
- **Параметры:** `str_ptr` - указатель на строку
- **⚠️ КРИТИЧНО:** Всегда вызывайте эту функцию после использования строк, возвращенных библиотекой
- **Пример:**
```python
# Получаем данные
data = owlwhisper.GetMyPeerID()

# Используем данные
print(ctypes.string_at(data).decode())

# ОБЯЗАТЕЛЬНО освобождаем память
owlwhisper.FreeString(data)
```

---

## 🔧 Типы данных

### C типы
```c
typedef int C.int;
typedef char* C.char;
```

### Python типы
```python
# Для строк используйте bytes
message = "Текст".encode('utf-8')

# Для чисел используйте int
limit = 100
```

---

## 📝 Примеры использования

### Полный пример Python клиента

```python
import ctypes
import json
import time

# Загружаем библиотеку
owlwhisper = ctypes.CDLL("./dist/libowlwhisper.so")

def main():
    try:
        # Запускаем
        print("🚀 Запуск Owl Whisper...")
        result = owlwhisper.StartOwlWhisper()
        if result != 0:
            print("❌ Ошибка запуска")
            return
        
        print("✅ Owl Whisper запущен")
        
        # Ждем подключения
        time.sleep(3)
        
        # Получаем статус
        status_data = owlwhisper.GetConnectionStatus()
        status_json = ctypes.string_at(status_data).decode()
        status = json.loads(status_json)
        owlwhisper.FreeString(status_data)
        
        print(f"🌐 Статус: {status}")
        
        # Получаем пиров
        peers_data = owlwhisper.GetPeers()
        peers_json = ctypes.string_at(peers_data).decode()
        peers = json.loads(peers_json)
        owlwhisper.FreeString(peers_data)
        
        print(f"👥 Пиры: {peers}")
        
        # Отправляем сообщение
        if len(peers) > 0:
            message = "Привет от Python!".encode('utf-8')
            result = owlwhisper.SendMessage(message)
            if result == 0:
                print("✅ Сообщение отправлено")
        
        # Ждем
        time.sleep(2)
        
        # Останавливаем
        print("🛑 Остановка...")
        owlwhisper.StopOwlWhisper()
        print("✅ Остановлен")
        
    except Exception as e:
        print(f"❌ Ошибка: {e}")

if __name__ == "__main__":
    main()
```

### Node.js пример

```javascript
const ffi = require('ffi-napi');
const ref = require('ref-napi');

// Загружаем библиотеку
const owlwhisper = ffi.Library('./dist/libowlwhisper', {
    'StartOwlWhisper': ['int', []],
    'StopOwlWhisper': ['int', []],
    'SendMessage': ['int', ['string']],
    'GetMyPeerID': ['string', []],
    'GetPeers': ['string', []],
    'GetConnectionStatus': ['string', []],
    'FreeString': ['void', ['string']]
});

// Использование
const result = owlwhisper.StartOwlWhisper();
if (result === 0) {
    console.log('✅ Запущен');
    
    const peerId = owlwhisper.GetMyPeerID();
    console.log('Peer ID:', peerId);
    owlwhisper.FreeString(peerId);
    
    owlwhisper.StopOwlWhisper();
}
```

---

## ⚠️ Важные замечания

### 🔒 Управление памятью
- **ВСЕГДА** вызывайте `FreeString()` после использования строк, возвращенных библиотекой
- Неиспользование `FreeString()` приведет к утечкам памяти
- В Python используйте `try/finally` для гарантированного освобождения памяти

### 🚫 Ограничения
- Библиотека должна быть запущена (`StartOwlWhisper`) перед вызовом других функций
- Не вызывайте функции после остановки (`StopOwlWhisper`)
- Peer ID должны быть валидными libp2p идентификаторами

### 🔄 Асинхронность
- Некоторые операции (подключение к пирам, обнаружение) происходят асинхронно
- Используйте `GetConnectionStatus()` для проверки текущего состояния
- Регулярно вызывайте `GetPeers()` для обновления списка пиров

---

## 🐛 Устранение неполадок

### Проблема: "Cannot open shared object file"
```bash
# Установите переменную окружения
export LD_LIBRARY_PATH=./dist:$LD_LIBRARY_PATH  # Linux
export DYLD_LIBRARY_PATH=./dist:$DYLD_LIBRARY_PATH  # macOS
```

### Проблема: "Segmentation fault"
- Убедитесь, что вызываете `FreeString()` после использования строк
- Проверьте, что библиотека запущена перед вызовом функций

### Проблема: Пустой список пиров
- **ИСПРАВЛЕНО:** Теперь библиотека корректно возвращает пиров из всех источников (mDNS, DHT, TCP, QUIC)
- Убедитесь, что прошло достаточно времени для обнаружения пиров (обычно 2-5 секунд)
- Проверьте, что ваш узел успешно подключился к bootstrap узлам

---

## 📁 Структура файлов

```
dist/
├── libowlwhisper.so      # Linux shared library
├── owlwhisper.dll        # Windows shared library  
├── libowlwhisper.dylib   # macOS shared library
└── README.md             # Документация по файлам

examples/
└── python_client.py      # Пример Python клиента

internal/core/
├── c_wrapper.go          # C обертки для экспорта
└── owlwhisper.h          # C заголовочный файл
```

---

## 🔗 Ссылки

- **libp2p:** https://libp2p.io/
- **Go CGo:** https://golang.org/cmd/cgo/
- **Python ctypes:** https://docs.python.org/3/library/ctypes.html
- **Node.js ffi-napi:** https://github.com/node-ffi-napi/node-ffi-napi

---

## 📞 Поддержка

Если у вас возникли проблемы:
1. Проверьте логи Go core (выводятся в консоль)
2. Убедитесь, что вызываете `FreeString()` после использования строк
3. Проверьте, что библиотека запущена перед вызовом функций
4. Убедитесь, что прошло достаточно времени для обнаружения пиров

**Последнее обновление:** 21 августа 2025