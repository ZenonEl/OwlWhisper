# 📦 **Управление контентом**

**Последнее обновление:** 23 августа 2025

Функции для анонсирования и поиска контента в DHT сети.

## 📋 **Функции**

### **`ProvideContent(contentID)`**

**Описание:** Анонсирует текущий узел как провайдера контента в DHT сети.

**Сигнатура:**
```c
int ProvideContent(char* contentID);
```

**Параметры:**
- `contentID` (char*) - идентификатор контента для анонсирования

**Возвращает:**
- `0` - успешное анонсирование
- `-1` - ошибка анонсирования

**Пример использования:**
```python
import ctypes

# Настройка типа возвращаемого значения
owlwhisper.ProvideContent.restype = ctypes.c_int

# Анонсируем себя как провайдера контента
content_id = "my-content-123".encode('utf-8')
result = owlwhisper.ProvideContent(content_id)
if result == 0:
    print("✅ Успешно анонсировали контент в сети")
else:
    print("❌ Не удалось анонсировать контент")
```

**Что происходит при анонсировании:**
1. Content ID хешируется для создания DHT ключа
2. Текущий узел анонсируется как провайдер в DHT
3. Другие узлы могут найти этот контент через `FindProvidersForContent`
4. Анонс автоматически распространяется по DHT сети

**Важные замечания:**
- ⚠️ **Content ID должен быть уникальным** - используйте хеши или UUID
- 💡 **Используйте для анонсирования** доступности контента
- 🔄 **Анонс автоматически распространяется** по DHT сети
- 📍 **Узел становится discoverable** для поиска контента

---

### **`FindProvidersForContent(contentID)`**

**Описание:** Ищет провайдеров контента в DHT сети.

**Сигнатура:**
```c
char* FindProvidersForContent(char* contentID);
```

**Параметры:**
- `contentID` (char*) - идентификатор искомого контента

**Возвращает:**
- `char*` - JSON строка со списком провайдеров (требует `FreeString()`)
- `NULL` - провайдеры не найдены или ошибка

**Структура возвращаемых данных:**
```json
[
  {
    "id": "12D3KooW...",
    "addrs": ["/ip4/192.168.1.100/tcp/1234"],
    "protocols": ["/owlwhisper/1.0.0"],
    "health": "good",
    "last_seen": 1755969188
  },
  {
    "id": "12D3KooW...",
    "addrs": ["/ip6/::1/tcp/1234"],
    "protocols": ["/owlwhisper/1.0.0"],
    "health": "excellent",
    "last_seen": 1755969189
  }
]
```

**Пример использования:**
```python
import ctypes
import json

# Настройка типа возвращаемого значения
owlwhisper.FindProvidersForContent.restype = ctypes.c_char_p

# Ищем провайдеров контента
content_id = "my-content-123".encode('utf-8')
providers_ptr = owlwhisper.FindProvidersForContent(content_id)
if providers_ptr:
    providers_json = ctypes.string_at(providers_ptr).decode()
    if not providers_json.startswith("ERROR"):
        providers = json.loads(providers_json)
        print(f"🔍 Найдено провайдеров: {len(providers)}")
        for provider in providers:
            print(f"   - {provider['id']} ({provider['health']})")
            print(f"     Адреса: {provider['addrs']}")
    else:
        print(f"❌ Ошибка поиска: {providers_json}")
    
    # ВАЖНО: Освобождаем память
    owlwhisper.FreeString(providers_ptr)
else:
    print("❌ Провайдеры не найдены")
```

**Что происходит при поиске:**
1. Content ID хешируется для создания DHT ключа
2. DHT сеть ищет провайдеров этого контента
3. Возвращается список всех найденных провайдеров
4. Включает информацию о здоровье и последнем появлении

**Важные замечания:**
- ⚠️ **ВСЕГДА вызывать `FreeString()`** после использования
- ⚠️ **Поиск может занять время** - DHT поиск асинхронный
- 💡 **Используйте для поиска** доступного контента
- 🔍 **Поиск работает глобально** - через всю DHT сеть

## 🔄 **Сценарии использования**

### **Анонсирование доступности файла:**

```python
def announce_file_availability(file_path):
    """Анонсирование доступности файла в сети"""
    try:
        import hashlib
        
        # Создаем Content ID из хеша файла
        with open(file_path, 'rb') as f:
            file_hash = hashlib.sha256(f.read()).hexdigest()
        
        content_id = f"file:{file_hash}".encode('utf-8')
        
        # Анонсируем в сети
        result = owlwhisper.ProvideContent(content_id)
        if result == 0:
            print(f"✅ Файл {file_path} анонсирован в сети")
            print(f"   Content ID: {file_hash}")
            return file_hash
        else:
            print(f"❌ Не удалось анонсировать файл {file_path}")
            return None
            
    except Exception as e:
        print(f"❌ Ошибка анонсирования файла: {e}")
        return None

# Пример использования
file_hash = announce_file_availability("./document.pdf")
if file_hash:
    print(f"Файл доступен по ID: {file_hash}")
```

### **Поиск доступных файлов:**

