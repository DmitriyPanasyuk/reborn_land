package handlers

import (
	"fmt"
	"log"
	"reborn_land/database"
	"reborn_land/models"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotHandlers struct {
	bot             *tgbotapi.BotAPI
	db              *database.DB
	waitingForName  map[int64]bool
	mineSessions    map[int64]*models.MineSession
	forestSessions  map[int64]*models.ForestSession
	miningTimers    map[int64]*time.Timer
	choppingTimers  map[int64]*time.Timer
	mineCooldowns   map[int64]time.Time // Время окончания кулдауна шахты
	forestCooldowns map[int64]time.Time // Время окончания кулдауна леса
	playerLocation  map[int64]string    // Текущее местоположение игрока
}

func New(bot *tgbotapi.BotAPI, db *database.DB) *BotHandlers {
	return &BotHandlers{
		bot:             bot,
		db:              db,
		waitingForName:  make(map[int64]bool),
		mineSessions:    make(map[int64]*models.MineSession),
		forestSessions:  make(map[int64]*models.ForestSession),
		miningTimers:    make(map[int64]*time.Timer),
		choppingTimers:  make(map[int64]*time.Timer),
		mineCooldowns:   make(map[int64]time.Time),
		forestCooldowns: make(map[int64]time.Time),
		playerLocation:  make(map[int64]string),
	}
}

func (h *BotHandlers) HandleUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		h.handleMessage(update.Message)
	}
	if update.CallbackQuery != nil {
		h.handleCallbackQuery(update.CallbackQuery)
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
	case "◀️ Назад":
		h.handleBack(message)
	case "⛏ Шахта":
		h.handleMine(message)
	case "🌾 Поле":
		h.handleField(message)
	case "🎣 Озеро":
		h.handleLake(message)
	case "🏞 Лес":
		h.handleForest(message)
	case "/create_axe":
		h.handleCreateAxe(message)
	case "/create_pickaxe":
		h.handleCreatePickaxe(message)
	case "/create_bow":
		h.handleCreateBow(message)
	case "/create_arrows":
		h.handleCreateArrows(message)
	case "/create_knife":
		h.handleCreateKnife(message)
	case "/create_fishing_rod":
		h.handleCreateFishingRod(message)
	case "/create_birch_plank":
		h.handleCreateBirchPlank(message)
	case "/eat":
		h.handleEat(message)
	case "🎯 Охота":
		h.handleHunting(message)
	case "🌿 Сбор":
		h.handleForestGathering(message)
	case "🪓 Рубка":
		h.handleChopping(message)
	default:
		// Неизвестная команда
		msg := tgbotapi.NewMessage(message.Chat.ID, "Неизвестная команда. Используйте /start для начала игры.")
		h.sendMessage(msg)
	}
}

func (h *BotHandlers) handleStart(message *tgbotapi.Message) {
	userID := message.From.ID

	// Проверяем, зарегистрирован ли пользователь
	exists, err := h.db.PlayerExists(userID)
	if err != nil {
		log.Printf("Error checking player existence: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	if exists {
		// Пользователь уже зарегистрирован
		player, err := h.db.GetPlayer(userID)
		if err != nil {
			log.Printf("Error getting player: %v", err)
			msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
			h.sendMessage(msg)
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
	h.sendMessage(msg)

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
		h.sendMessage(msg2)

		// Сразу после второго сообщения просим имя
		nameMsg := tgbotapi.NewMessage(message.Chat.ID, "Придумай себе имя:")
		h.sendMessage(nameMsg)

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
		h.sendMessage(msg)
		return
	}

	// Создаем игрока в базе данных
	player, err := h.db.CreatePlayer(userID, name)
	if err != nil {
		log.Printf("Error creating player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при регистрации. Попробуйте позже.")
		h.sendMessage(msg)
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
		h.sendMessage(msg)
		return
	}

	profileText := fmt.Sprintf(`👤 Профиль игрока
Имя: %s
Уровень: %d
Опыт: %d/100
Сытость: %d/100`, player.Name, player.Level, player.Experience, player.Satiety)

	msg := tgbotapi.NewMessage(message.Chat.ID, profileText)
	h.sendMessage(msg)
}

func (h *BotHandlers) handleInventory(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала зарегистрируйтесь с помощью команды /start")
		h.sendMessage(msg)
		return
	}

	// Получаем инвентарь
	inventory, err := h.db.GetPlayerInventory(player.ID)
	if err != nil {
		log.Printf("Error getting inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении инвентаря.")
		h.sendMessage(msg)
		return
	}

	if len(inventory) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "🎒 Ваш инвентарь пуст.")
		h.sendMessage(msg)
		return
	}

	// Формируем текст инвентаря
	inventoryText := "🎒 Ваш инвентарь:\n\n"
	for _, item := range inventory {
		if item.Type == "tool" && item.Durability > 0 {
			inventoryText += fmt.Sprintf("%s - %d шт. (Прочность: %d/100)\n", item.ItemName, item.Quantity, item.Durability)
		} else if item.ItemName == "Лесная ягода" {
			inventoryText += fmt.Sprintf("%s - %d шт. /eat\n", item.ItemName, item.Quantity)
		} else {
			inventoryText += fmt.Sprintf("%s - %d шт.\n", item.ItemName, item.Quantity)
		}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, inventoryText)
	h.sendMessage(msg)
}

