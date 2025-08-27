# 🔍 **Поиск пиров**

**Последнее обновление:** 23 августа 2025

Функции для поиска и обнаружения пиров в P2P сети.

## 📋 **Функции**

### **`FindPeer(peerID)`**

**Описание:** Ищет пира в сети по его Peer ID.

**Сигнатура:**
```c
char* FindPeer(char* peerID);
```

**Параметры:**
- `peerID` (char*) - Peer ID искомого пира

**Возвращает:**
- `char*` - JSON строка с информацией о пире (требует `FreeString()`)
- `NULL` - пир не найден или ошибка

**Структура возвращаемых данных:**
```json
{
  "id": "12D3KooW...",
  "addrs": ["/ip4/192.168.1.100/tcp/1234", "/ip6/::1/tcp/1234"],
  "protocols": ["/owlwhisper/1.0.0"],
  "agent_version": "owlwhisper/1.4.0"
}
```

**Пример использования:**
```python
import ctypes
import json

# Настройка типа возвращаемого значения
owlwhisper.FindPeer.restype = ctypes.c_char_p

# Поиск пира в сети
peer_id = "12D3KooW...".encode('utf-8')
peer_info_ptr = owlwhisper.FindPeer(peer_id)
if peer_info_ptr:
    peer_info_json = ctypes.string_at(peer_info_ptr).decode()
    if not peer_info_json.startswith("ERROR"):
        peer_info = json.loads(peer_info_json)
        print(f"✅ Пир найден: {peer_info['id']}")
        print(f"   Адреса: {peer_info['addrs']}")
        print(f"   Протоколы: {peer_info['protocols']}")
    else:
        print(f"❌ Ошибка поиска: {peer_info_json}")
    
    # ВАЖНО: Освобождаем память
    owlwhisper.FreeString(peer_info_ptr)
else:
    print("❌ Пир не найден")
```

**Что происходит при поиске:**
1. Запрос отправляется в DHT сеть
2. DHT ищет пира по его Peer ID
3. Возвращается информация о найденном пире
4. Включает адреса, протоколы и версию агента

**Важные замечания:**
- ⚠️ **ВСЕГДА вызывать `FreeString()`** после использования
- ⚠️ **Поиск может занять время** - DHT поиск асинхронный
- 💡 **Используйте для проверки** существования пира в сети
- 🔍 **Поиск работает глобально** - через всю DHT сеть

---

### **`GetMyPeerID()`**

**Описание:** Возвращает Peer ID текущего узла.

**Сигнатура:**
```c
char* GetMyPeerID();
```

**Параметры:** Нет

**Возвращает:**
- `char*` - строка с Peer ID (требует `FreeString()`)
- `NULL` - ошибка получения

**Пример использования:**
```python
import ctypes

# Настройка типа возвращаемого значения
owlwhisper.GetMyPeerID.restype = ctypes.c_char_p

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

**Что происходит при вызове:**
1. Извлекается Peer ID из libp2p узла
2. Возвращается в виде строки
3. Peer ID уникален для каждого узла

**Важные замечания:**
- ⚠️ **ВСЕГДА вызывать `FreeString()`** после использования
- 💡 **Используйте для идентификации** себя в сети
- 🔑 **Peer ID генерируется** из приватного ключа

---

### **`GetConnectedPeers()`**

**Описание:** Возвращает список всех подключенных пиров.

**Сигнатура:**
```c
char* GetConnectedPeers();
```

**Параметры:** Нет

**Возвращает:**
- `char*` - JSON строка со списком пиров (требует `FreeString()`)
- `NULL` - ошибка получения

**Структура возвращаемых данных:**
```json
[
  {
    "id": "12D3KooW...",
    "addrs": ["/ip4/192.168.1.100/tcp/1234"],
    "protocols": ["/owlwhisper/1.0.0"],
    "connection_quality": "good"
  },
  {
    "id": "12D3KooW...",
    "addrs": ["/ip6/::1/tcp/1234"],
    "protocols": ["/owlwhisper/1.0.0"],
    "connection_quality": "excellent"
  }
]
```

**Пример использования:**
```python
import ctypes
import json

# Настройка типа возвращаемого значения
owlwhisper.GetConnectedPeers.restype = ctypes.c_char_p

# Получение списка подключенных пиров
peers_ptr = owlwhisper.GetConnectedPeers()
if peers_ptr:
    peers_json = ctypes.string_at(peers_ptr).decode()
    peers = json.loads(peers_json)
    
    print(f"🔗 Подключенные пиры: {len(peers)}")
    for peer in peers:
        print(f"   - {peer['id']} ({peer['connection_quality']})")
        print(f"     Адреса: {peer['addrs']}")
    
    # ВАЖНО: Освобождаем память
    owlwhisper.FreeString(peers_ptr)
else:
    print("❌ Не удалось получить список пиров")
```

**Что происходит при вызове:**
1. Извлекается список активных соединений
2. Для каждого пира собирается информация
3. Возвращается JSON массив с данными

**Важные замечания:**
- ⚠️ **ВСЕГДА вызывать `FreeString()`** после использования
- 💡 **Используйте для мониторинга** активных соединений
- 🔄 **Список обновляется в реальном времени** - при подключении/отключении

## 🔄 **Сценарии использования**

### **Поиск конкретного пира:**

```python
def find_specific_peer(peer_id):
    """Поиск конкретного пира в сети"""
    try:
        peer_id_bytes = peer_id.encode('utf-8')
        peer_info_ptr = owlwhisper.FindPeer(peer_id_bytes)
        
        if peer_info_ptr:
            peer_info_json = ctypes.string_at(peer_info_ptr).decode()
            owlwhisper.FreeString(peer_info_ptr)
            
            if not peer_info_json.startswith("ERROR"):
                peer_info = json.loads(peer_info_json)
                print(f"✅ Пир найден: {peer_info['id']}")
                return peer_info
            else:
                print(f"❌ Ошибка поиска: {peer_info_json}")
                return None
        else:
            print("❌ Пир не найден")
            return None
            
    except Exception as e:
        print(f"❌ Исключение при поиске: {e}")
        return None

