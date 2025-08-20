package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"OwlWhisper/internal/core"
	"OwlWhisper/internal/tui/controller"
	"OwlWhisper/pkg/config"
	"OwlWhisper/pkg/interfaces"
)

// App представляет собой основное приложение
type App struct {
	coreService   interfaces.CoreService
	tuiController *controller.Controller
	config        *interfaces.Config

	ctx    context.Context
	cancel context.CancelFunc
}

// NewApp создает новое приложение
func NewApp() (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Загружаем конфигурацию
	cfg := config.DefaultConfig()

	// Создаем CORE сервис
	coreService, err := core.NewService(cfg)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать CORE сервис: %w", err)
	}

	// Создаем TUI контроллер (если включен)
	var tuiController *controller.Controller
	if cfg.UI.EnableTUI {
		tuiController = controller.NewController(coreService)
	}

	app := &App{
		coreService:   coreService,
		tuiController: tuiController,
		config:        cfg,
		ctx:           ctx,
		cancel:        cancel,
	}

	return app, nil
}

// Run запускает приложение
func (app *App) Run() error {
	// Запускаем CORE сервис
	if err := app.coreService.Start(app.ctx); err != nil {
		return fmt.Errorf("не удалось запустить CORE сервис: %w", err)
	}

	// Обрабатываем сигналы для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем TUI в отдельной горутине (если включен)
	if app.tuiController != nil {
		go func() {
			if err := app.tuiController.Start(); err != nil {
				log.Printf("Ошибка TUI: %v", err)
				app.cancel()
			}
		}()
	}

	// Ждем сигнала завершения
	<-sigChan
	log.Println("\n🛑 Получен сигнал завершения, останавливаем приложение...")

	// Graceful shutdown
	return app.Shutdown()
}

// Shutdown корректно останавливает приложение
func (app *App) Shutdown() error {
	// Останавливаем TUI
	if app.tuiController != nil {
		app.tuiController.Stop()
	}

	// Останавливаем CORE сервис
	if err := app.coreService.Stop(app.ctx); err != nil {
		log.Printf("⚠️ Ошибка остановки CORE сервиса: %v", err)
	}

	// Отменяем контекст
	app.cancel()

	log.Println("👋 Приложение остановлено")
	return nil
}
