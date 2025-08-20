package interfaces

// Config содержит конфигурацию для CORE сервиса
type Config struct {
	Network  NetworkConfig  `json:"network"`
	Security SecurityConfig `json:"security"`
	UI       UIConfig       `json:"ui"`
}

// SecurityConfig настройки безопасности
type SecurityConfig struct {
	EnableEncryption bool `json:"enableEncryption"`
	KeySize          int  `json:"keySize"`
}

// UIConfig настройки интерфейса
type UIConfig struct {
	EnableTUI bool `json:"enableTUI"`
	EnableAPI bool `json:"enableAPI"`
	APIPort   int  `json:"apiPort"`
}