func (h *BotHandlers) handleEat(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала зарегистрируйтесь с помощью команды /start")
		h.sendMessage(msg)
		return
	}

	// Получаем инвентарь
	inventory, err := h.db.GetPlayerInventory(player.ID)
	if err != nil {
		log.Printf("Error getting inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении инвентаря.")
		h.sendMessage(msg)
		return
	}

	// Ищем ягоды в инвентаре
	var berryItem *models.InventoryItem
	for i, item := range inventory {
		if item.ItemName == "Лесная ягода" && item.Quantity > 0 {
			berryItem = &inventory[i]
			break
		}
	}

	if berryItem == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет ягод для употребления.")
		h.sendMessage(msg)
		return
	}

	// Уменьшаем количество ягод в инвентаре
	err = h.db.ConsumeItem(player.ID, "Лесная ягода", 1)
	if err != nil {
		log.Printf("Error consuming berry: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при употреблении ягоды.")
		h.sendMessage(msg)
		return
	}

	// Сохраняем текущую сытость для расчета реального прироста
	oldSatiety := player.Satiety

	// Увеличиваем сытость на 5
	err = h.db.UpdatePlayerSatiety(player.ID, 5)
	if err != nil {
		log.Printf("Error updating satiety: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при обновлении сытости.")
		h.sendMessage(msg)
		return
	}

	// Получаем обновленные данные игрока
	updatedPlayer, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting updated player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении обновленных данных.")
		h.sendMessage(msg)
		return
	}

	// Вычисляем реальный прирост сытости
	actualSatietyGain := updatedPlayer.Satiety - oldSatiety

	// Отправляем сообщение об успешном употреблении
	responseText := fmt.Sprintf("Съеден предмет \"Ягода\", сытость пополнена на %d ед.\nСытость: %d/100", actualSatietyGain, updatedPlayer.Satiety)
	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	h.sendMessage(msg)
}

func (h *BotHandlers) handleGathering(message *tgbotapi.Message) {
	gatheringText := `🌿 Ты собрался в путь за ресурсами.

Выбери, куда хочешь отправиться:

🏞 Лес — древесина, охота, ягоды  
⛏ Шахта — камень, руда, уголь  
🌾 Поле — травы, злаки, редкие растения  
🎣 Озеро — рыбалка и вода`

	msg := tgbotapi.NewMessage(message.Chat.ID, gatheringText)
	h.sendGatheringKeyboard(msg)
}

func (h *BotHandlers) handleWorkplace(message *tgbotapi.Message) {
	workplaceText := `🔨 Ты подходишь к рабочему месту.

Здесь можно создавать новые предметы и обрабатывать ресурсы.`

	msg := tgbotapi.NewMessage(message.Chat.ID, workplaceText)
	h.sendWorkplaceKeyboard(msg)
}

func (h *BotHandlers) handleWorkbench(message *tgbotapi.Message) {
	workbenchText := `🛠 Доступные предметы для создания:

Березовый брус — /create_birch_plank
Простой топор — /create_axe
Простая кирка — /create_pickaxe
Простой лук — /create_bow
Стрелы — /create_arrows
Простой нож — /create_knife
Простая удочка — /create_fishing_rod`

	msg := tgbotapi.NewMessage(message.Chat.ID, workbenchText)
	h.sendMessage(msg)
}

func (h *BotHandlers) handleFurnace(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🧱 Функция печи пока в разработке...")
	h.sendMessage(msg)
}

func (h *BotHandlers) handleCampfire(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🔥 Функция костра пока в разработке...")
	h.sendMessage(msg)
}

func (h *BotHandlers) handleBack(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID

	// Проверяем, идет ли добыча ресурса или рубка
	if _, isMining := h.miningTimers[userID]; isMining {
		// Если идет добыча, не позволяем выйти
		msg := tgbotapi.NewMessage(chatID, "Идет добыча ресурса.")
		h.sendMessage(msg)
		return
	}

	if _, isChopping := h.choppingTimers[userID]; isChopping {
		// Если идет рубка, не позволяем выйти
		msg := tgbotapi.NewMessage(chatID, "Идет рубка дерева.")
		h.sendMessage(msg)
		return
	}

	// Проверяем, есть ли активная сессия шахты
	if session, exists := h.mineSessions[userID]; exists {
		// Удаляем сообщение с полем шахты
		deleteFieldMsg := tgbotapi.NewDeleteMessage(chatID, session.FieldMessageID)
		h.requestAPI(deleteFieldMsg)

		// Удаляем сообщение с информацией о шахте
		deleteInfoMsg := tgbotapi.NewDeleteMessage(chatID, session.InfoMessageID)
		h.requestAPI(deleteInfoMsg)

		// Удаляем сессию шахты
		delete(h.mineSessions, userID)

		// Возвращаемся в меню добычи
		msg := tgbotapi.NewMessage(chatID, "🌿 Выберите место для добычи ресурсов:")
		h.sendGatheringKeyboard(msg)
		return
	}

	// Проверяем, есть ли активная сессия леса
	if session, exists := h.forestSessions[userID]; exists {
		// Удаляем сообщение с полем леса
		deleteFieldMsg := tgbotapi.NewDeleteMessage(chatID, session.FieldMessageID)
		h.requestAPI(deleteFieldMsg)

		// Удаляем сообщение с информацией о лесе
		deleteInfoMsg := tgbotapi.NewDeleteMessage(chatID, session.InfoMessageID)
		h.requestAPI(deleteInfoMsg)

		// Удаляем сессию леса
		delete(h.forestSessions, userID)

		// Возвращаемся в меню леса
		msg := tgbotapi.NewMessage(chatID, "🌲 Ты входишь в густой лес. Под ногами хрустит трава, в кронах поют птицы, а где-то вдалеке слышен треск ветки — ты здесь не один...\n\nЗдесь ты можешь:\n🪓 Рубить деревья\n🎯 Охотиться на дичь\n🌿 Собирать травы и ягоды")
		h.sendForestKeyboard(msg)
		return
	}

	// Проверяем текущее местоположение игрока
	if location, exists := h.playerLocation[userID]; exists && location == "forest" {
		// Игрок в лесу - возвращаемся в меню добычи
		delete(h.playerLocation, userID) // Убираем местоположение
		msg := tgbotapi.NewMessage(chatID, "🌿 Выберите место для добычи ресурсов:")
		h.sendGatheringKeyboard(msg)
	} else {
		// Обычное возвращение в главное меню
		msg := tgbotapi.NewMessage(chatID, "🏠 Возвращаемся к главному меню.")
		h.sendWithKeyboard(msg)
	}
}

