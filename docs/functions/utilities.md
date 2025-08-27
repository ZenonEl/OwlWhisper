# 🛠️ **Утилиты**

**Последнее обновление:** 23 августа 2025

Утилитарные функции для управления памятью, логированием и вспомогательными операциями.

## 📋 **Функции**

### **`FreeString(stringPtr)`**

**Описание:** Освобождает память, выделенную для строки, возвращенной Core.

**Сигнатура:**
```c
void FreeString(char* stringPtr);
```

**Параметры:**
- `stringPtr` (char*) - указатель на строку для освобождения

**Возвращает:** Нет

**Пример использования:**
```python
import ctypes

# Настройка типов параметров
owlwhisper.FreeString.argtypes = [ctypes.c_char_p]

# Получение строки от Core
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
1. Проверяется валидность указателя
2. Освобождается память, выделенная для строки
3. Указатель становится невалидным
4. Память возвращается в систему

**Важные замечания:**
- ⚠️ **ВСЕГДА вызывать после использования** строк от Core
- ⚠️ **Не использовать строку после освобождения** - undefined behavior
- 💡 **Используйте для предотвращения** утечек памяти
- 🔒 **Потокобезопасно** - можно вызывать из разных потоков

---

### **`SetLogLevel(level)`**

**Описание:** Устанавливает уровень логирования для Core.

**Сигнатура:**
```c
int SetLogLevel(int level);
```

**Параметры:**
- `level` (int) - уровень логирования (0-4)

**Возвращает:**
- `0` - успешная установка
- `-1` - ошибка установки

**Уровни логирования:**
- `0` - **ERROR** - только ошибки
- `1` - **WARN** - предупреждения и ошибки
- `2` - **INFO** - информация, предупреждения и ошибки
- `3` - **DEBUG** - отладочная информация и выше
- `4` - **TRACE** - все сообщения

**Пример использования:**
```python
import ctypes

# Настройка типов параметров и возвращаемого значения
owlwhisper.SetLogLevel.argtypes = [ctypes.c_int]
owlwhisper.SetLogLevel.restype = ctypes.c_int

# Устанавливаем уровень логирования DEBUG
result = owlwhisper.SetLogLevel(3)
if result == 0:
    print("✅ Уровень логирования установлен в DEBUG")
else:
    print("❌ Не удалось установить уровень логирования")

# Устанавливаем минимальный уровень (только ошибки)
result = owlwhisper.SetLogLevel(0)
if result == 0:
    print("✅ Уровень логирования установлен в ERROR")
else:
    print("❌ Не удалось установить уровень логирования")
```

**Что происходит при вызове:**
1. Проверяется валидность уровня логирования
2. Устанавливается глобальный уровень для Core
3. Все последующие логи фильтруются по этому уровню
4. Изменения применяются немедленно

**Важные замечания:**
- ⚠️ **Уровень 0-4** - другие значения игнорируются
- 💡 **Используйте для отладки** и мониторинга
- 🔄 **Изменения применяются** ко всем логам
- 📝 **Логи выводятся** в stderr по умолчанию

---

### **`GetLogLevel()`**

**Описание:** Возвращает текущий уровень логирования Core.

**Сигнатура:**
```c
int GetLogLevel();
```

**Параметры:** Нет

**Возвращает:**
- `int` - текущий уровень логирования (0-4)

**Пример использования:**
```python
import ctypes

# Настройка типа возвращаемого значения
owlwhisper.GetLogLevel.restype = ctypes.c_int

# Получение текущего уровня логирования
current_level = owlwhisper.GetLogLevel()
print(f"📝 Текущий уровень логирования: {current_level}")

# Преобразование в человекочитаемый формат
level_names = {
    0: "ERROR",
    1: "WARN", 
    2: "INFO",
    3: "DEBUG",
    4: "TRACE"
}

