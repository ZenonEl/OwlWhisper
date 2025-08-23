package main

import (
	"fmt"
	"strings"
	"time"
)

/*
#cgo CFLAGS: -I../../internal/core
#cgo LDFLAGS: -L../../dist -lowlwhisper
#include "owlwhisper.h"
#include <stdlib.h>
*/
import "C"

func main() {
	fmt.Println("🧪 Тестирование системы событий OwlWhisper")
	fmt.Println(strings.Repeat("=", 60))

	// 1. Запускаем OwlWhisper
	fmt.Println("🚀 Запуск OwlWhisper...")
	result := C.StartOwlWhisper()
	if result != 0 {
		fmt.Println("❌ Ошибка запуска OwlWhisper")
		return
	}
	fmt.Println("✅ OwlWhisper запущен успешно")

	// 2. Получаем наш Peer ID
	fmt.Println("\n🔍 Получение нашего Peer ID...")
	peerIDPtr := C.GetMyPeerID()
	if peerIDPtr == nil {
		fmt.Println("❌ Не удалось получить Peer ID")
		return
	}
	peerID := C.GoString(peerIDPtr)
	C.FreeString(peerIDPtr)
	fmt.Printf("✅ Наш Peer ID: %s\n", peerID)

	// 3. Тестируем GetNextEvent в отдельной горутине
	fmt.Println("\n📡 Запуск слушателя событий...")
	eventChan := make(chan string, 10)

	go func() {
		for {
			eventPtr := C.GetNextEvent()
			if eventPtr == nil {
				// Нет событий, продолжаем ждать
				time.Sleep(100 * time.Millisecond)
				continue
			}

			event := C.GoString(eventPtr)
			C.FreeString(eventPtr)

			eventChan <- event
			fmt.Printf("📨 Получено событие: %s\n", event)
		}
	}()

	// 4. Ждем события в течение 10 секунд
	fmt.Println("⏳ Ожидание событий в течение 10 секунд...")
	timeout := time.After(10 * time.Second)

	eventCount := 0
	for {
		select {
		case event := <-eventChan:
			eventCount++
			fmt.Printf("📨 Событие #%d: %s\n", eventCount, event)

		case <-timeout:
			fmt.Printf("\n⏰ Таймаут ожидания. Получено событий: %d\n", eventCount)
			goto cleanup
		}
	}

cleanup:
	// 5. Останавливаем OwlWhisper
	fmt.Println("\n🛑 Остановка OwlWhisper...")
	C.StopOwlWhisper()
	fmt.Println("✅ OwlWhisper остановлен")

	fmt.Println("\n🎉 Тестирование завершено!")
	fmt.Println(strings.Repeat("=", 60))
}