func (h *BotHandlers) handleMine(message *tgbotapi.Message) {
	userID := message.From.ID

	// Проверяем, активен ли кулдаун шахты
	if cooldownEnd, exists := h.mineCooldowns[userID]; exists {
		if time.Now().Before(cooldownEnd) {
			// Кулдаун еще активен
			remainingTime := time.Until(cooldownEnd)
			msg := tgbotapi.NewMessage(message.Chat.ID,
				fmt.Sprintf("До обновления шахты осталось %d сек.", int(remainingTime.Seconds())))
			h.sendMessage(msg)
			return
		} else {
			// Кулдаун истек, удаляем его
			delete(h.mineCooldowns, userID)
		}
	}

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала зарегистрируйтесь с помощью команды /start")
		h.sendMessage(msg)
		return
	}

	// Проверяем сытость игрока
	if player.Satiety <= 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сытость 0. Необходимо поесть.")
		h.sendMessage(msg)
		return
	}

	// Получаем или создаем шахту
	mine, err := h.db.GetOrCreateMine(player.ID)
	if err != nil {
		log.Printf("Error getting mine: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при работе с шахтой.")
		h.sendMessage(msg)
		return
	}

	// Если шахта была истощена в базе данных, восстанавливаем её
	if mine.IsExhausted {
		if err := h.db.SetMineExhausted(player.ID, false); err != nil {
			log.Printf("Error setting mine exhausted: %v", err)
		}
		mine.IsExhausted = false
	}

	// Создаем новую сессию шахты
	h.createNewMineSession(userID, message.Chat.ID, mine)
}

func (h *BotHandlers) createNewMineSession(userID int64, chatID int64, mine *models.Mine) {
	// Генерируем случайное поле
	field := h.generateRandomMineField()

	// Показываем поле и получаем MessageID
	fieldMessageID, infoMessageID := h.showMineField(chatID, mine, field)

	// Создаем сессию
	session := &models.MineSession{
		PlayerID:       userID,
		Resources:      field,
		IsActive:       true,
		IsMining:       false,
		StartedAt:      time.Now(),
		FieldMessageID: fieldMessageID,
		InfoMessageID:  infoMessageID,
	}

	h.mineSessions[userID] = session
}

func (h *BotHandlers) showMineField(chatID int64, mine *models.Mine, field [][]string) (int, int) {
	// Создаем инлайн клавиатуру на основе переданного поля
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < 3; i++ {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3; j++ {
			cell := field[i][j]
			var callbackData string

			switch cell {
			case "🪨":
				callbackData = fmt.Sprintf("mine_stone_%d_%d", i, j)
			case "⚫":
				callbackData = fmt.Sprintf("mine_coal_%d_%d", i, j)
			default:
				callbackData = fmt.Sprintf("mine_empty_%d_%d", i, j)
				cell = " "
			}

			button := tgbotapi.NewInlineKeyboardButtonData(cell, callbackData)
			row = append(row, button)
		}
		keyboard = append(keyboard, row)
	}

	// Сначала отправляем поле шахты с инлайн кнопками
	fieldMsg := tgbotapi.NewMessage(chatID, "Выберите ресурс для добычи:")
	fieldMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	fieldResponse, _ := h.sendChattableWithResponse(fieldMsg)

	// Затем отправляем информационное сообщение с клавиатурой
	// Вычисляем опыт до следующего уровня
	expToNext := mine.Level*100 - mine.Experience

	infoText := fmt.Sprintf(`⛏ Шахта (Уровень %d)
До следующего уровня: %d опыта

Доступные ресурсы:
🪨 Камень
⚫ Уголь`, mine.Level, expToNext)

	mineKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	mineKeyboard.ResizeKeyboard = true

	infoMsg := tgbotapi.NewMessage(chatID, infoText)
	infoMsg.ReplyMarkup = mineKeyboard
	infoResponse, _ := h.sendChattableWithResponse(infoMsg)

	// Возвращаем ID поля и ID информационного сообщения
	return fieldResponse.MessageID, infoResponse.MessageID
}

func (h *BotHandlers) generateRandomMineField() [][]string {
	// Создаем пустое поле 3x3
	field := make([][]string, 3)
	for i := range field {
		field[i] = make([]string, 3)
	}

	// Доступные ресурсы
	availableResources := []string{"🪨", "⚫"}

	// Используем время для псевдослучайности
	now := time.Now()
	seed := now.UnixNano()

	// Создаем список всех позиций
	positions := [][2]int{
		{0, 0}, {0, 1}, {0, 2},
		{1, 0}, {1, 1}, {1, 2},
		{2, 0}, {2, 1}, {2, 2},
	}

	// Перемешиваем позиции с использованием времени
	for i := len(positions) - 1; i > 0; i-- {
		j := int((seed + int64(i*13)) % int64(i+1))
		positions[i], positions[j] = positions[j], positions[i]
	}

	// Размещаем 3 ресурса в первых 3 позициях
	for i := 0; i < 3; i++ {
		pos := positions[i]
		// Выбираем ресурс псевдослучайно
		resourceIndex := int((seed + int64(i*17) + int64(pos[0]*3) + int64(pos[1])) % int64(len(availableResources)))
		resourceType := availableResources[resourceIndex]
		field[pos[0]][pos[1]] = resourceType
	}

	return field
}

func (h *BotHandlers) createProgressBar(current, total int) string {
	// Создаем прогресс бар из 10 блоков
	barLength := 10
	filled := (current * barLength) / total
	if filled > barLength {
		filled = barLength
	}

	progressBar := ""
	for i := 0; i < barLength; i++ {
		if i < filled {
			progressBar += "🟩" // Заполненный блок
		} else {
			progressBar += "⬜" // Пустой блок
		}
	}

	return progressBar
}

func (h *BotHandlers) updateMiningProgress(userID int64, chatID int64, messageID int, resourceName string, totalDuration int, durability int, row, col int) {
	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime).Seconds()
			progress := int(elapsed)

			if progress >= totalDuration {
				// Добыча завершена
				h.completeMining(userID, chatID, resourceName, durability, messageID, row, col)
				return
			}

			// Обновляем прогресс бар
			percentage := int((elapsed / float64(totalDuration)) * 100)
			progressBar := h.createProgressBar(progress, totalDuration)

			newText := fmt.Sprintf(`Началась добыча ресурса "%s". Время добычи %d сек.
			
%s %d%%`, resourceName, totalDuration, progressBar, percentage)

			// Редактируем сообщение
			editMsg := tgbotapi.NewEditMessageText(chatID, messageID, newText)
			h.editMessage(editMsg)

		case <-time.After(time.Duration(totalDuration+1) * time.Second):
			// Таймаут на случай, если что-то пошло не так
			return
		}
	}
}

