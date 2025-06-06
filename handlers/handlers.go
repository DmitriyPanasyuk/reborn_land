package handlers

import (
	"fmt"
	"log"
	"reborn_land/database"
	"strings"
	"time"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotHandlers struct {
	bot            *tgbotapi.BotAPI
	db             *database.DB
	waitingForName map[int64]bool
}

func New(bot *tgbotapi.BotAPI, db *database.DB) *BotHandlers {
	return &BotHandlers{
		bot:            bot,
		db:             db,
		waitingForName: make(map[int64]bool),
	}
}

func (h *BotHandlers) HandleUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		h.handleMessage(update.Message)
	}
}

func (h *BotHandlers) handleMessage(message *tgbotapi.Message) {
	userID := message.From.ID

	// Проверяем, ждем ли мы от пользователя имя
	if h.waitingForName[userID] {
		h.handleNameInput(message)
		return
	}

	switch message.Text {
	case "/start":
		h.handleStart(message)
	case "/profile":
		h.handleProfile(message)
	case "🎒 Инвентарь":
		h.handleInventory(message)
	case "🌿 Добыча":
		h.handleGathering(message)
	case "🔨 Рабочее место":
		h.handleWorkplace(message)
	case "🛠 Верстак":
		h.handleWorkbench(message)
	case "🧱 Печь":
		h.handleFurnace(message)
	case "🔥 Костер":
		h.handleCampfire(message)
	default:
		// Неизвестная команда
		msg := tgbotapi.NewMessage(message.Chat.ID, "Неизвестная команда. Используйте /start для начала игры.")
		h.bot.Send(msg)
	}
}

