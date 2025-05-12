package config

import (
	"fmt"
	"os"
	"strconv"
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
	cfg := &Config{}

	// Настройки сервера
	port, err := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("неверный формат порта сервера: %v", err)
	}
	cfg.Server.Port = port

	// Настройки базы данных
	cfg.DB.Host = getEnv("DB_HOST", "localhost")
	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("неверный формат порта базы данных: %v", err)
	}
	cfg.DB.Port = dbPort
	cfg.DB.User = getEnv("DB_USER", "postgres")
	cfg.DB.Password = getEnv("DB_PASSWORD", "postgres")
	cfg.DB.DBName = getEnv("DB_NAME", "bank_db")

	// Настройки JWT
	cfg.JWT.SecretKey = getEnv("JWT_SECRET_KEY", "your-secret-key-here")
	jwtExpiresIn, err := strconv.Atoi(getEnv("JWT_EXPIRES_IN", "24"))
	if err != nil {
		return nil, fmt.Errorf("неверный формат времени жизни JWT: %v", err)
	}
	cfg.JWT.ExpiresIn = jwtExpiresIn

	// Настройки SMTP
	cfg.SMTP.Host = getEnv("SMTP_HOST", "smtp.gmail.com")
	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	if err != nil {
		return nil, fmt.Errorf("неверный формат порта SMTP: %v", err)
	}
	cfg.SMTP.Port = smtpPort
	cfg.SMTP.Username = getEnv("SMTP_USERNAME", "your-email@gmail.com")
	cfg.SMTP.Password = getEnv("SMTP_PASSWORD", "your-app-password")
	cfg.SMTP.From = getEnv("SMTP_FROM", "your-email@gmail.com")

	// Настройки карт
	cfg.CardPrivateKey = getEnv("CARD_PRIVATE_KEY", "your-card-private-key-here")
	cfg.CardPublicKey = getEnv("CARD_PUBLIC_KEY", "your-card-public-key-here")
	cfg.CardHMACKey = getEnv("CARD_HMAC_KEY", "your-card-hmac-key-here")

	return cfg, nil
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
