# 🌐 **Мониторинг сети**

**Последнее обновление:** 23 августа 2025

Функции для мониторинга состояния и производительности P2P сети.

## 📋 **Функции**

### **`GetNetworkStats()`**

**Описание:** Возвращает общую статистику сети и соединений.

**Сигнатура:**
```c
char* GetNetworkStats();
```

**Параметры:** Нет

**Возвращает:**
- `char*` - JSON строка со статистикой сети (требует `FreeString()`)
- `NULL` - ошибка получения

**Структура возвращаемых данных:**
```json
{
  "connected_peers": 5,
  "total_connections": 8,
  "network_status": "ready",
  "dht_status": "connected",
  "mdns_status": "active",
  "bootstrap_peers": 3,
  "uptime": 3600,
  "bytes_sent": 1024000,
  "bytes_received": 2048000,
  "last_activity": 1755969188
}
```

**Пример использования:**
```python
import ctypes
import json

# Настройка типа возвращаемого значения
owlwhisper.GetNetworkStats.restype = ctypes.c_char_p

# Получение статистики сети
stats_ptr = owlwhisper.GetNetworkStats()
if stats_ptr:
    stats_json = ctypes.string_at(stats_ptr).decode()
    stats = json.loads(stats_json)
    
    print(f"🌐 Статистика сети:")
    print(f"   Подключенных пиров: {stats.get('connected_peers', 0)}")
    print(f"   Всего соединений: {stats.get('total_connections', 0)}")
    print(f"   Статус сети: {stats.get('network_status', 'unknown')}")
    print(f"   Статус DHT: {stats.get('dht_status', 'unknown')}")
    print(f"   Статус mDNS: {stats.get('mdns_status', 'unknown')}")
    print(f"   Bootstrap пиров: {stats.get('bootstrap_peers', 0)}")
    print(f"   Время работы: {stats.get('uptime', 0)} секунд")
    print(f"   Отправлено: {stats.get('bytes_sent', 0)} байт")
    print(f"   Получено: {stats.get('bytes_received', 0)} байт")
    
    # ВАЖНО: Освобождаем память
    owlwhisper.FreeString(stats_ptr)
else:
    print("❌ Не удалось получить статистику сети")
```

**Что происходит при вызове:**
1. Собирается статистика по всем активным соединениям
2. Проверяется статус DHT и mDNS discovery
3. Подсчитывается трафик и время работы
4. Возвращается JSON с полной статистикой

**Важные замечания:**
- ⚠️ **ВСЕГДА вызывать `FreeString()`** после использования
- 💡 **Используйте для мониторинга** состояния сети
- 🔄 **Статистика обновляется в реальном времени**
- 📊 **Включает все аспекты** сетевой активности

---

### **`GetConnectionQuality(peerID)`**

**Описание:** Возвращает качество соединения с конкретным пиром.

**Сигнатура:**
```c
char* GetConnectionQuality(char* peerID);
```

**Параметры:**
- `peerID` (char*) - Peer ID пира для проверки качества

**Возвращает:**
- `char*` - JSON строка с качеством соединения (требует `FreeString()`)
- `NULL` - пир не найден или ошибка

**Структура возвращаемых данных:**
```json
{
  "peer_id": "12D3KooW...",
  "quality": "excellent",
  "latency": 15,
  "bandwidth": 1000000,
  "packet_loss": 0.01,
  "connection_time": 3600,
  "last_seen": 1755969188,
  "protocols": ["/owlwhisper/1.0.0"],
  "addresses": ["/ip4/192.168.1.100/tcp/1234"]
}
```