func (h *BotHandlers) handleStart(message *tgbotapi.Message) {
	userID := message.From.ID

	// Проверяем, зарегистрирован ли пользователь
	exists, err := h.db.PlayerExists(userID)
	if err != nil {
		log.Printf("Error checking player existence: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	if exists {
		// Пользователь уже зарегистрирован
		player, err := h.db.GetPlayer(userID)
		if err != nil {
			log.Printf("Error getting player: %v", err)
			msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
			h.bot.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("С возвращением, %s! 👋", player.Name))
		h.sendWithKeyboard(msg)
		return
	}

	// Начинаем регистрацию нового пользователя
	h.startRegistration(message)
}

func (h *BotHandlers) startRegistration(message *tgbotapi.Message) {
	// Первое сообщение
	welcomeText := `🏝 Добро пожаловать на Землю Возрождения!

Ты пришёл в край, где прежде не ступала нога человека. Нет ни домов, ни дорог — лишь бескрайняя дикая земля, богатая ресурсами, тайнами и возможностями.

🪨 В твоих руках — старая, но крепкая кирка. С неё начнётся твой путь.

🔧 Здесь нет ничего, но ты способен создать всё.
Построй свою хижину, разведай окрестности, добудь первые ресурсы и заложи фундамент новой цивилизации.
Всё — от костра до храмов — будет делом твоих рук.`

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	h.bot.Send(msg)

	// Ждем 2 секунды и отправляем второе сообщение
	go func() {
		time.Sleep(2 * time.Second)

		secondText := `🧭 Что тебя ждёт:
🪵 Добыча ресурсов (дерево, камень, пища)
🛖 Строительство и развитие поселения
🌄 Исследование новых территорий
🐺 Борьба с дикой природой
👥 Создание сообщества
🔮🔮 Открытие древних артефактов`

		msg2 := tgbotapi.NewMessage(message.Chat.ID, secondText)
		h.bot.Send(msg2)

		// Сразу после второго сообщения просим имя
		nameMsg := tgbotapi.NewMessage(message.Chat.ID, "Придумай себе имя:")
		h.bot.Send(nameMsg)

		// Отмечаем, что ждем имя от этого пользователя
		h.waitingForName[message.From.ID] = true
	}()
}

func (h *BotHandlers) handleNameInput(message *tgbotapi.Message) {
	userID := message.From.ID
	name := strings.TrimSpace(message.Text)

	// Проверяем длину имени
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 30 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Имя должно содержать от 1 до 30 символов. Попробуйте еще раз:")
		h.bot.Send(msg)
		return
	}

	// Создаем игрока в базе данных
	player, err := h.db.CreatePlayer(userID, name)
	if err != nil {
		log.Printf("Error creating player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при регистрации. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	// Убираем флаг ожидания имени
	delete(h.waitingForName, userID)

	// Отправляем сообщение об успешной регистрации
	successText := fmt.Sprintf(`✅ Регистрация прошла успешно!

Добро пожаловать, %s! 👋

Твой уровень: %d
Опыт: %d/100
Сытость: %d/100`, player.Name, player.Level, player.Experience, player.Satiety)

	msg := tgbotapi.NewMessage(message.Chat.ID, successText)
	h.sendWithKeyboard(msg)
}

func (h *BotHandlers) handleProfile(message *tgbotapi.Message) {
	userID := message.From.ID

	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала зарегистрируйтесь с помощью команды /start")
		h.bot.Send(msg)
		return
	}

	profileText := fmt.Sprintf(`👤 Профиль игрока
Имя: %s
Уровень: %d
Опыт: %d/100
Сытость: %d/100`, player.Name, player.Level, player.Experience, player.Satiety)

	msg := tgbotapi.NewMessage(message.Chat.ID, profileText)
	h.bot.Send(msg)
}

func (h *BotHandlers) handleInventory(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала зарегистрируйтесь с помощью команды /start")
		h.bot.Send(msg)
		return
	}

	// Получаем инвентарь
	inventory, err := h.db.GetPlayerInventory(player.ID)
	if err != nil {
		log.Printf("Error getting inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении инвентаря.")
		h.bot.Send(msg)
		return
	}

	if len(inventory) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "🎒 Ваш инвентарь пуст.")
		h.bot.Send(msg)
		return
	}

	// Формируем текст инвентаря
	inventoryText := "🎒 Ваш инвентарь:\n\n"
	for _, item := range inventory {
		if item.Type == "tool" && item.Durability > 0 {
			inventoryText += fmt.Sprintf("%s - %d шт. (Прочность: %d/100)\n", item.ItemName, item.Quantity, item.Durability)
		} else {
			inventoryText += fmt.Sprintf("%s - %d шт.\n", item.ItemName, item.Quantity)
		}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, inventoryText)
	h.bot.Send(msg)
}

func (h *BotHandlers) handleGathering(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🌿 Функция добычи ресурсов пока в разработке...")
	h.bot.Send(msg)
}

func (h *BotHandlers) handleWorkplace(message *tgbotapi.Message) {
	workplaceText := `🔨 Ты подходишь к рабочему месту.

Здесь можно создавать новые предметы и обрабатывать ресурсы.`

	msg := tgbotapi.NewMessage(message.Chat.ID, workplaceText)
	h.sendWorkplaceKeyboard(msg)
}

func (h *BotHandlers) handleWorkbench(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🛠 Функция верстака пока в разработке...")
	h.bot.Send(msg)
}

func (h *BotHandlers) handleFurnace(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🧱 Функция печи пока в разработке...")
	h.bot.Send(msg)
}

func (h *BotHandlers) handleCampfire(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🔥 Функция костра пока в разработке...")
	h.bot.Send(msg)
}

func (h *BotHandlers) sendWithKeyboard(msg tgbotapi.MessageConfig) {
	// Создаем клавиатуру
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🎒 Инвентарь"),
			tgbotapi.NewKeyboardButton("🌿 Добыча"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔨 Рабочее место"),
		),
	)
	keyboard.ResizeKeyboard = true
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}

func (h *BotHandlers) sendWorkplaceKeyboard(msg tgbotapi.MessageConfig) {
	// Создаем клавиатуру рабочего места
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🛠 Верстак"),
			tgbotapi.NewKeyboardButton("🧱 Печь"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔥 Костер"),
		),
	)
	keyboard.ResizeKeyboard = true
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}
