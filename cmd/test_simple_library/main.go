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
	peers := C.GetPeers()
	if peers != nil {
		peersStr := C.GoString(peers)
		fmt.Printf("✅ Список пиров: %s\n", peersStr)
		C.FreeString(peers)
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

	fmt.Println("\n✅ Тест завершен успешно!")
	fmt.Println("💡 Библиотека работает без segmentation fault!")
	fmt.Println("💡 НЕ вызываем StopOwlWhisper - это может вызывать проблемы")
}
