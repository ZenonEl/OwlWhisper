package main

/*
#cgo CFLAGS: -I../../internal/core
#cgo LDFLAGS: -L../../dist -lowlwhisper
#include "owlwhisper.h"
#include <stdlib.h>
#include <stdio.h>
*/
import "C"
import (
	"fmt"
	"strings"
	"unsafe"
)

func main() {
	fmt.Println("🧪 Тестирование функций поиска и анонсирования контента")
	fmt.Println(strings.Repeat("=", 60))

	// 1. Запускаем Core
	fmt.Println("\n1️⃣ Запуск Core...")
	result := C.StartOwlWhisper()
	if result != 0 {
		fmt.Println("❌ Не удалось запустить Core")
		return
	}
	fmt.Println("✅ Core успешно запущен")

	// 2. Получаем наш Peer ID
	fmt.Println("\n2️⃣ Получение нашего Peer ID...")
	myPeerID := C.GetMyPeerID()
	if myPeerID == nil {
		fmt.Println("❌ Не удалось получить Peer ID")
		return
	}
	myPeerIDStr := C.GoString(myPeerID)
	fmt.Printf("✅ Наш Peer ID: %s\n", myPeerIDStr)
	C.FreeString(myPeerID)

	// 3. Анонсируем себя как провайдера контента
	fmt.Println("\n3️⃣ Анонсирование контента...")
	testContentID := "test-content-123"
	contentIDC := C.CString(testContentID)
	defer C.free(unsafe.Pointer(contentIDC))

	success := C.ProvideContent(contentIDC)
	if success == 1 {
		fmt.Printf("✅ Успешно анонсировали контент: %s\n", testContentID)
	} else {
		fmt.Printf("❌ Не удалось анонсировать контент: %s\n", testContentID)
	}

	// 4. Ищем провайдеров контента
	fmt.Println("\n4️⃣ Поиск провайдеров контента...")
	providers := C.FindProvidersForContent(contentIDC)
	if providers == nil {
		fmt.Println("❌ Не удалось найти провайдеров")
	} else {
		providersStr := C.GoString(providers)
		if providersStr == "" {
			fmt.Println("ℹ️ Провайдеры не найдены (это нормально для нового контента)")
		} else {
			fmt.Printf("✅ Найдены провайдеры: %s\n", providersStr)
		}
		C.FreeString(providers)
	}

	// 5. Ищем пира по Peer ID
	fmt.Println("\n5️⃣ Поиск пира по Peer ID...")
	peerIDC := C.CString(myPeerIDStr)
	defer C.free(unsafe.Pointer(peerIDC))

	peerInfo := C.FindPeer(peerIDC)
	if peerInfo == nil {
		fmt.Println("❌ Не удалось найти пира")
	} else {
		peerInfoStr := C.GoString(peerInfo)
		if peerInfoStr == "" {
			fmt.Println("ℹ️ Пир не найден в DHT (это нормально для локального пира)")
		} else {
			fmt.Printf("✅ Пир найден: %s\n", peerInfoStr)
		}
		C.FreeString(peerInfo)
	}

	// 6. Получаем статистику сети
	fmt.Println("\n6️⃣ Получение статистики сети...")
	networkStats := C.GetNetworkStats()
	if networkStats == nil {
		fmt.Println("❌ Не удалось получить статистику сети")
	} else {
		statsStr := C.GoString(networkStats)
		fmt.Printf("✅ Статистика сети: %s\n", statsStr)
		C.FreeString(networkStats)
	}

	// 7. Останавливаем Core
	fmt.Println("\n7️⃣ Остановка Core...")
	stopResult := C.StopOwlWhisper()
	if stopResult != 1 {
		fmt.Println("❌ Не удалось остановить Core")
		return
	}
	fmt.Println("✅ Core успешно остановлен")

	fmt.Println("\n🎉 Тестирование завершено успешно!")
	fmt.Println(strings.Repeat("=", 60))
}
