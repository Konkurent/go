package config

import (
	"github.com/spf13/viper"
	"os"
)

// Config представляет конфигурацию приложения
type Config struct {
	Server struct {
		Port int
	}
	DB struct {
		Host     string
		Port     int
		User     string
		Password string
		DBName   string
	}
	JWT struct {
		SecretKey string
		ExpiresIn int // в часах
	}
	SMTP struct {
		Host     string
		Port     int
		Username string
		Password string
		From     string
	}
	CardPrivateKey string // Приватный ключ для подписи карт
	CardPublicKey  string // Публичный ключ для проверки подписи карт
	CardHMACKey    string // Ключ для HMAC-подписи карт
}

// NewConfig создает новый экземпляр конфигурации
func NewConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
