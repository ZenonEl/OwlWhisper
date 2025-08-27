package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"OwlWhisper/internal/core"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

// Состояния приложения
type AppState int

const (
	StateInitializing AppState = iota
	StateProfileInput
	StateCoreStarting
	StateReady
	StateError
)

// TUIApp основное приложение
type TUIApp struct {
	core          *core.CoreController
	contacts      map[string]string // nickname -> peerID
	nicknames     map[string]string // peerID -> nickname
	state         AppState
	profile       string
	peerID        string
	discriminator string
	contentID     string // ContentID для поиска
	errorMsg      string
	inputBuffer   string
	commandMode   bool
	outputLines   []string
}

// String возвращает строковое представление состояния
func (s AppState) String() string {
	switch s {
	case StateInitializing:
		return "Инициализация"
	case StateProfileInput:
		return "Ввод профиля"
	case StateCoreStarting:
		return "Запуск Core"
	case StateReady:
		return "Готов"
	case StateError:
		return "Ошибка"
	default:
		return "Неизвестно"
	}
}

// Init инициализирует приложение
func (a *TUIApp) Init() tea.Cmd {
	// Сразу переходим к вводу профиля
	a.state = StateProfileInput
	return tea.EnterAltScreen
}

// handleKeyPress обрабатывает нажатия клавиш
func (a *TUIApp) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.state {
	case StateProfileInput:
		return a.handleProfileInput(msg)
	case StateReady:
		return a.handleCommandInput(msg)
	case StateError:
		if msg.String() != "" {
			return a, tea.Quit
		}
	}
	return a, nil
}

// handleProfileInput обрабатывает ввод профиля
func (a *TUIApp) handleProfileInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		trimmed := strings.TrimSpace(a.inputBuffer)
		if trimmed != "" {
			a.profile = trimmed
			a.state = StateCoreStarting
			return a, a.startCore()
		}
		// Если ввод пустой, не делаем ничего
		return a, nil
	case "backspace":
		if len(a.inputBuffer) > 0 {
			a.inputBuffer = a.inputBuffer[:len(a.inputBuffer)-1]
		}
	case "ctrl+v":
		// Вставка из буфера обмена
		return a, a.pasteFromClipboard()
	case "ctrl+c":
		return a, tea.Quit
	default:
		if len(msg.String()) == 1 {
			a.inputBuffer += msg.String()
		}
	}
	return a, nil
}

// handleCommandInput обрабатывает ввод команд
func (a *TUIApp) handleCommandInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		trimmed := strings.TrimSpace(a.inputBuffer)
		if trimmed != "" {
			// Выполняем команду и очищаем буфер
			cmd := a.executeCommand(trimmed)
			a.inputBuffer = ""
			return a, cmd
		}
		// Если ввод пустой, не делаем ничего
		return a, nil
	case "backspace":
		if len(a.inputBuffer) > 0 {
			a.inputBuffer = a.inputBuffer[:len(a.inputBuffer)-1]
		}
	case "ctrl+v":
		// Вставка из буфера обмена
		return a, a.pasteFromClipboard()
	case "ctrl+c":
		return a, tea.Quit
	default:
		if len(msg.String()) == 1 {
			a.inputBuffer += msg.String()
		}
	}
	return a, nil
}

// Update обрабатывает сообщения
func (a *TUIApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		return a, nil
	case errorMsg:
		a.errorMsg = msg.err.Error()
		a.state = StateError
		return a, nil
	case coreStartedMsg:
		a.core = msg.controller
		a.peerID = msg.peerID
		a.discriminator = msg.discriminator
		a.contentID = msg.contentID // Обновляем contentID
		a.state = StateReady
		return a, nil
	case outputMsg:
		a.outputLines = append(a.outputLines, msg.line)
		// Ограничиваем количество строк вывода
		if len(a.outputLines) > 20 {
			a.outputLines = a.outputLines[len(a.outputLines)-20:]
		}
		return a, nil
	case pasteMsg:
		// Вставляем текст из буфера
		a.inputBuffer += msg.text
		return a, nil
	}
	return a, nil
}