level_name = level_names.get(current_level, "UNKNOWN")
print(f"📝 Уровень логирования: {level_name}")
```

**Что происходит при вызове:**
1. Извлекается текущий глобальный уровень логирования
2. Возвращается числовое значение (0-4)
3. Функция быстрая - только чтение

**Важные замечания:**
- 💡 **Используйте для проверки** текущих настроек
- 🔍 **Быстрая функция** - не влияет на производительность
- 📊 **Возвращает число** - нужно преобразовывать в название
- 🔄 **Актуальное значение** - всегда соответствует реальному состоянию

## 🔄 **Сценарии использования**

### **Безопасное управление памятью:**

```python
def safe_get_peer_info(peer_id):
    """Безопасное получение информации о пире с автоматическим освобождением памяти"""
    try:
        peer_id_bytes = peer_id.encode('utf-8')
        peer_info_ptr = owlwhisper.FindPeer(peer_id_bytes)
        
        if peer_info_ptr:
            try:
                # Получаем данные
                peer_info_json = ctypes.string_at(peer_info_ptr).decode()
                if not peer_info_json.startswith("ERROR"):
                    peer_info = json.loads(peer_info_json)
                    return peer_info
                else:
                    print(f"❌ Ошибка поиска: {peer_info_json}")
                    return None
            finally:
                # ВСЕГДА освобождаем память, даже при ошибке
                owlwhisper.FreeString(peer_info_ptr)
        else:
            print("❌ Пир не найден")
            return None
            
    except Exception as e:
        print(f"❌ Исключение при поиске пира: {e}")
        return None

# Пример использования
peer_info = safe_get_peer_info("12D3KooW...")
if peer_info:
    print(f"✅ Найден пир: {peer_info['id']}")
```

### **Автоматическое управление памятью с контекстным менеджером:**

```python
class SafeString:
    """Контекстный менеджер для безопасной работы со строками Core"""
    
    def __init__(self, string_ptr):
        self.string_ptr = string_ptr
        self._freed = False
    
    def __enter__(self):
        return self.string_ptr
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        if not self._freed and self.string_ptr:
            owlwhisper.FreeString(self.string_ptr)
            self._freed = True
    
    def get_string(self):
        """Получает строку и автоматически освобождает память"""
        if self.string_ptr:
            try:
                return ctypes.string_at(self.string_ptr).decode()
            finally:
                if not self._freed:
                    owlwhisper.FreeString(self.string_ptr)
                    self._freed = True
        return None

# Пример использования
def get_network_stats_safe():
    """Безопасное получение статистики сети"""
    stats_ptr = owlwhisper.GetNetworkStats()
    if stats_ptr:
        with SafeString(stats_ptr) as safe_ptr:
            stats_json = ctypes.string_at(safe_ptr).decode()
            return json.loads(stats_json)
    return None

# Или более простой вариант
def get_peer_id_safe():
    """Безопасное получение Peer ID"""
    peer_id_ptr = owlwhisper.GetMyPeerID()
    if peer_id_ptr:
        safe_string = SafeString(peer_id_ptr)
        return safe_string.get_string()
    return None

# Пример использования
stats = get_network_stats_safe()
if stats:
    print(f"Подключенных пиров: {stats.get('connected_peers', 0)}")

peer_id = get_peer_id_safe()
if peer_id:
    print(f"Мой Peer ID: {peer_id}")
```

### **Управление уровнем логирования:**

```python
def configure_logging_for_environment():
    """Настройка логирования в зависимости от окружения"""
    import os
    
    # Определяем окружение
    env = os.getenv('OWLWHISPER_ENV', 'production')
    
    if env == 'development':
        # В разработке - подробное логирование
        level = 3  # DEBUG
        print("🔧 Режим разработки - устанавливаем DEBUG логирование")
    elif env == 'testing':
        # В тестировании - средний уровень
        level = 2  # INFO
        print("🧪 Режим тестирования - устанавливаем INFO логирование")
    else:
        # В продакшене - только ошибки
        level = 0  # ERROR
        print("🚀 Продакшен режим - устанавливаем ERROR логирование")
    
    # Устанавливаем уровень
    result = owlwhisper.SetLogLevel(level)
    if result == 0:
        print(f"✅ Уровень логирования установлен: {level}")
        
        # Проверяем установку
        current_level = owlwhisper.GetLogLevel()
        if current_level == level:
            print("✅ Уровень логирования подтвержден")
        else:
            print(f"⚠️ Несоответствие уровня: установлен {level}, текущий {current_level}")
    else:
        print("❌ Не удалось установить уровень логирования")