**Пример использования:**
```python
import ctypes
import json

# Настройка типа возвращаемого значения
owlwhisper.GetConnectionQuality.restype = ctypes.c_char_p

# Проверка качества соединения с пиром
peer_id = "12D3KooW...".encode('utf-8')
quality_ptr = owlwhisper.GetConnectionQuality(peer_id)
if quality_ptr:
    quality_json = ctypes.string_at(quality_ptr).decode()
    if not quality_json.startswith("ERROR"):
        quality = json.loads(quality_json)
        
        print(f"📊 Качество соединения с {quality['peer_id']}:")
        print(f"   Общее качество: {quality['quality']}")
        print(f"   Задержка: {quality['latency']} мс")
        print(f"   Пропускная способность: {quality['bandwidth']} байт/с")
        print(f"   Потери пакетов: {quality['packet_loss']*100:.2f}%")
        print(f"   Время соединения: {quality['connection_time']} секунд")
    else:
        print(f"❌ Ошибка проверки качества: {quality_json}")
    
    # ВАЖНО: Освобождаем память
    owlwhisper.FreeString(quality_ptr)
else:
    print("❌ Не удалось проверить качество соединения")
```

**Что происходит при вызове:**
1. Проверяется существование соединения с пиром
2. Измеряется задержка и пропускная способность
3. Анализируется качество соединения
4. Возвращается детальная информация о соединении

**Важные замечания:**
- ⚠️ **ВСЕГДА вызывать `FreeString()`** после использования
- 💡 **Используйте для диагностики** проблем с соединением
- 🔍 **Требует активного соединения** с пиром
- 📊 **Качество оценивается** по нескольким параметрам

## 🔄 **Сценарии использования**

### **Мониторинг состояния сети:**

```python
def monitor_network_health():
    """Мониторинг общего состояния сети"""
    try:
        # Получаем статистику сети
        stats_ptr = owlwhisper.GetNetworkStats()
        if stats_ptr:
            stats_json = ctypes.string_at(stats_ptr).decode()
            stats = json.loads(stats_json)
            owlwhisper.FreeString(stats_ptr)
            
            print("🏥 Диагностика здоровья сети:")
            
            # Проверяем основные показатели
            connected_peers = stats.get('connected_peers', 0)
            network_status = stats.get('network_status', 'unknown')
            dht_status = stats.get('dht_status', 'unknown')
            
            if connected_peers > 0:
                print(f"✅ Подключенных пиров: {connected_peers}")
            else:
                print("❌ Нет подключенных пиров")
            
            if network_status == 'ready':
                print("✅ Сеть готова к работе")
            else:
                print(f"⚠️ Статус сети: {network_status}")
            
            if dht_status == 'connected':
                print("✅ DHT подключен")
            else:
                print(f"⚠️ Статус DHT: {dht_status}")
            
            # Анализируем трафик
            bytes_sent = stats.get('bytes_sent', 0)
            bytes_received = stats.get('bytes_received', 0)
            
            if bytes_sent > 0 or bytes_received > 0:
                print(f"📊 Трафик: отправлено {bytes_sent}, получено {bytes_received} байт")
            else:
                print("📊 Нет сетевой активности")
            
            return True
        else:
            print("❌ Не удалось получить статистику сети")
            return False
            
    except Exception as e:
        print(f"❌ Ошибка мониторинга: {e}")
        return False

# Пример использования
monitor_network_health()
```

### **Анализ качества соединений:**

```python
def analyze_connection_quality():
    """Анализ качества всех соединений"""
    try:
        # Получаем список подключенных пиров
        peers_ptr = owlwhisper.GetConnectedPeers()
        if peers_ptr:
            peers_json = ctypes.string_at(peers_ptr).decode()
            peers = json.loads(peers_json)
            owlwhisper.FreeString(peers_ptr)
            
            print("🔍 Анализ качества соединений:")
            
            quality_stats = {
                'excellent': 0,
                'good': 0,
                'poor': 0,
                'bad': 0,
                'unknown': 0
            }
            
            # Анализируем качество каждого соединения
            for peer in peers:
                peer_id = peer['id']
                quality = peer.get('connection_quality', 'unknown')
                quality_stats[quality] = quality_stats.get(quality, 0) + 1
                
                # Получаем детальную информацию о качестве
                quality_ptr = owlwhisper.GetConnectionQuality(peer_id.encode('utf-8'))
                if quality_ptr:
                    quality_json = ctypes.string_at(quality_ptr).decode()
                    if not quality_json.startswith("ERROR"):
                        quality_details = json.loads(quality_json)
                        
                        print(f"   {peer_id}:")
                        print(f"     Качество: {quality_details['quality']}")
                        print(f"     Задержка: {quality_details.get('latency', 'N/A')} мс")
                        print(f"     Пропускная способность: {quality_details.get('bandwidth', 'N/A')} байт/с")
                    
                    owlwhisper.FreeString(quality_ptr)
            
            # Выводим общую статистику
            print("\n📊 Общая статистика качества:")
            for quality, count in quality_stats.items():
                if count > 0:
                    print(f"   {quality}: {count}")
            
            return quality_stats
        else:
            print("❌ Не удалось получить список пиров")
            return {}
            
    except Exception as e:
        print(f"❌ Ошибка анализа качества: {e}")
        return {}

# Пример использования
quality_stats = analyze_connection_quality()
print(f"Всего соединений: {sum(quality_stats.values())}")
```