// View отображает интерфейс
func (a *TUIApp) View() string {
	switch a.state {
	case StateInitializing:
		return a.renderInitializing()
	case StateProfileInput:
		return a.renderProfileInput()
	case StateCoreStarting:
		return a.renderCoreStarting()
	case StateReady:
		return a.renderReady()
	case StateError:
		return a.renderError()
	default:
		return "Неизвестное состояние"
	}
}

// renderInitializing отображает экран инициализации
func (a *TUIApp) renderInitializing() string {
	return `
🦉 Owl Whisper TUI Client
========================

Инициализация...
`
}

// renderProfileInput отображает экран ввода профиля
func (a *TUIApp) renderProfileInput() string {
	return fmt.Sprintf(`
🦉 Owl Whisper TUI Client
========================

Введите имя профиля: %s█
(Нажмите Enter для подтверждения)

`, a.inputBuffer)
}

// renderCoreStarting отображает экран запуска Core
func (a *TUIApp) renderCoreStarting() string {
	return fmt.Sprintf(`
🦉 Owl Whisper TUI Client
========================

Профиль: %s
Запуск Core...
`, a.profile)
}

// renderReady отображает основной экран
func (a *TUIApp) renderReady() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`
🦉 Owl Whisper TUI Client
========================

Профиль: %s#%s
Peer ID: %s
Content ID: %s
Состояние: %s

`, a.profile, a.discriminator, a.peerID, a.contentID, a.state.String()))

	// Команды
	sb.WriteString(`
Команды:
/help - Справка
/peers - Список пиров
/contacts - Список контактов
/find id <peer_id> - Поиск по Peer ID
/find name <nickname#discriminator> - Поиск по никнейму
/add <peer_id> <nickname> - Добавить контакт
/msg <nickname> <текст> - Отправить сообщение
/network - Статус сети и технические пиры
/status - Статус анонсирования и поиска
/diag - Полная диагностика системы
/quit - Выход

`)

	// Вывод
	if len(a.outputLines) > 0 {
		sb.WriteString("Вывод:\n")
		for _, line := range a.outputLines {
			sb.WriteString(line + "\n")
		}
	}

	// Ввод команды
	sb.WriteString(fmt.Sprintf("Введите команду (начните с /): %s█", a.inputBuffer))

	return sb.String()
}

// renderError отображает экран ошибки
func (a *TUIApp) renderError() string {
	return fmt.Sprintf(`
🦉 Owl Whisper TUI Client
========================

❌ Ошибка: %s

Нажмите любую клавишу для выхода...
`, a.errorMsg)
}

