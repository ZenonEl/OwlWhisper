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

### 2.5. Создание нового профиля

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

### 2.6. Прямая генерация ключей (для продвинутых)

```python
# Генерируем сырые байты ключа напрямую
key_bytes = owlwhisper.GenerateNewKeyBytes()
if key_bytes:
    # Получаем сырые байты ключа
    raw_key = ctypes.string_at(key_bytes)
    
    print(f"🔑 Сырые байты ключа: {len(raw_key)} байт")
    
    # Освобождаем память
    owlwhisper.FreeString(key_bytes)
    
    # Запускаем с сырыми байтами
    result = owlwhisper.StartOwlWhisperWithKey(raw_key, len(raw_key))
else:
    print("❌ Ошибка генерации ключа")
```

### 3. Мульти-профильная система

```python
import ctypes
import os
import base64
import json

# Настройка типов для функций
owlwhisper.StartOwlWhisperWithKey.argtypes = [ctypes.c_char_p, ctypes.c_int]
owlwhisper.StartOwlWhisperWithKey.restype = ctypes.c_int
owlwhisper.GenerateNewKeyPair.restype = ctypes.c_char_p

# Вариант 1: Создание нового профиля
def create_new_profile(nickname):
    # Генерируем новую пару ключей
    key_data = owlwhisper.GenerateNewKeyPair()
    if not key_data:
        print("❌ Ошибка генерации ключей")
        return None
    
    # Декодируем данные ключа
    json_str = ctypes.string_at(key_data).decode()
    key_info = json.loads(json_str)
    
    print(f"✅ Создан новый профиль:")
    print(f"   Peer ID: {key_info['peer_id']}")
    print(f"   Никнейм: {nickname}")
    
    # Получаем приватный ключ
    private_key = base64.b64decode(key_info['private_key'])
    
    # Освобождаем память
    owlwhisper.FreeString(key_data)
    
    # Теперь можно зашифровать и сохранить ключ
    return private_key, key_info['peer_id']

# Вариант 2: Использование существующего профиля
def load_existing_profile(profile_path):
    # Загружаем зашифрованный ключ из профиля
    with open(os.path.expanduser(profile_path), "rb") as f:
        encrypted_key = f.read()

    # Расшифровываем ключ (это делает Python клиент)
    decrypted_key = decrypt_key(encrypted_key, user_password)

    # Запускаем с профилем
    result = owlwhisper.StartOwlWhisperWithKey(decrypted_key, len(decrypted_key))
    if result == 0:
        print("✅ Owl Whisper запущен с профилем")
        
        # Работаем с профилем...
        profile = owlwhisper.GetMyProfile()
        print(f"Профиль: {ctypes.string_at(profile).decode()}")
        owlwhisper.FreeString(profile)
        
        return True
    else:
        print("❌ Ошибка запуска профиля")
        return False

# Пример использования
if __name__ == "__main__":
    # Создаем новый профиль
    new_key, new_peer_id = create_new_profile("Рабочий")
    if new_key:
        # Запускаем с новым ключом
        result = owlwhisper.StartOwlWhisperWithKey(new_key, len(new_key))
        if result == 0:
            print("✅ Новый профиль запущен!")
            
            # Останавливаем
            owlwhisper.StopOwlWhisper()
```

---

## 🏗️ Архитектура мульти-профильной системы

### Принципы проектирования

**Owl Whisper Core** остается **"беспрофильным" (profile-agnostic)** и не знает о существовании нескольких аккаунтов. Вместо этого он принимает ключ как аргумент, что обеспечивает:

- ✅ **Модульность** - Core фокусируется только на P2P сети
- ✅ **Безопасность** - ключи никогда не хранятся в Core
- ✅ **Гибкость** - клиент полностью контролирует управление профилями
- ✅ **Масштабируемость** - легко добавлять новые профили без изменения Core

### Распределение ответственности

