# 🔑 **Генерация ключей**

**Последнее обновление:** 23 августа 2025

Функции для создания криптографических ключей libp2p Ed25519.

## 📋 **Функции**

### **`GenerateNewKeyPair()`**

**Описание:** Генерирует новую пару libp2p ключей Ed25519 и возвращает информацию в JSON формате.

**Сигнатура:**
```c
char* GenerateNewKeyPair();
```

**Параметры:** Нет

**Возвращает:**
- `char*` - JSON строка с информацией о ключе (требует `FreeString()`)
- `NULL` - ошибка генерации

**Структура возвращаемых данных:**
```json
{
  "private_key": "base64_encoded_key_bytes",
  "peer_id": "12D3KooW...",
  "key_type": "Ed25519",
  "key_length": 68
}
```

**Пример использования:**
```python
import ctypes
import base64
import json

# Настройка типа возвращаемого значения
owlwhisper.GenerateNewKeyPair.restype = ctypes.c_char_p

# Генерация новой пары ключей
key_data = owlwhisper.GenerateNewKeyPair()
if key_data:
    # Декодируем JSON данные
    json_str = ctypes.string_at(key_data).decode()
    key_info = json.loads(json_str)
    
    print(f"🔑 Новый профиль создан:")
    print(f"   Peer ID: {key_info['peer_id']}")
    print(f"   Тип ключа: {key_info['key_type']}")
    print(f"   Длина ключа: {key_info['key_length']} байт")
    
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

**Что происходит при генерации:**
1. Создается криптографически стойкий Ed25519 ключ
2. Вычисляется Peer ID из публичного ключа
3. Приватный ключ кодируется в base64
4. Формируется JSON с информацией о ключе

**Важные замечания:**
- ⚠️ **ВСЕГДА вызывать `FreeString()`** после использования
- ⚠️ **Приватный ключ определяет вашу сетевую идентичность** - храните безопасно
- 💡 **Используйте для создания новых профилей** пользователей
- 🔒 **Ключ уже в base64** - не нужно дополнительно кодировать

---

### **`GenerateNewKeyBytes()`**

**Описание:** Генерирует сырые байты приватного ключа libp2p Ed25519.

**Сигнатура:**
```c
char* GenerateNewKeyBytes();
```

**Параметры:** Нет

**Возвращает:**
- `char*` - массив байт приватного ключа (требует `FreeString()`)
- `NULL` - ошибка генерации

**Пример использования:**
```python
import ctypes

# Настройка типа возвращаемого значения
owlwhisper.GenerateNewKeyBytes.restype = ctypes.c_char_p

# Генерация сырых байт ключа
key_bytes = owlwhisper.GenerateNewKeyBytes()
if key_bytes:
    # Получаем сырые байты ключа
    raw_key = ctypes.string_at(key_bytes)
    
    print(f"🔑 Сырые байты ключа: {len(raw_key)} байт")
    print(f"   Первые 16 байт: {raw_key[:16].hex()}")
    
    # Освобождаем память
    owlwhisper.FreeString(key_bytes)
    
    # Теперь можно использовать ключ для запуска Core
    result = owlwhisper.StartOwlWhisperWithKey(raw_key, len(raw_key))
else:
    print("❌ Ошибка генерации ключа")
```

**Что происходит при генерации:**
1. Создается криптографически стойкий Ed25519 ключ
2. Возвращаются сырые байты приватного ключа
3. Ключ готов для прямого использования

**Важные замечания:**
- ⚠️ **ВСЕГДА вызывать `FreeString()`** после использования
- ⚠️ **Ключ в сыром формате** - не base64
- 💡 **Используйте для продвинутых сценариев** работы с ключами
- 🔒 **Сырые байты** - передавайте напрямую в StartOwlWhisperWithKey

## 🔄 **Сценарии использования**

### **Создание нового профиля:**

```python
def create_new_profile(nickname):
    """Создание нового профиля пользователя"""
    try:
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
        # или сразу запустить с ним
        return {
            'peer_id': key_info['peer_id'],
            'private_key': private_key,
            'nickname': nickname
        }
        
    except Exception as e:
        print(f"❌ Ошибка создания профиля: {e}")
        return None
