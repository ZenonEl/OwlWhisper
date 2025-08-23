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
	fmt.Println("🔒 ТЕСТ ЗАЩИЩЕННЫХ ПИРОВ")
	fmt.Println("==========================")

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

	// Получаем список защищенных пиров (должен быть пустым)
	fmt.Println("\n🔒 Проверяем список защищенных пиров...")
	protectedPeers := C.GetProtectedPeers()
	if protectedPeers == nil {
		fmt.Println("❌ Не удалось получить список защищенных пиров")
		return
	}
	defer C.FreeString(protectedPeers)

	protectedList := C.GoString(protectedPeers)
	fmt.Printf("📋 Защищенные пиры: %s\n", protectedList)

	// Проверяем, является ли наш PeerID защищенным (должен быть false)
	fmt.Println("\n🔍 Проверяем, защищен ли наш PeerID...")
	isProtected := C.IsProtectedPeer(C.CString(myID))
	if isProtected == 1 {
		fmt.Println("❌ Наш PeerID не должен быть защищенным")
	} else {
		fmt.Println("✅ Наш PeerID не защищен (как и должно быть)")
	}

	// Добавляем наш PeerID в защищенные (тест)
	fmt.Println("\n🔒 Добавляем наш PeerID в защищенные...")
	addResult := C.AddProtectedPeer(C.CString(myID))
	if addResult != 0 {
		fmt.Println("❌ Не удалось добавить PeerID в защищенные")
	} else {
		fmt.Println("✅ PeerID добавлен в защищенные")
	}

	// Проверяем снова
	fmt.Println("\n🔍 Проверяем статус защиты...")
	isProtected = C.IsProtectedPeer(C.CString(myID))
	if isProtected == 1 {
		fmt.Println("✅ PeerID теперь защищен")
	} else {
		fmt.Println("❌ PeerID не защищен после добавления")
	}

	// Получаем обновленный список защищенных пиров
	fmt.Println("\n📋 Получаем обновленный список защищенных пиров...")
	protectedPeers = C.GetProtectedPeers()
	if protectedPeers == nil {
		fmt.Println("❌ Не удалось получить обновленный список")
		return
	}
	defer C.FreeString(protectedPeers)

	protectedList = C.GoString(protectedPeers)
	fmt.Printf("📋 Защищенные пиры: %s\n", protectedList)

	// Удаляем PeerID из защищенных
	fmt.Println("\n🔓 Удаляем PeerID из защищенных...")
	removeResult := C.RemoveProtectedPeer(C.CString(myID))
	if removeResult != 0 {
		fmt.Println("❌ Не удалось удалить PeerID из защищенных")
	} else {
		fmt.Println("✅ PeerID удален из защищенных")
	}

	// Финальная проверка
	fmt.Println("\n🔍 Финальная проверка статуса...")
	isProtected = C.IsProtectedPeer(C.CString(myID))
	if isProtected == 1 {
		fmt.Println("❌ PeerID все еще защищен после удаления")
	} else {
		fmt.Println("✅ PeerID больше не защищен")
	}

	// Получаем финальный список
	fmt.Println("\n📋 Финальный список защищенных пиров...")
	protectedPeers = C.GetProtectedPeers()
	if protectedPeers == nil {
		fmt.Println("❌ Не удалось получить финальный список")
		return
	}
	defer C.FreeString(protectedPeers)

	protectedList = C.GoString(protectedPeers)
	fmt.Printf("📋 Защищенные пиры: %s\n", protectedList)

	// Останавливаем OwlWhisper
	fmt.Println("\n🛑 Останавливаем OwlWhisper...")
	C.StopOwlWhisper()
	fmt.Println("✅ OwlWhisper остановлен")

	fmt.Println("\n🎉 Тест защищенных пиров завершен!")
}