| Задача | Кто отвечает? | Детали |
|--------|---------------|---------|
| **Генерация ключей** | **Core (Go)** | `GenerateNewKeyPair()` создает Ed25519 ключи в libp2p формате |
| **Хранение профилей** | **Клиент (Python/Flet)** | Управляет директориями `~/.config/owlwhisper/profiles/<uuid>/` |
| **Шифрование ключей** | **Клиент (Python/Flet)** | Использует `cryptography` для AES-256-GCM + Argon2 |
| **Запрос паролей** | **Клиент (Python/Flet)** | UI экран входа/разблокировки |
| **Запуск P2P узла** | **Core (Go)** | Принимает `[]byte` ключа как аргумент |
| **Хранение данных** | **Клиент (Python/Flet)** | Управляет `profile.db` для каждого профиля |

### Структура хранения

```
~/.config/owlwhisper/
├── profiles.json                    # Мастер-файл со списком профилей
└── profiles/
    ├── <profile_uuid_1>/           # Рабочий профиль
    │   ├── identity.key.encrypted  # ЗАШИФРОВАННЫЙ приватный ключ
    │   ├── profile.db              # SQLite база для этого профиля
    │   └── settings.json           # Настройки UI
    │
    └── <profile_uuid_2>/           # Личный профиль
        ├── identity.key.encrypted
        ├── profile.db
        └── settings.json
```

### 🔑 Формат ключей

**Важно:** Go библиотека ожидает ключи в **libp2p формате**, а не в Protobuf или сырых байтах!

- **✅ Правильно:** 
  - `GenerateNewKeyPair()` - для получения JSON с информацией о ключе
  - `GenerateNewKeyBytes()` - для получения сырых байтов ключа
- **❌ Неправильно:** `secrets.token_bytes(32)` или Protobuf сериализация
- **🔧 Формат:** Ed25519 ключи, сериализованные через `crypto.MarshalPrivateKey()`
- **📏 Размер:** Обычно 68 байт для Ed25519 ключей
- **🆔 PeerID:** Автоматически генерируется из публичного ключа

### 🔄 Два способа работы с ключами

#### Способ 1: JSON с метаданными (рекомендуется для UI)
```python
# Получаем полную информацию о ключе
key_data = owlwhisper.GenerateNewKeyPair()
json_str = ctypes.string_at(key_data).decode()
key_info = json.loads(json_str)

# Ключ уже в base64, декодируем для использования
private_key = base64.b64decode(key_info['private_key'])
result = owlwhisper.StartOwlWhisperWithKey(private_key, len(private_key))

owlwhisper.FreeString(key_data)
```

#### Способ 2: Сырые байты (для продвинутых)
```python
# Получаем только байты ключа
key_bytes = owlwhisper.GenerateNewKeyBytes()
raw_key = ctypes.string_at(key_bytes)

# Используем напрямую
result = owlwhisper.StartOwlWhisperWithKey(raw_key, len(raw_key))

owlwhisper.FreeString(key_bytes)
```

---

## 📚 API Reference

### 🔧 Основные функции

#### `StartOwlWhisper() -> int`
Запускает Owl Whisper core с автоматически загруженным ключом (для обратной совместимости).
- **Возвращает:** `0` при успехе, `1` при ошибке
- **Пример:**
```python
result = owlwhisper.StartOwlWhisper()
if result == 0:
    print("✅ Запущен")
else:
    print("❌ Ошибка запуска")
```

#### `StartOwlWhisperWithKey(keyBytes: bytes, keyLength: int) -> int`
Запускает Owl Whisper core с переданным ключом (для мульти-профильных систем).
- **Параметры:**
  - `keyBytes` - байты приватного ключа
  - `keyLength` - длина ключа в байтах
