package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config представляет конфигурацию приложения
type Config struct {
	// Сетевые настройки
	Network struct {
		ListenPort      int      `json:"listen_port"`
		BootstrapNodes  []string `json:"bootstrap_nodes"`
		RelayNodes      []string `json:"relay_nodes"`
		STUNServers     []string `json:"stun_servers"`
		EnableRelay     bool     `json:"enable_relay"`
		EnableNAT       bool     `json:"enable_nat"`
		EnableHolePunch bool     `json:"enable_hole_punch"`
	} `json:"network"`

	// Настройки чата
	Chat struct {
		MaxMessageLength int  `json:"max_message_length"`
		MessageHistory   int  `json:"message_history"`
		AutoSave         bool `json:"auto_save"`
	} `json:"chat"`

	// Настройки безопасности
	Security struct {
		EnableTLS     bool   `json:"enable_tls"`
		EnableNoise   bool   `json:"enable_noise"`
		MinTLSVersion string `json:"min_tls_version"`
	} `json:"security"`

	// Настройки логирования
	Logging struct {
		Level      string `json:"level"`
		OutputFile string `json:"output_file"`
		Console    bool   `json:"console"`
	} `json:"logging"`

	// Настройки UI
	UI struct {
		Theme          string `json:"theme"`
		ShowTimestamps bool   `json:"show_timestamps"`
		ShowPeerIDs    bool   `json:"show_peer_ids"`
	} `json:"ui"`
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	config := &Config{}

	// Сетевые настройки по умолчанию
	config.Network.ListenPort = 0 // 0 означает автоматический выбор порта
	config.Network.BootstrapNodes = []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	}
	config.Network.RelayNodes = []string{
		"/dnsaddr/relay.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/relay.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	}
	config.Network.STUNServers = []string{
		"stun:stun.l.google.com:19302",
		"stun:stun1.l.google.com:19302",
		"stun:stun2.l.google.com:19302",
	}
	config.Network.EnableRelay = true
	config.Network.EnableNAT = true
	config.Network.EnableHolePunch = true

	// Настройки чата по умолчанию
	config.Chat.MaxMessageLength = 1000
	config.Chat.MessageHistory = 100
	config.Chat.AutoSave = true

	// Настройки безопасности по умолчанию
	config.Security.EnableTLS = true
	config.Security.EnableNoise = false
	config.Security.MinTLSVersion = "1.3"

	// Настройки логирования по умолчанию
	config.Logging.Level = "info"
	config.Logging.OutputFile = ""
	config.Logging.Console = true

	// Настройки UI по умолчанию
	config.UI.Theme = "default"
	config.UI.ShowTimestamps = true
	config.UI.ShowPeerIDs = false

	return config
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	if configPath == "" {
		// Ищем конфиг в стандартных местах
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configPath = filepath.Join(homeDir, ".owlwhisper", "config.json")
		}
	}

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			err = json.Unmarshal(data, config)
			if err != nil {
				return nil, err
			}
		}
	}

	return config, nil
}

// SaveConfig сохраняет конфигурацию в файл
func (c *Config) SaveConfig(configPath string) error {
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = filepath.Join(homeDir, ".owlwhisper", "config.json")
	}

	// Создаем директорию если не существует
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
