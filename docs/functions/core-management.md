# 🚀 **Управление Core**

**Последнее обновление:** 23 августа 2025

Функции для управления жизненным циклом Owl Whisper Core.

## 📋 **Функции**

### **`StartOwlWhisper()`**

**Описание:** Запускает Owl Whisper Core с автоматически сгенерированным ключом.

**Сигнатура:**
```c
int StartOwlWhisper();
```

**Параметры:** Нет

**Возвращает:**
- `0` - успешный запуск
- `-1` - ошибка запуска

**Пример использования:**
```python
import ctypes

# Настройка типа возвращаемого значения
owlwhisper.StartOwlWhisper.restype = ctypes.c_int

# Запуск Core
result = owlwhisper.StartOwlWhisper()
if result == 0:
    print("✅ Owl Whisper Core запущен")
else:
    print("❌ Ошибка запуска Owl Whisper Core")
```

**Что происходит при запуске:**
1. Генерируется новая пара libp2p ключей Ed25519
2. Создается P2P узел с libp2p
3. Запускается mDNS discovery для локального поиска
4. Запускается DHT discovery для глобального поиска
5. Устанавливаются соединения с bootstrap узлами

**Важные замечания:**
- ⚠️ **Не вызывать повторно** - Core уже запущен
- ⚠️ **Проверять возвращаемое значение** - 0 = успех, -1 = ошибка
- 💡 **Используйте для быстрого старта** без сохранения профиля

---

### **`StartOwlWhisperWithKey(keyBytes, keyLength)`**

**Описание:** Запускает Owl Whisper Core с переданным приватным ключом.

**Сигнатура:**
```c
int StartOwlWhisperWithKey(char* keyBytes, int keyLength);
```

**Параметры:**
- `keyBytes` (char*) - массив байт приватного ключа
- `keyLength` (int) - длина ключа в байтах

**Возвращает:**
- `0` - успешный запуск
- `-1` - ошибка запуска

**Пример использования:**
```python
import ctypes
import base64

# Настройка типов параметров и возвращаемого значения
owlwhisper.StartOwlWhisperWithKey.argtypes = [ctypes.c_char_p, ctypes.c_int]
owlwhisper.StartOwlWhisperWithKey.restype = ctypes.c_int

# Загрузка существующего ключа
with open("private_key.bin", "rb") as f:
    private_key = f.read()

# Запуск с ключом
result = owlwhisper.StartOwlWhisperWithKey(private_key, len(private_key))
if result == 0:
    print("✅ Owl Whisper Core запущен с существующим ключом")
else:
    print("❌ Ошибка запуска с ключом")
```

**Что происходит при запуске:**
1. Восстанавливается Peer ID из переданного ключа
2. Создается P2P узел с существующим ключом
3. Запускается mDNS discovery для локального поиска
4. Запускается DHT discovery для глобального поиска
5. Устанавливаются соединения с bootstrap узлами

**Важные замечания:**
- ⚠️ **Ключ должен быть в правильном формате** - libp2p Ed25519
- ⚠️ **Не передавать публичный ключ** - только приватный
- 💡 **Используйте для восстановления профиля** или запуска с существующим ключом
- 🔒 **Безопасно храните ключ** - он определяет вашу сетевую идентичность

---

### **`StopOwlWhisper()`**

**Описание:** Останавливает Owl Whisper Core и освобождает все ресурсы.

**Сигнатура:**
```c
int StopOwlWhisper();
```

**Параметры:** Нет

**Возвращает:**
- `0` - успешная остановка
- `-1` - ошибка остановки

**Пример использования:**
```python
import ctypes

# Настройка типа возвращаемого значения
owlwhisper.StopOwlWhisper.restype = ctypes.c_int

# Остановка Core
result = owlwhisper.StopOwlWhisper()
if result == 0:
    print("✅ Owl Whisper Core остановлен")
else:
    print("❌ Ошибка остановки")
```

**Что происходит при остановке:**
1. Закрываются все сетевые соединения
2. Останавливается mDNS discovery
3. Останавливается DHT discovery
4. Сохраняется DHT routing table в кэш
5. Освобождаются все ресурсы libp2p
6. Останавливается EventManager

**Важные замечания:**
- ⚠️ **Вызывать перед завершением программы** - иначе ресурсы не освободятся
- ⚠️ **Не вызывать если Core не запущен** - может вызвать ошибки
- 💡 **Используйте для корректного завершения** работы с Core
- 🔄 **После остановки можно запустить заново** - Core полностью перезапускается

## 🔄 **Жизненный цикл Core**

### **Типичная последовательность:**

```python
import ctypes

# 1. Запуск
result = owlwhisper.StartOwlWhisper()
if result != 0:
    print("❌ Не удалось запустить Core")
    exit(1)

print("✅ Core запущен")

# 2. Работа с Core
# ... выполнение операций ...

# 3. Остановка
result = owlwhisper.StopOwlWhisper()
if result != 0:
    print("❌ Не удалось остановить Core")
else:
    print("✅ Core остановлен")
```

### **Обработка ошибок:**

```python
def safe_start_core():
    """Безопасный запуск Core с обработкой ошибок"""
    try:
        result = owlwhisper.StartOwlWhisper()
        if result == 0:
            print("✅ Core запущен")
            return True
        else:
            print(f"❌ Ошибка запуска Core: {result}")
            return False
    except Exception as e:
        print(f"❌ Исключение при запуске Core: {e}")
        return False

def safe_stop_core():
    """Безопасная остановка Core с обработкой ошибок"""
    try:
        result = owlwhisper.StopOwlWhisper()
        if result == 0:
            print("✅ Core остановлен")
            return True
        else:
            print(f"❌ Ошибка остановки Core: {result}")
            return False
    except Exception as e:
        print(f"❌ Исключение при остановке Core: {e}")
        return False
```

## ⚠️ **Важные замечания**

### **Безопасность:**
- **Не передавайте ключи в открытом виде** - используйте шифрование
- **Проверяйте возвращаемые значения** от всех функций
- **Обрабатывайте исключения** при работе с Core

### **Производительность:**
- **Запуск занимает 2-5 секунд** - время на подключение к bootstrap узлам
- **Остановка занимает 1-2 секунды** - время на сохранение состояния
- **Не запускайте/останавливайте Core часто** - используйте один экземпляр

### **Совместимость:**
- **Core полностью перезапускается** при каждом Start/Stop
- **Состояние не сохраняется** между запусками (кроме DHT routing table)
- **Используйте один экземпляр** для всего жизненного цикла приложения

## 🔗 **Связанные функции**

- [Генерация ключей](../functions/key-generation.md) - создание ключей для StartOwlWhisperWithKey
- [Система событий](../functions/events-system.md) - мониторинг состояния Core
- [Мониторинг сети](../functions/network-monitoring.md) - проверка работоспособности

---

**Последнее обновление:** 23 августа 2025  
**Автор:** Core Development Team 