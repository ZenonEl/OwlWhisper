# 🔗 **Управление соединениями**

**Последнее обновление:** 23 августа 2025

Функции для управления защищенными соединениями и лимитами сети.

## 📋 **Функции**

### **`AddProtectedPeer(peerID)`**

**Описание:** Добавляет пира в список защищенных соединений.

**Сигнатура:**
```c
int AddProtectedPeer(char* peerID);
```

**Параметры:**
- `peerID` (char*) - Peer ID пира для добавления в защищенные

**Возвращает:**
- `0` - успешное добавление
- `-1` - ошибка добавления

**Пример использования:**
```python
import ctypes

# Настройка типа возвращаемого значения
owlwhisper.AddProtectedPeer.restype = ctypes.c_int

# Добавляем пира в защищенные соединения
peer_id = "12D3KooW...".encode('utf-8')
result = owlwhisper.AddProtectedPeer(peer_id)
if result == 0:
    print("✅ Пир добавлен в защищенные соединения")
else:
    print("❌ Не удалось добавить пира в защищенные")
```

**Что происходит при добавлении:**
1. Пир добавляется в список защищенных соединений
2. ConnectionManager автоматически управляет переподключением
3. Соединение защищено от автоматического закрытия
4. При разрыве соединения автоматически восстанавливается

**Важные замечания:**
- ⚠️ **Защищенные пиры не закрываются** автоматически
- 💡 **Используйте для важных соединений** - друзья, серверы
- 🔄 **Автоматическое переподключение** при разрыве
- 📍 **Лимит защищенных соединений** - обычно 10-20 пиров

---

### **`RemoveProtectedPeer(peerID)`**

**Описание:** Удаляет пира из списка защищенных соединений.

**Сигнатура:**
```c
int RemoveProtectedPeer(char* peerID);
```

**Параметры:**
- `peerID` (char*) - Peer ID пира для удаления из защищенных

**Возвращает:**
- `0` - успешное удаление
- `-1` - ошибка удаления

**Пример использования:**
```python
import ctypes

# Настройка типа возвращаемого значения
owlwhisper.RemoveProtectedPeer.restype = ctypes.c_int

# Удаляем пира из защищенных соединений
peer_id = "12D3KooW...".encode('utf-8')
result = owlwhisper.RemoveProtectedPeer(peer_id)
if result == 0:
    print("✅ Пир удален из защищенных соединений")
else:
    print("❌ Не удалось удалить пира из защищенных")
```

**Что происходит при удалении:**
1. Пир удаляется из списка защищенных соединений
2. Соединение может быть автоматически закрыто
3. Автоматическое переподключение прекращается
4. Пир становится обычным (незащищенным)

**Важные замечания:**
- ⚠️ **Соединение может разорваться** после удаления
- 💡 **Используйте для управления** важностью соединений
- 🔄 **Переподключение прекращается** автоматически
- 📍 **Пир остается в сети** - только меняется статус защиты

---

### **`GetProtectedPeers()`**

**Описание:** Возвращает список защищенных пиров.

**Сигнатура:**
```c
char* GetProtectedPeers();
```

**Параметры:** Нет

**Возвращает:**
- `char*` - JSON строка со списком защищенных пиров (требует `FreeString()`)
- `NULL` - ошибка получения

**Структура возвращаемых данных:**
```json
[
  {
    "id": "12D3KooW...",
    "addrs": ["/ip4/192.168.1.100/tcp/1234"],
    "protocols": ["/owlwhisper/1.0.0"],
    "connection_quality": "excellent",
    "protected_since": 1755969188
  },
  {
    "id": "12D3KooW...",
    "addrs": ["/ip6/::1/tcp/1234"],
    "protocols": ["/owlwhisper/1.0.0"],
    "connection_quality": "good",
    "protected_since": 1755969189
  }
]
```

**Пример использования:**
```python
import ctypes
import json

# Настройка типа возвращаемого значения
owlwhisper.GetProtectedPeers.restype = ctypes.c_char_p

# Получение списка защищенных пиров
protected_peers_ptr = owlwhisper.GetProtectedPeers()
if protected_peers_ptr:
    protected_peers_json = ctypes.string_at(protected_peers_ptr).decode()
    protected_peers = json.loads(protected_peers_json)
    
    print(f"🛡️ Защищенные пиры: {len(protected_peers)}")
    for peer in protected_peers:
        print(f"   - {peer['id']} ({peer['connection_quality']})")
        print(f"     Защищен с: {peer['protected_since']}")
    
    # ВАЖНО: Освобождаем память
    owlwhisper.FreeString(protected_peers_ptr)
else:
    print("❌ Не удалось получить список защищенных пиров")
```

**Что происходит при вызове:**
1. Извлекается список защищенных соединений
2. Для каждого пира собирается расширенная информация
3. Включается время добавления в защищенные
4. Возвращается JSON массив с данными

