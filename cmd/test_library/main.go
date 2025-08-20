package main

import (
	"fmt"
	"time"
)

// #cgo CFLAGS: -I../../internal/core
// #cgo LDFLAGS: -L../../dist -lowlwhisper
// #include "owlwhisper.h"
import "C"

func main() {
	fmt.Println("🦉 Тестируем Owl Whisper shared library")
	fmt.Println("==================================================")

	// Запускаем
	fmt.Println("🚀 Запуск...")
	result := C.StartOwlWhisper()
	if result == 0 {
		fmt.Println("✅ Owl Whisper запущен")
	} else {
		fmt.Println("❌ Ошибка запуска")
		return
	}

	// Ждем немного
	time.Sleep(2 * time.Second)

	// Получаем Peer ID
	fmt.Println("👤 Получаем Peer ID...")
	peerID := C.GetMyPeerID()
	if peerID != nil {
		goPeerID := C.GoString(peerID)
		fmt.Printf("   Peer ID: %s\n", goPeerID)
		C.FreeString(peerID)
	}

	// Получаем статус
	fmt.Println("🌐 Получаем статус...")
	status := C.GetConnectionStatus()
	if status != nil {
		goStatus := C.GoString(status)
		fmt.Printf("   Статус: %s\n", goStatus)
		C.FreeString(status)
	}

	// Отправляем сообщение
	fmt.Println("💬 Отправляем сообщение...")
	testMsg := C.CString("Тест от Go!")
	result = C.SendMessage(testMsg)
	if result == 0 {
		fmt.Println("✅ Сообщение отправлено")
	} else {
		fmt.Println("❌ Ошибка отправки")
	}

	// Ждем еще немного
	time.Sleep(2 * time.Second)

	// Останавливаем
	fmt.Println("🛑 Остановка...")
	result = C.StopOwlWhisper()
	if result == 0 {
		fmt.Println("✅ Owl Whisper остановлен")
	} else {
		fmt.Println("❌ Ошибка остановки")
	}

	fmt.Println("🎉 Тест завершен!")
}