```python
def find_file_providers(file_hash):
    """Поиск провайдеров файла в сети"""
    try:
        content_id = f"file:{file_hash}".encode('utf-8')
        providers_ptr = owlwhisper.FindProvidersForContent(content_id)
        
        if providers_ptr:
            providers_json = ctypes.string_at(providers_ptr).decode()
            owlwhisper.FreeString(providers_ptr)
            
            if not providers_json.startswith("ERROR"):
                providers = json.loads(providers_json)
                print(f"🔍 Найдено провайдеров файла: {len(providers)}")
                
                # Сортируем по качеству соединения
                sorted_providers = sorted(providers, key=lambda p: p.get('health', 'unknown'))
                
                for provider in sorted_providers:
                    print(f"   - {provider['id']} ({provider['health']})")
                    print(f"     Адреса: {provider['addrs']}")
                
                return sorted_providers
            else:
                print(f"❌ Ошибка поиска: {providers_json}")
                return []
        else:
            print("❌ Провайдеры не найдены")
            return []
            
    except Exception as e:
        print(f"❌ Ошибка поиска файла: {e}")
        return []

# Пример использования
providers = find_file_providers("abc123...")
if providers:
    print(f"Можно скачать файл у {len(providers)} провайдеров")
```

### **Анонсирование сервиса:**

```python
def announce_service(service_name, service_type):
    """Анонсирование доступности сервиса в сети"""
    try:
        import uuid
        
        # Создаем уникальный Content ID для сервиса
        service_id = f"service:{service_type}:{service_name}:{uuid.uuid4().hex[:8]}"
        content_id = service_id.encode('utf-8')
        
        # Анонсируем сервис
        result = owlwhisper.ProvideContent(content_id)
        if result == 0:
            print(f"✅ Сервис {service_name} анонсирован в сети")
            print(f"   Content ID: {service_id}")
            return service_id
        else:
            print(f"❌ Не удалось анонсировать сервис {service_name}")
            return None
            
    except Exception as e:
        print(f"❌ Ошибка анонсирования сервиса: {e}")
        return None

# Пример использования
service_id = announce_service("chat-server", "messaging")
if service_id:
    print(f"Сервис доступен по ID: {service_id}")
```

### **Поиск сервисов:**

```python
def find_services(service_type):
    """Поиск сервисов определенного типа в сети"""
    try:
        # Ищем сервисы по типу
        search_pattern = f"service:{service_type}:".encode('utf-8')
        providers_ptr = owlwhisper.FindProvidersForContent(search_pattern)
        
        if providers_ptr:
            providers_json = ctypes.string_at(providers_ptr).decode()
            owlwhisper.FreeString(providers_ptr)
            
            if not providers_json.startswith("ERROR"):
                providers = json.loads(providers_json)
                print(f"🔍 Найдено сервисов типа {service_type}: {len(providers)}")
                
                for provider in providers:
                    print(f"   - {provider['id']} ({provider['health']})")
                    print(f"     Адреса: {provider['addrs']}")
                
                return providers
            else:
                print(f"❌ Ошибка поиска: {providers_json}")
                return []
        else:
            print("❌ Сервисы не найдены")
            return []
            
    except Exception as e:
        print(f"❌ Ошибка поиска сервисов: {e}")
        return []

# Пример использования
messaging_services = find_services("messaging")
if messaging_services:
    print(f"Доступно {len(messaging_services)} сервисов обмена сообщениями")
```

### **Мониторинг анонсированного контента:**

```python
def monitor_announced_content():
    """Мониторинг анонсированного контента"""
    try:
        # Получаем список подключенных пиров
        peers_ptr = owlwhisper.GetConnectedPeers()
        if peers_ptr:
            peers_json = ctypes.string_at(peers_ptr).decode()
            peers = json.loads(peers_json)
            owlwhisper.FreeString(peers_ptr)
            
            print(f"🌐 Мониторинг анонсированного контента:")
            print(f"   Подключенных пиров: {len(peers)}")
            
            # Проверяем, есть ли пиры, ищущие наш контент
            for peer in peers:
                # Здесь можно добавить логику проверки интереса к нашему контенту
                print(f"   - {peer['id']} ({peer['connection_quality']})")
            
            return len(peers)
        else:
            print("❌ Не удалось получить информацию о пирах")
            return 0
            
    except Exception as e:
        print(f"❌ Ошибка мониторинга: {e}")
        return 0

# Пример использования
peer_count = monitor_announced_content()
print(f"Всего пиров для распространения контента: {peer_count}")
```

## ⚠️ **Важные замечания**

### **Content ID стратегии:**
- **Используйте хеши** для файлов - SHA256, MD5
- **Используйте UUID** для сервисов - уникальная идентификация
- **Используйте префиксы** для категорий - `file:`, `service:`, `user:`
- **Избегайте конфликтов** - длинные и уникальные идентификаторы

### **Производительность:**
- **`ProvideContent()`** - быстрый, локальное анонсирование
- **`FindProvidersForContent()`** - медленный, требует DHT поиска
- **Анонсы распространяются** автоматически по DHT сети
- **Поиск может занять** от нескольких секунд до минут

### **Надежность:**
- **DHT поиск** может не найти всех провайдеров
- **Анонсы могут потеряться** при проблемах с сетью
- **Провайдеры могут быть недоступны** при поиске
- **Используйте fallback** механизмы для критичного контента

### **Безопасность:**
- **Content ID публичен** - не содержит приватной информации
- **Провайдеры видны** всем участникам сети
- **Нет аутентификации** - любой может анонсировать контент
- **Проверяйте контент** перед использованием

### **Ограничения:**
- **Только поиск по Content ID** - нет поиска по содержимому
- **Нет метаданных** - только список провайдеров
- **Нет версионирования** - один Content ID = один контент
- **Нет приоритизации** - все провайдеры равны

## 🔗 **Связанные функции**

- [Поиск пиров](../functions/peer-discovery.md) - поиск конкретных пиров
- [Система событий](../functions/events-system.md) - уведомления о сетевых изменениях
- [Управление Core](../functions/core-management.md) - запуск Core для работы с DHT
- [Утилиты](../functions/utilities.md) - управление памятью для строк

---

**Последнее обновление:** 23 августа 2025  
**Автор:** Core Development Team 