func (h *BotHandlers) handleField(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🌾 Функция поля пока в разработке...")
	h.sendMessage(msg)
}

func (h *BotHandlers) handleLake(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🎣 Функция озера пока в разработке...")
	h.sendMessage(msg)
}

func (h *BotHandlers) handleForest(message *tgbotapi.Message) {
	userID := message.From.ID

	// Устанавливаем местоположение игрока
	h.playerLocation[userID] = "forest"

	forestText := `🌲 Ты входишь в густой лес. Под ногами хрустит трава, в кронах поют птицы, а где-то вдалеке слышен треск ветки — ты здесь не один...

Здесь ты можешь:
🪓 Рубить деревья  
🎯 Охотиться на дичь  
🌿 Собирать травы и ягоды`

	msg := tgbotapi.NewMessage(message.Chat.ID, forestText)
	h.sendForestKeyboard(msg)
}

func (h *BotHandlers) handleHunting(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🎯 Функция охоты пока в разработке...")
	h.sendMessage(msg)
}

func (h *BotHandlers) handleChopping(message *tgbotapi.Message) {
	userID := message.From.ID

	// Проверяем, активен ли кулдаун леса
	if cooldownEnd, exists := h.forestCooldowns[userID]; exists {
		if time.Now().Before(cooldownEnd) {
			// Кулдаун еще активен
			remainingTime := time.Until(cooldownEnd)
			msg := tgbotapi.NewMessage(message.Chat.ID,
				fmt.Sprintf("До обновления леса осталось %d сек.", int(remainingTime.Seconds())))
			h.sendMessage(msg)
			return
		} else {
			// Кулдаун истек, удаляем его
			delete(h.forestCooldowns, userID)
		}
	}

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала зарегистрируйтесь с помощью команды /start")
		h.sendMessage(msg)
		return
	}

	// Проверяем сытость игрока
	if player.Satiety <= 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сытость 0. Необходимо поесть.")
		h.sendMessage(msg)
		return
	}

	// Получаем или создаем лес
	forest, err := h.db.GetOrCreateForest(player.ID)
	if err != nil {
		log.Printf("Error getting forest: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при работе с лесом.")
		h.sendMessage(msg)
		return
	}

	// Если лес был истощен в базе данных, восстанавливаем его
	if forest.IsExhausted {
		if err := h.db.SetForestExhausted(player.ID, false); err != nil {
			log.Printf("Error setting forest exhausted: %v", err)
		}
		forest.IsExhausted = false
	}

	// Создаем новую сессию леса
	h.createNewForestSession(userID, message.Chat.ID, forest)
}

func (h *BotHandlers) createNewForestSession(userID int64, chatID int64, forest *models.Forest) {
	// Генерируем случайное поле
	field := h.generateRandomForestField()

	// Показываем поле и получаем MessageID
	fieldMessageID, infoMessageID := h.showForestField(chatID, forest, field)

	// Создаем сессию
	session := &models.ForestSession{
		PlayerID:       userID,
		Resources:      field,
		IsActive:       true,
		IsChopping:     false,
		StartedAt:      time.Now(),
		FieldMessageID: fieldMessageID,
		InfoMessageID:  infoMessageID,
	}

	h.forestSessions[userID] = session
}

func (h *BotHandlers) generateRandomForestField() [][]string {
	// Создаем пустое поле 3x3
	field := make([][]string, 3)
	for i := range field {
		field[i] = make([]string, 3)
	}

	// Доступные ресурсы для леса
	availableResources := []string{"🌳"}

	// Используем время для псевдослучайности
	now := time.Now()
	seed := now.UnixNano()

	// Создаем список всех позиций
	positions := [][2]int{
		{0, 0}, {0, 1}, {0, 2},
		{1, 0}, {1, 1}, {1, 2},
		{2, 0}, {2, 1}, {2, 2},
	}

	// Перемешиваем позиции с использованием времени
	for i := len(positions) - 1; i > 0; i-- {
		j := int((seed + int64(i*13)) % int64(i+1))
		positions[i], positions[j] = positions[j], positions[i]
	}

	// Размещаем 3 ресурса в первых 3 позициях
	for i := 0; i < 3; i++ {
		pos := positions[i]
		// Выбираем ресурс псевдослучайно
		resourceIndex := int((seed + int64(i*17) + int64(pos[0]*3) + int64(pos[1])) % int64(len(availableResources)))
		resourceType := availableResources[resourceIndex]
		field[pos[0]][pos[1]] = resourceType
	}

	return field
}

func (h *BotHandlers) showForestField(chatID int64, forest *models.Forest, field [][]string) (int, int) {
	// Создаем инлайн клавиатуру на основе переданного поля
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < 3; i++ {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3; j++ {
			cell := field[i][j]
			var callbackData string

			switch cell {
			case "🌳":
				callbackData = fmt.Sprintf("forest_birch_%d_%d", i, j)
			default:
				callbackData = fmt.Sprintf("forest_empty_%d_%d", i, j)
				cell = " "
			}

			button := tgbotapi.NewInlineKeyboardButtonData(cell, callbackData)
			row = append(row, button)
		}
		keyboard = append(keyboard, row)
	}

	// Сначала отправляем поле леса с инлайн кнопками
	fieldMsg := tgbotapi.NewMessage(chatID, "Выберите дерево для рубки:")
	fieldMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	fieldResponse, _ := h.sendChattableWithResponse(fieldMsg)

	// Затем отправляем информационное сообщение с клавиатурой
	// Вычисляем опыт до следующего уровня
	expToNext := forest.Level*100 - forest.Experience

	infoText := fmt.Sprintf(`🪓 Рубка (Уровень %d)
До следующего уровня: %d опыта

Доступные ресурсы:
🌳 Береза`, forest.Level, expToNext)

	forestKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	forestKeyboard.ResizeKeyboard = true

	infoMsg := tgbotapi.NewMessage(chatID, infoText)
	infoMsg.ReplyMarkup = forestKeyboard
	infoResponse, _ := h.sendChattableWithResponse(infoMsg)

	// Возвращаем ID поля и ID информационного сообщения
	return fieldResponse.MessageID, infoResponse.MessageID
}

