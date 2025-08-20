package tui

import (
	"bufio"
	"log"
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
	log.Println("🦉 Добро пожаловать в Owl Whisper!")
	log.Println("🔗 P2P мессенджер с приоритетом на приватность")
	log.Println()
	log.Println("Доступные команды:")
	log.Println("  /help          - Показать справку")
	log.Println("  /peers         - Показать подключенных пиров")
	log.Println("  /quit          - Выйти из приложения")
	log.Println()
	log.Println("Просто введите сообщение для отправки всем подключенным пирам")
	log.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		message := scanner.Text()

		// Обрабатываем команды
		if message == "/quit" {
			log.Println("👋 Выход из приложения...")
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
				log.Printf("❌ Ошибка отправки: %v", err)
			} else {
				log.Printf("📤 Отправлено: %s", message)
			}
		}
	}

	return scanner.Err()
}

// showHelp показывает справку
func (h *Handler) showHelp() {
	log.Println("📚 Справка по командам:")
	log.Println("  /help          - Показать эту справку")
	log.Println("  /peers         - Показать подключенных пиров")
	log.Println("  /quit          - Выйти из приложения")
	log.Println()
	log.Println("💡 Просто введите текст для отправки сообщения всем подключенным пирам")
}

// showPeers показывает список подключенных пиров
func (h *Handler) showPeers() {
	h.mu.Lock()
	peers := h.controller.GetPeers()
	h.mu.Unlock()

	if len(peers) == 0 {
		log.Println("🔌 Нет подключенных пиров")
		return
	}

	log.Printf("🔌 Подключенные пиры (%d):", len(peers))
	for _, peer := range peers {
		log.Printf("  🟢 %s", peer.ShortString())
	}
}
