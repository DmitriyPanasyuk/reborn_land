package main

import (
	"log"
	"reborn_land/config"
	"reborn_land/database"
	"reborn_land/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	if cfg.TelegramToken == "" {
		log.Fatal("TELEGRAM_TOKEN is required. Please set it in environment variables or .env file")
	}

	// Подключаемся к базе данных
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Создаем бота
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Создаем обработчики
	botHandlers := handlers.New(bot, db)

	// Настраиваем получение обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Обрабатываем обновления
	for update := range updates {
		botHandlers.HandleUpdate(update)
	}
}