**Важные замечания:**
- ⚠️ **ВСЕГДА вызывать `FreeString()`** после использования
- 💡 **Используйте для мониторинга** важных соединений
- 🔄 **Список обновляется в реальном времени** - при добавлении/удалении
- 📊 **Включает метаданные** о защищенных соединениях

## 🔄 **Сценарии использования**

### **Управление списком друзей:**

```python
def manage_friends_list():
    """Управление списком друзей как защищенных пиров"""
    try:
        # Список друзей (в реальном приложении загружается из базы)
        friends = ["12D3KooW...", "12D3KooW...", "12D3KooW..."]
        
        print("👥 Управление списком друзей:")
        
        # Добавляем всех друзей в защищенные
        for friend_id in friends:
            peer_id = friend_id.encode('utf-8')
            result = owlwhisper.AddProtectedPeer(peer_id)
            
            if result == 0:
                print(f"✅ {friend_id} добавлен в защищенные")
            else:
                print(f"❌ Не удалось добавить {friend_id}")
        
        # Получаем текущий список защищенных
        protected_peers_ptr = owlwhisper.GetProtectedPeers()
        if protected_peers_ptr:
            protected_peers_json = ctypes.string_at(protected_peers_ptr).decode()
            protected_peers = json.loads(protected_peers_json)
            owlwhisper.FreeString(protected_peers_ptr)
            
            print(f"🛡️ Всего защищенных пиров: {len(protected_peers)}")
            
        return True
        
    except Exception as e:
        print(f"❌ Ошибка управления друзьями: {e}")
        return False

# Пример использования
manage_friends_list()
```

### **Автоматическое управление важными соединениями:**

```python
def auto_manage_important_connections():
    """Автоматическое управление важными соединениями"""
    try:
        # Получаем список подключенных пиров
        connected_peers_ptr = owlwhisper.GetConnectedPeers()
        if connected_peers_ptr:
            connected_peers_json = ctypes.string_at(connected_peers_ptr).decode()
            connected_peers = json.loads(connected_peers_json)
            owlwhisper.FreeString(connected_peers_ptr)
            
            # Получаем список защищенных пиров
            protected_peers_ptr = owlwhisper.GetProtectedPeers()
            if protected_peers_ptr:
                protected_peers_json = ctypes.string_at(protected_peers_ptr).decode()
                protected_peers = json.loads(protected_peers_json)
                owlwhisper.FreeString(protected_peers_ptr)
                
                protected_ids = {peer['id'] for peer in protected_peers}
                
                print("🔍 Анализ важных соединений:")
                
                # Анализируем качество соединений
                for peer in connected_peers:
                    peer_id = peer['id']
                    quality = peer.get('connection_quality', 'unknown')
                    
                    if quality in ['excellent', 'good'] and peer_id not in protected_ids:
                        print(f"💡 {peer_id} имеет хорошее соединение - добавляем в защищенные")
                        
                        # Добавляем в защищенные
                        peer_id_bytes = peer_id.encode('utf-8')
                        result = owlwhisper.AddProtectedPeer(peer_id_bytes)
                        
                        if result == 0:
                            print(f"✅ {peer_id} добавлен в защищенные")
                        else:
                            print(f"❌ Не удалось добавить {peer_id}")
                    
                    elif quality in ['poor', 'bad'] and peer_id in protected_ids:
                        print(f"⚠️ {peer_id} имеет плохое соединение - убираем из защищенных")
                        
                        # Убираем из защищенных
                        peer_id_bytes = peer_id.encode('utf-8')
                        result = owlwhisper.RemoveProtectedPeer(peer_id_bytes)
                        
                        if result == 0:
                            print(f"✅ {peer_id} убран из защищенных")
                        else:
                            print(f"❌ Не удалось убрать {peer_id}")
                
                return True
            else:
                print("❌ Не удалось получить защищенных пиров")
                return False
        else:
            print("❌ Не удалось получить подключенных пиров")
            return False
            
    except Exception as e:
        print(f"❌ Ошибка автоматического управления: {e}")
        return False

# Пример использования
auto_manage_important_connections()
```

### **Мониторинг защищенных соединений:**