func (h *BotHandlers) handleForestGathering(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🌿 Функция сбора растений пока в разработке...")
	h.sendMessage(msg)
}

func (h *BotHandlers) handleCreateAxe(message *tgbotapi.Message) {
	h.showRecipe(message, "Простой топор")
}

func (h *BotHandlers) handleCreatePickaxe(message *tgbotapi.Message) {
	h.showRecipe(message, "Простая кирка")
}

func (h *BotHandlers) handleCreateBow(message *tgbotapi.Message) {
	h.showRecipe(message, "Простой лук")
}

func (h *BotHandlers) handleCreateArrows(message *tgbotapi.Message) {
	h.showRecipe(message, "Стрелы")
}

func (h *BotHandlers) handleCreateKnife(message *tgbotapi.Message) {
	h.showRecipe(message, "Простой нож")
}

func (h *BotHandlers) handleCreateFishingRod(message *tgbotapi.Message) {
	h.showRecipe(message, "Простая удочка")
}

func (h *BotHandlers) handleCreateBirchPlank(message *tgbotapi.Message) {
	h.showRecipe(message, "Березовый брус")
}

func (h *BotHandlers) showRecipe(message *tgbotapi.Message, itemName string) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала зарегистрируйтесь с помощью команды /start")
		h.sendMessage(msg)
		return
	}

	// Получаем рецепт
	recipe, err := h.db.GetRecipeRequirements(itemName)
	if err != nil {
		log.Printf("Error getting recipe: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Ошибка получения рецепта.")
		h.sendMessage(msg)
		return
	}

	// Формируем текст рецепта
	recipeText := fmt.Sprintf(`Для изготовления предмета "%s" необходимо следующее:`, itemName)
	canCraft := true

	for _, ingredient := range recipe {
		playerQuantity, err := h.db.GetItemQuantityInInventory(player.ID, ingredient.ItemName)
		if err != nil {
			log.Printf("Error getting inventory quantity: %v", err)
			playerQuantity = 0
		}

		if playerQuantity < ingredient.Quantity {
			canCraft = false
		}

		recipeText += fmt.Sprintf("\n%s - %d/%d шт.", ingredient.ItemName, playerQuantity, ingredient.Quantity)
	}

	// Добавляем кнопку "Создать"
	var buttonText string
	if canCraft {
		buttonText = "Создать ✅"
	} else {
		buttonText = "Создать ❌"
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, recipeText)

	// Создаем инлайн клавиатуру с кнопкой создать
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("craft_%s", itemName)),
		),
	)
	msg.ReplyMarkup = keyboard
	h.sendMessage(msg)
}

func (h *BotHandlers) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	data := callback.Data

	// Обрабатываем callback'и от шахты
	if strings.HasPrefix(data, "mine_stone_") {
		parts := strings.Split(data, "_")
		if len(parts) == 4 {
			row, col := parts[2], parts[3]
			h.startMiningAtPosition(userID, callback.Message.Chat.ID, "Камень", 10, callback.ID, row, col)
		}
	} else if strings.HasPrefix(data, "mine_coal_") {
		parts := strings.Split(data, "_")
		if len(parts) == 4 {
			row, col := parts[2], parts[3]
			h.startMiningAtPosition(userID, callback.Message.Chat.ID, "Уголь", 20, callback.ID, row, col)
		}
	} else if strings.HasPrefix(data, "mine_empty_") {
		// Пустая ячейка
		callbackConfig := tgbotapi.NewCallback(callback.ID, "Здесь нет ресурсов!")
		h.requestAPI(callbackConfig)
	} else if strings.HasPrefix(data, "forest_birch_") {
		// Обрабатываем callback'и от леса
		parts := strings.Split(data, "_")
		if len(parts) == 4 {
			row, col := parts[2], parts[3]
			h.startChoppingAtPosition(userID, callback.Message.Chat.ID, "Береза", 10, callback.ID, row, col)
		}
	} else if strings.HasPrefix(data, "forest_empty_") {
		// Пустая ячейка
		callbackConfig := tgbotapi.NewCallback(callback.ID, "Здесь нет деревьев!")
		h.requestAPI(callbackConfig)
	} else {
		// Остальные callback (крафт и т.д.)
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "🔨 Функция создания предметов пока в разработке...")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		h.requestAPI(callbackConfig)
	}
}

func (h *BotHandlers) startMiningAtPosition(userID int64, chatID int64, resourceName string, duration int, callbackID string, rowStr, colStr string) {
	row, _ := strconv.Atoi(rowStr)
	col, _ := strconv.Atoi(colStr)

	h.startMining(userID, chatID, resourceName, duration, callbackID, row, col)
}

func (h *BotHandlers) startMining(userID int64, chatID int64, resourceName string, duration int, callbackID string, row, col int) {
	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		return
	}

	// Проверяем наличие кирки
	hasTool, durability, err := h.db.HasToolInInventory(player.ID, "Простая кирка")
	if err != nil {
		log.Printf("Error checking tool: %v", err)
		return
	}

	if !hasTool {
		msg := tgbotapi.NewMessage(chatID, `В инвентаре нет предмета "Простая кирка".`)
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}

	// Отвечаем на callback
	callbackConfig := tgbotapi.NewCallback(callbackID, "")
	h.requestAPI(callbackConfig)

	// Удаляем предыдущее сообщение о результате добычи, если оно существует
	if session, exists := h.mineSessions[userID]; exists && session.ResultMessageID != 0 {
		deleteResultMsg := tgbotapi.NewDeleteMessage(chatID, session.ResultMessageID)
		h.requestAPI(deleteResultMsg)
		session.ResultMessageID = 0 // Сбрасываем ID
	}

	// Отправляем сообщение о начале добычи (не удаляем информационное сообщение)
	initialText := fmt.Sprintf(`Началась добыча ресурса "%s". Время добычи %d сек.
		
%s 0%%`, resourceName, duration, h.createProgressBar(0, 10))

	miningMsg := tgbotapi.NewMessage(chatID, initialText)
	sentMsg, _ := h.sendMessageWithResponse(miningMsg)

	// Запускаем горутину для обновления прогресс бара
	go h.updateMiningProgress(userID, chatID, sentMsg.MessageID, resourceName, duration, durability, row, col)

	// Создаем заглушку таймера (основная логика теперь в updateMiningProgress)
	timer := time.NewTimer(time.Duration(duration) * time.Second)
	h.miningTimers[userID] = timer
}

