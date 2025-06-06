package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	DatabaseURL   string
}

func Load() *Config {
	// Загружаем переменные окружения из .env файла, если он существует
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	return &Config{
		TelegramToken: getEnv("TELEGRAM_TOKEN", ""),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://user:password@localhost/reborn_land?sslmode=disable"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