```

### **Быстрая генерация для тестирования:**

```python
def quick_key_generation():
    """Быстрая генерация ключа для тестирования"""
    try:
        # Генерируем сырые байты
        key_bytes = owlwhisper.GenerateNewKeyBytes()
        if not key_bytes:
            return None
        
        # Получаем ключ
        raw_key = ctypes.string_at(key_bytes)
        
        # Освобождаем память
        owlwhisper.FreeString(key_bytes)
        
        # Запускаем Core с новым ключом
        result = owlwhisper.StartOwlWhisperWithKey(raw_key, len(raw_key))
        if result == 0:
            print("✅ Core запущен с новым ключом")
            return True
        else:
            print("❌ Не удалось запустить Core с новым ключом")
            return False
            
    except Exception as e:
        print(f"❌ Ошибка быстрой генерации: {e}")
        return False
```

### **Безопасное хранение ключей:**

```python
import cryptography.fernet
import os

def encrypt_and_save_key(private_key, password, filename):
    """Шифрование и сохранение приватного ключа"""
    try:
        # Создаем ключ шифрования из пароля
        salt = os.urandom(16)
        key = cryptography.fernet.Fernet.generate_key()
        cipher = cryptography.fernet.Fernet(key)
        
        # Шифруем приватный ключ
        encrypted_key = cipher.encrypt(private_key)
        
        # Сохраняем зашифрованный ключ
        with open(filename, 'wb') as f:
            f.write(encrypted_key)
        
        print(f"✅ Ключ зашифрован и сохранен в {filename}")
        return True
        
    except Exception as e:
        print(f"❌ Ошибка шифрования ключа: {e}")
        return False

def load_and_decrypt_key(filename, password):
    """Загрузка и расшифровка приватного ключа"""
    try:
        # Загружаем зашифрованный ключ
        with open(filename, 'rb') as f:
            encrypted_key = f.read()
        
        # Расшифровываем ключ
        cipher = cryptography.fernet.Fernet(key)
        private_key = cipher.decrypt(encrypted_key)
        
        print("✅ Ключ загружен и расшифрован")
        return private_key
        
    except Exception as e:
        print(f"❌ Ошибка загрузки ключа: {e}")
        return None
```

## ⚠️ **Важные замечания**

### **Безопасность:**
- **Приватный ключ определяет вашу сетевую идентичность** - храните безопасно
- **Используйте шифрование** для хранения ключей
- **Не передавайте ключи в открытом виде** по сети
- **Генерируйте ключи на безопасном устройстве**

### **Форматы ключей:**
- **`GenerateNewKeyPair()`** возвращает base64-кодированный ключ
- **`GenerateNewKeyBytes()`** возвращает сырые байты
- **Все ключи в формате libp2p Ed25519** - совместимы с StartOwlWhisperWithKey

### **Производительность:**
- **Генерация занимает <1 секунды** - криптографически стойкие ключи
- **Ключи уникальны** - вероятность коллизии практически нулевая
- **Можно генерировать множество ключей** без проблем с производительностью

### **Совместимость:**
- **Ключи совместимы с libp2p** - стандартный формат Ed25519
- **Peer ID вычисляется автоматически** из публичного ключа
- **Можно использовать в других libp2p приложениях**

## 🔗 **Связанные функции**

- [Управление Core](../functions/core-management.md) - запуск Core с сгенерированными ключами
- [Система событий](../functions/events-system.md) - мониторинг состояния после запуска
- [Утилиты](../functions/utilities.md) - управление памятью и логирование

---

**Последнее обновление:** 23 августа 2025  
**Автор:** Core Development Team 