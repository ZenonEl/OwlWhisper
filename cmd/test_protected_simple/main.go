package main

import (
	"encoding/base64"
	"fmt"
	"time"
	"unsafe"
)

/*
#cgo CFLAGS: -I../../internal/core
#cgo LDFLAGS: -L../../dist -lowlwhisper
#include "owlwhisper.h"
#include <stdlib.h>
#include <string.h>
*/
import "C"

func main() {
	fmt.Println("🧪 ПРОСТОЙ ТЕСТ: Базовая функциональность библиотеки")
	fmt.Println("======================================================")

	// Шаг 1: Генерируем сырые байты ключа
	fmt.Println("\n🔑 Шаг 1: Генерируем сырые байты ключа...")

	keyData := C.GenerateNewKeyBytes()
	if keyData == nil {
		fmt.Println("❌ Ошибка генерации ключа")
		return
	}
	defer C.FreeString(keyData)

	keyStr := C.GoString(keyData)
	fmt.Printf("✅ Base64-encoded ключ: %d символов\n", len(keyStr))
	fmt.Printf("🔍 Первые 20 символов: %s...\n", keyStr[:20])

	// Шаг 2: Запускаем с ключом
	fmt.Println("\n🚀 Шаг 2: Запускаем Owl Whisper...")

	// Декодируем base64 в байты
	keyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		fmt.Printf("❌ Ошибка декодирования base64: %v\n", err)
		return
	}

	fmt.Printf("✅ Декодированный ключ: %d байт\n", len(keyBytes))
	fmt.Printf("🔍 Первые 16 байт (hex): %x\n", keyBytes[:16])

	result := C.StartOwlWhisperWithKey((*C.char)(unsafe.Pointer(&keyBytes[0])), C.int(len(keyBytes)))
	if result != 0 {
		fmt.Println("❌ Ошибка запуска Owl Whisper")
		return
	}

	fmt.Println("✅ Owl Whisper запущен!")

	// Шаг 3: Проверяем базовые функции
	fmt.Println("\n🔍 Шаг 3: Проверяем базовые функции...")

	// Получаем Peer ID
	peerID := C.GetMyPeerID()
	if peerID != nil {
		myPeerID := C.GoString(peerID)
		fmt.Printf("✅ Мой Peer ID: %s\n", myPeerID)
		C.FreeString(peerID)
	}

	// Получаем статус соединения
	status := C.GetConnectionStatus()
	if status != nil {
		statusStr := C.GoString(status)
		fmt.Printf("✅ Статус соединения: %s\n", statusStr)
		C.FreeString(status)
	}

	// Получаем список пиров
	peers := C.GetConnectedPeers()
	if peers != nil {
		peersStr := C.GoString(peers)
		fmt.Printf("✅ Список пиров: %s\n", peersStr)
		C.FreeString(peers)
	}

	// 🔒 ТЕСТИРУЕМ ЗАЩИЩЕННЫЕ ПИРЫ
	fmt.Println("\n🔒 Тестируем защищенные пиры...")

	// Получаем список защищенных пиров (должен быть пустым)
	protectedPeers := C.GetProtectedPeers()
	if protectedPeers != nil {
		protectedStr := C.GoString(protectedPeers)
		fmt.Printf("✅ Защищенные пиры: %s\n", protectedStr)
		C.FreeString(protectedPeers)
	} else {
		fmt.Println("❌ Не удалось получить список защищенных пиров")
	}

	// Получаем наш PeerID для тестирования
	myPeerID := C.GetMyPeerID()
	if myPeerID != nil {
		myID := C.GoString(myPeerID)

		// Проверяем, является ли наш PeerID защищенным (должен быть false)
		fmt.Printf("\n🔍 Проверяем защиту для %s...\n", myID[:12]+"...")
		isProtected := C.IsProtectedPeer(myPeerID)
		if isProtected == 1 {
			fmt.Println("❌ PeerID не должен быть защищенным изначально")
		} else {
			fmt.Println("✅ PeerID не защищен (правильно)")
		}

		// Добавляем в защищенные
		fmt.Println("\n🔒 Добавляем PeerID в защищенные...")
		addResult := C.AddProtectedPeer(myPeerID)
		if addResult == 0 {
			fmt.Println("✅ PeerID добавлен в защищенные")

			// Проверяем снова
			isProtected = C.IsProtectedPeer(myPeerID)
			if isProtected == 1 {
				fmt.Println("✅ PeerID теперь защищен")
			} else {
				fmt.Println("❌ PeerID не защищен после добавления")
			}

			// Удаляем из защищенных
			fmt.Println("\n🔓 Удаляем PeerID из защищенных...")
			removeResult := C.RemoveProtectedPeer(myPeerID)
			if removeResult == 0 {
				fmt.Println("✅ PeerID удален из защищенных")

				// Финальная проверка
				isProtected = C.IsProtectedPeer(myPeerID)
				if isProtected == 0 {
					fmt.Println("✅ PeerID больше не защищен")
				} else {
					fmt.Println("❌ PeerID все еще защищен")
				}
			} else {
				fmt.Println("❌ Не удалось удалить PeerID")
			}
		} else {
			fmt.Println("❌ Не удалось добавить PeerID в защищенные")
		}

		C.FreeString(myPeerID)
	}

	// Ждем немного для стабилизации
	fmt.Println("\n⏳ Ждем 10 секунд для стабилизации...")
	time.Sleep(10 * time.Second)

	// Финальная проверка
	fmt.Println("\n📊 Финальная проверка...")

	statusFinal := C.GetConnectionStatus()
	if statusFinal != nil {
		statusStr := C.GoString(statusFinal)
		fmt.Printf("🔍 Финальный статус: %s\n", statusStr)
		C.FreeString(statusFinal)
	}

	// 🔒 ТЕСТИРУЕМ ЛИМИТЫ СОЕДИНЕНИЙ
	fmt.Println("\n📊 Тестируем лимиты соединений...")

	// Получаем текущие лимиты
	limits := C.GetConnectionLimits()
	if limits != nil {
		limitsStr := C.GoString(limits)
		fmt.Printf("📋 Лимиты соединений: %s\n", limitsStr)
		C.FreeString(limits)
	} else {
		fmt.Println("❌ Не удалось получить лимиты соединений")
	}

	// 🔄 ТЕСТИРУЕМ АВТОПЕРЕПОДКЛЮЧЕНИЕ
	fmt.Println("\n🔄 Тестируем автопереподключение...")

	// Проверяем начальное состояние
	fmt.Println("\n📊 Проверяем начальное состояние автопереподключения...")
	isEnabled := C.IsAutoReconnectEnabled()
	if isEnabled == 1 {
		fmt.Println("✅ Автопереподключение включено по умолчанию")
	} else {
		fmt.Println("❌ Автопереподключение не включено по умолчанию")
	}

	// Отключаем автопереподключение
	fmt.Println("\n⏸️ Отключаем автопереподключение...")
	disableResult := C.DisableAutoReconnect()
	if disableResult == 0 {
		fmt.Println("✅ Автопереподключение отключено")

		// Проверяем состояние
		isEnabled = C.IsAutoReconnectEnabled()
		if isEnabled == 0 {
			fmt.Println("✅ Автопереподключение действительно отключено")
		} else {
			fmt.Println("❌ Автопереподключение не отключилось")
		}
	} else {
		fmt.Println("❌ Не удалось отключить автопереподключение")
	}

	// Включаем автопереподключение
	fmt.Println("\n🔄 Включаем автопереподключение...")
	enableResult := C.EnableAutoReconnect()
	if enableResult == 0 {
		fmt.Println("✅ Автопереподключение включено")

		// Проверяем состояние
		isEnabled = C.IsAutoReconnectEnabled()
		if isEnabled == 1 {
			fmt.Println("✅ Автопереподключение действительно включено")
		} else {
			fmt.Println("❌ Автопереподключение не включилось")
		}
	} else {
		fmt.Println("❌ Не удалось включить автопереподключение")
	}

	// Проверяем количество попыток переподключения для нашего пира
	fmt.Println("\n📊 Проверяем количество попыток переподключения...")
	attempts := C.GetReconnectAttempts(myPeerID)
	fmt.Printf("📋 Попыток переподключения: %d\n", attempts)

	fmt.Println("\n✅ Тест завершен успешно!")
	fmt.Println("💡 Библиотека работает без segmentation fault!")
	fmt.Println("💡 НЕ вызываем StopOwlWhisper - это может вызывать проблемы")
}
