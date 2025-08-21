package main

import (
	"crypto/rand"
	"fmt"
	"time"
	"unsafe"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// #cgo CFLAGS: -I../../internal/core
// #cgo LDFLAGS: -L../../dist -lowlwhisper
// #include "owlwhisper.h"
import "C"

func main() {
	fmt.Println("🔑 Тестируем инъекцию ключей в Core")
	fmt.Println("=====================================")

	// Настраиваем логирование - отключаем для чистого вывода
	fmt.Println("🔇 Отключаем логи...")
	C.SetLogLevel(0) // SILENT

	// Тест 1: Создание нового ключа
	fmt.Println("\n🔑 Тест 1: Создание нового ключа...")
	newPrivKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		fmt.Printf("❌ Ошибка генерации ключа: %v\n", err)
		return
	}

	// Сериализуем ключ в байты
	keyBytes, err := crypto.MarshalPrivateKey(newPrivKey)
	if err != nil {
		fmt.Printf("❌ Ошибка сериализации ключа: %v\n", err)
		return
	}

	fmt.Printf("✅ Сгенерирован новый ключ размером %d байт\n", len(keyBytes))

	// Тест 2: Запуск с переданным ключом
	fmt.Println("\n🚀 Тест 2: Запуск с переданным ключом...")
	result := C.StartOwlWhisperWithKey((*C.char)(unsafe.Pointer(&keyBytes[0])), C.int(len(keyBytes)))
	if result == 0 {
		fmt.Println("✅ Owl Whisper запущен с переданным ключом")
	} else {
		fmt.Println("❌ Ошибка запуска с ключом")
		return
	}

	// Ждем немного для инициализации
	time.Sleep(2 * time.Second)

	// Тест 3: Проверка Peer ID
	fmt.Println("\n👤 Тест 3: Проверка Peer ID...")
	peerID := C.GetMyPeerID()
	if peerID != nil {
		goPeerID := C.GoString(peerID)
		fmt.Printf("   Peer ID: %s\n", goPeerID)
		C.FreeString(peerID)
	}

	// Тест 4: Проверка статуса соединения
	fmt.Println("\n🌐 Тест 4: Проверка статуса соединения...")
	status := C.GetConnectionStatus()
	if status != nil {
		goStatus := C.GoString(status)
		fmt.Printf("   Статус: %s\n", goStatus)
		C.FreeString(status)
	}

	// Тест 5: Проверка профиля
	fmt.Println("\n👤 Тест 5: Проверка профиля...")
	profile := C.GetMyProfile()
	if profile != nil {
		goProfile := C.GoString(profile)
		fmt.Printf("   Профиль: %s\n", goProfile)
		C.FreeString(profile)
	}

	// Тест 6: Обновление профиля
	fmt.Println("\n📝 Тест 6: Обновление профиля...")
	testNickname := C.CString("KeyInjectionTest")
	result = C.UpdateMyProfile(testNickname)
	if result == 0 {
		fmt.Println("✅ Профиль обновлен")
	} else {
		fmt.Println("❌ Ошибка обновления профиля")
	}

	// Тест 7: Проверка обновленного профиля
	fmt.Println("\n👤 Тест 7: Проверка обновленного профиля...")
	updatedProfile := C.GetMyProfile()
	if updatedProfile != nil {
		goUpdatedProfile := C.GoString(updatedProfile)
		fmt.Printf("   Обновленный профиль: %s\n", goUpdatedProfile)
		C.FreeString(updatedProfile)
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
