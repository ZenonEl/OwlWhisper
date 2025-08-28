# 🔗 **Подключение к пирам**

**Файл:** `docs/functions/peer-connection.md`  
**Версия:** v1.5  
**Последнее обновление:** 28 августа 2025

## 🎯 **Обзор**

Функции подключения к пирам обеспечивают установление соединений с найденными пирами. Включают в себя автоматическое управление relay, hole punching и fallback механизмы.

## 📋 **Доступные функции**

### **1. Connect**
```c
int Connect(char* peerID, char* addrs);
```

**Описание:** Подключается к пиру по Peer ID и адресам.

**Параметры:**
- `peerID` - Peer ID пира для подключения
- `addrs` - JSON массив адресов пира

**Возвращает:**
- `0` - успех
- `-1` - ошибка

**Формат адресов:**
```json
[
  "/ip4/192.168.1.100/tcp/1234",
  "/ip4/10.0.0.1/tcp/5678",
  "/dns4/relay.example.com/tcp/443/wss/p2p/12D3KooW..."
]
```

**Пример использования:**
```c
char* addrs = "[\"/ip4/192.168.1.100/tcp/1234\"]";
int result = Connect("12D3KooW...", addrs);
if (result == 0) {
    printf("✅ Успешно подключились к пиру\n");
} else {
    printf("❌ Ошибка подключения\n");
}
```

**Логика работы:**
1. **Проверка** - не подключены ли уже
2. **Парсинг** адресов из JSON
3. **Создание** AddrInfo структуры
4. **Подключение** через libp2p
5. **Автоматический** fallback через relay

---

### **2. SetupAutoRelayWithDHT**
```c
int SetupAutoRelayWithDHT();
```

**Описание:** Настраивает автоматическое использование relay через DHT routing table.

**Параметры:** Нет

**Возвращает:**
- `0` - успех
- `-1` - ошибка

**Пример использования:**
```c
int result = SetupAutoRelayWithDHT();
if (result == 0) {
    printf("✅ AutoRelay с DHT настроен\n");
} else {
    printf("❌ Ошибка настройки AutoRelay\n");
}
```

**Логика работы:**
1. **Получение** DHT из discovery manager
2. **Настройка** peer source из routing table
3. **Включение** EnableAutoRelayWithPeerSource
4. **Конфигурация** boot delay и max candidates

---

### **3. ConnectToPeer (существующая)**
```c
int ConnectToPeer(char* peerID);
```

**Описание:** Подключается к пиру по Peer ID (использует FindPeer для поиска адресов).

**Параметры:**
- `peerID` - Peer ID пира для подключения

**Возвращает:**
- `0` - успех
- `-1` - ошибка

**Пример использования:**
```c
int result = ConnectToPeer("12D3KooW...");
if (result == 0) {
    printf("✅ Успешно подключились к пиру\n");
} else {
    printf("❌ Ошибка подключения\n");
}
```

## 🔄 **Логика подключения**

### **Этапы подключения:**
1. **Поиск адресов** - через DHT или кэш
2. **Проверка соединения** - не подключены ли уже
3. **Попытка прямого подключения** - через hole punching
4. **Fallback через relay** - если прямое не удается
5. **Установка соединения** - создание network.Conn

### **NAT Traversal:**
- **Hole Punching** - автоматический обход NAT
- **AutoNATv2** - определение типа NAT
- **Relay** - fallback для сложных NAT
- **AutoRelay** - автоматический поиск relay

## ⚡ **Преимущества новых функций**

### **Connect() vs ConnectToPeer():**
- **Connect()** - более гибкий, принимает адреса
- **ConnectToPeer()** - проще в использовании
- **Оба** поддерживают автоматический fallback

### **SetupAutoRelayWithDHT():**
- **Интеграция** с существующей DHT инфраструктурой
- **Автоматический** peer source из routing table
- **Оптимизация** для конкретной сети

## 🚨 **Важные замечания**

### **Управление памятью:**
- **Connect()** не освобождает переданные строки
- **ConnectToPeer()** не освобождает переданные строки
- Строки должны быть валидными до завершения функции

### **Производительность:**
- **Connect()** быстрее (адреса уже известны)
- **ConnectToPeer()** медленнее (нужно искать адреса)
- **SetupAutoRelayWithDHT()** выполняется один раз при инициализации

### **Сетевой трафик:**
- **Hole punching** - минимальный трафик
- **Relay** - дополнительный трафик через relay
- **DHT lookup** - только для ConnectToPeer()

## 🔧 **Интеграция с другими функциями**

### **Discovery:**
- **FindPeersOnce()** возвращает адреса для Connect()
- **StartAggressiveDiscovery()** автоматически использует Connect()
- **FindProvidersForContent()** возвращает адреса для Connect()

### **События:**
- **PeerConnected** генерируется при успешном подключении
- **PeerDisconnected** генерируется при отключении
- Используйте **GetNextEvent()** для мониторинга

### **Кэширование:**
- **SavePeerToCache()** сохраняет успешные подключения
- **LoadPeerFromCache()** загружает кэшированные адреса
- **Connect()** может использовать кэшированные адреса

## 📊 **Примеры использования**

### **Подключение к найденному пиру:**
```c
// Ищем пира
char* providers = FindProvidersForContent("content-id");
if (providers != NULL) {
    // Парсим JSON и подключаемся
    // ... парсинг JSON ...
    int result = Connect(peerID, addrs);
    if (result == 0) {
        printf("✅ Подключились к %s\n", peerID);
    }
    FreeString(providers);
}
```

### **Настройка AutoRelay при инициализации:**
```c
// Запускаем Core
StartOwlWhisper();

// Настраиваем AutoRelay
SetupAutoRelayWithDHT();

// Теперь все подключения будут использовать relay при необходимости
```

### **Комбинированное использование:**
```c
// Настраиваем AutoRelay
SetupAutoRelayWithDHT();

// Запускаем агрессивный поиск
StartAggressiveDiscovery("rendezvous");

// Обрабатываем события подключения
while (1) {
    char* event = GetNextEvent();
    if (event != NULL) {
        if (strstr(event, "PeerConnected") != NULL) {
            printf("🟢 Новое подключение!\n");
        }
        FreeString(event);
    }
    sleep(1);
}
```

## 🎯 **Рекомендации**

1. **Используйте Connect()** когда адреса уже известны
2. **Используйте ConnectToPeer()** для простых случаев
3. **Настройте SetupAutoRelayWithDHT()** при инициализации
4. **Мониторьте события** через GetNextEvent()
5. **Кэшируйте успешные подключения** для повторного использования

## 🔍 **Отладка подключений**

### **Проверка статуса:**
```c
// Проверяем подключение
char* status = GetConnectionStatus();
if (status != NULL) {
    printf("Статус: %s\n", status);
    FreeString(status);
}
```

### **Качество соединения:**
```c
// Получаем качество соединения
char* quality = GetConnectionQuality("peer-id");
if (quality != NULL) {
    printf("Качество: %s\n", quality);
    FreeString(quality);
}
```

### **Статистика сети:**
```c
// Получаем статистику сети
char* stats = GetNetworkStats();
if (stats != NULL) {
    printf("Статистика: %s\n", stats);
    FreeString(stats);
}
```

---

**Последнее обновление:** 28 августа 2025  
**Автор:** Core Development Team
