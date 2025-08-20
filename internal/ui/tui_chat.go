package ui

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"OwlWhisper/pkg/interfaces"

	"github.com/libp2p/go-libp2p/core/peer"
)

// TUIChat представляет TUI интерфейс для чата
type TUIChat struct {
	chatService    interfaces.IChatService
	contactService interfaces.IContactService
	networkService interfaces.INetworkService
	ctx            context.Context
	cancel         context.CancelFunc
	scanner        *bufio.Scanner
}

// NewTUIChat создает новый экземпляр TUIChat
func NewTUIChat(
	chatService interfaces.IChatService,
	contactService interfaces.IContactService,
	networkService interfaces.INetworkService,
) *TUIChat {
	ctx, cancel := context.WithCancel(context.Background())

	return &TUIChat{
		chatService:    chatService,
		contactService: contactService,
		networkService: networkService,
		ctx:            ctx,
		cancel:         cancel,
		scanner:        bufio.NewScanner(os.Stdin),
	}
}

// Start запускает TUI интерфейс
func (t *TUIChat) Start() error {
	// Запускаем сетевой сервис
	if err := t.networkService.Start(t.ctx); err != nil {
		return fmt.Errorf("failed to start network service: %w", err)
	}

	// Обрабатываем сигналы для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем обработку команд в фоне
	go t.handleCommands()

	// Показываем приветственное сообщение
	t.showWelcome()

	// Ждем сигнала завершения
	<-sigChan
	log.Println("\n🛑 Получен сигнал завершения, останавливаем приложение...")

	// Останавливаем сетевой сервис
	if err := t.networkService.Stop(t.ctx); err != nil {
		log.Printf("Warning: failed to stop network service: %v", err)
	}

	t.cancel()
	return nil
}

// showWelcome показывает приветственное сообщение
func (t *TUIChat) showWelcome() {
	fmt.Println("🦉 Добро пожаловать в Owl Whisper!")
	fmt.Println("🔗 P2P мессенджер с приоритетом на приватность")
	fmt.Println("")
	fmt.Println("Доступные команды:")
	fmt.Println("  /help          - Показать справку")
	fmt.Println("  /contacts      - Показать контакты")
	fmt.Println("  /connect <id>  - Подключиться к пиру")
	fmt.Println("  /msg <id>      - Отправить сообщение")
	fmt.Println("  /history <id>  - Показать историю сообщений")
	fmt.Println("  /peers         - Показать подключенных пиров")
	fmt.Println("  /quit          - Выйти из приложения")
	fmt.Println("")
	fmt.Println("Просто введите сообщение для отправки всем подключенным пирам")
	fmt.Println("")
}

// handleCommands обрабатывает команды пользователя
func (t *TUIChat) handleCommands() {
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			fmt.Print("🦉 > ")
			if !t.scanner.Scan() {
				return
			}

			input := strings.TrimSpace(t.scanner.Text())
			if input == "" {
				continue
			}

			// Обрабатываем команды
			if strings.HasPrefix(input, "/") {
				t.handleCommand(input)
			} else {
				// Отправляем сообщение всем подключенным пирам
				t.broadcastMessage(input)
			}
		}
	}
}

// handleCommand обрабатывает команду
func (t *TUIChat) handleCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "/help":
		t.showWelcome()

	case "/contacts":
		t.showContacts()

	case "/connect":
		if len(parts) < 2 {
			fmt.Println("❌ Использование: /connect <peer_id>")
			return
		}
		t.connectToPeer(parts[1])

	case "/msg":
		if len(parts) < 3 {
			fmt.Println("❌ Использование: /msg <peer_id> <сообщение>")
			return
		}
		peerID := parts[1]
		message := strings.Join(parts[2:], " ")
		t.sendMessage(peerID, message)

	case "/history":
		if len(parts) < 2 {
			fmt.Println("❌ Использование: /history <peer_id>")
			return
		}
		t.showMessageHistory(parts[1])

	case "/peers":
		t.showConnectedPeers()

	case "/quit":
		fmt.Println("👋 До свидания!")
		t.cancel()

	default:
		fmt.Printf("❌ Неизвестная команда: %s\n", parts[0])
		fmt.Println("Введите /help для справки")
	}
}

