package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ICoreController - это публичный интерфейс для всего Core слоя
type ICoreController interface {
	// Start запускает Core контроллер
	Start() error

	// Stop останавливает Core контроллер
	Stop() error

	// Broadcast отправляет данные всем подключенным пирам
	Broadcast(data []byte) error

	// Send отправляет данные конкретному пиру
	Send(peerID peer.ID, data []byte) error

	// GetMyID возвращает ID текущего узла
	GetMyID() string

	// GetPeers возвращает список подключенных пиров
	GetPeers() []peer.ID

	// Messages возвращает канал для получения ВСЕХ входящих данных
	Messages() <-chan RawMessage

	// GetHost возвращает узел
	GetHost() host.Host

	// Новые методы для работы с профилями
	GetMyProfile() *ProfileInfo
	UpdateMyProfile(nickname string) error
	GetPeerProfile(peerID peer.ID) *ProfileInfo
	SendProfileToPeer(peerID peer.ID) error
}

// ProfileInfo представляет профиль пользователя
type ProfileInfo struct {
	Nickname      string
	Discriminator string
	DisplayName   string
	PeerID        string
	LastSeen      time.Time
	IsOnline      bool
}

// CoreController реализует ICoreController интерфейс
type CoreController struct {
	node      *Node
	discovery *DiscoveryManager

	ctx    context.Context
	cancel context.CancelFunc

	// Мьютекс для безопасного доступа
	mu sync.RWMutex

	// Статус работы
	running bool

	// Кэшированный профиль пользователя
	userProfile *UserProfile
}

// NewCoreController создает новый Core контроллер (для обратной совместимости)
func NewCoreController(ctx context.Context) (*CoreController, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Создаем Node
	node, err := NewNode(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать Node: %w", err)
	}

	return createControllerFromNode(ctx, cancel, node)
}

// NewCoreControllerWithKey создает новый Core контроллер с переданным ключом
func NewCoreControllerWithKey(ctx context.Context, privKey crypto.PrivKey) (*CoreController, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Создаем PersistenceManager
	persistence, err := NewPersistenceManager()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать PersistenceManager: %w", err)
	}

	// Создаем Node с переданным ключом
	node, err := NewNodeWithKey(ctx, privKey, persistence)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать Node с ключом: %w", err)
	}

	return createControllerFromNode(ctx, cancel, node)
}

// NewCoreControllerWithKeyBytes создает новый Core контроллер с переданными байтами ключа
func NewCoreControllerWithKeyBytes(ctx context.Context, keyBytes []byte) (*CoreController, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Создаем PersistenceManager
	persistence, err := NewPersistenceManager()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать PersistenceManager: %w", err)
	}

	// Создаем Node с переданными байтами ключа
	node, err := NewNodeWithKeyBytes(ctx, keyBytes, persistence)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать Node с байтами ключа: %w", err)
	}

	return createControllerFromNode(ctx, cancel, node)
}

// createControllerFromNode создает контроллер из готового узла
func createControllerFromNode(ctx context.Context, cancel context.CancelFunc, node *Node) (*CoreController, error) {
	// Создаем DiscoveryManager с callback для новых пиров
	discovery, err := NewDiscoveryManager(ctx, node.GetHost(), func(pi peer.AddrInfo) {
		// Когда найден новый пир, добавляем его в Node
		node.AddPeer(pi.ID)
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось создать DiscoveryManager: %w", err)
	}

	// Загружаем профиль пользователя
	userProfile, err := node.persistence.LoadProfile()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("не удалось загрузить профиль: %w", err)
	}

	controller := &CoreController{
		node:        node,
		discovery:   discovery,
		ctx:         ctx,
		cancel:      cancel,
		userProfile: userProfile,
	}

	return controller, nil
}

// Start запускает Core контроллер
func (c *CoreController) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("контроллер уже запущен")
	}

	// Запускаем Node
	if err := c.node.Start(); err != nil {
		return fmt.Errorf("не удалось запустить Node: %w", err)
	}

	// Запускаем Discovery
	if err := c.discovery.Start(); err != nil {
		return fmt.Errorf("не удалось запустить Discovery: %w", err)
	}

	c.running = true
	Info("🚀 Core контроллер запущен")

	return nil
}

