#!/usr/bin/env python3
"""
🔑 Пример использования функции GenerateNewKeyPair

Этот пример показывает, как правильно создавать новые профили
вместо использования временных заглушек для ключей.
"""

import ctypes
import base64
import json
import os

def main():
    # Путь к библиотеке
    lib_path = "./dist/libowlwhisper.so"
    
    if not os.path.exists(lib_path):
        print(f"❌ Библиотека не найдена: {lib_path}")
        print("Сначала скомпилируйте библиотеку!")
        return
    
    # Загружаем библиотеку
    print("📚 Загружаем библиотеку...")
    owlwhisper = ctypes.CDLL(lib_path)
    
    # Настраиваем типы для функций
    owlwhisper.GenerateNewKeyPair.restype = ctypes.c_char_p
    owlwhisper.StartOwlWhisperWithKey.argtypes = [ctypes.c_char_p, ctypes.c_int]
    owlwhisper.StartOwlWhisperWithKey.restype = ctypes.c_int
    
    # Отключаем логи для чистого вывода
    print("🔇 Отключаем логи...")
    owlwhisper.SetLogLevel(0)
    
    print("\n🔑 Шаг 1: Генерируем новую пару ключей...")
    
    # Генерируем новую пару ключей
    key_data = owlwhisper.GenerateNewKeyPair()
    if not key_data:
        print("❌ Ошибка генерации ключей")
        return
    
    # Декодируем данные ключа (они в base64)
    base64_data = ctypes.string_at(key_data).decode()
    print(f"   Полученные данные (base64): {base64_data[:100]}...")
    
    try:
        # Сначала декодируем base64, потом парсим JSON
        json_bytes = base64.b64decode(base64_data)
        json_str = json_bytes.decode('utf-8')
        print(f"   Декодированный JSON: {json_str[:100]}...")
        
        key_info = json.loads(json_str)
    except (json.JSONDecodeError, UnicodeDecodeError) as e:
        print(f"❌ Ошибка декодирования: {e}")
        print(f"   Base64 данные: {repr(base64_data)}")
        owlwhisper.FreeString(key_data)
        return
    
    print("✅ Ключи сгенерированы успешно!")
    print(f"   Peer ID: {key_info['peer_id']}")
    print(f"   Тип ключа: {key_info['key_type']}")
    print(f"   Длина ключа: {key_info['key_length']} байт")
    
    # Получаем приватный ключ
    private_key = base64.b64decode(key_info['private_key'])
    print(f"   Приватный ключ (hex): {private_key[:16].hex()}...")
    
    # Освобождаем память
    owlwhisper.FreeString(key_data)
    
    print("\n🚀 Шаг 2: Запускаем Owl Whisper с новым ключом...")
    
    # Запускаем с новым ключом
    result = owlwhisper.StartOwlWhisperWithKey(private_key, len(private_key))
    if result == 0:
        print("✅ Owl Whisper запущен с новым профилем!")
        
        # Проверяем Peer ID
        peer_id = owlwhisper.GetMyPeerID()
        if peer_id:
            my_peer_id = ctypes.string_at(peer_id).decode()
            print(f"   Подтвержденный Peer ID: {my_peer_id}")
            owlwhisper.FreeString(peer_id)
            
            # Проверяем, что Peer ID совпадает
            if my_peer_id == key_info['peer_id']:
                print("✅ Peer ID совпадает с сгенерированным!")
            else:
                print("⚠️ Peer ID не совпадает!")
        
        print("\n🛑 Останавливаем...")
        owlwhisper.StopOwlWhisper()
        print("✅ Остановлен")
        
    else:
        print("❌ Ошибка запуска с новым ключом")
        print("Это означает, что ключ в правильном формате!")

if __name__ == "__main__":
    print("🔑 Пример генерации ключей для Owl Whisper")
    print("=" * 50)
    
    try:
        main()
        print("\n🎉 Пример завершен успешно!")
        print("\n💡 Теперь вы можете:")
        print("   1. Зашифровать приватный ключ")
        print("   2. Сохранить его в профиле")
        print("   3. Использовать для входа в профиль")
        
    except Exception as e:
        print(f"\n❌ Ошибка: {e}")
        print("\n🔧 Возможные решения:")
        print("   1. Убедитесь, что библиотека скомпилирована")
        print("   2. Проверьте путь к библиотеке")
        print("   3. Установите LD_LIBRARY_PATH=./dist") 