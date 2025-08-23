package tui

import (
	"bufio"
	"os"
	"sync"

	"OwlWhisper/internal/core"
)

// Handler обрабатывает пользовательский ввод
type Handler struct {
	controller core.ICoreController
	mu         sync.Mutex
}

// NewHandler создает новый TUI обработчик
func NewHandler(controller core.ICoreController) *Handler {
	return &Handler{
		controller: controller,
	}
}

// Start запускает обработку пользовательского ввода
func (h *Handler) Start() error {
	core.Info("🦉 Добро пожаловать в Owl Whisper!")
	core.Info("🔗 P2P мессенджер с приоритетом на приватность")
	core.Info("")
	core.Info("Доступные команды:")
	core.Info("  /help          - Показать справку")
	core.Info("  /peers         - Показать подключенных пиров")
	core.Info("  /quit          - Выйти из приложения")
	core.Info("")
	core.Info("Просто введите сообщение для отправки всем подключенным пирам")
	core.Info("")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		message := scanner.Text()

		// Обрабатываем команды
		if message == "/quit" {
			core.Info("👋 Выход из приложения...")
			return nil
		}

		if message == "/help" {
			h.showHelp()
			continue
		}

		if message == "/peers" {
			h.showPeers()
			continue
		}

		// Отправляем сообщение всем пирам
		if message != "" {
			if err := h.controller.Broadcast([]byte(message)); err != nil {
				core.Error("❌ Ошибка отправки: %v", err)
			} else {
				core.Info("📤 Отправлено: %s", message)
			}
		}
	}

	return scanner.Err()
}

// showHelp показывает справку
func (h *Handler) showHelp() {
	core.Info("📚 Справка по командам:")
	core.Info("  /help          - Показать эту справку")
	core.Info("  /peers         - Показать подключенных пиров")
	core.Info("  /quit          - Выйти из приложения")
	core.Info("")
	core.Info("💡 Просто введите текст для отправки сообщения всем подключенным пирам")
}

// showPeers показывает список подключенных пиров
func (h *Handler) showPeers() {
	h.mu.Lock()
	peers := h.controller.GetConnectedPeers()
	h.mu.Unlock()

	if len(peers) == 0 {
		core.Info("🔌 Нет подключенных пиров")
		return
	}

	core.Info("🔌 Подключенные пиры (%d):", len(peers))
	for _, peer := range peers {
		core.Info("  🟢 %s", peer.ShortString())
	}
}