### **Мониторинг производительности:**

```python
def monitor_network_performance():
    """Мониторинг производительности сети"""
    try:
        import time
        
        print("⚡ Мониторинг производительности сети:")
        
        # Получаем начальную статистику
        start_stats_ptr = owlwhisper.GetNetworkStats()
        if start_stats_ptr:
            start_stats_json = ctypes.string_at(start_stats_ptr).decode()
            start_stats = json.loads(start_stats_json)
            owlwhisper.FreeString(start_stats_ptr)
            
            start_time = time.time()
            start_bytes_sent = start_stats.get('bytes_sent', 0)
            start_bytes_received = start_stats.get('bytes_received', 0)
            
            print(f"📊 Начальные показатели:")
            print(f"   Отправлено: {start_bytes_sent} байт")
            print(f"   Получено: {start_bytes_received} байт")
            
            # Ждем некоторое время
            print("⏳ Ожидание 10 секунд для измерения...")
            time.sleep(10)
            
            # Получаем конечную статистику
            end_stats_ptr = owlwhisper.GetNetworkStats()
            if end_stats_ptr:
                end_stats_json = ctypes.string_at(end_stats_ptr).decode()
                end_stats = json.loads(end_stats_json)
                owlwhisper.FreeString(end_stats_ptr)
                
                end_time = time.time()
                end_bytes_sent = end_stats.get('bytes_sent', 0)
                end_bytes_received = end_stats.get('bytes_received', 0)
                
                # Вычисляем скорость передачи
                duration = end_time - start_time
                bytes_sent_diff = end_bytes_sent - start_bytes_sent
                bytes_received_diff = end_bytes_received - start_bytes_received
                
                sent_speed = bytes_sent_diff / duration if duration > 0 else 0
                received_speed = bytes_received_diff / duration if duration > 0 else 0
                
                print(f"📊 Конечные показатели:")
                print(f"   Отправлено: {end_bytes_sent} байт")
                print(f"   Получено: {end_bytes_received} байт")
                print(f"   Время измерения: {duration:.2f} секунд")
                print(f"   Скорость отправки: {sent_speed:.2f} байт/с")
                print(f"   Скорость получения: {received_speed:.2f} байт/с")
                
                return {
                    'sent_speed': sent_speed,
                    'received_speed': received_speed,
                    'duration': duration
                }
            else:
                print("❌ Не удалось получить конечную статистику")
                return None
        else:
            print("❌ Не удалось получить начальную статистику")
            return None
            
    except Exception as e:
        print(f"❌ Ошибка мониторинга производительности: {e}")
        return None

# Пример использования
performance = monitor_network_performance()
if performance:
    print(f"Средняя скорость: {performance['sent_speed']:.2f} байт/с отправка, {performance['received_speed']:.2f} байт/с получение")
```

### **Автоматическая диагностика проблем:**

