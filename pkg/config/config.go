package config

import (
	"OwlWhisper/pkg/interfaces"
	"time"
)

// Config содержит конфигурацию приложения
type Config struct {
	// Network настройки сети
	Network interfaces.NetworkConfig `json:"network"`

	// Security настройки безопасности
	Security interfaces.SecurityConfig `json:"security"`

	// UI настройки интерфейса
	UI interfaces.UIConfig `json:"ui"`
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *interfaces.Config {
	return &interfaces.Config{
		Network: interfaces.NetworkConfig{
			ListenPort:       0,
			EnableNAT:        true,
			EnableHolePunch:  true,
			EnableRelay:      true,
			DiscoveryTimeout: 30 * time.Second,
		},
		Security: interfaces.SecurityConfig{
			EnableEncryption: true,
			KeySize:          256,
		},
		UI: interfaces.UIConfig{
			EnableTUI: true,
			EnableAPI: true,
			APIPort:   8080,
		},
	}
}