# Пример использования
configure_logging_for_environment()
```

### **Динамическое управление логированием:**

```python
class LoggingManager:
    """Менеджер логирования с динамическим управлением"""
    
    def __init__(self):
        self.current_level = owlwhisper.GetLogLevel()
        self.level_names = {
            0: "ERROR", 1: "WARN", 2: "INFO", 3: "DEBUG", 4: "TRACE"
        }
    
    def set_level(self, level):
        """Устанавливает уровень логирования"""
        if level not in self.level_names:
            print(f"❌ Неверный уровень логирования: {level}")
            return False
        
        result = owlwhisper.SetLogLevel(level)
        if result == 0:
            self.current_level = level
            print(f"✅ Уровень логирования изменен на {self.level_names[level]}")
            return True
        else:
            print(f"❌ Не удалось установить уровень {self.level_names[level]}")
            return False
    
    def get_level(self):
        """Получает текущий уровень логирования"""
        return self.current_level
    
    def get_level_name(self):
        """Получает название текущего уровня"""
        return self.level_names.get(self.current_level, "UNKNOWN")
    
    def increase_level(self):
        """Увеличивает уровень логирования"""
        new_level = min(self.current_level + 1, 4)
        return self.set_level(new_level)
    
    def decrease_level(self):
        """Уменьшает уровень логирования"""
        new_level = max(self.current_level - 1, 0)
        return self.set_level(new_level)
    
    def set_debug(self):
        """Устанавливает DEBUG уровень"""
        return self.set_level(3)
    
    def set_error_only(self):
        """Устанавливает только ERROR логирование"""
        return self.set_level(0)
    
    def status(self):
        """Показывает текущий статус логирования"""
        print(f"📝 Текущий уровень логирования: {self.current_level} ({self.get_level_name()})")

# Пример использования
logging_manager = LoggingManager()
logging_manager.status()

# Переключаемся в режим отладки
logging_manager.set_debug()

# Проверяем статус
logging_manager.status()

# Увеличиваем уровень
logging_manager.increase_level()
logging_manager.status()
```

### **Автоматическая очистка памяти:**

```python
def cleanup_all_strings(string_pointers):
    """Автоматическая очистка всех строк"""
    cleaned_count = 0
    
    for string_ptr in string_pointers:
        if string_ptr:
            try:
                owlwhisper.FreeString(string_ptr)
                cleaned_count += 1
            except Exception as e:
                print(f"⚠️ Ошибка при освобождении строки: {e}")
    
    print(f"🧹 Очищено строк: {cleaned_count}")
    return cleaned_count

# Пример использования
def batch_get_peer_info(peer_ids):
    """Пакетное получение информации о пирах"""
    string_pointers = []
    
    try:
        for peer_id in peer_ids:
            peer_id_bytes = peer_id.encode('utf-8')
            peer_info_ptr = owlwhisper.FindPeer(peer_id_bytes)
            
            if peer_info_ptr:
                string_pointers.append(peer_info_ptr)
                
                # Обрабатываем данные
                peer_info_json = ctypes.string_at(peer_info_ptr).decode()
                if not peer_info_json.startswith("ERROR"):
                    peer_info = json.loads(peer_info_json)
                    print(f"✅ {peer_id}: {peer_info['id']}")
                else:
                    print(f"❌ {peer_id}: {peer_info_json}")
            else:
                print(f"❌ {peer_id}: не найден")
        
        return True
        
    finally:
        # Автоматически очищаем все строки
        cleanup_all_strings(string_pointers)

# Пример использования
peer_list = ["12D3KooW...", "12D3KooW...", "12D3KooW..."]
batch_get_peer_info(peer_list)
```

## ⚠️ **Важные замечания**

### **Управление памятью:**
- **ВСЕГДА вызывать `FreeString()`** после использования строк от Core
- **Не использовать строку после освобождения** - undefined behavior
- **Освобождать даже при ошибках** - используйте try-finally
- **Потокобезопасно** - можно вызывать из разных потоков

### **Уровни логирования:**
- **0 (ERROR)** - только критические ошибки
- **1 (WARN)** - предупреждения и ошибки
- **2 (INFO)** - основная информация
- **3 (DEBUG)** - отладочная информация
- **4 (TRACE)** - все сообщения

### **Производительность:**
- **`FreeString()`** - быстрая, локальная операция
- **`SetLogLevel()`** - быстрая, локальная операция
- **`GetLogLevel()`** - мгновенная, только чтение
- **Логирование не влияет** на производительность сети

### **Надежность:**
- **Автоматическая проверка** валидности указателей
- **Graceful degradation** при ошибках
- **Потокобезопасность** для всех операций
- **Защита от двойного освобождения**

### **Ограничения:**
- **Только строки от Core** - не работает с другими строками
- **Нет управления файлами логов** - только stderr
- **Нет ротации логов** - все в одном потоке
- **Нет структурированного логирования** - только текст

## 🔗 **Связанные функции**

- [Управление Core](../functions/core-management.md) - запуск Core для использования утилит
- [Система событий](../functions/events-system.md) - логирование событий
- [Мониторинг сети](../functions/network-monitoring.md) - логирование сетевых проблем
- [Все функции, возвращающие строки](../functions/) - требуют `FreeString()`

---

**Последнее обновление:** 23 августа 2025  
**Автор:** Core Development Team 