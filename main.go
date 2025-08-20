package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"OwlWhisper/internal/core"
	"OwlWhisper/internal/storage"
	"OwlWhisper/internal/transport"
	"OwlWhisper/internal/ui"
	"OwlWhisper/pkg/config"
)

func main() {
	// Парсим флаги командной строки
	configPath := flag.String("config", "", "Путь к файлу конфигурации")
	listenPort := flag.Int("port", 0, "Порт для прослушивания (0 = автоматический)")
	destAddr := flag.String("d", "", "Адрес для прямого подключения")
	dbPath := flag.String("db", "", "Путь к базе данных SQLite")
	flag.Parse()

	// Загружаем конфигурацию
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Printf("Warning: failed to load config: %v, using defaults", err)
		cfg = config.DefaultConfig()
	}

	// Переопределяем порт если указан в командной строке
	if *listenPort != 0 {
		cfg.Network.ListenPort = *listenPort
	}

	// Создаем директорию для данных приложения
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	appDir := filepath.Join(homeDir, ".owlwhisper")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		log.Fatalf("Failed to create app directory: %v", err)
	}

	// Определяем путь к базе данных
	if *dbPath == "" {
		*dbPath = filepath.Join(appDir, "owlwhisper.db")
	}

	// Создаем репозиторий для хранения данных
	log.Printf("📁 Используем базу данных: %s", *dbPath)
	storageRepo, err := storage.NewSQLiteRepository(*dbPath)
	if err != nil {
		log.Fatalf("Failed to create storage repository: %v", err)
	}
	defer storageRepo.Close()

	// Создаем транспортный слой
	log.Printf("🌐 Создаем транспортный слой...")
	transportLayer, err := transport.NewLibp2pTransport(
		cfg.Network.ListenPort,
		cfg.Security.EnableTLS,
		cfg.Security.EnableNoise,
		cfg.Network.EnableNAT,
		cfg.Network.EnableHolePunch,
		cfg.Network.EnableRelay,
	)
	if err != nil {
		log.Fatalf("Failed to create transport layer: %v", err)
	}

	// Создаем сервисы
	log.Printf("🔧 Создаем core сервисы...")
	chatService := core.NewChatService(storageRepo, transportLayer)
	contactService := core.NewContactService(storageRepo, transportLayer)
	networkService := core.NewNetworkService(transportLayer, contactService, chatService)

	// Создаем TUI интерфейс
	log.Printf("🖥️ Создаем TUI интерфейс...")
	tuiChat := ui.NewTUIChat(chatService, contactService, networkService)

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обрабатываем сигналы для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем приложение в фоне
	go func() {
		if err := tuiChat.Start(); err != nil {
			log.Printf("Error starting TUI: %v", err)
			cancel()
		}
	}()

	// Если указан адрес для прямого подключения, подключаемся
	if *destAddr != "" {
		go func() {
			// Ждем немного, чтобы основной узел успел запуститься
			select {
			case <-time.After(5 * time.Second):
				log.Printf("🔗 Попытка прямого подключения к %s", *destAddr)
				if err := transportLayer.ConnectDirectly(context.Background(), *destAddr); err != nil {
					log.Printf("❌ Ошибка прямого подключения: %v", err)
				} else {
					log.Printf("✅ Прямое подключение установлено")
				}
			case <-ctx.Done():
				return
			}
		}()
	}

	// Ждем сигнала завершения
	<-sigChan
	log.Println("\n🛑 Получен сигнал завершения, останавливаем приложение...")

	// Останавливаем все сервисы
	cancel()

	// Останавливаем сетевой сервис
	if err := networkService.Stop(context.Background()); err != nil {
		log.Printf("Warning: failed to stop network service: %v", err)
	}

	log.Println("👋 Приложение остановлено")
}
