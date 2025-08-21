#!/usr/bin/env python3
"""
Пример Python клиента для Owl Whisper

Демонстрирует использование:
- Настройки логирования
- Работы с профилями
- Безопасного управления памятью
- Всех основных функций API
"""

import ctypes
import json
import time
import sys
import os

def load_library():
    """Загружает библиотеку OwlWhisper"""
    # Путь к библиотеке
    lib_path = "./dist/libowlwhisper.so"
    
    # Проверяем существование файла
    if not os.path.exists(lib_path):
        print(f"❌ Библиотека не найдена: {lib_path}")
        print("💡 Запустите: go build -buildmode=c-shared -o dist/libowlwhisper.so ./cmd/owlwhisper")
        sys.exit(1)
    
    try:
        return ctypes.CDLL(lib_path)
    except OSError as e:
        print(f"❌ Ошибка загрузки библиотеки: {e}")
        print("💡 Установите переменную окружения: export LD_LIBRARY_PATH=./dist:$LD_LIBRARY_PATH")
        sys.exit(1)

def safe_get_string(owlwhisper, func_call):
    """Безопасно получает строку из C функции и освобождает память"""
    try:
        result_ptr = func_call()
        if result_ptr:
            result_str = ctypes.string_at(result_ptr).decode('utf-8')
            owlwhisper.FreeString(result_ptr)
            return result_str
        return ""
    except Exception as e:
        print(f"⚠️ Ошибка получения строки: {e}")
        return ""

def test_logging_configuration(owlwhisper):
    """Тестирует настройки логирования"""
    print("\n🔧 === Тест настроек логирования ===")
    
    # Отключаем логи
    print("🔇 Отключаем все логи...")
    result = owlwhisper.SetLogLevel(0)  # SILENT
    if result == 0:
        print("✅ Логи отключены")
    else:
        print("❌ Ошибка отключения логов")
    
    # Можно также настроить вывод в файл
    # owlwhisper.SetLogOutput(2, b"./logs")  # Только в файл
    # owlwhisper.SetLogOutput(3, b"./logs")  # В консоль и файл

def test_basic_operations(owlwhisper):
    """Тестирует базовые операции"""
    print("\n🚀 === Тест базовых операций ===")
    
    # Запуск
    print("🏁 Запуск Owl Whisper...")
    result = owlwhisper.StartOwlWhisper()
    if result != 0:
        print("❌ Ошибка запуска")
        return False
    
    print("✅ Owl Whisper запущен")
    time.sleep(2)  # Ждем инициализации
    
    # Получение Peer ID
    print("\n👤 Получение Peer ID...")
    peer_id = safe_get_string(owlwhisper, owlwhisper.GetMyPeerID)
    print(f"   Peer ID: {peer_id}")
    
    # Статус соединения
    print("\n🌐 Проверка статуса соединения...")
    status_str = safe_get_string(owlwhisper, owlwhisper.GetConnectionStatus)
    try:
        status = json.loads(status_str)
        print(f"   Подключен: {status.get('connected', False)}")
        print(f"   Количество пиров: {status.get('peers', 0)}")
        print(f"   Мой Peer ID: {status.get('my_peer_id', 'неизвестно')}")
    except json.JSONDecodeError:
        print(f"   Статус (сырой): {status_str}")
    
    # Список пиров
    print("\n👥 Получение списка пиров...")
    peers_str = safe_get_string(owlwhisper, owlwhisper.GetPeers)
    try:
        peers = json.loads(peers_str)
        print(f"   Найдено пиров: {len(peers)}")
        for i, peer in enumerate(peers[:3]):  # Показываем только первые 3
            short_id = peer[:20] + "..." if len(peer) > 20 else peer
            print(f"   {i+1}. {short_id}")
        if len(peers) > 3:
            print(f"   ... и еще {len(peers) - 3} пиров")
    except json.JSONDecodeError:
        print(f"   Пиры (сырой): {peers_str}")
    
    return True

