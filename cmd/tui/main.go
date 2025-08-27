package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Инициализируем TUI приложение
	app := NewTUIApp()

	// Запускаем bubbletea
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal("Ошибка запуска TUI:", err)
		os.Exit(1)
	}
}

// NewTUIApp создает новое TUI приложение
func NewTUIApp() *TUIApp {
	return &TUIApp{
		core:          nil,
		contacts:      make(map[string]string), // nickname -> peerID
		nicknames:     make(map[string]string), // peerID -> nickname
		state:         StateInitializing,
		profile:       "",
		peerID:        "",
		discriminator: "",
	}
}
