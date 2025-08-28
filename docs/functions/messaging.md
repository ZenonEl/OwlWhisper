# 📨 **Универсальный обмен данными**

**Последнее обновление:** 28 августа 2025

Функции для универсального обмена данными в формате `[]byte` между пирами.

## 🎯 **Обзор**

**Универсальный обмен данными** - это механизм для передачи любых типов данных между пирами в сети. В отличие от устаревших текстовых функций, новые методы работают с `[]byte`, что позволяет передавать:

- Текстовые сообщения (UTF-8)
- Бинарные файлы
- Структурированные данные (JSON, Protobuf)
- Медиа контент
- Любые другие типы данных

## 📋 **Функции**

### **`Send(peerID, data)`**

**Описание:** Отправляет данные конкретному пиру через существующий поток.

**Сигнатура:**
```c
int Send(char* peerID, char* data, int dataLength);
```

**Параметры:**
- `peerID` (char*) - Peer ID получателя
- `data` (char*) - данные для отправки в формате []byte
- `dataLength` (int) - длина данных в байтах

**Возвращает:**
- `0` - успешная отправка
- `-1` - ошибка отправки

**Пример использования:**
```python
import ctypes

# Настройка типов параметров и возвращаемого значения
owlwhisper.Send.argtypes = [ctypes.c_char_p, ctypes.c_char_p, ctypes.c_int]
owlwhisper.Send.restype = ctypes.c_int

# Отправка текстового сообщения
peer_id = "12D3KooW...".encode('utf-8')
message = "Привет, мир!".encode('utf-8')
result = owlwhisper.Send(peer_id, message, len(message))
if result == 0:
    print("✅ Данные отправлены успешно")
else:
    print("❌ Ошибка отправки данных")
```

**Что происходит при отправке:**
1. Проверяется существование потока к указанному пиру
2. Данные отправляются через libp2p поток
3. Возвращается результат операции

**Важные замечания:**
- ⚠️ **Пир должен быть подключен** - иначе отправка не удастся
- ⚠️ **Данные передаются как есть** - без дополнительной сериализации
- 💡 **Используйте для любых типов данных** - текст, файлы, структуры

---

### **`CreateStream(peerID)`**

**Описание:** Создает новый поток к указанному пиру.

**Сигнатура:**
```c
int CreateStream(char* peerID);
```

**Параметры:**
- `peerID` (char*) - Peer ID пира для подключения

**Возвращает:**
- `0` - успешное создание потока
- `-1` - ошибка создания потока

**Пример использования:**
```python
import ctypes

# Настройка типов параметров и возвращаемого значения
owlwhisper.CreateStream.argtypes = [ctypes.c_char_p]
owlwhisper.CreateStream.restype = ctypes.c_int

# Создание потока к пиру
peer_id = "12D3KooW...".encode('utf-8')
result = owlwhisper.CreateStream(peer_id)
if result == 0:
    print("✅ Поток создан успешно")
else:
    print("❌ Ошибка создания потока")
```

**Что происходит при создании:**
1. Устанавливается соединение с указанным пиром
2. Создается libp2p поток
3. Поток сохраняется для последующего использования

**Важные замечания:**
- ⚠️ **Пир должен быть доступен** в сети
- ⚠️ **Поток создается с таймаутом** из конфигурации
- 💡 **Используйте перед отправкой данных** если поток не существует

---

## 📥 **Получение входящих сообщений**

Входящие сообщения получаются через систему событий `GetNextEvent()`. См. [Система событий](../functions/events-system.md) для подробностей.

**Пример получения сообщений:**
```python
import ctypes
import json

# Настройка типа возвращаемого значения
owlwhisper.GetNextEvent.restype = ctypes.c_char_p

def event_listener():
    """Слушатель событий для получения входящих сообщений"""
    while True:
        try:
            event_ptr = owlwhisper.GetNextEvent()
            if event_ptr:
                event_json = ctypes.string_at(event_ptr).decode()
                owlwhisper.FreeString(event_ptr)
                
                event = json.loads(event_json)
                
                if event['type'] == 'NewMessage':
                    sender_id = event['payload']['senderID']
                    data = event['payload']['data']
                    
                    print(f"📨 Сообщение от {sender_id}")
                    
                    # Обработка данных в зависимости от типа
                    if data.startswith(b'FILE:'):
                        handle_file_data(sender_id, data)
                    elif data.startswith(b'JSON:'):
                        handle_json_data(sender_id, data)
                    else:
                        handle_text_data(sender_id, data)
                        
        except Exception as e:
            print(f"❌ Ошибка в цикле событий: {e}")
            time.sleep(1)

# Запускаем слушатель в отдельном потоке
import threading
thread = threading.Thread(target=event_listener, daemon=True)
thread.start()
```

## 🔄 **Сценарии использования**