# Пример использования
peer_info = find_specific_peer("12D3KooW...")
if peer_info:
    print(f"Адреса: {peer_info['addrs']}")
```

### **Проверка онлайн статуса:**

```python
def check_peer_online_status(peer_id):
    """Проверка онлайн статуса пира"""
    try:
        # Сначала ищем в подключенных пирах
        connected_peers_ptr = owlwhisper.GetConnectedPeers()
        if connected_peers_ptr:
            peers_json = ctypes.string_at(connected_peers_ptr).decode()
            connected_peers = json.loads(peers_json)
            owlwhisper.FreeString(connected_peers_ptr)
            
            # Проверяем, есть ли пир в списке подключенных
            for peer in connected_peers:
                if peer['id'] == peer_id:
                    return "online", peer['connection_quality']
        
        # Если не найден в подключенных, ищем в сети
        peer_info = find_specific_peer(peer_id)
        if peer_info:
            return "available", "unknown"
        else:
            return "offline", "unknown"
            
    except Exception as e:
        print(f"❌ Ошибка проверки статуса: {e}")
        return "error", "unknown"

# Пример использования
status, quality = check_peer_online_status("12D3KooW...")
print(f"Статус: {status}, Качество: {quality}")
```

### **Мониторинг сети:**

```python
def monitor_network_peers():
    """Мониторинг количества пиров в сети"""
    try:
        # Получаем список подключенных пиров
        peers_ptr = owlwhisper.GetConnectedPeers()
        if peers_ptr:
            peers_json = ctypes.string_at(peers_ptr).decode()
            peers = json.loads(peers_json)
            owlwhisper.FreeString(peers_ptr)
            
            print(f"🌐 Сеть: {len(peers)} активных пиров")
            
            # Анализируем качество соединений
            quality_stats = {}
            for peer in peers:
                quality = peer.get('connection_quality', 'unknown')
                quality_stats[quality] = quality_stats.get(quality, 0) + 1
            
            print("📊 Качество соединений:")
            for quality, count in quality_stats.items():
                print(f"   {quality}: {count}")
            
            return len(peers), quality_stats
            
        else:
            print("❌ Не удалось получить информацию о пирах")
            return 0, {}
            
    except Exception as e:
        print(f"❌ Ошибка мониторинга: {e}")
        return 0, {}

# Пример использования
peer_count, quality_stats = monitor_network_peers()
print(f"Всего пиров: {peer_count}")
```

### **Поиск пиров по протоколу:**

```python
def find_peers_by_protocol(protocol):
    """Поиск пиров, поддерживающих определенный протокол"""
    try:
        peers_ptr = owlwhisper.GetConnectedPeers()
        if peers_ptr:
            peers_json = ctypes.string_at(peers_ptr).decode()
            peers = json.loads(peers_json)
            owlwhisper.FreeString(peers_ptr)
            
            # Фильтруем пиров по протоколу
            protocol_peers = []
            for peer in peers:
                if protocol in peer.get('protocols', []):
                    protocol_peers.append(peer)
            
            print(f"🔍 Пиры с протоколом {protocol}: {len(protocol_peers)}")
            for peer in protocol_peers:
                print(f"   - {peer['id']}")
            
            return protocol_peers
            
        else:
            print("❌ Не удалось получить список пиров")
            return []
            
    except Exception as e:
        print(f"❌ Ошибка поиска по протоколу: {e}")
        return []

# Пример использования
owlwhisper_peers = find_peers_by_protocol("/owlwhisper/1.0.0")
print(f"Найдено OwlWhisper пиров: {len(owlwhisper_peers)}")
```

## ⚠️ **Важные замечания**

### **Производительность:**
- **`GetConnectedPeers()`** - быстрый, возвращает локальные данные
- **`FindPeer()`** - медленный, требует DHT поиска
- **`GetMyPeerID()`** - мгновенный, локальные данные

### **Надежность:**
- **DHT поиск** может не найти пира, даже если он онлайн
- **Список подключенных** актуален только для прямых соединений
- **Peer ID уникален** - не может быть дубликатов

### **Ограничения:**
- **Поиск только по Peer ID** - нет поиска по адресу или имени
- **Только активные соединения** - не показывает всех известных пиров
- **Нет геолокации** - адреса IP без географической информации

### **Безопасность:**
- **Peer ID публичен** - не содержит приватной информации
- **Адреса могут быть приватными** - NAT, VPN, Tor
- **Протоколы открыты** - любой может увидеть поддерживаемые протоколы

## 🔗 **Связанные функции**

- [Управление Core](../functions/core-management.md) - запуск Core для поиска пиров
- [Система событий](../functions/events-system.md) - уведомления о подключении/отключении пиров
- [Управление соединениями](../functions/connection-management.md) - управление качеством соединений
- [Утилиты](../functions/utilities.md) - управление памятью для строк

---

**Последнее обновление:** 23 августа 2025  
**Автор:** Core Development Team 