func (h *BotHandlers) completeMining(userID int64, chatID int64, resourceName string, oldDurability int, messageID int, row, col int) {
	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		return
	}

	// Добавляем ресурс в инвентарь
	if err := h.db.AddItemToInventory(player.ID, resourceName, 1); err != nil {
		log.Printf("Error adding item to inventory: %v", err)
	}

	// Обновляем прочность кирки и сытость (при добыче игрок тратит энергию)
	if err := h.db.UpdateItemDurability(player.ID, "Простая кирка", 1); err != nil {
		log.Printf("Error updating item durability: %v", err)
	}
	if err := h.db.UpdatePlayerSatiety(player.ID, -1); err != nil {
		log.Printf("Error updating player satiety: %v", err)
	}

	// Добавляем опыт шахте и проверяем повышение уровня
	levelUp, newLevel, err := h.db.UpdateMineExperience(player.ID, 2)
	if err != nil {
		log.Printf("Error updating mine experience: %v", err)
		return
	}

	// Получаем обновленные данные
	updatedPlayer, _ := h.db.GetPlayer(userID)
	mine, _ := h.db.GetOrCreateMine(player.ID)

	// Удаляем сообщение о добыче
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)

	// Показываем результат
	resultText := fmt.Sprintf(`✅ Ты добыл %s!
Получено опыта: 2
Сытость: %d/100
Прочность кирки: %d/100
До следующего уровня: %d опыта`,
		resourceName,
		updatedPlayer.Satiety,
		oldDurability-1,
		mine.Level*100-mine.Experience)

	msg := tgbotapi.NewMessage(chatID, resultText)
	resultResponse, _ := h.sendMessageWithResponse(msg)

	// Если уровень повысился, показываем сообщение о повышении
	if levelUp {
		levelUpText := fmt.Sprintf("🎉 Поздравляем! Уровень шахты повышен до %d уровня!", newLevel)
		levelUpMsg := tgbotapi.NewMessage(chatID, levelUpText)
		h.sendMessage(levelUpMsg)
	}

	// Сохраняем ID сообщения с результатом в сессии
	if session, exists := h.mineSessions[userID]; exists {
		session.ResultMessageID = resultResponse.MessageID
	}

	// Кнопка "Назад" остается активной, не нужно восстанавливать

	// Убираем таймер
	delete(h.miningTimers, userID)

	// Обновляем поле - убираем добытый ресурс
	if session, exists := h.mineSessions[userID]; exists {
		session.Resources[row][col] = ""

		// Проверяем, остались ли ресурсы в поле
		totalResources := 0
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if session.Resources[i][j] != "" {
					totalResources++
				}
			}
		}

		if totalResources > 0 {
			// Обновляем инлайн клавиатуру с новым состоянием поля
			h.updateMineField(chatID, session.Resources, session.FieldMessageID)
			// Обновляем информационное сообщение с актуальными данными
			h.updateMineInfoMessage(userID, chatID, mine, session.InfoMessageID)
		} else {
			// Поле истощено, устанавливаем кулдаун
			if err := h.db.ExhaustMine(userID); err != nil {
				log.Printf("Error exhausting mine: %v", err)
			}

			// Устанавливаем таймер кулдауна на 60 секунд
			h.mineCooldowns[userID] = time.Now().Add(60 * time.Second)

			// Удаляем сообщение с полем шахты
			deleteFieldMsg := tgbotapi.NewDeleteMessage(chatID, session.FieldMessageID)
			h.requestAPI(deleteFieldMsg)

			// Удаляем сообщение с информацией о шахте
			deleteInfoMsg := tgbotapi.NewDeleteMessage(chatID, session.InfoMessageID)
			h.requestAPI(deleteInfoMsg)

			exhaustMsg := tgbotapi.NewMessage(chatID, `⚠️ Шахта истощена! Необходимо подождать 1 минуту до восстановления ресурсов.
Нажми кнопку "⛏ Шахта" чтобы проверить готовность.`)
			h.sendGatheringKeyboard(exhaustMsg)

			// Удаляем сессию
			delete(h.mineSessions, userID)
		}
	}
}

func (h *BotHandlers) updateMineField(chatID int64, field [][]string, messageID int) {
	text := "Выберите ресурс для добычи:"

	// Создаем инлайн клавиатуру на основе переданного поля
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < 3; i++ {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3; j++ {
			cell := field[i][j]
			var callbackData string

			switch cell {
			case "🪨":
				callbackData = fmt.Sprintf("mine_stone_%d_%d", i, j)
			case "⚫":
				callbackData = fmt.Sprintf("mine_coal_%d_%d", i, j)
			default:
				callbackData = fmt.Sprintf("mine_empty_%d_%d", i, j)
				cell = " "
			}

			button := tgbotapi.NewInlineKeyboardButtonData(cell, callbackData)
			row = append(row, button)
		}
		keyboard = append(keyboard, row)
	}

	// Редактируем существующее сообщение вместо отправки нового
	editMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, tgbotapi.NewInlineKeyboardMarkup(keyboard...))
	h.editMessage(editMsg)
}

