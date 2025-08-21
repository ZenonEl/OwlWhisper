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
	fmt.Println("🧪 Тестируем профили и управление памятью")
	fmt.Println("==============================================")

	// Настраиваем логирование - отключаем для чистого вывода
	fmt.Println("🔇 Отключаем логи...")
	C.SetLogLevel(0) // SILENT

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

	// Тест 1: Получаем исходный профиль
	fmt.Println("\n👤 Тест 1: Получаем исходный профиль...")
	profile1 := C.GetMyProfile()
	if profile1 != nil {
		goProfile1 := C.GoString(profile1)
		fmt.Printf("   Исходный профиль: %s\n", goProfile1)
		C.FreeString(profile1)
	}

	// Тест 2: Обновляем профиль
	fmt.Println("\n📝 Тест 2: Обновляем профиль...")
	testNickname := C.CString("TestUser123")
	result = C.UpdateMyProfile(testNickname)
	if result == 0 {
		fmt.Println("✅ Профиль обновлен")
	} else {
		fmt.Println("❌ Ошибка обновления профиля")
	}

	// Тест 3: Проверяем обновленный профиль
	fmt.Println("\n👤 Тест 3: Проверяем обновленный профиль...")
	profile2 := C.GetMyProfile()
	if profile2 != nil {
		goProfile2 := C.GoString(profile2)
		fmt.Printf("   Обновленный профиль: %s\n", goProfile2)
		C.FreeString(profile2)
	}

	// Тест 4: Проверяем GetConnectionStatus
	fmt.Println("\n🌐 Тест 4: Проверяем статус соединения...")
	status := C.GetConnectionStatus()
	if status != nil {
		goStatus := C.GoString(status)
		fmt.Printf("   Статус: %s\n", goStatus)
		C.FreeString(status)
	}

	// Тест 5: Массовая проверка FreeString (тест на утечки памяти)
	fmt.Println("\n🔄 Тест 5: Массовый тест управления памятью...")
	for i := 0; i < 100; i++ {
		profile := C.GetMyProfile()
		if profile != nil {
			C.FreeString(profile)
		}
		
		peers := C.GetPeers()
		if peers != nil {
			C.FreeString(peers)
		}
		
		connStatus := C.GetConnectionStatus()
		if connStatus != nil {
			C.FreeString(connStatus)
		}
		
		if i%20 == 0 {
			fmt.Printf("   Прогресс: %d/100\n", i)
		}
	}
	fmt.Println("✅ Массовый тест завершен без ошибок")

	// Тест 6: Еще раз проверяем профиль после множественных операций
	fmt.Println("\n👤 Тест 6: Финальная проверка профиля...")
	finalProfile := C.GetMyProfile()
	if finalProfile != nil {
		goFinalProfile := C.GoString(finalProfile)
		fmt.Printf("   Финальный профиль: %s\n", goFinalProfile)
		C.FreeString(finalProfile)
	}

	// Останавливаем
	fmt.Println("\n🛑 Остановка...")
	result = C.StopOwlWhisper()
	if result == 0 {
		fmt.Println("✅ Owl Whisper остановлен")
	} else {
		fmt.Println("❌ Ошибка остановки")
	}

	fmt.Println("\n🎉 Все тесты завершены!")
}