// startCore запускает Core
func (a *TUIApp) startCore() tea.Cmd {
	return func() tea.Msg {
		// Создаем новый ключ для профиля
		keyBytes, err := a.generateNewKeyBytes()
		if err != nil {
			return errorMsg{err: fmt.Errorf("ошибка генерации ключа: %w", err)}
		}

		// Создаем Core контроллер
		ctx := context.Background()
		controller, err := core.NewCoreControllerWithKeyBytes(ctx, keyBytes)
		if err != nil {
			return errorMsg{err: fmt.Errorf("ошибка создания Core: %w", err)}
		}

		// Запускаем Core
		if err := controller.Start(); err != nil {
			return errorMsg{err: fmt.Errorf("ошибка запуска Core: %w", err)}
		}

		// Получаем Peer ID
		peerID := controller.GetMyID()

		// Вычисляем дискриминатор
		discriminator := ""
		if len(peerID) >= 6 {
			discriminator = peerID[len(peerID)-6:]
		}

		// Анонсируем профиль в DHT
		profileContentID := a.computeContentID(a.profile + "#" + discriminator)
		fmt.Printf("🔍 Генерируем ContentID для профиля: %s#%s\n", a.profile, discriminator)
		fmt.Printf("🔑 Вычисленный ContentID: %s\n", profileContentID)
		fmt.Printf("📢 Начинаем анонсирование в DHT...\n")

		// Детальная диагностика перед анонсированием
		fmt.Printf("🔍 Детальная диагностика:\n")
		fmt.Printf("  - Peer ID: %s\n", peerID)
		fmt.Printf("  - Discriminator: %s\n", discriminator)
		fmt.Printf("  - Content ID: %s\n", profileContentID)
		fmt.Printf("  - Timestamp: %s\n", time.Now().Format("15:04:05"))

		// Анонсируем в DHT
		fmt.Printf("📡 Вызываем controller.ProvideContent...\n")
		err = controller.ProvideContent(profileContentID)
		if err != nil {
			// Не критично - продолжаем работу
			fmt.Printf("❌ Ошибка анонсирования в DHT: %v\n", err)
			fmt.Printf("🔍 Детали ошибки: %T: %v\n", err, err)
		} else {
			fmt.Printf("✅ Успешно анонсирован в DHT!\n")
			fmt.Printf("🌐 Теперь другие пиры могут найти вас по ContentID: %s\n", profileContentID)
		}

		// Проверяем статус DHT после анонсирования
		fmt.Printf("🔍 Проверяем статус DHT после анонсирования...\n")
		if dhtSize := controller.GetDHTRoutingTableSize(); dhtSize > 0 {
			fmt.Printf("📊 DHT Routing Table: %d пиров\n", dhtSize)
		} else {
			fmt.Printf("⚠️ DHT Routing Table пуста\n")
		}

		// Успешно запустили Core
		return coreStartedMsg{
			controller:    controller,
			peerID:        peerID,
			discriminator: discriminator,
			contentID:     profileContentID,
		}
	}
}

// cmdStatus показывает статус анонсирования и поиска
func (a *TUIApp) cmdStatus() tea.Cmd {
	return func() tea.Msg {
		debugMsg := "Выполняю команду /status"

		if a.core == nil {
			return outputMsg{line: debugMsg + "\n❌ Core не запущен"}
		}

		var sb strings.Builder
		sb.WriteString(debugMsg + "\n📊 Статус анонсирования:\n")

		// Информация о профиле
		sb.WriteString(fmt.Sprintf("👤 Профиль: %s#%s\n", a.profile, a.discriminator))
		sb.WriteString(fmt.Sprintf("🆔 Peer ID: %s\n", a.peerID))
		sb.WriteString(fmt.Sprintf("🔑 Content ID: %s\n", a.contentID))

		// Проверяем статус DHT
		host := a.core.GetHost()
		if host != nil {
			// Получаем статистику сети
			stats := a.core.GetNetworkStats()
			if len(stats) > 0 {
				sb.WriteString("\n🌐 Статус DHT:\n")
				for key, value := range stats {
					if key == "dht" {
						sb.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
					}
				}
			}
		}

		// Проверяем анонсирование
		sb.WriteString("\n📢 Проверка анонсирования:\n")
		sb.WriteString("  Попробуйте найти себя командой:\n")
		sb.WriteString(fmt.Sprintf("  /find name %s#%s\n", a.profile, a.discriminator))

		// Детальная информация для отладки
		sb.WriteString("\n🔍 Детальная отладка:\n")
		sb.WriteString("  1. Проверьте логи Core для анонсирования\n")
		sb.WriteString("  2. Убедитесь что DHT запущен в режиме сервера\n")
		sb.WriteString("  3. Проверьте подключение к bootstrap узлам\n")
		sb.WriteString("  4. Подождите несколько минут после анонсирования\n")

		return outputMsg{line: sb.String()}
	}
}