func (h *BotHandlers) updateMineInfoMessage(userID int64, chatID int64, mine *models.Mine, messageID int) {
	// Вычисляем опыт до следующего уровня
	expToNext := mine.Level*100 - mine.Experience

	infoText := fmt.Sprintf(`⛏ Шахта (Уровень %d)
До следующего уровня: %d опыта

Доступные ресурсы:
🪨 Камень
⚫ Уголь`, mine.Level, expToNext)

	// Удаляем старое сообщение
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)

	// Создаем новое сообщение с клавиатурой
	mineKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	mineKeyboard.ResizeKeyboard = true

	newMsg := tgbotapi.NewMessage(chatID, infoText)
	newMsg.ReplyMarkup = mineKeyboard
	newResponse, _ := h.sendMessageWithResponse(newMsg)

	// Обновляем ID сообщения в сессии
	if session, exists := h.mineSessions[userID]; exists {
		session.InfoMessageID = newResponse.MessageID
	}
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
	h.sendMessage(msg)
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
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	keyboard.ResizeKeyboard = true
	msg.ReplyMarkup = keyboard
	h.sendMessage(msg)
}

func (h *BotHandlers) sendMineKeyboard(msg tgbotapi.MessageConfig) {
	// Создаем клавиатуру шахты
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	keyboard.ResizeKeyboard = true
	msg.ReplyMarkup = keyboard
	h.sendMessage(msg)
}

func (h *BotHandlers) sendGatheringKeyboard(msg tgbotapi.MessageConfig) {
	// Создаем клавиатуру добычи ресурсов
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⛏ Шахта"),
			tgbotapi.NewKeyboardButton("🌾 Поле"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🎣 Озеро"),
			tgbotapi.NewKeyboardButton("🏞 Лес"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	keyboard.ResizeKeyboard = true
	msg.ReplyMarkup = keyboard
	h.sendMessage(msg)
}

func (h *BotHandlers) sendForestKeyboard(msg tgbotapi.MessageConfig) {
	// Создаем клавиатуру леса
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🎯 Охота"),
			tgbotapi.NewKeyboardButton("🌿 Сбор"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🪓 Рубка"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	keyboard.ResizeKeyboard = true
	msg.ReplyMarkup = keyboard
	h.sendMessage(msg)
}

// Вспомогательная функция для отправки сообщений с обработкой ошибок
func (h *BotHandlers) sendMessage(msg tgbotapi.MessageConfig) {
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

// Вспомогательная функция для отправки сообщений с возвратом результата
func (h *BotHandlers) sendMessageWithResponse(msg tgbotapi.MessageConfig) (tgbotapi.Message, error) {
	response, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}
	return response, err
}

// Вспомогательная функция для редактирования сообщений с обработкой ошибок
func (h *BotHandlers) editMessage(editMsg tgbotapi.Chattable) {
	if _, err := h.bot.Send(editMsg); err != nil {
		log.Printf("Failed to edit message: %v", err)
	}
}

// Вспомогательная функция для отправки любого Chattable с возвратом результата
func (h *BotHandlers) sendChattableWithResponse(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	response, err := h.bot.Send(c)
	if err != nil {
		log.Printf("Failed to send chattable: %v", err)
	}
	return response, err
}

// Вспомогательная функция для отправки запросов к Telegram API с обработкой ошибок
func (h *BotHandlers) requestAPI(c tgbotapi.Chattable) {
	if _, err := h.bot.Request(c); err != nil {
		log.Printf("Failed to send API request: %v", err)
	}
}

// Функции для рубки леса
func (h *BotHandlers) startChoppingAtPosition(userID int64, chatID int64, resourceName string, duration int, callbackID string, rowStr, colStr string) {
	row, _ := strconv.Atoi(rowStr)
	col, _ := strconv.Atoi(colStr)

	h.startChopping(userID, chatID, resourceName, duration, callbackID, row, col)
}

func (h *BotHandlers) startChopping(userID int64, chatID int64, resourceName string, duration int, callbackID string, row, col int) {
	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		return
	}

	// Проверяем наличие топора
	hasTool, durability, err := h.db.HasToolInInventory(player.ID, "Простой топор")
	if err != nil {
		log.Printf("Error checking tool: %v", err)
		return
	}

	if !hasTool {
		msg := tgbotapi.NewMessage(chatID, `В инвентаре нет предмета "Простой топор".`)
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}

	// Отвечаем на callback
	callbackConfig := tgbotapi.NewCallback(callbackID, "")
	h.requestAPI(callbackConfig)

	// Удаляем предыдущее сообщение о результате рубки, если оно существует
	if session, exists := h.forestSessions[userID]; exists && session.ResultMessageID != 0 {
		deleteResultMsg := tgbotapi.NewDeleteMessage(chatID, session.ResultMessageID)
		h.requestAPI(deleteResultMsg)
		session.ResultMessageID = 0 // Сбрасываем ID
	}

	// Отправляем сообщение о начале рубки
	initialText := fmt.Sprintf(`Идет рубка дерева "%s". Время рубки %d сек.
		
%s 0%%`, resourceName, duration, h.createProgressBar(0, 10))

	choppingMsg := tgbotapi.NewMessage(chatID, initialText)
	sentMsg, _ := h.sendMessageWithResponse(choppingMsg)

	// Запускаем горутину для обновления прогресс бара
	go h.updateChoppingProgress(userID, chatID, sentMsg.MessageID, resourceName, duration, durability, row, col)

	// Создаем заглушку таймера
	timer := time.NewTimer(time.Duration(duration) * time.Second)
	h.choppingTimers[userID] = timer
}

func (h *BotHandlers) updateChoppingProgress(userID int64, chatID int64, messageID int, resourceName string, totalDuration int, durability int, row, col int) {
	startTime := time.Now()

	for {
		elapsed := time.Since(startTime)
		if elapsed >= time.Duration(totalDuration)*time.Second {
			break
		}

		percentage := int((elapsed.Seconds() / float64(totalDuration)) * 100)
		progressText := fmt.Sprintf(`Идет рубка дерева "%s". Время рубки %d сек.
			
%s %d%%`, resourceName, totalDuration, h.createProgressBar(percentage, 100), percentage)

		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, progressText)
		h.editMessage(editMsg)

		time.Sleep(1 * time.Second)
	}

	// После завершения показываем 100% и завершаем
	finalText := fmt.Sprintf(`Идет рубка дерева "%s". Время рубки %d сек.
		
%s 100%%`, resourceName, totalDuration, h.createProgressBar(100, 100))

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, finalText)
	h.editMessage(editMsg)

	// Завершаем рубку
	h.completeChopping(userID, chatID, resourceName, durability, messageID, row, col)
}