// showContacts показывает список контактов
func (t *TUIChat) showContacts() {
	contacts, err := t.contactService.GetContacts(t.ctx)
	if err != nil {
		fmt.Printf("❌ Ошибка получения контактов: %v\n", err)
		return
	}

	if len(contacts) == 0 {
		fmt.Println("📝 Контакты не найдены")
		return
	}

	fmt.Println("📝 Контакты:")
	for _, contact := range contacts {
		status := "🔴"
		if contact.IsOnline {
			status = "🟢"
		}
		fmt.Printf("  %s %s (%s) - %s\n",
			status,
			contact.Nickname,
			contact.PeerID[:12]+"...",
			contact.LastSeen.Format("15:04:05"),
		)
	}
}

// connectToPeer подключается к указанному пиру
func (t *TUIChat) connectToPeer(peerIDStr string) {
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		fmt.Printf("❌ Неверный формат PeerID: %v\n", err)
		return
	}

	if err := t.networkService.ConnectToPeer(t.ctx, peerID); err != nil {
		fmt.Printf("❌ Ошибка подключения: %v\n", err)
		return
	}

	fmt.Printf("✅ Подключились к %s\n", peerID.ShortString())
}

// sendMessage отправляет сообщение указанному пиру
func (t *TUIChat) sendMessage(peerIDStr, message string) {
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		fmt.Printf("❌ Неверный формат PeerID: %v\n", err)
		return
	}

	if err := t.chatService.SendMessage(t.ctx, peerID, message); err != nil {
		fmt.Printf("❌ Ошибка отправки сообщения: %v\n", err)
		return
	}

	fmt.Printf("📤 Сообщение отправлено к %s\n", peerID.ShortString())
}

// showMessageHistory показывает историю сообщений с указанным пиром
func (t *TUIChat) showMessageHistory(peerIDStr string) {
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		fmt.Printf("❌ Неверный формат PeerID: %v\n", err)
		return
	}

	messages, err := t.chatService.GetMessages(t.ctx, peerID, 20, 0)
	if err != nil {
		fmt.Printf("❌ Ошибка получения истории: %v\n", err)
		return
	}

	if len(messages) == 0 {
		fmt.Println("📜 История сообщений пуста")
		return
	}

	fmt.Printf("📜 История сообщений с %s:\n", peerID.ShortString())
	for _, msg := range messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		if msg.FromPeer == peerID.String() {
			fmt.Printf("  [%s] %s: %s\n", timestamp, "Они", msg.Content)
		} else {
			fmt.Printf("  [%s] %s: %s\n", timestamp, "Вы", msg.Content)
		}
	}
}

// showConnectedPeers показывает подключенных пиров
func (t *TUIChat) showConnectedPeers() {
	peers := t.networkService.GetConnectedPeers()

	if len(peers) == 0 {
		fmt.Println("🔌 Нет подключенных пиров")
		return
	}

	fmt.Println("🔌 Подключенные пиры:")
	for _, peerID := range peers {
		contact := t.networkService.GetPeerInfo(peerID)
		if contact != nil {
			fmt.Printf("  🟢 %s (%s)\n", contact.Nickname, peerID.ShortString())
		} else {
			fmt.Printf("  🟢 %s\n", peerID.ShortString())
		}
	}
}

// broadcastMessage отправляет сообщение всем подключенным пирам
func (t *TUIChat) broadcastMessage(message string) {
	peers := t.networkService.GetConnectedPeers()

	if len(peers) == 0 {
		fmt.Println("❌ Нет подключенных пиров для отправки сообщения")
		return
	}

	successCount := 0
	for _, peerID := range peers {
		if err := t.chatService.SendMessage(t.ctx, peerID, message); err != nil {
			fmt.Printf("❌ Ошибка отправки к %s: %v\n", peerID.ShortString(), err)
		} else {
			successCount++
		}
	}

	if successCount > 0 {
		fmt.Printf("📤 Сообщение отправлено %d пирам\n", successCount)
	}
}
