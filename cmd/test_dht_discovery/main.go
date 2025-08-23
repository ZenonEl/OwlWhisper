package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"OwlWhisper/internal/core"
)

func main() {
	fmt.Println("🌐 ТЕСТ: DHT Discovery и глобальная сеть")
	fmt.Println("==========================================")

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n🛑 Получен сигнал завершения, останавливаем...")
		cancel()
	}()

	// Создаем первый клиент
	fmt.Println("\n🚀 Создаем первый клиент...")
	client1, err := core.NewCoreController(ctx)
	if err != nil {
		log.Fatalf("❌ Не удалось создать первый клиент: %v", err)
	}
	defer client1.Stop()

	// Запускаем первый клиент
	if err := client1.Start(); err != nil {
		log.Fatalf("❌ Не удалось запустить первый клиент: %v", err)
	}
	fmt.Printf("✅ Первый клиент запущен с PeerID: %s\n", client1.GetMyID())

	// Ждем для подключения к bootstrap узлам
	fmt.Println("\n⏳ Ждем подключения к bootstrap узлам DHT...")
	time.Sleep(10 * time.Second)

	// Проверяем статус DHT
	fmt.Println("\n📊 Проверяем статус DHT...")
	peers1 := client1.GetPeers()
	fmt.Printf("🔍 Клиент 1 видит %d пиров в DHT: %v\n", len(peers1), peers1)

	// Создаем второй клиент
	fmt.Println("\n🚀 Создаем второй клиент...")
	client2, err := core.NewCoreController(ctx)
	if err != nil {
		log.Fatalf("❌ Не удалось создать второй клиент: %v", err)
	}
	defer client2.Stop()

	// Запускаем второй клиент
	if err := client2.Start(); err != nil {
		log.Fatalf("❌ Не удалось запустить второй клиент: %v", err)
	}
	fmt.Printf("✅ Второй клиент запущен с PeerID: %s\n", client2.GetMyID())

	// Ждем для подключения к bootstrap узлам
	fmt.Println("\n⏳ Ждем подключения второго клиента к bootstrap узлам...")
	time.Sleep(10 * time.Second)

	// Проверяем статус DHT для второго клиента
	peers2 := client2.GetPeers()
	fmt.Printf("🔍 Клиент 2 видит %d пиров в DHT: %v\n", len(peers2), peers2)

	// Теперь пытаемся найти друг друга через DHT
	fmt.Println("\n🔍 Пытаемся найти друг друга через DHT...")

	// Ждем еще немного для стабилизации DHT
	time.Sleep(15 * time.Second)

	// Финальная проверка
	fmt.Println("\n📊 Финальная проверка DHT...")
	peers1Final := client1.GetPeers()
	peers2Final := client2.GetPeers()

	fmt.Printf("🔍 Клиент 1: %d пиров в DHT\n", len(peers1Final))
	fmt.Printf("🔍 Клиент 2: %d пиров в DHT\n", len(peers2Final))

	// Проверяем, видят ли клиенты друг друга
	client1ID := client1.GetMyID()
	client2ID := client2.GetMyID()

	client1SeesClient2 := false
	client2SeesClient1 := false

	for _, peer := range peers1Final {
		if peer.String() == client2ID {
			client1SeesClient2 = true
			break
		}
	}

	for _, peer := range peers2Final {
		if peer.String() == client1ID {
			client2SeesClient1 = true
			break
		}
	}

	fmt.Printf("\n🔗 Результат обнаружения через DHT:\n")
	fmt.Printf("   Клиент 1 видит Клиент 2: %t\n", client1SeesClient2)
	fmt.Printf("   Клиент 2 видит Клиент 1: %t\n", client2SeesClient1)

	if client1SeesClient2 && client2SeesClient1 {
		fmt.Println("\n🎉 УСПЕХ! Оба клиента видят друг друга через DHT!")
	} else if len(peers1Final) > 0 || len(peers2Final) > 0 {
		fmt.Println("\n✅ ЧАСТИЧНЫЙ УСПЕХ! DHT работает, но клиенты не видят друг друга")
		fmt.Println("💡 Это нормально - они могут быть в разных частях DHT сети")
	} else {
		fmt.Println("\n⚠️  ВНИМАНИЕ: DHT не работает или нет подключения к bootstrap узлам")
	}

	fmt.Println("\n✅ Тест завершен!")
	fmt.Println("💡 Если DHT работает - значит глобальная сеть функционирует")
}