```python
def monitor_protected_connections():
    """Мониторинг защищенных соединений"""
    try:
        # Получаем список защищенных пиров
        protected_peers_ptr = owlwhisper.GetProtectedPeers()
        if protected_peers_ptr:
            protected_peers_json = ctypes.string_at(protected_peers_ptr).decode()
            protected_peers = json.loads(protected_peers_json)
            owlwhisper.FreeString(protected_peers_ptr)
            
            print(f"🛡️ Мониторинг защищенных соединений:")
            print(f"   Всего защищенных: {len(protected_peers)}")
            
            # Анализируем качество защищенных соединений
            quality_stats = {}
            for peer in protected_peers:
                quality = peer.get('connection_quality', 'unknown')
                quality_stats[quality] = quality_stats.get(quality, 0) + 1
            
            print("📊 Качество защищенных соединений:")
            for quality, count in quality_stats.items():
                print(f"   {quality}: {count}")
            
            # Проверяем время защиты
            current_time = int(time.time())
            for peer in protected_peers:
                protected_since = peer.get('protected_since', 0)
                protection_duration = current_time - protected_since
                
                if protection_duration > 3600:  # Более часа
                    print(f"⏰ {peer['id']} защищен {protection_duration//3600} часов")
            
            return len(protected_peers), quality_stats
            
        else:
            print("❌ Не удалось получить защищенных пиров")
            return 0, {}
            
    except Exception as e:
        print(f"❌ Ошибка мониторинга: {e}")
        return 0, {}

# Пример использования
protected_count, quality_stats = monitor_protected_connections()
print(f"Всего защищенных соединений: {protected_count}")
```

### **Управление серверными соединениями:**

```python
def manage_server_connections():
    """Управление соединениями с серверами"""
    try:
        # Список важных серверов
        servers = [
            {"id": "12D3KooW...", "name": "Main Server", "type": "core"},
            {"id": "12D3KooW...", "name": "Backup Server", "type": "backup"},
            {"id": "12D3KooW...", "name": "File Server", "type": "storage"}
        ]
        
        print("🖥️ Управление серверными соединениями:")
        
        # Добавляем все серверы в защищенные
        for server in servers:
            peer_id = server['id'].encode('utf-8')
            result = owlwhisper.AddProtectedPeer(peer_id)
            
            if result == 0:
                print(f"✅ {server['name']} ({server['type']}) добавлен в защищенные")
            else:
                print(f"❌ Не удалось добавить {server['name']}")
        
        # Проверяем статус
        protected_peers_ptr = owlwhisper.GetProtectedPeers()
        if protected_peers_ptr:
            protected_peers_json = ctypes.string_at(protected_peers_ptr).decode()
            protected_peers = json.loads(protected_peers_json)
            owlwhisper.FreeString(protected_peers_ptr)
            
            print(f"🛡️ Серверов в защищенных: {len(protected_peers)}")
            
            # Проверяем качество серверных соединений
            for peer in protected_peers:
                if peer['id'] in [s['id'] for s in servers]:
                    server_info = next(s for s in servers if s['id'] == peer['id'])
                    quality = peer.get('connection_quality', 'unknown')
                    print(f"   {server_info['name']}: {quality}")
        
        return True
        
    except Exception as e:
        print(f"❌ Ошибка управления серверами: {e}")
        return False

# Пример использования
manage_server_connections()
```

## ⚠️ **Важные замечания**

### **Лимиты соединений:**
- **Защищенные соединения:** обычно 10-20 пиров
- **Инфраструктурные соединения:** обычно 5-10 пиров
- **Общий лимит:** зависит от конфигурации libp2p
- **Автоматическое управление** - Core сам закрывает лишние

### **Качество соединений:**
- **excellent** - отличное соединение, стабильное
- **good** - хорошее соединение, стабильное
- **poor** - плохое соединение, нестабильное
- **bad** - очень плохое соединение, часто разрывается

### **Автоматическое поведение:**
- **Защищенные пиры** автоматически переподключаются
- **Обычные пиры** могут быть закрыты при нехватке ресурсов
- **ConnectionManager** сам управляет лимитами
- **Fallback механизм** использует кэшированные адреса

### **Производительность:**
- **`AddProtectedPeer()`** - мгновенный, локальная операция
- **`RemoveProtectedPeer()`** - мгновенный, локальная операция
- **`GetProtectedPeers()`** - быстрый, локальные данные
- **Автоматическое управление** не влияет на производительность

### **Надежность:**
- **Защищенные пиры** не закрываются автоматически
- **При разрыве** автоматически восстанавливается соединение
- **Fallback механизм** использует кэшированные адреса
- **Graceful degradation** при проблемах с сетью

### **Ограничения:**
- **Только Peer ID** - нет управления по адресу
- **Нет приоритизации** - все защищенные равны
- **Нет группировки** - простой список
- **Нет метаданных** - только статус защиты

## 🔗 **Связанные функции**

- [Поиск пиров](../functions/peer-discovery.md) - поиск пиров для добавления в защищенные
- [Система событий](../functions/events-system.md) - уведомления о подключении/отключении
- [Мониторинг сети](../functions/network-monitoring.md) - проверка качества соединений
- [Утилиты](../functions/utilities.md) - управление памятью для строк

---

**Последнее обновление:** 23 августа 2025  
**Автор:** Core Development Team 