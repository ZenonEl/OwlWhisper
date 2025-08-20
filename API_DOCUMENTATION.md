# 🦉 Owl Whisper API Documentation

## **📚 Обзор**

Owl Whisper предоставляет **прямой доступ к функциям** через Go shared library (`.so`, `.dll`, `.dylib`). Прямой вызов функций через ctypes/FFI.

---

## **🔧 Установка и компиляция**

### **1. Компиляция Go в shared library:**

```bash
# Linux
go build -buildmode=c-shared -o dist/libowlwhisper.so ./cmd/owlwhisper

# Windows
go build -buildmode=c-shared -o dist/owlwhisper.dll ./cmd/owlwhisper

# macOS
go build -buildmode=c-shared -o dist/libowlwhisper.dylib ./cmd/owlwhisper
```

### **2. Python пример:**
```python
import ctypes
owlwhisper = ctypes.CDLL("./dist/libowlwhisper.so")
```

---

## **🚀 Основные функции**

### **Инициализация и управление**

#### `StartOwlWhisper()`
- **Описание:** Запускает Owl Whisper
- **Возвращает:** `int` (0 = успех, -1 = ошибка)
- **Пример:** `result = owlwhisper.StartOwlWhisper()`

#### `StopOwlWhisper()`
- **Описание:** Останавливает Owl Whisper
- **Возвращает:** `int` (0 = успех, -1 = ошибка)
- **Пример:** `result = owlwhisper.StopOwlWhisper()`

---

### **Отправка сообщений**

#### `SendMessage(text)`
- **Описание:** Отправляет сообщение всем подключенным пирам
- **Параметры:** `char* text` - текст сообщения
- **Возвращает:** `int` (0 = успех, -1 = ошибка)
- **Пример:** `result = owlwhisper.SendMessage("Привет!")`

#### `SendMessageToPeer(peerID, text)`
- **Описание:** Отправляет сообщение конкретному пиру
- **Параметры:** 
  - `char* peerID` - ID пира
  - `char* text` - текст сообщения
- **Возвращает:** `int` (0 = успех, -1 = ошибка)
- **Пример:** `result = owlwhisper.SendMessageToPeer("peer123", "Привет!")`

---

### **Получение информации**

#### `GetMyPeerID()`
- **Описание:** Возвращает ваш Peer ID
- **Возвращает:** `char*` - строка с Peer ID
- **Пример:** `peer_id = owlwhisper.GetMyPeerID()`
- **Важно:** Не забудьте вызвать `FreeString(peer_id)` после использования!

#### `GetPeers()`
- **Описание:** Возвращает список подключенных пиров
- **Возвращает:** `char*` - JSON массив строк
- **Пример:** `peers = owlwhisper.GetPeers()`
- **Важно:** Не забудьте вызвать `FreeString(peers)` после использования!

#### `GetConnectionStatus()`
- **Описание:** Возвращает статус сетевого подключения
- **Возвращает:** `char*` - JSON объект с информацией
- **Пример:** `status = owlwhisper.GetConnectionStatus()`
- **Важно:** Не забудьте вызвать `FreeString(status)` после использования!

---

### **История чата**

#### `GetChatHistory(peerID)`
- **Описание:** Возвращает историю чата с пиром
- **Параметры:** `char* peerID` - ID пира
- **Возвращает:** `char*` - JSON массив сообщений
- **Пример:** `history = owlwhisper.GetChatHistory("peer123")`
- **Важно:** Не забудьте вызвать `FreeString(history)` после использования!

#### `GetChatHistoryLimit(peerID, limit)`
- **Описание:** Возвращает ограниченную историю чата с пиром
- **Параметры:** 
  - `char* peerID` - ID пира
  - `int limit` - максимальное количество сообщений
- **Возвращает:** `char*` - JSON массив сообщений
- **Пример:** `history = owlwhisper.GetChatHistoryLimit("peer123", 50)`
- **Важно:** Не забудьте вызвать `FreeString(history)` после использования!

---

### **Подключение к пирам**

#### `ConnectToPeer(peerID)`
- **Описание:** Подключается к пиру по ID
- **Параметры:** `char* peerID` - ID пира для подключения
- **Возвращает:** `int` (0 = успех, -1 = ошибка)
- **Пример:** `result = owlwhisper.ConnectToPeer("peer123")`

---

### **Управление памятью**

#### `FreeString(str)`
- **Описание:** Освобождает память, выделенную для строк
- **Параметры:** `char* str` - строка для освобождения
- **Возвращает:** `void`
- **Пример:** `owlwhisper.FreeString(peer_id)`
- **Важно:** **ВСЕГДА** вызывайте эту функцию для строк, возвращенных библиотекой!

---

## **📝 Примеры использования**

### **Python (ctypes)**
```python
import ctypes
import json

# Загружаем библиотеку
owlwhisper = ctypes.CDLL("./dist/libowlwhisper.so")

# Настраиваем типы возвращаемых значений
owlwhisper.GetMyPeerID.restype = ctypes.c_char_p
owlwhisper.GetPeers.restype = ctypes.c_char_p
owlwhisper.GetConnectionStatus.restype = ctypes.c_char_p
owlwhisper.GetChatHistory.restype = ctypes.c_char_p

# Запускаем
owlwhisper.StartOwlWhisper()

# Получаем Peer ID
peer_id = owlwhisper.GetMyPeerID()
print(f"Мой ID: {peer_id.decode('utf-8')}")
owlwhisper.FreeString(peer_id)

# Отправляем сообщение
owlwhisper.SendMessage("Привет всем!")

# Останавливаем
owlwhisper.StopOwlWhisper()
```

### **JavaScript/Node.js (ffi-napi)**
```javascript
const ffi = require('ffi-napi');

const owlwhisper = ffi.Library('./dist/libowlwhisper.so', {
  'StartOwlWhisper': ['int', []],
  'SendMessage': ['int', ['string']],
  'GetMyPeerID': ['string', []],
  'StopOwlWhisper': ['int', []]
});

owlwhisper.StartOwlWhisper();
const peerId = owlwhisper.GetMyPeerID();
console.log(`Мой ID: ${peerId}`);
owlwhisper.StopOwlWhisper();
```

---

## **⚠️ Важные замечания**

1. **Управление памятью:** Всегда вызывайте `FreeString()` для строк, возвращенных библиотекой
2. **Кодировка:** Строки передаются как UTF-8
3. **Ошибки:** Функции возвращают 0 при успехе, -1 при ошибке
4. **JSON:** Некоторые функции возвращают JSON строки для сложных данных
5. **Потокобезопасность:** Библиотека не является потокобезопасной

---

## **🔍 Отладка**

### **Проверка загрузки библиотеки:**
```python
try:
    owlwhisper = ctypes.CDLL("./dist/libowlwhisper.so")
    print("✅ Библиотека загружена")
except Exception as e:
    print(f"❌ Ошибка: {e}")
```

### **Проверка функций:**
```python
# Проверяем что функция существует
if hasattr(owlwhisper, 'StartOwlWhisper'):
    print("✅ Функция StartOwlWhisper найдена")
else:
    print("❌ Функция StartOwlWhisper не найдена")
```

---

## **📁 Структура файлов**

```
dist/
├── libowlwhisper.so      # Linux shared library
├── owlwhisper.dll        # Windows shared library  
├── libowlwhisper.dylib   # macOS shared library
└── README.md             # Документация по дистрибутиву

examples/
└── python_client.py      # Пример Python клиента
```