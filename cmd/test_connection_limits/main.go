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
	fmt.Println("🔒 ТЕСТ ЛИМИТОВ СОЕДИНЕНИЙ")
	fmt.Println("============================")

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

	// Проверяем начальные лимиты соединений
	fmt.Println("\n📊 Проверяем начальные лимиты соединений...")
	limits := C.GetConnectionLimits()
	if limits != nil {
		limitsStr := C.GoString(limits)
		fmt.Printf("📋 Лимиты соединений: %s\n", limitsStr)
		C.FreeString(limits)
	} else {
		fmt.Println("❌ Не удалось получить лимиты соединений")
	}

	// Получаем список защищенных пиров (должен быть пустым)
	fmt.Println("\n🔒 Проверяем список защищенных пиров...")
	protectedPeers := C.GetProtectedPeers()
	if protectedPeers != nil {
		protectedStr := C.GoString(protectedPeers)
		fmt.Printf("📋 Защищенные пиры: %s\n", protectedStr)
		C.FreeString(protectedPeers)
	} else {
		fmt.Println("❌ Не удалось получить список защищенных пиров")
	}

	// Тестируем добавление защищенного пира
	fmt.Println("\n🔒 Тестируем добавление защищенного пира...")
	addResult := C.AddProtectedPeer(myPeerID)
	if addResult == 0 {
		fmt.Println("✅ PeerID добавлен в защищенные")

		// Проверяем лимиты после добавления
		fmt.Println("\n📊 Проверяем лимиты после добавления защищенного пира...")
		limits = C.GetConnectionLimits()
		if limits != nil {
			limitsStr := C.GoString(limits)
			fmt.Printf("📋 Лимиты соединений: %s\n", limitsStr)
			C.FreeString(limits)
		}

		// Удаляем защищенный пир
		fmt.Println("\n🔓 Удаляем защищенный пир...")
		removeResult := C.RemoveProtectedPeer(myPeerID)
		if removeResult == 0 {
			fmt.Println("✅ PeerID удален из защищенных")

			// Проверяем лимиты после удаления
			fmt.Println("\n📊 Проверяем лимиты после удаления защищенного пира...")
			limits = C.GetConnectionLimits()
			if limits != nil {
				limitsStr := C.GoString(limits)
				fmt.Printf("📋 Лимиты соединений: %s\n", limitsStr)
				C.FreeString(limits)
			}
		} else {
			fmt.Println("❌ Не удалось удалить защищенный пир")
		}
	} else {
		fmt.Println("❌ Не удалось добавить PeerID в защищенные")
	}

	// Получаем финальную статистику сети
	fmt.Println("\n📊 Финальная статистика сети...")
	networkStats := C.GetNetworkStats()
	if networkStats != nil {
		statsStr := C.GoString(networkStats)
		fmt.Printf("📋 Статистика сети: %s\n", statsStr)
		C.FreeString(networkStats)
	}

	fmt.Println("\n🎉 Тест лимитов соединений завершен!")
}