// cmdNetwork показывает технические пиры и статус сети
func (a *TUIApp) cmdNetwork() tea.Cmd {
	return func() tea.Msg {
		debugMsg := "Выполняю команду /network"

		if a.core == nil {
			return outputMsg{line: debugMsg + "\n❌ Core не запущен"}
		}

		var sb strings.Builder
		sb.WriteString(debugMsg + "\n🌐 Статус сети:\n")

		// Получаем хост
		host := a.core.GetHost()
		if host == nil {
			return outputMsg{line: debugMsg + "\n❌ Хост недоступен"}
		}

		// Все пиры из всех протоколов
		allPeers := host.Network().Peers()
		sb.WriteString(fmt.Sprintf("📊 Всего пиров в сети: %d\n", len(allPeers)))

		if len(allPeers) > 0 {
			sb.WriteString("🔗 Технические пиры:\n")
			for _, p := range allPeers {
				sb.WriteString(fmt.Sprintf("  %s\n", p.String()))
			}
		}

		// Защищенные пиры
		protectedPeers := a.core.GetProtectedPeers()
		if len(protectedPeers) > 0 {
			sb.WriteString("🛡️ Защищенные пиры:\n")
			for _, p := range protectedPeers {
				sb.WriteString(fmt.Sprintf("  %s\n", p.String()))
			}
		}

		// Статистика сети
		stats := a.core.GetNetworkStats()
		if len(stats) > 0 {
			sb.WriteString("📈 Статистика сети:\n")
			for key, value := range stats {
				sb.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
			}
		}

		return outputMsg{line: sb.String()}
	}
}

// pasteFromClipboard вставляет текст из буфера обмена
func (a *TUIApp) pasteFromClipboard() tea.Cmd {
	return func() tea.Msg {
		// В Linux используем xclip для получения содержимого буфера
		cmd := exec.Command("xclip", "-o", "-selection", "clipboard")
		output, err := cmd.Output()
		if err != nil {
			return outputMsg{line: "❌ Ошибка вставки из буфера: " + err.Error()}
		}

		// Убираем лишние символы и вставляем
		text := strings.TrimSpace(string(output))
		if text != "" {
			return pasteMsg{text: text}
		}

		return outputMsg{line: "⚠️ Буфер обмена пуст"}
	}
}

// generateNewKeyBytes генерирует новые байты ключа
func (a *TUIApp) generateNewKeyBytes() ([]byte, error) {
	return core.GenerateKeyBytes()
}

// executeCommand выполняет команду
func (a *TUIApp) executeCommand(cmd string) tea.Cmd {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return func() tea.Msg {
			return outputMsg{line: "Команда не распознана"}
		}
	}

	// Отладочная информация
	debugMsg := fmt.Sprintf("Выполняю команду: %s (части: %v)", cmd, parts)

	switch parts[0] {
	case "/help":
		return func() tea.Msg {
			return outputMsg{line: debugMsg + "\nДоступные команды: /help, /peers, /contacts, /find, /add, /msg, /network, /status, /dhtinfo, /quit"}
		}
	case "/peers":
		return a.cmdPeers()
	case "/contacts":
		return a.cmdContacts()
	case "/find":
		return a.cmdFind(parts[1:])
	case "/add":
		return a.cmdAdd(parts[1:])
	case "/msg":
		return a.cmdMsg(parts[1:])
	case "/network":
		return a.cmdNetwork()
	case "/status":
		return a.cmdStatus()
	case "/dhtinfo":
		return a.cmdDHTInfo()
	case "/diag":
		return a.cmdDiag()
	case "/quit":
		return tea.Quit
	default:
		return func() tea.Msg {
			return outputMsg{line: fmt.Sprintf("%s\nНеизвестная команда: %s", debugMsg, parts[0])}
		}
	}
}

// cmdPeers показывает список пиров
func (a *TUIApp) cmdPeers() tea.Cmd {
	return func() tea.Msg {
		debugMsg := "Выполняю команду /peers"

		if a.core == nil {
			return outputMsg{line: debugMsg + "\n❌ Core не запущен"}
		}

		peers := a.core.GetConnectedPeers()
		if len(peers) == 0 {
			return outputMsg{line: debugMsg + "\n📊 Нет подключенных пиров"}
		}

		var sb strings.Builder
		sb.WriteString(debugMsg + "\n📊 Подключенные пиры:\n")
		for _, p := range peers {
			sb.WriteString(fmt.Sprintf("  %s\n", p.String()))
		}

		return outputMsg{line: sb.String()}
	}
}