func (h *BotHandlers) completeChopping(userID int64, chatID int64, resourceName string, oldDurability int, messageID int, row, col int) {
	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		return
	}

	// Добавляем ресурс в инвентарь
	if err := h.db.AddItemToInventory(player.ID, resourceName, 1); err != nil {
		log.Printf("Error adding item to inventory: %v", err)
	}

	// Обновляем прочность топора и сытость
	if err := h.db.UpdateItemDurability(player.ID, "Простой топор", 1); err != nil {
		log.Printf("Error updating item durability: %v", err)
	}
	if err := h.db.UpdatePlayerSatiety(player.ID, -1); err != nil {
		log.Printf("Error updating player satiety: %v", err)
	}

	// Добавляем опыт лесу и проверяем повышение уровня
	levelUp, newLevel, err := h.db.UpdateForestExperience(player.ID, 2)
	if err != nil {
		log.Printf("Error updating forest experience: %v", err)
		return
	}

	// Получаем обновленные данные
	updatedPlayer, _ := h.db.GetPlayer(userID)
	forest, _ := h.db.GetOrCreateForest(player.ID)

	// Удаляем сообщение о рубке
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)

	// Показываем результат
	resultText := fmt.Sprintf(`✅ Ты срубил дерево "%s"!
Получено опыта: 2
Сытость: %d/100
Прочность топора: %d/100
До следующего уровня: %d опыта`,
		resourceName,
		updatedPlayer.Satiety,
		oldDurability-1,
		forest.Level*100-forest.Experience)

	msg := tgbotapi.NewMessage(chatID, resultText)
	resultResponse, _ := h.sendMessageWithResponse(msg)

	// Если уровень повысился, показываем сообщение о повышении
	if levelUp {
		levelUpText := fmt.Sprintf("🎉 Поздравляем! Уровень леса повышен до %d уровня!", newLevel)
		levelUpMsg := tgbotapi.NewMessage(chatID, levelUpText)
		h.sendMessage(levelUpMsg)
	}

	// Сохраняем ID сообщения с результатом в сессии
	if session, exists := h.forestSessions[userID]; exists {
		session.ResultMessageID = resultResponse.MessageID
	}

	// Убираем таймер
	delete(h.choppingTimers, userID)

	// Обновляем поле - убираем срубленное дерево
	if session, exists := h.forestSessions[userID]; exists {
		session.Resources[row][col] = ""

		// Проверяем, остались ли ресурсы в поле
		totalResources := 0
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if session.Resources[i][j] != "" {
					totalResources++
				}
			}
		}

		if totalResources > 0 {
			// Обновляем инлайн клавиатуру с новым состоянием поля
			h.updateForestField(chatID, session.Resources, session.FieldMessageID)
			// Обновляем информационное сообщение с актуальными данными
			h.updateForestInfoMessage(userID, chatID, forest, session.InfoMessageID)
		} else {
			// Поле истощено, устанавливаем кулдаун
			if err := h.db.ExhaustForest(userID); err != nil {
				log.Printf("Error exhausting forest: %v", err)
			}

			// Устанавливаем таймер кулдауна на 60 секунд
			h.forestCooldowns[userID] = time.Now().Add(60 * time.Second)

			// Удаляем сообщение с полем леса
			deleteFieldMsg := tgbotapi.NewDeleteMessage(chatID, session.FieldMessageID)
			h.requestAPI(deleteFieldMsg)

			// Удаляем сообщение с информацией о лесе
			deleteInfoMsg := tgbotapi.NewDeleteMessage(chatID, session.InfoMessageID)
			h.requestAPI(deleteInfoMsg)

			exhaustMsg := tgbotapi.NewMessage(chatID, `⚠️ Лес истощен! Необходимо подождать 1 минуту до восстановления деревьев.
Нажми кнопку "🪓 Рубка" чтобы проверить готовность.`)
			h.sendForestKeyboard(exhaustMsg)

			// Удаляем сессию
			delete(h.forestSessions, userID)
		}
	}
}

func (h *BotHandlers) updateForestField(chatID int64, field [][]string, messageID int) {
	text := "Выберите дерево для рубки:"

	// Создаем инлайн клавиатуру на основе переданного поля
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < 3; i++ {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3; j++ {
			cell := field[i][j]
			var callbackData string

			switch cell {
			case "🌳":
				callbackData = fmt.Sprintf("forest_birch_%d_%d", i, j)
			default:
				callbackData = fmt.Sprintf("forest_empty_%d_%d", i, j)
				cell = " "
			}

			button := tgbotapi.NewInlineKeyboardButtonData(cell, callbackData)
			row = append(row, button)
		}
		keyboard = append(keyboard, row)
	}

	// Редактируем существующее сообщение вместо отправки нового
	editMsg := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, tgbotapi.NewInlineKeyboardMarkup(keyboard...))
	h.editMessage(editMsg)
}

func (h *BotHandlers) updateForestInfoMessage(userID int64, chatID int64, forest *models.Forest, messageID int) {
	// Вычисляем опыт до следующего уровня
	expToNext := forest.Level*100 - forest.Experience

	infoText := fmt.Sprintf(`🪓 Рубка (Уровень %d)
До следующего уровня: %d опыта

Доступные ресурсы:
🌳 Береза`, forest.Level, expToNext)

	// Удаляем старое сообщение
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)

	// Создаем новое сообщение с клавиатурой
	forestKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	forestKeyboard.ResizeKeyboard = true

	newMsg := tgbotapi.NewMessage(chatID, infoText)
	newMsg.ReplyMarkup = forestKeyboard
	newResponse, _ := h.sendMessageWithResponse(newMsg)

	// Обновляем ID сообщения в сессии
	if session, exists := h.forestSessions[userID]; exists {
		session.InfoMessageID = newResponse.MessageID
	}
}
