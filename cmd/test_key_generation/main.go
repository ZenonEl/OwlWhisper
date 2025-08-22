package main

/*
#cgo CFLAGS: -I../../internal/core
#cgo LDFLAGS: -L../../dist -lowlwhisper
#include "owlwhisper.h"
#include <stdlib.h>
*/
import "C"
import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"unsafe"
)

func main() {
	fmt.Println("🔑 Тестируем генерацию ключей в Core")
	fmt.Println("=====================================")

	// Настраиваем логирование - отключаем для чистого вывода
	fmt.Println("🔇 Отключаем логи...")
	C.SetLogLevel(0) // SILENT

	// Тест 1: Генерация новой пары ключей
	fmt.Println("\n🔑 Тест 1: Генерация новой пары ключей...")
	keyPairData := C.GenerateNewKeyPair()
	if keyPairData == nil {
		fmt.Println("❌ Ошибка генерации ключей")
		return
	}

	// Получаем данные
	goKeyData := C.GoString(keyPairData)
	fmt.Printf("✅ Получены данные ключа: %s\n", goKeyData[:50]+"...")

	// Освобождаем память
	C.FreeString(keyPairData)

	// Декодируем base64
	jsonBytes, err := base64.StdEncoding.DecodeString(goKeyData)
	if err != nil {
		fmt.Printf("❌ Ошибка декодирования base64: %v\n", err)
		return
	}

	// Парсим JSON
	var keyInfo map[string]interface{}
	err = json.Unmarshal(jsonBytes, &keyInfo)
	if err != nil {
		fmt.Printf("❌ Ошибка парсинга JSON: %v\n", err)
		return
	}

	// Выводим информацию о ключе
	fmt.Println("\n📋 Информация о сгенерированном ключе:")
	fmt.Printf("   Тип ключа: %v\n", keyInfo["key_type"])
	fmt.Printf("   Длина ключа: %v байт\n", keyInfo["key_length"])
	fmt.Printf("   Peer ID: %v\n", keyInfo["peer_id"])

	// Тест 2: Запуск с сгенерированным ключом
	fmt.Println("\n🚀 Тест 2: Запуск с сгенерированным ключом...")
	
	// Получаем приватный ключ из JSON (он в base64)
	privateKeyBase64 := keyInfo["private_key"].(string)
	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		fmt.Printf("❌ Ошибка декодирования ключа: %v\n", err)
		return
	}
	
	// Запускаем с ключом
	result := C.StartOwlWhisperWithKey((*C.char)(unsafe.Pointer(&privateKeyBytes[0])), C.int(len(privateKeyBytes)))
	if result == 0 {
		fmt.Println("✅ Owl Whisper запущен с сгенерированным ключом")
		
		// Проверяем Peer ID
		peerID := C.GetMyPeerID()
		if peerID != nil {
			goPeerID := C.GoString(peerID)
			fmt.Printf("   Подтвержденный Peer ID: %s\n", goPeerID)
			C.FreeString(peerID)
		}
		
		// Останавливаем
		fmt.Println("\n🛑 Остановка...")
		stopResult := C.StopOwlWhisper()
		if stopResult == 0 {
			fmt.Println("✅ Owl Whisper остановлен")
		} else {
			fmt.Println("❌ Ошибка остановки")
		}
	} else {
		fmt.Println("❌ Ошибка запуска с сгенерированным ключом")
	}

	fmt.Println("\n🎉 Тест генерации ключей завершен!")
} 