// Stop останавливает Core контроллер
func (c *CoreController) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	// Останавливаем Discovery
	if err := c.discovery.Stop(); err != nil {
		Warn("⚠️ Ошибка остановки Discovery: %v", err)
	}

	// Останавливаем Node
	if err := c.node.Stop(); err != nil {
		Warn("⚠️ Ошибка остановки Discovery: %v", err)
	}

	// Отменяем контекст
	c.cancel()

	c.running = false
	Info("🛑 Core контроллер остановлен")

	return nil
}

// Broadcast отправляет данные всем подключенным пирам
func (c *CoreController) Broadcast(data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return fmt.Errorf("контроллер не запущен")
	}

	return c.node.Broadcast(data)
}

// Send отправляет данные конкретному пиру
func (c *CoreController) Send(peerID peer.ID, data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return fmt.Errorf("контроллер не запущен")
	}

	return c.node.Send(peerID, data)
}

// GetMyID возвращает ID текущего узла
func (c *CoreController) GetMyID() string {
	return c.node.GetMyID()
}

// GetPeers возвращает список подключенных пиров
func (c *CoreController) GetPeers() []peer.ID {
	return c.node.GetPeers()
}

// Messages возвращает канал для получения входящих сообщений
func (c *CoreController) Messages() <-chan RawMessage {
	return c.node.Messages()
}

// GetHost возвращает узел
func (c *CoreController) GetHost() host.Host {
	return c.node.GetHost()
}

// IsRunning проверяет, запущен ли контроллер
func (c *CoreController) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.running
}

// IsConnected проверяет, подключен ли указанный пир
func (c *CoreController) IsConnected(peerID peer.ID) bool {
	return c.node.IsConnected(peerID)
}

// GetMyProfile возвращает профиль текущего узла
func (c *CoreController) GetMyProfile() *ProfileInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	peerID := c.GetMyID()

	// Генерируем discriminator из последних 6 символов PeerID
	discriminator := ""
	if len(peerID) >= 6 {
		discriminator = "#" + peerID[len(peerID)-6:]
	}

	// Используем сохраненный профиль
	nickname := "Anonymous"
	displayName := "Anonymous" + discriminator
	if c.userProfile != nil {
		nickname = c.userProfile.Nickname
		if nickname != "" && nickname != "Anonymous" {
			displayName = nickname + discriminator
		}
	}

	return &ProfileInfo{
		Nickname:      nickname,
		Discriminator: discriminator,
		DisplayName:   displayName,
		PeerID:        peerID,
		LastSeen:      time.Now(),
		IsOnline:      true,
	}
}

// UpdateMyProfile обновляет профиль текущего узла
func (c *CoreController) UpdateMyProfile(nickname string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Обновляем кэшированный профиль
	if c.userProfile == nil {
		c.userProfile = &UserProfile{
			Nickname:    nickname,
			DisplayName: nickname,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	} else {
		c.userProfile.Nickname = nickname
		c.userProfile.DisplayName = nickname
		c.userProfile.UpdatedAt = time.Now()
	}

	// Сохраняем в файл
	if err := c.node.persistence.SaveProfile(c.userProfile); err != nil {
		Error("❌ Ошибка сохранения профиля: %v", err)
		return fmt.Errorf("не удалось сохранить профиль: %w", err)
	}

	Info("📝 Профиль обновлен: %s", nickname)
	return nil
}

// GetPeerProfile возвращает профиль указанного пира
func (c *CoreController) GetPeerProfile(peerID peer.ID) *ProfileInfo {
	// TODO: Реализовать получение профиля из кэша или запрос
	// Пока возвращаем базовую информацию
	discriminator := ""
	peerIDStr := peerID.String()
	if len(peerIDStr) >= 6 {
		discriminator = "#" + peerIDStr[len(peerIDStr)-6:]
	}

	return &ProfileInfo{
		Nickname:      "Unknown",
		Discriminator: discriminator,
		DisplayName:   "Unknown" + discriminator,
		PeerID:        peerIDStr,
		LastSeen:      time.Now(),
		IsOnline:      c.IsConnected(peerID),
	}
}

// SendProfileToPeer отправляет профиль указанному пиру
func (c *CoreController) SendProfileToPeer(peerID peer.ID) error {
	// TODO: Реализовать отправку ProfileInfo через Protobuf
	// Пока просто логируем
	Info("📤 Отправка профиля к %s", peerID.ShortString())
	return nil
}