- **Возвращает:** `0` при успехе, `1` при ошибке
- **Пример:**
```python
# Загружаем зашифрованный ключ из профиля
with open("profile/identity.key.encrypted", "rb") as f:
    key_data = f.read()
    
# Расшифровываем ключ (это делает Python клиент)
decrypted_key = decrypt_key(key_data, user_password)

# Запускаем с расшифрованным ключом
result = owlwhisper.StartOwlWhisperWithKey(decrypted_key, len(decrypted_key))
if result == 0:
    print("✅ Owl Whisper запущен с профилем")
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

#### `GenerateNewKeyPair() -> str*`
Генерирует новую пару libp2p ключей Ed25519.
- **Возвращает:** Base64-закодированную JSON-строку с информацией о ключе
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **Структура возвращаемых данных:**
```json
{
  "private_key": "base64_encoded_key_bytes",
  "peer_id": "12D3KooW...",
  "key_type": "Ed25519",
  "key_length": 68
}
```
- **Пример:**
```python
import base64
```

#### `GenerateNewKeyBytes() -> str*`
Генерирует сырые байты libp2p ключа Ed25519 для прямого использования.
- **Возвращает:** Сырые байты ключа в libp2p формате
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **💡 Использование:** Для случаев, когда нужны только байты ключа без JSON метаданных
- **📏 Размер:** Обычно 68 байт для Ed25519 ключей
- **Пример:**
```python
# Получаем сырые байты ключа
key_bytes = owlwhisper.GenerateNewKeyBytes()
if key_bytes:
    raw_key = ctypes.string_at(key_bytes)
    
    # Запускаем с сырыми байтами
    result = owlwhisper.StartOwlWhisperWithKey(raw_key, len(raw_key))
    
    # Освобождаем память
    owlwhisper.FreeString(key_bytes)
```
import json

# Генерируем новую пару ключей
key_data = owlwhisper.GenerateNewKeyPair()
if key_data:
    # Декодируем base64
    json_str = ctypes.string_at(key_data).decode()
    key_info = json.loads(json_str)
    
    print(f"Новый Peer ID: {key_info['peer_id']}")
    print(f"Тип ключа: {key_info['key_type']}")
    print(f"Длина ключа: {key_info['key_length']} байт")
    
    # Получаем приватный ключ для использования
    private_key = base64.b64decode(key_info['private_key'])
    
    # Освобождаем память
    owlwhisper.FreeString(key_data)
    
    # Теперь можно использовать ключ для запуска
    result = owlwhisper.StartOwlWhisperWithKey(private_key, len(private_key))
else:
    print("❌ Ошибка генерации ключей")
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

#### `GetConnectedPeers() -> str*`
Получает список всех подключенных пиров.
- **Возвращает:** JSON-массив с Peer ID'ами
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **💡 Изменение:** Переименовано с `GetPeers()` для ясности
- **Пример:**
```python
import json

peers_data = owlwhisper.GetConnectedPeers()
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

### 📊 Сетевая диагностика и мониторинг

#### `GetNetworkStats() -> str*`
Получает детальную статистику сети для отладки.
- **Возвращает:** JSON-объект с подробной информацией о сети
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **Пример:**
```python
import json

stats_data = owlwhisper.GetNetworkStats()
stats_json = ctypes.string_at(stats_data).decode()
stats = json.loads(stats_json)
print(f"Статус: {stats['status']}")
print(f"Всего пиров: {stats['total_peers']}")
print(f"Подключенных пиров: {stats['connected_peers']}")
print(f"Всего соединений: {stats['total_connections']}")
print(f"Протоколы: {stats['protocols']}")
owlwhisper.FreeString(stats_data)
```

#### `GetConnectionQuality(peer_id: str) -> str*`
Получает качество соединения с конкретным пиром.
- **Параметры:** `peer_id` - Peer ID пира
- **Возвращает:** JSON-объект с информацией о качестве соединения
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **Пример:**
```python
import json

peer_id = "12D3KooW...".encode('utf-8')
quality_data = owlwhisper.GetConnectionQuality(peer_id)
quality_json = ctypes.string_at(quality_data).decode()
quality = json.loads(quality_json)
print(f"Статус: {quality['status']}")
print(f"Соединений: {quality['total_connections']}")
print(f"Стримов: {quality['total_streams']}")
print(f"Протоколы: {quality['protocols']}")
owlwhisper.FreeString(quality_data)
```