### **Отправка текстовых сообщений:**

```python
def send_text_message(peer_id, text):
    """Отправка текстового сообщения"""
    try:
        peer_id_bytes = peer_id.encode('utf-8')
        text_bytes = text.encode('utf-8')
        
        result = owlwhisper.Send(peer_id_bytes, text_bytes, len(text_bytes))
        
        if result == 0:
            print(f"✅ Текстовое сообщение отправлено: {text}")
            return True
        else:
            print(f"❌ Ошибка отправки текста: {result}")
            return False
            
    except Exception as e:
        print(f"❌ Исключение при отправке: {e}")
        return False

# Пример использования
send_text_message("12D3KooW...", "Привет! Как дела?")
```

### **Отправка файлов:**

```python
import os

def send_file(peer_id, file_path):
    """Отправка файла"""
    try:
        with open(file_path, 'rb') as f:
            file_data = f.read()
        
        # Добавляем заголовок для идентификации типа
        header = b'FILE:' + os.path.basename(file_path).encode('utf-8') + b':'
        data_to_send = header + file_data
        
        peer_id_bytes = peer_id.encode('utf-8')
        
        result = owlwhisper.Send(peer_id_bytes, data_to_send, len(data_to_send))
        
        if result == 0:
            print(f"✅ Файл {file_path} отправлен успешно")
            return True
        else:
            print(f"❌ Ошибка отправки файла: {result}")
            return False
            
    except Exception as e:
        print(f"❌ Исключение при отправке файла: {e}")
        return False

# Пример использования
send_file("12D3KooW...", "document.pdf")
```

### **Отправка структурированных данных:**

```python
import json

def send_json_data(peer_id, data_dict):
    """Отправка JSON данных"""
    try:
        json_str = json.dumps(data_dict, ensure_ascii=False)
        json_bytes = json_str.encode('utf-8')
        
        # Добавляем заголовок для JSON
        header = b'JSON:'
        data_to_send = header + json_bytes
        
        peer_id_bytes = peer_id.encode('utf-8')
        
        result = owlwhisper.Send(peer_id_bytes, data_to_send, len(data_to_send))
        
        if result == 0:
            print(f"✅ JSON данные отправлены: {data_dict}")
            return True
        else:
            print(f"❌ Ошибка отправки JSON: {result}")
            return False
            
    except Exception as e:
        print(f"❌ Исключение при отправке JSON: {e}")
        return False

# Пример использования
data = {
    "type": "chat_message",
    "content": "Привет!",
    "timestamp": "2025-08-28T10:00:00Z",
    "metadata": {"priority": "high"}
}
send_json_data("12D3KooW...", data)
```

### **Отправка бинарных данных:**

```python
def send_binary_data(peer_id, binary_data, data_type="binary"):
    """Отправка бинарных данных"""
    try:
        # Добавляем заголовок с типом данных
        header = f"BINARY:{data_type}:".encode('utf-8')
        data_to_send = header + binary_data
        
        peer_id_bytes = peer_id.encode('utf-8')
        
        result = owlwhisper.Send(peer_id_bytes, data_to_send, len(data_to_send))
        
        if result == 0:
            print(f"✅ Бинарные данные типа {data_type} отправлены")
            return True
        else:
            print(f"❌ Ошибка отправки бинарных данных: {result}")
            return False
            
    except Exception as e:
        print(f"❌ Исключение при отправке бинарных данных: {e}")
        return False

# Пример использования
image_data = b'\x89PNG\r\n\x1a\n...'  # PNG файл
send_binary_data("12D3KooW...", image_data, "image/png")
```

## ⚠️ **Важные замечания**

### **Текущие возможности:**
- **Универсальный обмен данными** - поддержка любых типов данных
- **Автоматическое управление потоками** - создание и переиспользование
- **Callback система** - автоматическая обработка входящих данных
- **Безопасность** - все данные передаются через защищенные libp2p соединения

### **Ограничения:**
- **Размер данных** - ограничен размером libp2p потока
- **Таймауты** - настраиваются через NodeConfig
- **Потокобезопасность** - callback должен быть потокобезопасным

### **Рекомендации:**
- **Используйте заголовки** для идентификации типа данных
- **Обрабатывайте ошибки** отправки и получения
- **Используйте callback** для автоматической обработки входящих данных
- **Планируйте размер данных** в соответствии с ограничениями сети

## 🔗 **Связанные функции**

- [Управление Core](../functions/core-management.md) - запуск и остановка Core
- [Подключение к пирам](../functions/peer-connection.md) - установка соединений
- [Система событий](../functions/events-system.md) - альтернативный способ получения событий
- [Поиск пиров](../functions/peer-discovery.md) - поиск получателей для отправки

---

**Последнее обновление:** 28 августа 2025  
**Автор:** Core Development Team 