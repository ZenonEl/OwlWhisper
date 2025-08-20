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
	node      *core.Node
	discovery *core.DiscoveryManager
	tui       *tui.Handler
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewApp создает новое приложение
func NewApp() (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем узел
	node, err := core.NewNode(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать узел: %w", err)
	}

	// Создаем менеджер обнаружения
	discovery := core.NewDiscoveryManager(ctx, node.GetHost())

	// Создаем TUI обработчик
	tuiHandler := tui.NewHandler(node)

	app := &App{
		node:      node,
		discovery: discovery,
		tui:       tuiHandler,
		ctx:       ctx,
		cancel:    cancel,
	}

	return app, nil
}

// Run запускает приложение
func (app *App) Run() error {
	// Запускаем узел
	if err := app.node.Start(); err != nil {
		return fmt.Errorf("не удалось запустить узел: %w", err)
	}

	// Запускаем discovery
	if err := app.discovery.Start(); err != nil {
		return fmt.Errorf("не удалось запустить discovery: %w", err)
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
	// Останавливаем discovery
	if err := app.discovery.Stop(); err != nil {
		log.Printf("⚠️ Ошибка остановки discovery: %v", err)
	}

	// Останавливаем узел
	if err := app.node.Close(); err != nil {
		log.Printf("⚠️ Ошибка остановки узла: %v", err)
	}

	// Отменяем контекст
	app.cancel()

	log.Println("👋 Приложение остановлено")
	return nil
}
