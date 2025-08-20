package controller

import (
	"bufio"
	"log"
	"os"
	"sync"

	"OwlWhisper/pkg/interfaces"
)

// Controller управляет TUI интерфейсом
type Controller struct {
	coreService interfaces.CoreService
	running     bool
	mu          sync.RWMutex
}

// NewController создает новый TUI контроллер
func NewController(coreService interfaces.CoreService) *Controller {
	return &Controller{
		coreService: coreService,
	}
}

// Start запускает TUI контроллер
func (c *Controller) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.mu.Unlock()

	log.Println("🦉 Добро пожаловать в Owl Whisper!")
	log.Println("🔗 P2P мессенджер с приоритетом на приватность")
	log.Println()
	log.Println("Доступные команды:")
	log.Println("  /help          - Показать справку")
	log.Println("  /peers         - Показать подключенных пиров")
	log.Println("  /status        - Показать статус сервиса")
	log.Println("  /quit          - Выйти из приложения")
	log.Println()
	log.Println("Просто введите сообщение для отправки всем подключенным пирам")
	log.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		message := scanner.Text()

		// Проверяем, что контроллер все еще запущен
		c.mu.RLock()
		if !c.running {
			c.mu.RUnlock()
			break
		}
		c.mu.RUnlock()

		// Обрабатываем команды
		if message == "/quit" {
			log.Println("👋 Выход из TUI...")
			break
		}

		if message == "/help" {
			c.showHelp()
			continue
		}

		if message == "/peers" {
			c.showPeers()
			continue
		}

		if message == "/status" {
			c.showStatus()
			continue
		}

		// Отправляем сообщение всем пирам
		if message != "" {
			// TODO: Реализовать отправку сообщений через CORE сервис
			log.Printf("📤 Сообщение: %s (пока не реализовано)", message)
		}
	}

	return scanner.Err()
}

// Stop останавливает TUI контроллер
func (c *Controller) Stop() {
	c.mu.Lock()
	c.running = false
	c.mu.Unlock()
}

// showHelp показывает справку
func (c *Controller) showHelp() {
	log.Println("📚 Справка по командам:")
	log.Println("  /help          - Показать эту справку")
	log.Println("  /peers         - Показать подключенных пиров")
	log.Println("  /status        - Показать статус сервиса")
	log.Println("  /quit          - Выйти из приложения")
	log.Println()
	log.Println("💡 Просто введите текст для отправки сообщения всем подключенным пирам")
}

// showPeers показывает список подключенных пиров
func (c *Controller) showPeers() {
	peers := c.coreService.Network().GetPeers()

	if len(peers) == 0 {
		log.Println("🔌 Нет подключенных пиров")
		return
	}

	log.Printf("🔌 Подключенные пиры (%d):", len(peers))
	for _, peer := range peers {
		log.Printf("  🟢 %s", peer.ShortString())
	}
}

// showStatus показывает статус сервиса
func (c *Controller) showStatus() {
	status := c.coreService.GetStatus()

	log.Println("📊 Статус сервиса:")
	log.Printf("  🚀 Запущен: %v", status.Running)
	log.Printf("  👥 Пиров: %d", status.PeersCount)
	log.Printf("  🌐 Тип сети: %s", status.NetworkType)
	log.Printf("  ⏱️  Время работы: %d сек", status.Uptime)
}