// cmdContacts показывает список контактов
func (a *TUIApp) cmdContacts() tea.Cmd {
	return func() tea.Msg {
		debugMsg := "Выполняю команду /contacts"

		if len(a.contacts) == 0 {
			return outputMsg{line: debugMsg + "\n📋 Нет добавленных контактов"}
		}

		var sb strings.Builder
		sb.WriteString(debugMsg + "\n📋 Контакты:\n")
		for nickname, peerID := range a.contacts {
			// Проверяем онлайн статус
			online := "❌"
			if a.core != nil {
				peers := a.core.GetConnectedPeers()
				for _, p := range peers {
					if p.String() == peerID {
						online = "✅"
						break
					}
				}
			}
			sb.WriteString(fmt.Sprintf("  %s %s (%s)\n", online, nickname, peerID))
		}

		return outputMsg{line: sb.String()}
	}
}

// cmdFind выполняет поиск
func (a *TUIApp) cmdFind(args []string) tea.Cmd {
	if len(args) < 2 {
		return func() tea.Msg {
			return outputMsg{line: "❌ Использование: /find id <peer_id> или /find name <nickname#discriminator>"}
		}
	}

	switch args[0] {
	case "id":
		if len(args) < 2 {
			return func() tea.Msg {
				return outputMsg{line: "❌ Использование: /find id <peer_id>"}
			}
		}
		return a.findByPeerID(args[1])
	case "name":
		if len(args) < 2 {
			return func() tea.Msg {
				return outputMsg{line: "❌ Использование: /find name <nickname#discriminator>"}
			}
		}
		return a.findByName(args[1])
	default:
		return func() tea.Msg {
			return outputMsg{line: "❌ Использование: /find id <peer_id> или /find name <nickname#discriminator>"}
		}
	}
}

// findByPeerID ищет пира по Peer ID
func (a *TUIApp) findByPeerID(peerIDStr string) tea.Cmd {
	return func() tea.Msg {
		debugMsg := fmt.Sprintf("Выполняю поиск по Peer ID: %s", peerIDStr)

		if a.core == nil {
			return outputMsg{line: debugMsg + "\n❌ Core не запущен"}
		}

		// Парсим Peer ID
		peerID, err := peer.Decode(peerIDStr)
		if err != nil {
			return outputMsg{line: fmt.Sprintf("%s\n❌ Неверный Peer ID: %v", debugMsg, err)}
		}

		// Детальная диагностика перед поиском
		fmt.Printf("🔍 Поиск по Peer ID: %s\n", peerIDStr)
		fmt.Printf("🔑 Парсированный Peer ID: %s\n", peerID.String())
		fmt.Printf("📡 Начинаем поиск пира в DHT...\n")
		fmt.Printf("⏱️ Ожидаем результат поиска (таймаут 30 сек)...\n")
		fmt.Printf("🔍 Детальная диагностика перед поиском:\n")
		fmt.Printf("  - Наш Peer ID: %s\n", a.peerID)
		fmt.Printf("  - Наш Content ID: %s\n", a.contentID)
		fmt.Printf("  - Timestamp: %s\n", time.Now().Format("15:04:05"))
		fmt.Printf("  - Core статус: %s\n", a.state.String())

		// Ищем пира
		fmt.Printf("📡 Вызываем a.core.FindPeer...\n")
		addrInfo, err := a.core.FindPeer(peerID)
		if err != nil {
			fmt.Printf("❌ Ошибка поиска пира: %v\n", err)
			fmt.Printf("🔍 Детали ошибки: %T: %v\n", err, err)
			fmt.Printf("🔍 Возможные причины:\n")
			fmt.Printf("  - Пир офлайн или не найден в DHT\n")
			fmt.Printf("  - DHT еще не готов к поиску\n")
			fmt.Printf("  - Проблема с сетью\n")
			return outputMsg{line: fmt.Sprintf("%s\n❌ Пир не найден: %v", debugMsg, err)}
		}

		// Успешный результат
		fmt.Printf("✅ Пир найден: %s\n", addrInfo.ID.String())
		fmt.Printf("📍 Адреса пира: %v\n", addrInfo.Addrs)
		fmt.Printf("🔍 Детали найденного пира:\n")
		fmt.Printf("  - Peer ID: %s\n", addrInfo.ID.String())
		fmt.Printf("  - Количество адресов: %d\n", len(addrInfo.Addrs))
		fmt.Printf("  - Timestamp: %s\n", time.Now().Format("15:04:05"))

		// Подключаемся к найденному пиру
		// TODO: Реализовать подключение через Core

		return outputMsg{line: fmt.Sprintf("%s\n✅ Пир найден: %s\n📍 Адрес: %v", debugMsg, addrInfo.ID.String(), addrInfo.Addrs)}
	}
}