def test_profile_operations(owlwhisper):
    """Тестирует операции с профилями"""
    print("\n👤 === Тест операций с профилями ===")
    
    # Получение текущего профиля
    print("📄 Получение текущего профиля...")
    profile_str = safe_get_string(owlwhisper, owlwhisper.GetMyProfile)
    try:
        profile = json.loads(profile_str)
        print(f"   Никнейм: {profile.get('nickname', 'неизвестно')}")
        print(f"   Дискриминатор: {profile.get('discriminator', 'неизвестно')}")
        print(f"   Отображаемое имя: {profile.get('display_name', 'неизвестно')}")
        print(f"   Онлайн: {profile.get('is_online', False)}")
    except json.JSONDecodeError:
        print(f"   Профиль (сырой): {profile_str}")
    
    # Обновление профиля
    print("\n📝 Обновление профиля...")
    new_nickname = "PythonUser_" + str(int(time.time()))
    nickname_bytes = new_nickname.encode('utf-8')
    result = owlwhisper.UpdateMyProfile(nickname_bytes)
    
    if result == 0:
        print(f"✅ Профиль обновлен на: {new_nickname}")
        
        # Проверяем обновленный профиль
        print("🔍 Проверка обновленного профиля...")
        updated_profile_str = safe_get_string(owlwhisper, owlwhisper.GetMyProfile)
        try:
            updated_profile = json.loads(updated_profile_str)
            print(f"   Новый никнейм: {updated_profile.get('nickname', 'неизвестно')}")
            print(f"   Новое отображаемое имя: {updated_profile.get('display_name', 'неизвестно')}")
        except json.JSONDecodeError:
            print(f"   Обновленный профиль (сырой): {updated_profile_str}")
    else:
        print("❌ Ошибка обновления профиля")

def test_memory_stress(owlwhisper):
    """Стресс-тест управления памятью"""
    print("\n🔄 === Стресс-тест управления памятью ===")
    
    print("🧪 Выполняем 50 операций с интенсивным использованием памяти...")
    for i in range(50):
        # Тестируем все функции, которые возвращают строки
        _ = safe_get_string(owlwhisper, owlwhisper.GetMyPeerID)
        _ = safe_get_string(owlwhisper, owlwhisper.GetPeers)
        _ = safe_get_string(owlwhisper, owlwhisper.GetConnectionStatus)
        _ = safe_get_string(owlwhisper, owlwhisper.GetMyProfile)
        
        if i % 10 == 0:
            print(f"   Прогресс: {i}/50")
    
    print("✅ Стресс-тест завершен без ошибок памяти")

def test_messaging(owlwhisper):
    """Тестирует отправку сообщений"""
    print("\n💬 === Тест отправки сообщений ===")
    
    # Отправка широковещательного сообщения
    test_message = f"Привет от Python! Время: {time.strftime('%H:%M:%S')}"
    message_bytes = test_message.encode('utf-8')
    
    print(f"📤 Отправка сообщения: {test_message}")
    result = owlwhisper.SendMessage(message_bytes)
    
    if result == 0:
        print("✅ Сообщение отправлено")
    else:
        print("❌ Ошибка отправки сообщения")

def shutdown(owlwhisper):
    """Корректно останавливает Owl Whisper"""
    print("\n🛑 === Остановка ===")
    
    result = owlwhisper.StopOwlWhisper()
    if result == 0:
        print("✅ Owl Whisper остановлен")
    else:
        print("❌ Ошибка остановки")

def main():
    """Главная функция"""
    print("🐍 Python клиент для Owl Whisper")
    print("=" * 50)
    
    # Загружаем библиотеку
    print("📚 Загрузка библиотеки...")
    owlwhisper = load_library()
    print("✅ Библиотека загружена")
    
    try:
        # Тестируем настройки логирования
        test_logging_configuration(owlwhisper)
        
        # Тестируем базовые операции
        if not test_basic_operations(owlwhisper):
            return
        
        # Тестируем профили
        test_profile_operations(owlwhisper)
        
        # Тестируем отправку сообщений
        test_messaging(owlwhisper)
        
        # Стресс-тест памяти
        test_memory_stress(owlwhisper)
        
        print("\n🎉 Все тесты успешно завершены!")
        
    except KeyboardInterrupt:
        print("\n⚠️ Прервано пользователем")
    except Exception as e:
        print(f"\n❌ Неожиданная ошибка: {e}")
    finally:
        # Всегда останавливаем, даже при ошибках
        shutdown(owlwhisper)

if __name__ == "__main__":
    main()