### 🔍 Поиск и обнаружение пиров

#### `FindPeer(peer_id: str) -> str*`
Ищет пира в сети по PeerID.
- **Параметры:** `peer_id` - Peer ID для поиска
- **Возвращает:** JSON-объект с информацией о найденном пире
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **💡 Примечание:** Пока ищет только среди уже подключенных пиров
- **Пример:**
```python
import json

peer_id = "12D3KooW...".encode('utf-8')
peer_data = owlwhisper.FindPeer(peer_id)
if peer_data:
    peer_json = ctypes.string_at(peer_data).decode()
    peer_info = json.loads(peer_json)
    print(f"Найден пир: {peer_info['id']}")
    print(f"Адреса: {peer_info['addrs']}")
    owlwhisper.FreeString(peer_data)
```

#### `FindPeerByNickname(nickname: str) -> str*`
Ищет пира по никнейму в локальной базе данных.
- **Параметры:** `nickname` - никнейм для поиска
- **Возвращает:** JSON-объект с информацией о профиле пира
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **💡 Примечание:** Пока возвращает заглушку (не реализовано)
- **Пример:**
```python
import json

nickname = "Друг".encode('utf-8')
profile_data = owlwhisper.FindPeerByNickname(nickname)
if profile_data:
    profile_json = ctypes.string_at(profile_data).decode()
    profile = json.loads(profile_json)
    print(f"Найден профиль: {profile['nickname']}")
    owlwhisper.FreeString(profile_data)
```

### 🔑 Генерация ключей

#### `GenerateNewKeyPair() -> str*`
Генерирует новую пару libp2p ключей Ed25519.
- **Возвращает:** Base64-закодированную JSON-строку с информацией о ключе
- **⚠️ Важно:** Не забудьте вызвать `FreeString()` после использования
- **Структура возвращаемых данных:**
```json
{
  "private_key": "base64_encoded_key_bytes",
  "peer_id": "12D3KooW...",
  "key_type": "Ed25519",
  "key_length": 68
}
```
- **Пример:**
```python
import base64
import json

# Генерируем новую пару ключей
key_data = owlwhisper.GenerateNewKeyPair()
if key_data:
    # Декодируем base64
    json_str = ctypes.string_at(key_data).decode()
    key_info = json.loads(json_str)
    
    print(f"Новый Peer ID: {key_info['peer_id']}")
    print(f"Тип ключа: {key_info['key_type']}")
    print(f"Длина ключа: {key_info['key_length']} байт")
    
    # Получаем приватный ключ для использования
    private_key = base64.b64decode(key_info['private_key'])
    
    # Освобождаем память
    owlwhisper.FreeString(key_data)
    
    # Теперь можно использовать ключ для запуска
    result = owlwhisper.StartOwlWhisperWithKey(private_key, len(private_key))
else:
    print("❌ Ошибка генерации ключей")
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
        # Настраиваем логирование (опционально)
        # owlwhisper.SetLogLevel(0)  # Отключить все логи
        # owlwhisper.SetLogOutput(2, "./logs")  # Логи только в файл
        
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
        peers_data = owlwhisper.GetConnectedPeers()
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
    'StartOwlWhisperWithKey': ['int', ['string', 'int']],
    'StopOwlWhisper': ['int', []],
    'GenerateNewKeyPair': ['string', []],
    'SendMessage': ['int', ['string']],
    'GetMyPeerID': ['string', []],
    'GetConnectedPeers': ['string', []],
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

// Пример создания нового профиля
console.log('🔑 Создаем новый профиль...');
const keyData = owlwhisper.GenerateNewKeyPair();
if (keyData) {
    const keyInfo = JSON.parse(Buffer.from(keyData, 'base64').toString());
    console.log('Новый Peer ID:', keyInfo.peer_id);
    console.log('Тип ключа:', keyInfo.key_type);
    
    // Освобождаем память
    owlwhisper.FreeString(keyData);
    
    // Теперь можно использовать ключ
    const privateKey = Buffer.from(keyInfo.private_key, 'base64');
    const startResult = owlwhisper.StartOwlWhisperWithKey(privateKey, privateKey.length);
    if (startResult === 0) {
        console.log('✅ Новый профиль запущен!');
        owlwhisper.StopOwlWhisper();
    }
}
```

