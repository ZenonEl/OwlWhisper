package core

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	// Константы для файлов
	configDir     = ".config/owlwhisper"
	identityFile  = "identity.key"
	peerCacheFile = "peer.cache"
	profileFile   = "profile.json"

	// Максимальное количество пиров для кэширования
	maxCachedPeers = 50

	// Время жизни кэшированного пира
	peerCacheTTL = 24 * time.Hour
)

// PeerCacheEntry представляет запись в кэше пиров
type PeerCacheEntry struct {
	PeerID    string    `json:"peer_id"`
	Addresses []string  `json:"addresses"`
	LastSeen  time.Time `json:"last_seen"`
	Healthy   bool      `json:"healthy"` // Был ли пир "здоровым" при последнем соединении
}

// UserProfile представляет профиль пользователя для сохранения
type UserProfile struct {
	Nickname    string    `json:"nickname"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PersistenceManager управляет сохранением и загрузкой данных
type PersistenceManager struct {
	configPath string
}

// NewPersistenceManager создает новый менеджер персистентности
func NewPersistenceManager() (*PersistenceManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить домашнюю директорию: %w", err)
	}

	configPath := filepath.Join(homeDir, configDir)

	// Создаем директорию если не существует
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать конфигурационную директорию: %w", err)
	}

	return &PersistenceManager{
		configPath: configPath,
	}, nil
}

// LoadOrCreateIdentity загружает существующий ключ или создает новый
func (pm *PersistenceManager) LoadOrCreateIdentity() (crypto.PrivKey, error) {
	identityPath := filepath.Join(pm.configPath, identityFile)

	// Пытаемся загрузить существующий ключ
	if data, err := os.ReadFile(identityPath); err == nil {
		privKey, err := crypto.UnmarshalPrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("не удалось десериализовать ключ: %w", err)
		}
		return privKey, nil
	}

	// Создаем новый ключ
	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("не удалось сгенерировать новый ключ: %w", err)
	}

	// Сохраняем новый ключ
	keyData, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("не удалось сериализовать ключ: %w", err)
	}

	if err := os.WriteFile(identityPath, keyData, 0600); err != nil {
		return nil, fmt.Errorf("не удалось сохранить ключ: %w", err)
	}

	return privKey, nil
}

// SavePeerToCache сохраняет пира в кэш
func (pm *PersistenceManager) SavePeerToCache(peerID peer.ID, addresses []string, healthy bool) error {
	// Загружаем существующий кэш
	cache, err := pm.loadPeerCache()
	if err != nil {
		// Если файл не существует, создаем новый кэш
		cache = make(map[string]PeerCacheEntry)
	}

	// Обновляем или добавляем пира
	cache[peerID.String()] = PeerCacheEntry{
		PeerID:    peerID.String(),
		Addresses: addresses,
		LastSeen:  time.Now(),
		Healthy:   healthy,
	}

	// Очищаем устаревшие записи
	pm.cleanupExpiredEntries(cache)

	// Сохраняем обновленный кэш
	return pm.savePeerCache(cache)
}

// LoadPeerFromCache загружает пира из кэша
func (pm *PersistenceManager) LoadPeerFromCache(peerID peer.ID) (*PeerCacheEntry, error) {
	cache, err := pm.loadPeerCache()
	if err != nil {
		return nil, err
	}

	entry, exists := cache[peerID.String()]
	if !exists {
		return nil, fmt.Errorf("пир %s не найден в кэше", peerID.String())
	}

	// Проверяем, не устарел ли пир
	if time.Since(entry.LastSeen) > peerCacheTTL {
		// Удаляем устаревшую запись
		delete(cache, peerID.String())
		pm.savePeerCache(cache)
		return nil, fmt.Errorf("пир %s устарел", peerID.String())
	}

	return &entry, nil
}

// GetAllCachedPeers возвращает всех кэшированных пиров
func (pm *PersistenceManager) GetAllCachedPeers() ([]PeerCacheEntry, error) {
	cache, err := pm.loadPeerCache()
	if err != nil {
		return nil, err
	}

	// Очищаем устаревшие записи
	pm.cleanupExpiredEntries(cache)
	pm.savePeerCache(cache)

	var entries []PeerCacheEntry
	for _, entry := range cache {
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetHealthyCachedPeers возвращает только "здоровых" кэшированных пиров
func (pm *PersistenceManager) GetHealthyCachedPeers() ([]PeerCacheEntry, error) {
	allPeers, err := pm.GetAllCachedPeers()
	if err != nil {
		return nil, err
	}

	var healthyPeers []PeerCacheEntry
	for _, peer := range allPeers {
		if peer.Healthy {
			healthyPeers = append(healthyPeers, peer)
		}
	}

	return healthyPeers, nil
}

// RemovePeerFromCache удаляет пира из кэша
func (pm *PersistenceManager) RemovePeerFromCache(peerID peer.ID) error {
	cache, err := pm.loadPeerCache()
	if err != nil {
		return err
	}

	delete(cache, peerID.String())
	return pm.savePeerCache(cache)
}

// ClearPeerCache очищает весь кэш пиров
func (pm *PersistenceManager) ClearPeerCache() error {
	cachePath := filepath.Join(pm.configPath, peerCacheFile)
	return os.Remove(cachePath)
}

// loadPeerCache загружает кэш пиров из файла
func (pm *PersistenceManager) loadPeerCache() (map[string]PeerCacheEntry, error) {
	cachePath := filepath.Join(pm.configPath, peerCacheFile)
	
	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]PeerCacheEntry), nil
		}
		return nil, fmt.Errorf("не удалось прочитать кэш пиров: %w", err)
	}

	var cache map[string]PeerCacheEntry
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("не удалось десериализовать кэш пиров: %w", err)
	}

	return cache, nil
}

// savePeerCache сохраняет кэш пиров в файл
func (pm *PersistenceManager) savePeerCache(cache map[string]PeerCacheEntry) error {
	cachePath := filepath.Join(pm.configPath, peerCacheFile)
	
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("не удалось сериализовать кэш пиров: %w", err)
	}

	return os.WriteFile(cachePath, data, 0644)
}

// cleanupExpiredEntries очищает устаревшие записи из кэша
func (pm *PersistenceManager) cleanupExpiredEntries(cache map[string]PeerCacheEntry) {
	now := time.Now()
	for peerID, entry := range cache {
		if now.Sub(entry.LastSeen) > peerCacheTTL {
			delete(cache, peerID)
		}
	}
}

// SavePeerCache сохраняет кэш пиров
func (pm *PersistenceManager) SavePeerCache(peers []peer.ID, addresses map[peer.ID][]string) error {
	cachePath := filepath.Join(pm.configPath, peerCacheFile)

	var entries []PeerCacheEntry
	now := time.Now()

	// Создаем записи для кэширования
	for _, p := range peers {
		if len(entries) >= maxCachedPeers {
			break
		}

		entry := PeerCacheEntry{
			PeerID:   p.String(),
			LastSeen: now,
			Healthy:  true, // Считаем пира здоровым если он в списке активных
		}

		// Добавляем адреса если есть
		if addrs, exists := addresses[p]; exists {
			entry.Addresses = make([]string, len(addrs))
			for i, addr := range addrs {
				entry.Addresses[i] = addr // addrs уже являются строками
			}
		}

		entries = append(entries, entry)
	}

	// Сохраняем в JSON
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("не удалось сериализовать кэш пиров: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("не удалось сохранить кэш пиров: %w", err)
	}

	return nil
}

// LoadPeerCache загружает кэш пиров
func (pm *PersistenceManager) LoadPeerCache() ([]PeerCacheEntry, error) {
	cachePath := filepath.Join(pm.configPath, peerCacheFile)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []PeerCacheEntry{}, nil // Пустой кэш
		}
		return nil, fmt.Errorf("не удалось прочитать кэш пиров: %w", err)
	}

	var entries []PeerCacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("не удалось десериализовать кэш пиров: %w", err)
	}

	// Фильтруем устаревшие записи
	var validEntries []PeerCacheEntry
	now := time.Now()

	for _, entry := range entries {
		if now.Sub(entry.LastSeen) < peerCacheTTL {
			validEntries = append(validEntries, entry)
		}
	}

	return validEntries, nil
}

// GetConfigPath возвращает путь к конфигурационной директории
func (pm *PersistenceManager) GetConfigPath() string {
	return pm.configPath
}

// SaveProfile сохраняет профиль пользователя
func (pm *PersistenceManager) SaveProfile(profile *UserProfile) error {
	profilePath := filepath.Join(pm.configPath, profileFile)

	// Обновляем время изменения
	profile.UpdatedAt = time.Now()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now()
	}

	// Сериализуем в JSON
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("не удалось сериализовать профиль: %w", err)
	}

	// Сохраняем в файл
	if err := os.WriteFile(profilePath, data, 0600); err != nil {
		return fmt.Errorf("не удалось сохранить профиль: %w", err)
	}

	return nil
}

// LoadProfile загружает профиль пользователя
func (pm *PersistenceManager) LoadProfile() (*UserProfile, error) {
	profilePath := filepath.Join(pm.configPath, profileFile)

	// Проверяем существование файла
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		// Файл не существует, возвращаем профиль по умолчанию
		return &UserProfile{
			Nickname:    "Anonymous",
			DisplayName: "Anonymous",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	// Читаем файл
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл профиля: %w", err)
	}

	// Десериализуем JSON
	var profile UserProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("не удалось десериализовать профиль: %w", err)
	}

	return &profile, nil
}