// findByName ищет пира по никнейму
func (a *TUIApp) findByName(nameWithDisc string) tea.Cmd {
	return func() tea.Msg {
		debugMsg := fmt.Sprintf("Выполняю поиск по никнейму: %s", nameWithDisc)

		if a.core == nil {
			return outputMsg{line: debugMsg + "\n❌ Core не запущен"}
		}

		// Вычисляем ContentID из никнейма
		contentID := a.computeContentID(nameWithDisc)
		debugMsg += fmt.Sprintf("\n🔍 Вычисленный ContentID: %s", contentID)

		// Детальное логирование поиска
		fmt.Printf("🔍 Поиск по никнейму: %s\n", nameWithDisc)
		fmt.Printf("🔑 Вычисленный ContentID: %s\n", contentID)
		fmt.Printf("📡 Начинаем поиск провайдеров в DHT...\n")
		fmt.Printf("⏱️ Ожидаем результат поиска (таймаут 60 сек)...\n")
		fmt.Printf("🔍 Детальная диагностика перед поиском:\n")
		fmt.Printf("  - Наш Peer ID: %s\n", a.peerID)
		fmt.Printf("  - Наш Content ID: %s\n", a.contentID)
		fmt.Printf("  - Timestamp: %s\n", time.Now().Format("15:04:05"))
		fmt.Printf("  - Core статус: %s\n", a.state.String())

		// Ищем провайдеров
		providers, err := a.core.FindProvidersForContent(contentID)
		if err != nil {
			fmt.Printf("❌ Ошибка поиска провайдеров: %v\n", err)
			fmt.Printf("🔍 Возможные причины:\n")
			fmt.Printf("  - Пир не анонсировал себя в DHT\n")
			fmt.Printf("  - DHT еще не готов к поиску\n")
			fmt.Printf("  - Проблема с сетью\n")
			return outputMsg{line: fmt.Sprintf("%s\n❌ Ошибка поиска: %v", debugMsg, err)}
		}

		fmt.Printf("📊 Найдено провайдеров: %d\n", len(providers))

		if len(providers) == 0 {
			fmt.Printf("⚠️ Провайдеры не найдены для ContentID: %s\n", contentID)
			fmt.Printf("🔍 Возможные причины:\n")
			fmt.Printf("  - Пир не анонсировал себя в DHT\n")
			fmt.Printf("  - DHT еще не готов к поиску\n")
			fmt.Printf("  - Проблема с сетью\n")
			return outputMsg{line: fmt.Sprintf("%s\n❌ Пир не найден", debugMsg)}
		}

		// Берем первого провайдера
		provider := providers[0]
		fmt.Printf("✅ Найден провайдер: %s\n", provider.ID.String())
		fmt.Printf("📍 Адреса провайдера: %v\n", provider.Addrs)

		// Детальная информация о найденном провайдере
		fmt.Printf("🔍 Детали найденного провайдера:\n")
		fmt.Printf("  - Peer ID: %s\n", provider.ID.String())
		fmt.Printf("  - Short Peer ID: %s\n", provider.ID.ShortString())
		fmt.Printf("  - Количество адресов: %d\n", len(provider.Addrs))
		fmt.Printf("  - Timestamp: %s\n", time.Now().Format("15:04:05"))
		fmt.Printf("  - Content ID для поиска: %s\n", contentID)

		// TODO: Реализовать подключение к провайдеру
		fmt.Printf("🔗 TODO: Подключение к провайдеру %s\n", provider.ID.ShortString())

		return outputMsg{line: fmt.Sprintf("%s\n✅ Пир найден: %s", debugMsg, provider.ID.String())}
	}
}

