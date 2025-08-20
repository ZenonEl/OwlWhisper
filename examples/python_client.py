#!/usr/bin/env python3
"""
Python клиент для Owl Whisper через Go библиотеку
Использует ctypes для прямого вызова функций
"""

import ctypes
import json
import os
import sys

# Путь к Go библиотеке
LIBRARY_PATH = "../dist/libowlwhisper.so"

def load_library():
    """Загружает Go библиотеку"""
    if not os.path.exists(LIBRARY_PATH):
        print(f"❌ Библиотека не найдена: {LIBRARY_PATH}")
        print("Сначала скомпилируйте Go в shared library:")
        print("go build -buildmode=c-shared -o dist/libowlwhisper.so ./cmd/owlwhisper")
        sys.exit(1)
    
    try:
        lib = ctypes.CDLL(LIBRARY_PATH)
        print("✅ Библиотека загружена успешно")
        return lib
    except Exception as e:
        print(f"❌ Ошибка загрузки библиотеки: {e}")
        sys.exit(1)

def main():
    """Основная функция"""
    print("🦉 Owl Whisper Python Client")
    print("=" * 40)
    
    # Загружаем библиотеку
    owlwhisper = load_library()
    
    # Настраиваем типы возвращаемых значений
    owlwhisper.GetMyPeerID.restype = ctypes.c_char_p
    owlwhisper.GetPeers.restype = ctypes.c_char_p
    owlwhisper.GetConnectionStatus.restype = ctypes.c_char_p
    owlwhisper.GetChatHistory.restype = ctypes.c_char_p
    owlwhisper.GetChatHistoryLimit.restype = ctypes.c_char_p
    
    print("\n🚀 Запуск Owl Whisper...")
    result = owlwhisper.StartOwlWhisper()
    if result == 0:
        print("✅ Owl Whisper запущен")
    else:
        print("❌ Ошибка запуска")
        return
    
    try:
        # Получаем информацию о себе
        print("\n👤 Мой Peer ID:")
        peer_id = owlwhisper.GetMyPeerID()
        if peer_id:
            print(f"   {peer_id.decode('utf-8')}")
            # Освобождаем память
            owlwhisper.FreeString(peer_id)
        
        # Получаем статус подключения
        print("\n🌐 Статус подключения:")
        status = owlwhisper.GetConnectionStatus()
        if status:
            try:
                status_data = json.loads(status.decode('utf-8'))
                print(f"   Подключен: {status_data.get('connected', 'Unknown')}")
                print(f"   Пиров: {status_data.get('peers', 'Unknown')}")
            except json.JSONDecodeError:
                print(f"   {status.decode('utf-8')}")
            finally:
                owlwhisper.FreeString(status)
        
        # Получаем список пиров
        print("\n👥 Список пиров:")
        peers = owlwhisper.GetPeers()
        if peers:
            try:
                peers_data = json.loads(peers.decode('utf-8'))
                if peers_data:
                    for i, peer in enumerate(peers_data, 1):
                        print(f"   {i}. {peer}")
                else:
                    print("   Пиров пока нет")
            except json.JSONDecodeError:
                print(f"   {peers.decode('utf-8')}")
            finally:
                owlwhisper.FreeString(peers)
        
        # Отправляем тестовое сообщение
        print("\n💬 Отправка тестового сообщения...")
        test_message = "Привет от Python клиента! 🐍"
        result = owlwhisper.SendMessage(test_message.encode('utf-8'))
        if result == 0:
            print("✅ Сообщение отправлено")
        else:
            print("❌ Ошибка отправки")
        
        # Получаем историю чата (если есть)
        print("\n📚 История чата:")
        history = owlwhisper.GetChatHistory(b"test-peer")
        if history:
            try:
                history_data = json.loads(history.decode('utf-8'))
                if history_data:
                    for msg in history_data:
                        print(f"   [{msg.get('timestamp', 'Unknown')}] {msg.get('text', 'Unknown')}")
                else:
                    print("   История пуста")
            except json.JSONDecodeError:
                print(f"   {history.decode('utf-8')}")
            finally:
                owlwhisper.FreeString(history)
        
        print("\n⏳ Ожидание 5 секунд...")
        import time
        time.sleep(5)
        
    except KeyboardInterrupt:
        print("\n\n⏹️ Прерывание пользователем")
    finally:
        # Останавливаем Owl Whisper
        print("\n🛑 Остановка Owl Whisper...")
        result = owlwhisper.StopOwlWhisper()
        if result == 0:
            print("✅ Owl Whisper остановлен")
        else:
            print("❌ Ошибка остановки")

if __name__ == "__main__":
    main() 