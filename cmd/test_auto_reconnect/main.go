package main

import (
	"C"
	"fmt"
	"time"
)

/*
#cgo CFLAGS: -I../../internal/core
#cgo LDFLAGS: -L../../dist -lowlwhisper
#include "owlwhisper.h"
#include <stdlib.h>
#include <string.h>
*/

func main() {
	fmt.Println("🔄 ТЕСТ АВТОПЕРЕПОДКЛЮЧЕНИЯ")
	fmt.Println("==============================")

	// Генерируем новый ключ
	fmt.Println("\n🔑 Генерируем новый ключ...")
	keyBytes := C.GenerateNewKeyBytes()
	if keyBytes == nil {
		fmt.Println("❌ Не удалось сгенерировать ключ")
		return
	}
	defer C.FreeString(keyBytes)

	keyStr := C.GoString(keyBytes)
	fmt.Printf("✅ Ключ сгенерирован, длина: %d байт\n", len(keyStr))

	// Запускаем OwlWhisper
	fmt.Println("\n🚀 Запускаем OwlWhisper...")
	result := C.StartOwlWhisperWithKey(C.CString(keyStr), C.int(len(keyStr)))
	if result != 0 {
		fmt.Println("❌ Не удалось запустить OwlWhisper")
		return
	}
	fmt.Println("✅ OwlWhisper запущен")

	// Ждем стабилизации сети
	fmt.Println("\n⏳ Ждем стабилизации сети (15 секунд)...")
	time.Sleep(15 * time.Second)

	// Получаем наш PeerID
	fmt.Println("\n👤 Получаем наш PeerID...")
	myPeerID := C.GetMyPeerID()
	if myPeerID == nil {
		fmt.Println("❌ Не удалось получить PeerID")
		return
	}
	defer C.FreeString(myPeerID)

	myID := C.GoString(myPeerID)
	fmt.Printf("✅ Наш PeerID: %s\n", myID)

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

	// Добавляем пира в защищенные для тестирования попыток переподключения
	fmt.Println("\n🔒 Добавляем пира в защищенные для тестирования...")
	addResult := C.AddProtectedPeer(myPeerID)
	if addResult == 0 {
		fmt.Println("✅ Пир добавлен в защищенные")

		// Проверяем количество попыток переподключения
		fmt.Println("\n📊 Проверяем количество попыток переподключения...")
		attempts := C.GetReconnectAttempts(myPeerID)
		fmt.Printf("📋 Попыток переподключения: %d\n", attempts)

		// Удаляем пира из защищенных
		fmt.Println("\n🔓 Удаляем пира из защищенных...")
		removeResult := C.RemoveProtectedPeer(myPeerID)
		if removeResult == 0 {
			fmt.Println("✅ Пир удален из защищенных")
		} else {
			fmt.Println("❌ Не удалось удалить пира")
		}
	} else {
		fmt.Println("❌ Не удалось добавить пира в защищенные")
	}

	// Получаем финальную статистику
	fmt.Println("\n📊 Финальная статистика...")

	// Лимиты соединений
	limits := C.GetConnectionLimits()
	if limits != nil {
		limitsStr := C.GoString(limits)
		fmt.Printf("📋 Лимиты соединений: %s\n", limitsStr)
		C.FreeString(limits)
	}

	// Статистика сети
	networkStats := C.GetNetworkStats()
	if networkStats != nil {
		statsStr := C.GoString(networkStats)
		fmt.Printf("📋 Статистика сети: %s\n", statsStr)
		C.FreeString(networkStats)
	}

	fmt.Println("\n🎉 Тест автопереподключения завершен!")
}