// computeContentID вычисляет ContentID из никнейма
func (a *TUIApp) computeContentID(nameWithDisc string) string {
	// Используем правильный CIDv1 вместо простого хэша
	hash := sha256.Sum256([]byte(nameWithDisc))

	// Создаем multihash
	mh, err := multihash.Encode(hash[:], multihash.SHA2_256)
	if err != nil {
		// Fallback на простой хэш если multihash не удался
		return fmt.Sprintf("%x", hash)
	}

	// Создаем CIDv1 с кодеком raw
	cidV1 := cid.NewCidV1(cid.Raw, mh)
	return cidV1.String()
}

// cmdAdd добавляет контакт
func (a *TUIApp) cmdAdd(args []string) tea.Cmd {
	if len(args) < 2 {
		return func() tea.Msg {
			return outputMsg{line: "❌ Использование: /add <peer_id> <nickname>"}
		}
	}

	return func() tea.Msg {
		debugMsg := fmt.Sprintf("Выполняю команду /add с аргументами: %v", args)

		peerID := args[0]
		nickname := args[1]

		// Проверяем валидность Peer ID
		_, err := peer.Decode(peerID)
		if err != nil {
			return outputMsg{line: fmt.Sprintf("%s\n❌ Неверный Peer ID: %v", debugMsg, err)}
		}

		// Добавляем контакт
		a.contacts[nickname] = peerID
		a.nicknames[peerID] = nickname

		return outputMsg{line: fmt.Sprintf("%s\n✅ Контакт %s добавлен с Peer ID: %s", debugMsg, nickname, peerID)}
	}
}

// cmdMsg отправляет сообщение
func (a *TUIApp) cmdMsg(args []string) tea.Cmd {
	if len(args) < 2 {
		return func() tea.Msg {
			return outputMsg{line: "Использование: /msg <nickname> <текст>"}
		}
	}

	return func() tea.Msg {
		nickname := args[0]
		message := strings.Join(args[1:], " ")

		// Находим Peer ID
		peerID, exists := a.contacts[nickname]
		if !exists {
			return outputMsg{line: fmt.Sprintf("Контакт %s не найден", nickname)}
		}

		if a.core == nil {
			return outputMsg{line: "Core не запущен"}
		}

		// Парсим Peer ID
		peer, err := peer.Decode(peerID)
		if err != nil {
			return outputMsg{line: fmt.Sprintf("Неверный Peer ID: %v", err)}
		}

		// Отправляем сообщение
		err = a.core.Send(peer, []byte(message))
		if err != nil {
			return outputMsg{line: fmt.Sprintf("Ошибка отправки: %v", err)}
		}

		return outputMsg{line: fmt.Sprintf("Сообщение отправлено %s", nickname)}
	}
}

// Сообщения для bubbletea
type errorMsg struct {
	err error
}

type coreStartedMsg struct {
	controller    *core.CoreController
	peerID        string
	discriminator string
	contentID     string // Added contentID to the struct
}

type outputMsg struct {
	line string
}

