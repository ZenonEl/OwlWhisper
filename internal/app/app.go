package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"OwlWhisper/internal/core"
	"OwlWhisper/internal/tui"
)

// App представляет собой основное приложение
type App struct {
	coreController *core.CoreController
	tui            *tui.Handler
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewApp создает новое приложение
func NewApp() (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем Core контроллер
	coreController, err := core.NewCoreController(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать Core контроллер: %w", err)
	}

	// Создаем TUI обработчик
	tuiHandler := tui.NewHandler(coreController)

	app := &App{
		coreController: coreController,
		tui:            tuiHandler,
		ctx:            ctx,
		cancel:         cancel,
	}

	return app, nil
}

// Run запускает приложение
func (app *App) Run() error {
	// Запускаем Core контроллер
	if err := app.coreController.Start(); err != nil {
		return fmt.Errorf("не удалось запустить Core контроллер: %w", err)
	}

	// Обрабатываем сигналы для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем TUI в отдельной горутине
	go func() {
		if err := app.tui.Start(); err != nil {
			log.Printf("Ошибка TUI: %v", err)
			app.cancel()
		}
	}()

	// Ждем сигнала завершения
	<-sigChan
	log.Println("\n🛑 Получен сигнал завершения, останавливаем приложение...")

	// Graceful shutdown
	return app.Shutdown()
}

// Shutdown корректно останавливает приложение
func (app *App) Shutdown() error {
	// Останавливаем Core контроллер
	if err := app.coreController.Stop(); err != nil {
		log.Printf("⚠️ Ошибка остановки Core контроллера: %v", err)
	}

	// Отменяем контекст
	app.cancel()

	log.Println("👋 Приложение остановлено")
	return nil
}