```python
def diagnose_network_issues():
    """Автоматическая диагностика сетевых проблем"""
    try:
        print("🔧 Автоматическая диагностика сети:")
        
        issues = []
        
        # Получаем статистику сети
        stats_ptr = owlwhisper.GetNetworkStats()
        if stats_ptr:
            stats_json = ctypes.string_at(stats_ptr).decode()
            stats = json.loads(stats_json)
            owlwhisper.FreeString(stats_ptr)
            
            # Проверяем подключенных пиров
            connected_peers = stats.get('connected_peers', 0)
            if connected_peers == 0:
                issues.append("Нет подключенных пиров")
            elif connected_peers < 3:
                issues.append(f"Мало подключенных пиров: {connected_peers}")
            
            # Проверяем статус сети
            network_status = stats.get('network_status', 'unknown')
            if network_status != 'ready':
                issues.append(f"Сеть не готова: {network_status}")
            
            # Проверяем статус DHT
            dht_status = stats.get('dht_status', 'unknown')
            if dht_status != 'connected':
                issues.append(f"DHT не подключен: {dht_status}")
            
            # Проверяем bootstrap пиров
            bootstrap_peers = stats.get('bootstrap_peers', 0)
            if bootstrap_peers == 0:
                issues.append("Нет bootstrap пиров")
            elif bootstrap_peers < 2:
                issues.append(f"Мало bootstrap пиров: {bootstrap_peers}")
            
            # Проверяем время работы
            uptime = stats.get('uptime', 0)
            if uptime < 60:
                issues.append(f"Сеть работает мало времени: {uptime} секунд")
            
            # Выводим результаты диагностики
            if issues:
                print("❌ Обнаружены проблемы:")
                for issue in issues:
                    print(f"   - {issue}")
                
                print("\n💡 Рекомендации:")
                if "Нет подключенных пиров" in issues:
                    print("   - Проверьте подключение к интернету")
                    print("   - Убедитесь, что Core запущен")
                if "DHT не подключен" in issues:
                    print("   - Подождите подключения к DHT")
                    print("   - Проверьте firewall настройки")
                if "Мало bootstrap пиров" in issues:
                    print("   - Проверьте доступность bootstrap узлов")
                    print("   - Используйте fallback механизм")
            else:
                print("✅ Проблем не обнаружено")
            
            return issues
        else:
            print("❌ Не удалось получить статистику для диагностики")
            return ["Не удалось получить статистику сети"]
            
    except Exception as e:
        print(f"❌ Ошибка диагностики: {e}")
        return ["Ошибка диагностики"]

# Пример использования
network_issues = diagnose_network_issues()
if network_issues:
    print(f"Найдено {len(network_issues)} проблем с сетью")
```

## ⚠️ **Важные замечания**

### **Метрики качества соединений:**
- **excellent** - задержка < 50мс, потери < 1%
- **good** - задержка 50-100мс, потери 1-5%
- **poor** - задержка 100-200мс, потери 5-10%
- **bad** - задержка > 200мс, потери > 10%

### **Статусы сети:**
- **ready** - сеть готова к работе
- **connecting** - подключение к bootstrap узлам
- **bootstrap_failed** - не удалось подключиться к bootstrap
- **fallback** - используется fallback механизм

### **Статусы DHT:**
- **connected** - DHT подключен и работает
- **connecting** - подключение к DHT
- **disconnected** - DHT отключен
- **error** - ошибка DHT

### **Производительность:**
- **`GetNetworkStats()`** - быстрый, локальные данные
- **`GetConnectionQuality()`** - медленный, требует измерения
- **Статистика обновляется** в реальном времени
- **Измерения качества** могут занять время

### **Надежность:**
- **Статистика точна** для активных соединений
- **Качество соединения** может меняться со временем
- **DHT статус** может быть нестабильным
- **Используйте усреднение** для долгосрочного мониторинга

### **Ограничения:**
- **Только активные соединения** - не показывает разорванные
- **Качество оценивается** по текущему состоянию
- **Нет исторических данных** - только текущие значения
- **Нет предсказания** будущих проблем

## 🔗 **Связанные функции**

- [Поиск пиров](../functions/peer-discovery.md) - получение списка пиров для анализа
- [Управление соединениями](../functions/connection-management.md) - управление качеством соединений
- [Система событий](../functions/events-system.md) - уведомления о сетевых изменениях
- [Утилиты](../functions/utilities.md) - управление памятью для строк

---

**Последнее обновление:** 23 августа 2025  
**Автор:** Core Development Team 