## 🔧 **Функции для настройки логирования**

### **SetLogLevel**
Устанавливает уровень логирования.

**Параметры:**
- `level` (int): Уровень логирования
  - 0: SILENT - логи отключены
  - 1: ERROR - только ошибки
  - 2: WARN - предупреждения и ошибки
  - 3: INFO - информация, предупреждения и ошибки
  - 4: DEBUG - все логи

**Возвращает:** 0 при успехе, 1 при ошибке

### **SetLogOutput**
Настраивает вывод логов.

**Параметры:**
- `output` (int): Тип вывода
  - 0: NONE - логи отключены
  - 1: CONSOLE - только в консоль
  - 2: FILE - только в файл
  - 3: BOTH - в консоль и файл
- `log_dir` (char*): Директория для логов (для FILE и BOTH)

**Возвращает:** 0 при успехе, 1 при ошибке

**Примеры использования:**
```python
# Отключить все логи
owlwhisper.SetLogLevel(0)

# Только ошибки в консоль
owlwhisper.SetLogLevel(1)

# Все логи в файл
owlwhisper.SetLogOutput(2, "./logs")

# Все логи в консоль и файл
owlwhisper.SetLogOutput(3, "./logs")
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
- Регулярно вызывайте `GetConnectedPeers()` для обновления списка пиров

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

### Проблема: Сложность работы с ключами
- **РЕШЕНИЕ:** ✅ **ДОБАВЛЕНЫ ДВЕ ФУНКЦИИ** для упрощения работы с ключами
- **`GenerateNewKeyPair()`** - возвращает JSON с метаданными (рекомендуется для UI)
- **`GenerateNewKeyBytes()`** - возвращает сырые байты ключа (для продвинутых)
- **Статус:** Проблема полностью решена в версии от 23 августа 2025

### Проблема: "proto: cannot parse invalid wire-format data"
- **РЕШЕНИЕ:** Используйте `GenerateNewKeyPair()` для создания ключей вместо `secrets.token_bytes(32)`
- **Причина:** Go библиотека ожидает ключи в libp2p формате, а не случайные байты
- **Формат:** Ed25519 ключи, сериализованные через `crypto.MarshalPrivateKey()`
- **Размер:** Обычно 68 байт для Ed25519 ключей

### Проблема: "panic: assignment to entry in nil map"
- **РЕШЕНИЕ:** ✅ **ИСПРАВЛЕНО** - теперь map `peers` корректно инициализируется
- **Причина:** Map не был инициализирован в конструкторе `NewNodeWithKey`
- **Статус:** Проблема полностью решена в версии от 23 августа 2025

### Проблема: "Segmentation fault при вызове StopOwlWhisper"
- **РЕШЕНИЕ:** ⚠️ **ИЗВЕСТНАЯ ПРОБЛЕМА** - избегайте вызова `StopOwlWhisper` в тестах
- **Причина:** Возможные race conditions при graceful shutdown
- **Временное решение:** Не вызывайте `StopOwlWhisper` в тестовых сценариях
- **Статус:** Исследуется, планируется исправление в следующих версиях

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

**Последнее обновление:** 23 августа 2025 (Этап 1 завершен: добавлены сетевые методы, переименован GetPeers, исправлены критические баги)