// cmdDHTInfo показывает информацию о DHT для отладки
func (a *TUIApp) cmdDHTInfo() tea.Cmd {
	return func() tea.Msg {
		debugMsg := "Выполняю команду /dhtinfo"

		if a.core == nil {
			return outputMsg{line: debugMsg + "\n❌ Core не запущен"}
		}

		var sb strings.Builder
		sb.WriteString(debugMsg + "\n📊 Информация о DHT:\n")

		// Размер DHT routing table
		rtSize := a.core.GetDHTRoutingTableSize()
		sb.WriteString(fmt.Sprintf("📈 Размер routing table: %d пиров\n", rtSize))

		// Интерпретация размера
		if rtSize == 0 {
			sb.WriteString("⚠️ DHT routing table пуста - узел еще не готов к поиску\n")
		} else if rtSize < 10 {
			sb.WriteString("🔄 DHT routing table мала - узел еще разогревается\n")
		} else if rtSize < 50 {
			sb.WriteString("✅ DHT routing table в норме - узел готов к поиску\n")
		} else {
			sb.WriteString("🚀 DHT routing table большая - узел полностью готов\n")
		}

		// Рекомендации
		sb.WriteString("\n💡 Рекомендации:\n")
		if rtSize < 10 {
			sb.WriteString("  - Подождите еще 1-2 минуты для разогрева DHT\n")
			sb.WriteString("  - Проверьте подключение к bootstrap узлам\n")
		} else {
			sb.WriteString("  - DHT готов к работе\n")
			sb.WriteString("  - Можно выполнять поиск\n")
		}

		return outputMsg{line: sb.String()}
	}
}

// cmdDiag показывает полную диагностику системы
func (a *TUIApp) cmdDiag() tea.Cmd {
	return func() tea.Msg {
		debugMsg := "Выполняю команду /diag"

		if a.core == nil {
			return outputMsg{line: debugMsg + "\n❌ Core не запущен"}
		}

		var sb strings.Builder
		sb.WriteString(debugMsg + "\n")
		sb.WriteString("--- ДИАГНОСТИКА OWL WHISPER ---\n")

		// Основная информация
		sb.WriteString(fmt.Sprintf("PeerID: %s\n", a.peerID))
		sb.WriteString(fmt.Sprintf("ContentID: %s\n", a.contentID))
		sb.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format("15:04:05")))

		// DHT статус
		sb.WriteString("\n--- DHT ---\n")
		rtSize := a.core.GetDHTRoutingTableSize()
		sb.WriteString(fmt.Sprintf("Routing Table Size: %d peers\n", rtSize))

		// Статус DHT
		if rtSize == 0 {
			sb.WriteString("⚠️ DHT не готов - узел еще разогревается\n")
		} else if rtSize < 10 {
			sb.WriteString("🔄 DHT разогревается - подождите еще\n")
		} else if rtSize < 50 {
			sb.WriteString("✅ DHT готов к работе\n")
		} else {
			sb.WriteString("🚀 DHT полностью готов\n")
		}

		// Соединения
		sb.WriteString("\n--- СОЕДИНЕНИЯ ---\n")
		peers := a.core.GetConnectedPeers()
		sb.WriteString(fmt.Sprintf("Active Connections (/peers): %d peers\n", len(peers)))

		// Защищенные пиры
		protectedPeers := a.core.GetProtectedPeers()
		sb.WriteString(fmt.Sprintf("Protected Peers (ConnMgr): %d peers\n", len(protectedPeers)))

		// Статистика сети
		stats := a.core.GetNetworkStats()
		if len(stats) > 0 {
			sb.WriteString("\n--- СТАТИСТИКА СЕТИ ---\n")
			for key, value := range stats {
				sb.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
			}
		}

		// Рекомендации
		sb.WriteString("\n--- РЕКОМЕНДАЦИИ ---\n")
		if rtSize < 10 {
			sb.WriteString("  - Подождите 2-3 минуты для разогрева DHT\n")
			sb.WriteString("  - Проверьте подключение к bootstrap узлам\n")
		} else if len(peers) == 0 {
			sb.WriteString("  - DHT готов, но нет активных соединений\n")
			sb.WriteString("  - Попробуйте поиск - создаст соединения\n")
		} else {
			sb.WriteString("  - Система готова к работе\n")
			sb.WriteString("  - Можно выполнять поиск и обмен сообщениями\n")
		}

		return outputMsg{line: sb.String()}
	}
}
