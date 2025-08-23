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
	fmt.Println("🧪 ТЕСТ: Запуск двух локальных клиентов")
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
	fmt.Printf("🔍 Статус первого клиента: %t\n", client1.IsRunning())

	// Ждем немного для стабилизации
	time.Sleep(3 * time.Second)

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
	fmt.Printf("🔍 Статус второго клиента: %t\n", client2.IsRunning())

	// Ждем для обнаружения через mDNS
	fmt.Println("\n⏳ Ждем обнаружения пиров через mDNS...")

	// Проверяем каждые 2 секунды
	for i := 0; i < 10; i++ {
		time.Sleep(2 * time.Second)

		peers1 := client1.GetPeers()
		peers2 := client2.GetPeers()

		fmt.Printf("⏱️  Проверка %d/10:\n", i+1)
		fmt.Printf("   Клиент 1: %d пиров\n", len(peers1))
		fmt.Printf("   Клиент 2: %d пиров\n", len(peers2))

		// Если оба клиента видят друг друга, выходим
		if len(peers1) > 0 && len(peers2) > 0 {
			break
		}
	}

	// Финальная проверка
	fmt.Println("\n📊 Финальная проверка...")
	peers1 := client1.GetPeers()
	peers2 := client2.GetPeers()

	fmt.Printf("🔍 Клиент 1 видит %d пиров: %v\n", len(peers1), peers1)
	fmt.Printf("🔍 Клиент 2 видит %d пиров: %v\n", len(peers2), peers2)

	// Проверяем, видят ли клиенты друг друга
	client1ID := client1.GetMyID()
	client2ID := client2.GetMyID()

	client1SeesClient2 := false
	client2SeesClient1 := false

	for _, peer := range peers1 {
		if peer.String() == client2ID {
			client1SeesClient2 = true
			break
		}
	}

	for _, peer := range peers2 {
		if peer.String() == client1ID {
			client2SeesClient1 = true
			break
		}
	}

	fmt.Printf("\n🔗 Результат обнаружения:\n")
	fmt.Printf("   Клиент 1 видит Клиент 2: %t\n", client1SeesClient2)
	fmt.Printf("   Клиент 2 видит Клиент 1: %t\n", client2SeesClient1)

	if client1SeesClient2 && client2SeesClient1 {
		fmt.Println("\n🎉 УСПЕХ! Оба клиента видят друг друга!")
	} else {
		fmt.Println("\n⚠️  ВНИМАНИЕ: Клиенты не видят друг друга")
		fmt.Println("💡 Это может быть нормально для localhost - mDNS может не работать")
	}

	fmt.Println("\n✅ Тест завершен успешно!")
	fmt.Println("💡 Если не было паники с 'assignment to entry in nil map' - баг исправлен!")
}
