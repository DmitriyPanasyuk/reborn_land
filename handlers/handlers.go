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
	bot                     *tgbotapi.BotAPI
	db                      *database.DB
	waitingForName          map[int64]bool
	waitingForCraftQuantity map[int64]string // Ожидание количества для крафта (значение - название предмета)
	mineSessions            map[int64]*models.MineSession
	forestSessions          map[int64]*models.ForestSession
	gatheringSessions       map[int64]*models.GatheringSession
	miningTimers            map[int64]*time.Timer
	choppingTimers          map[int64]*time.Timer
	gatheringTimers         map[int64]*time.Timer
	craftingTimers          map[int64]*time.Timer // Таймеры для крафта
	mineCooldowns           map[int64]time.Time   // Время окончания кулдауна шахты
	forestCooldowns         map[int64]time.Time   // Время окончания кулдауна леса
	gatheringCooldowns      map[int64]time.Time   // Время окончания кулдауна сбора
	playerLocation          map[int64]string      // Текущее местоположение игрока
}

func New(bot *tgbotapi.BotAPI, db *database.DB) *BotHandlers {
	return &BotHandlers{
		bot:                     bot,
		db:                      db,
		waitingForName:          make(map[int64]bool),
		waitingForCraftQuantity: make(map[int64]string),
		mineSessions:            make(map[int64]*models.MineSession),
		forestSessions:          make(map[int64]*models.ForestSession),
		gatheringSessions:       make(map[int64]*models.GatheringSession),
		miningTimers:            make(map[int64]*time.Timer),
		choppingTimers:          make(map[int64]*time.Timer),
		gatheringTimers:         make(map[int64]*time.Timer),
		craftingTimers:          make(map[int64]*time.Timer),
		mineCooldowns:           make(map[int64]time.Time),
		forestCooldowns:         make(map[int64]time.Time),
		gatheringCooldowns:      make(map[int64]time.Time),
		playerLocation:          make(map[int64]string),
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

	// Проверяем, ждем ли мы количество для крафта
	if itemName, exists := h.waitingForCraftQuantity[userID]; exists {
		h.handleCraftQuantityInput(message, itemName)
		return
	}

	// Проверяем, идет ли крафт
	if _, exists := h.craftingTimers[userID]; exists {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Нельзя совершать действия пока идет создание предметов.")
		h.sendMessage(msg)
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
	case "📜 Квесты":
		h.handleQuest(message)
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
	case "/create_simple_hut":
		h.handleCreateSimpleHut(message)
	case "/eat":
		h.handleEat(message)
	case "🎯 Охота":
		h.handleHunting(message)
	case "🌿 Сбор":
		h.handleForestGathering(message)
	case "🪓 Рубка":
		h.handleChopping(message)
	case "📖 Лор":
		h.handleLore(message)
	case "🗓️ Ежедневные":
		h.handleDailyQuests(message)
	case "📆 Еженедельные":
		h.handleWeeklyQuests(message)
	case "🏘️ Постройки":
		h.handleBuildings(message)
	case "/look":
		h.handleLookPages(message)
	case "/read":
		h.handleReadPage(message)
	case "/read1":
		h.handleReadPage1(message)
	case "/read2":
		h.handleReadPage2(message)
	case "/read3":
		h.handleReadPage3(message)
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

	// Запускаем последовательность сообщений с задержками
	go func() {
		// Ждем 2 секунды и отправляем второе сообщение
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

		// Ждем еще 2 секунды и отправляем третье сообщение
		time.Sleep(2 * time.Second)

		thirdText := `Мир пал — не в огне и не в крови,
а в молчании. Цивилизации исчезли, города заросли, 
знания рассыпались, словно пыль. 
Никто не помнит, что случилось. 
Осталась только Земля. 
Дикая, первобытная. Но она помнит...

Ты — один из первых, кто пробудился. 
Без имени, без памяти. Но с искрой внутри. 
Искрой Возрождения. Всё, что ты построишь, — будет первым шагом к пробуждению этого мира. 
А может, и правды...`

		msg3 := tgbotapi.NewMessage(message.Chat.ID, thirdText)
		h.sendMessage(msg3)

		// Сразу после третьего сообщения просим имя
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

func (h *BotHandlers) handleCraftQuantityInput(message *tgbotapi.Message, itemName string) {
	userID := message.From.ID
	quantityStr := strings.TrimSpace(message.Text)

	// Проверяем, что введено число
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Введите корректное количество (положительное число):")
		h.sendMessage(msg)
		return
	}

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Проверяем количество березы в инвентаре для березового бруса
	if itemName == "Березовый брус" {
		birchQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "Береза")
		if err != nil {
			log.Printf("Error getting birch quantity: %v", err)
			birchQuantity = 0
		}

		// Для березового бруса нужно 2 березы за 1 брус
		requiredBirch := quantity * 2
		if birchQuantity < requiredBirch {
			msg := tgbotapi.NewMessage(message.Chat.ID, `Недостаточно предмета "Береза".`)
			h.sendMessage(msg)
			// Убираем флаг ожидания количества
			delete(h.waitingForCraftQuantity, userID)
			return
		}

		// Начинаем крафт
		h.startCrafting(userID, message.Chat.ID, itemName, quantity)
	}

	// Убираем флаг ожидания количества
	delete(h.waitingForCraftQuantity, userID)
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
Telegram ID: %d
Уровень: %d
Опыт: %d/100
Сытость: %d/100`, player.Name, player.TelegramID, player.Level, player.Experience, player.Satiety)

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

	// Разделяем предметы на категории
	var regularItems []models.InventoryItem
	var pages []models.InventoryItem

	for _, item := range inventory {
		if strings.Contains(item.ItemName, "📖 Страница") {
			pages = append(pages, item)
		} else {
			regularItems = append(regularItems, item)
		}
	}

	// Проверяем, есть ли обычные предметы
	if len(regularItems) == 0 && len(pages) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "🎒 Ваш инвентарь пуст.")
		h.sendMessage(msg)
		return
	}

	// Формируем текст инвентаря
	inventoryText := "🎒 Ваш инвентарь:\n\n"

	// Добавляем обычные предметы
	for _, item := range regularItems {
		if item.Type == "tool" && item.Durability > 0 {
			inventoryText += fmt.Sprintf("%s - %d шт. (Прочность: %d/100)\n", item.ItemName, item.Quantity, item.Durability)
		} else if item.ItemName == "Лесная ягода" {
			inventoryText += fmt.Sprintf("%s - %d шт. /eat\n", item.ItemName, item.Quantity)
		} else {
			inventoryText += fmt.Sprintf("%s - %d шт.\n", item.ItemName, item.Quantity)
		}
	}

	// Добавляем раздел страниц, если они есть
	if len(pages) > 0 {
		inventoryText += "\n📖 Страницы: /look\n"
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

	if _, isGathering := h.gatheringTimers[userID]; isGathering {
		// Если идет сбор, не позволяем выйти
		msg := tgbotapi.NewMessage(chatID, "Идет сбор ягод.")
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

	// Проверяем, есть ли активная сессия сбора
	if session, exists := h.gatheringSessions[userID]; exists {
		// Удаляем сообщение с полем сбора
		deleteFieldMsg := tgbotapi.NewDeleteMessage(chatID, session.FieldMessageID)
		h.requestAPI(deleteFieldMsg)

		// Удаляем сообщение с информацией о сборе
		deleteInfoMsg := tgbotapi.NewDeleteMessage(chatID, session.InfoMessageID)
		h.requestAPI(deleteInfoMsg)

		// Удаляем сессию сбора
		delete(h.gatheringSessions, userID)

		// Возвращаемся в меню леса
		msg := tgbotapi.NewMessage(chatID, "🌲 Ты входишь в густой лес. Под ногами хрустит трава, в кронах поют птицы, а где-то вдалеке слышен треск ветки — ты здесь не один...\n\nЗдесь ты можешь:\n🪓 Рубить деревья\n🎯 Охотиться на дичь\n🌿 Собирать травы и ягоды")
		h.sendForestKeyboard(msg)
		return
	}

	// Проверяем текущее местоположение игрока
	if location, exists := h.playerLocation[userID]; exists {
		switch location {
		case "forest":
			// Игрок в лесу - возвращаемся в меню добычи
			delete(h.playerLocation, userID) // Убираем местоположение
			msg := tgbotapi.NewMessage(chatID, "🌿 Выберите место для добычи ресурсов:")
			h.sendGatheringKeyboard(msg)
		case "quest":
			// Игрок в меню квестов - возвращаемся в главное меню
			delete(h.playerLocation, userID) // Убираем местоположение
			msg := tgbotapi.NewMessage(chatID, "🏠 Возвращаемся к главному меню.")
			h.sendWithKeyboard(msg)
		default:
			// Обычное возвращение в главное меню
			msg := tgbotapi.NewMessage(chatID, "🏠 Возвращаемся к главному меню.")
			h.sendWithKeyboard(msg)
		}
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
	expToNext := (mine.Level * 100) - mine.Experience

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
	expToNext := (forest.Level * 100) - forest.Experience

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
	userID := message.From.ID

	// Проверяем, активен ли кулдаун сбора
	if cooldownEnd, exists := h.gatheringCooldowns[userID]; exists {
		if time.Now().Before(cooldownEnd) {
			// Кулдаун еще активен
			remainingTime := time.Until(cooldownEnd)
			msg := tgbotapi.NewMessage(message.Chat.ID,
				fmt.Sprintf("До обновления ягодных кустов осталось %d сек.", int(remainingTime.Seconds())))
			h.sendMessage(msg)
			return
		} else {
			// Кулдаун истек, удаляем его
			delete(h.gatheringCooldowns, userID)
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

	// Получаем или создаем данные о сборе
	gathering, err := h.db.GetOrCreateGathering(player.ID)
	if err != nil {
		log.Printf("Error getting gathering: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при работе со сбором.")
		h.sendMessage(msg)
		return
	}

	// Если сбор был истощен в базе данных, восстанавливаем его
	if gathering.IsExhausted {
		if err := h.db.SetGatheringExhausted(player.ID, false); err != nil {
			log.Printf("Error setting gathering exhausted: %v", err)
		}
	}

	// Создаем новую сессию сбора
	h.createNewGatheringSession(userID, message.Chat.ID)
}

func (h *BotHandlers) createNewGatheringSession(userID int64, chatID int64) {
	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player in createNewGatheringSession: %v", err)
		return
	}

	// Получаем или создаем данные о сборе
	gathering, err := h.db.GetOrCreateGathering(player.ID)
	if err != nil {
		log.Printf("Error getting gathering in createNewGatheringSession: %v", err)
		return
	}

	// Генерируем случайное поле
	field := h.generateRandomGatheringField()

	// Показываем поле и получаем MessageID
	fieldMessageID, infoMessageID := h.showGatheringField(chatID, field, gathering)

	// Создаем сессию
	session := &models.GatheringSession{
		PlayerID:       userID,
		Resources:      field,
		IsActive:       true,
		StartedAt:      time.Now(),
		FieldMessageID: fieldMessageID,
		InfoMessageID:  infoMessageID,
	}

	h.gatheringSessions[userID] = session
}

func (h *BotHandlers) generateRandomGatheringField() [][]string {
	// Создаем пустое поле 3x3
	field := make([][]string, 3)
	for i := range field {
		field[i] = make([]string, 3)
	}

	// Доступные ресурсы для сбора
	availableResources := []string{"🍇"}

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

func (h *BotHandlers) showGatheringField(chatID int64, field [][]string, gathering *models.Gathering) (int, int) {
	// Создаем инлайн клавиатуру на основе переданного поля
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < 3; i++ {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3; j++ {
			cell := field[i][j]
			var callbackData string

			switch cell {
			case "🍇":
				callbackData = fmt.Sprintf("gathering_berry_%d_%d", i, j)
			default:
				callbackData = fmt.Sprintf("gathering_empty_%d_%d", i, j)
				cell = " "
			}

			button := tgbotapi.NewInlineKeyboardButtonData(cell, callbackData)
			row = append(row, button)
		}
		keyboard = append(keyboard, row)
	}

	// Сначала отправляем поле сбора с инлайн кнопками
	fieldMsg := tgbotapi.NewMessage(chatID, "Выберите ресурс для сбора:")
	fieldMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	fieldResponse, _ := h.sendChattableWithResponse(fieldMsg)

	// Затем отправляем информационное сообщение с клавиатурой
	// Вычисляем опыт до следующего уровня
	expToNext := (gathering.Level * 100) - gathering.Experience

	infoText := fmt.Sprintf(`🌿 Сбор (Уровень %d)
До следующего уровня: %d опыта

Доступные ресурсы:
🍇 Ягоды`, gathering.Level, expToNext)

	gatheringKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	gatheringKeyboard.ResizeKeyboard = true

	infoMsg := tgbotapi.NewMessage(chatID, infoText)
	infoMsg.ReplyMarkup = gatheringKeyboard
	infoResponse, _ := h.sendChattableWithResponse(infoMsg)

	// Возвращаем ID поля и ID информационного сообщения
	return fieldResponse.MessageID, infoResponse.MessageID
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
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала зарегистрируйтесь с помощью команды /start")
		h.sendMessage(msg)
		return
	}

	// Проверяем количество березы в инвентаре
	birchQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "Береза")
	if err != nil {
		log.Printf("Error getting birch quantity: %v", err)
		birchQuantity = 0
	}

	// Показываем рецепт с кнопкой
	recipeText := fmt.Sprintf(`Для изготовления предмета "Березовый брус" необходимо следующее:
Береза - %d/%d шт.`, birchQuantity, 2)

	// Проверяем, можно ли создать хотя бы один предмет
	canCraft := birchQuantity >= 2
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
			tgbotapi.NewInlineKeyboardButtonData(buttonText, "craft_Березовый брус"),
		),
	)
	msg.ReplyMarkup = keyboard
	h.sendMessage(msg)
}

func (h *BotHandlers) handleCreateSimpleHut(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала зарегистрируйтесь с помощью команды /start")
		h.sendMessage(msg)
		return
	}

	// Определяем требования для строительства простой хижины
	requirements := []struct {
		ItemName string
		Quantity int
	}{
		{"Береза", 20},
		{"Березовый брус", 10},
		{"Камень", 15},
		{"Лесная ягода", 10},
	}

	// Формируем текст рецепта
	recipeText := "Для строительства необходимо следующее:"
	canBuild := true

	for _, req := range requirements {
		playerQuantity, err := h.db.GetItemQuantityInInventory(player.ID, req.ItemName)
		if err != nil {
			log.Printf("Error getting inventory quantity: %v", err)
			playerQuantity = 0
		}

		if playerQuantity < req.Quantity {
			canBuild = false
		}

		recipeText += fmt.Sprintf("\n%s - %d/%d шт.", req.ItemName, playerQuantity, req.Quantity)
	}

	// Добавляем кнопку "Создать"
	var buttonText string
	if canBuild {
		buttonText = "Создать ✅"
	} else {
		buttonText = "Создать ❌"
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, recipeText)

	// Создаем инлайн клавиатуру с кнопкой создать
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buttonText, "craft_Простая хижина"),
		),
	)
	msg.ReplyMarkup = keyboard
	h.sendMessage(msg)
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
	} else if strings.HasPrefix(data, "gathering_berry_") {
		// Обрабатываем callback'и от сбора
		parts := strings.Split(data, "_")
		if len(parts) == 4 {
			row, col := parts[2], parts[3]
			h.startGatheringAtPosition(userID, callback.Message.Chat.ID, "Лесная ягода", 10, callback.ID, row, col)
		}
	} else if strings.HasPrefix(data, "gathering_empty_") {
		// Пустая ячейка
		callbackConfig := tgbotapi.NewCallback(callback.ID, "Здесь нет ягод!")
		h.requestAPI(callbackConfig)
	} else if strings.HasPrefix(data, "craft_") {
		// Обрабатываем крафт
		itemName := strings.TrimPrefix(data, "craft_")
		h.handleCraftCallback(userID, callback.Message.Chat.ID, itemName, callback.ID)
	} else if strings.HasPrefix(data, "quest_accept_") {
		// Принятие квеста
		questIDStr := strings.TrimPrefix(data, "quest_accept_")
		questID, _ := strconv.Atoi(questIDStr)
		h.handleQuestAccept(userID, callback.Message.Chat.ID, questID, callback.ID, callback.Message.MessageID)
	} else if strings.HasPrefix(data, "quest_decline_") {
		// Отказ от квеста
		h.handleQuestDecline(callback.Message.Chat.ID, callback.ID, callback.Message.MessageID)
	} else {
		// Остальные callback
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "🔨 Функция пока в разработке...")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		h.requestAPI(callbackConfig)
	}
}

func (h *BotHandlers) handleQuestAccept(userID int64, chatID int64, questID int, callbackID string, messageID int) {
	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		callbackConfig := tgbotapi.NewCallback(callbackID, "Ошибка получения данных игрока")
		h.requestAPI(callbackConfig)
		return
	}

	// Активируем квест
	err = h.db.UpdateQuestStatus(player.ID, questID, "active")
	if err != nil {
		log.Printf("Error updating quest status: %v", err)
		callbackConfig := tgbotapi.NewCallback(callbackID, "Ошибка активации квеста")
		h.requestAPI(callbackConfig)
		return
	}

	// Отвечаем на callback
	callbackConfig := tgbotapi.NewCallback(callbackID, "Квест принят!")
	h.requestAPI(callbackConfig)

	// Удаляем сообщение с предложением квеста
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)
}

func (h *BotHandlers) handleQuestDecline(chatID int64, callbackID string, messageID int) {
	// Отвечаем на callback
	callbackConfig := tgbotapi.NewCallback(callbackID, "Квест отклонен")
	h.requestAPI(callbackConfig)

	// Удаляем сообщение с предложением квеста
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)
}

func (h *BotHandlers) handleCraftCallback(userID int64, chatID int64, itemName string, callbackID string) {
	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		callbackConfig := tgbotapi.NewCallback(callbackID, "Ошибка получения данных игрока")
		h.requestAPI(callbackConfig)
		return
	}

	// Обрабатываем крафт березового бруса
	if itemName == "Березовый брус" {
		// Проверяем количество березы
		birchQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "Береза")
		if err != nil {
			log.Printf("Error getting birch quantity: %v", err)
			birchQuantity = 0
		}

		if birchQuantity < 2 {
			callbackConfig := tgbotapi.NewCallback(callbackID, `Недостаточно предмета "Береза"`)
			h.requestAPI(callbackConfig)
			return
		}

		// Отвечаем на callback
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)

		// Спрашиваем количество
		msg := tgbotapi.NewMessage(chatID, "Введи сколько предметов хочешь создать:")
		h.sendMessage(msg)

		// Отмечаем, что ждем количество для крафта
		h.waitingForCraftQuantity[userID] = itemName
	} else {
		// Для других предметов пока заглушка
		callbackConfig := tgbotapi.NewCallback(callbackID, "Функция пока в разработке")
		h.requestAPI(callbackConfig)
	}
}

func (h *BotHandlers) startMiningAtPosition(userID int64, chatID int64, resourceName string, duration int, callbackID string, rowStr, colStr string) {
	row, _ := strconv.Atoi(rowStr)
	col, _ := strconv.Atoi(colStr)

	h.startMining(userID, chatID, resourceName, duration, callbackID, row, col)
}

func (h *BotHandlers) startMining(userID int64, chatID int64, resourceName string, duration int, callbackID string, row, col int) {
	// Проверяем, идет ли уже добыча в шахте, рубка в лесу или крафт
	if _, exists := h.miningTimers[userID]; exists {
		msg := tgbotapi.NewMessage(chatID, "Нельзя начинать новую добычу, пока не закончена текущая.")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}
	if _, exists := h.choppingTimers[userID]; exists {
		msg := tgbotapi.NewMessage(chatID, "Нельзя начинать новую добычу, пока не закончена текущая.")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}
	if _, exists := h.craftingTimers[userID]; exists {
		msg := tgbotapi.NewMessage(chatID, "Нельзя совершать действия пока идет создание предметов.")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}

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

	// Проверяем квест на добычу камня
	if resourceName == "Камень" {
		h.checkStoneQuestProgress(userID, chatID, player.ID)
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
		(mine.Level*100)-mine.Experience)

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
	expToNext := (mine.Level * 100) - mine.Experience

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
			tgbotapi.NewKeyboardButton("📜 Квесты"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏘️ Постройки"),
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
	// Проверяем, идет ли уже рубка в лесу, добыча в шахте или крафт
	if _, exists := h.choppingTimers[userID]; exists {
		msg := tgbotapi.NewMessage(chatID, "Нельзя начинать новую добычу, пока не закончена текущая.")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}
	if _, exists := h.miningTimers[userID]; exists {
		msg := tgbotapi.NewMessage(chatID, "Нельзя начинать новую добычу, пока не закончена текущая.")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}
	if _, exists := h.craftingTimers[userID]; exists {
		msg := tgbotapi.NewMessage(chatID, "Нельзя совершать действия пока идет создание предметов.")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}

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
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime).Seconds()
			progress := int(elapsed)

			if progress >= totalDuration {
				// Рубка завершена
				h.completeChopping(userID, chatID, resourceName, durability, messageID, row, col)
				return
			}

			// Обновляем прогресс бар
			percentage := int((elapsed / float64(totalDuration)) * 100)
			progressBar := h.createProgressBar(progress, totalDuration)

			newText := fmt.Sprintf(`Началась рубка дерева "%s". Время рубки %d сек.
			
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

	// Проверяем квест на рубку березы
	if resourceName == "Береза" {
		h.checkBirchQuestProgress(userID, chatID, player.ID)
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
		(forest.Level*100)-forest.Experience)

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
	expToNext := (forest.Level * 100) - forest.Experience

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

func (h *BotHandlers) handleQuest(message *tgbotapi.Message) {
	userID := message.From.ID

	// Устанавливаем местоположение игрока
	h.playerLocation[userID] = "quest"

	questText := `📜 Квесты

Выберите тип квестов:`

	msg := tgbotapi.NewMessage(message.Chat.ID, questText)
	h.sendQuestKeyboard(msg)
}

func (h *BotHandlers) sendQuestKeyboard(msg tgbotapi.MessageConfig) {
	// Создаем клавиатуру квестов
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📖 Лор"),
			tgbotapi.NewKeyboardButton("🗓️ Ежедневные"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📆 Еженедельные"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	keyboard.ResizeKeyboard = true
	msg.ReplyMarkup = keyboard
	h.sendMessage(msg)
}

func (h *BotHandlers) handleLore(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Сначала зарегистрируйтесь с помощью команды /start")
		h.sendMessage(msg)
		return
	}

	// Проверяем квест 1
	quest, err := h.db.GetPlayerQuest(player.ID, 1)
	if err != nil {
		log.Printf("Error getting quest: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении квеста.")
		h.sendMessage(msg)
		return
	}

	if quest == nil || quest.Status == "available" {
		// Квест еще не создан или доступен для принятия
		if quest == nil {
			// Создаем квест, если его нет
			err := h.db.CreateQuest(player.ID, 1, 5) // Квест 1: нарубить 5 березы
			if err != nil {
				log.Printf("Error creating quest: %v", err)
				msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при создании квеста.")
				h.sendMessage(msg)
				return
			}
		}

		// Показываем предложение квеста
		questText := `🪓 Квест 1: Дерево под топор
Задание: Наруби 5 берёзы
Награда: 🎖 10 опыта + 📖 Страница 1 «Забытая тишина»`

		msg := tgbotapi.NewMessage(message.Chat.ID, questText)

		// Создаем инлайн кнопки
		acceptBtn := tgbotapi.NewInlineKeyboardButtonData("Принять", "quest_accept_1")
		declineBtn := tgbotapi.NewInlineKeyboardButtonData("Отказ", "quest_decline_1")
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(acceptBtn, declineBtn),
		)
		msg.ReplyMarkup = keyboard
		h.sendMessage(msg)
		return
	}

	if quest.Status == "active" {
		// Квест активен, показываем прогресс
		activeText := fmt.Sprintf(`Активный квест: 🪓 Квест 1: Дерево под топор
Задание: Наруби 5 берёзы (%d/5)
Награда: 🎖 10 опыта + 📖 Страница 1 «Забытая тишина»`, quest.Progress)

		msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
		h.sendMessage(msg)
		return
	}

	if quest.Status == "completed" {
		// Квест 1 выполнен, проверяем квест 2
		quest2, err := h.db.GetPlayerQuest(player.ID, 2)
		if err != nil {
			log.Printf("Error getting quest 2: %v", err)
			msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении квеста.")
			h.sendMessage(msg)
			return
		}

		if quest2 == nil || quest2.Status == "available" {
			// Квест 2 еще не создан или доступен для принятия
			if quest2 == nil {
				// Создаем квест 2, если его нет
				err := h.db.CreateQuest(player.ID, 2, 3) // Квест 2: добыть 3 камня
				if err != nil {
					log.Printf("Error creating quest 2: %v", err)
					msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при создании квеста.")
					h.sendMessage(msg)
					return
				}
			}

			// Показываем предложение квеста 2
			questText := `⛏ Квест 2: Вглубь
Задание: Добудь 3 камня
Награда: 🎖 10 опыта + 📖 Страница 2 «Пыль веков»`

			msg := tgbotapi.NewMessage(message.Chat.ID, questText)

			// Создаем инлайн кнопки
			acceptBtn := tgbotapi.NewInlineKeyboardButtonData("Принять", "quest_accept_2")
			declineBtn := tgbotapi.NewInlineKeyboardButtonData("Отказ", "quest_decline_2")
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(acceptBtn, declineBtn),
			)
			msg.ReplyMarkup = keyboard
			h.sendMessage(msg)
			return
		}

		if quest2.Status == "active" {
			// Квест 2 активен, показываем прогресс
			activeText := fmt.Sprintf(`Активный квест: ⛏ Квест 2: Вглубь
Задание: Добудь 3 камня (%d/3)
Награда: 🎖 10 опыта + 📖 Страница 2 «Пыль веков»`, quest2.Progress)

			msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
			h.sendMessage(msg)
			return
		}

		if quest2.Status == "completed" {
			// Квест 2 выполнен, проверяем квест 3
			quest3, err := h.db.GetPlayerQuest(player.ID, 3)
			if err != nil {
				log.Printf("Error getting quest 3: %v", err)
				msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении квеста.")
				h.sendMessage(msg)
				return
			}

			if quest3 == nil || quest3.Status == "available" {
				// Квест 3 еще не создан или доступен для принятия
				if quest3 == nil {
					// Создаем квест 3, если его нет
					err := h.db.CreateQuest(player.ID, 3, 3) // Квест 3: создать 3 березовых бруса
					if err != nil {
						log.Printf("Error creating quest 3: %v", err)
						msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при создании квеста.")
						h.sendMessage(msg)
						return
					}
				}

				// Показываем предложение квеста 3
				questText := `🪚 Квест 3: Руки мастера
Задание: Создай 3 берёзовых бруса
Награда: 🎖 10 опыта + 📖 Страница 3 «Первый шаг»`

				msg := tgbotapi.NewMessage(message.Chat.ID, questText)

				// Создаем инлайн кнопки
				acceptBtn := tgbotapi.NewInlineKeyboardButtonData("Принять", "quest_accept_3")
				declineBtn := tgbotapi.NewInlineKeyboardButtonData("Отказ", "quest_decline_3")
				keyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(acceptBtn, declineBtn),
				)
				msg.ReplyMarkup = keyboard
				h.sendMessage(msg)
				return
			}

			if quest3.Status == "active" {
				// Квест 3 активен, показываем прогресс
				activeText := fmt.Sprintf(`Активный квест: 🪚 Квест 3: Руки мастера
Задание: Создай 3 берёзовых бруса (%d/3)
Награда: 🎖 10 опыта + 📖 Страница 3 «Первый шаг»`, quest3.Progress)

				msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
				h.sendMessage(msg)
				return
			}

			if quest3.Status == "completed" {
				// Все квесты выполнены, показываем заглушку
				msg := tgbotapi.NewMessage(message.Chat.ID, "Цепочка квестов в разработке.")
				h.sendMessage(msg)
				return
			}
		}
	}
}

func (h *BotHandlers) handleDailyQuests(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "🗓️ Функция ежедневных квестов пока в разработке...")
	h.sendMessage(msg)
}

func (h *BotHandlers) handleWeeklyQuests(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "📆 Функция еженедельных квестов пока в разработке...")
	h.sendMessage(msg)
}

func (h *BotHandlers) handleBuildings(message *tgbotapi.Message) {
	buildingsText := `🏘️ Доступные постройки:
Простая хижина /create_simple_hut`

	msg := tgbotapi.NewMessage(message.Chat.ID, buildingsText)
	h.sendMessage(msg)
}

func (h *BotHandlers) handleLookPages(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Получаем инвентарь игрока
	inventory, err := h.db.GetPlayerInventory(player.ID)
	if err != nil {
		log.Printf("Error getting inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Фильтруем только страницы
	var pages []string
	for _, item := range inventory {
		if strings.Contains(item.ItemName, "📖 Страница 1") {
			pages = append(pages, fmt.Sprintf("%s - %d шт. /read1",
				item.ItemName,
				item.Quantity))
		} else if strings.Contains(item.ItemName, "📖 Страница 2") {
			pages = append(pages, fmt.Sprintf("%s - %d шт. /read2",
				item.ItemName,
				item.Quantity))
		} else if strings.Contains(item.ItemName, "📖 Страница 3") {
			pages = append(pages, fmt.Sprintf("%s - %d шт. /read3",
				item.ItemName,
				item.Quantity))
		}
	}

	if len(pages) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "📖 У вас пока нет страниц.")
		h.sendMessage(msg)
		return
	}

	// Формируем сообщение со страницами
	text := "📖 Ваши страницы:\n\n" + strings.Join(pages, "\n")
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	h.sendMessage(msg)
}

func (h *BotHandlers) handleReadPage(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Получаем инвентарь игрока
	inventory, err := h.db.GetPlayerInventory(player.ID)
	if err != nil {
		log.Printf("Error getting inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Находим все страницы в инвентаре
	var availablePages []string
	pageTexts := make(map[string]string)

	// Определяем доступные страницы и их тексты
	for _, item := range inventory {
		if strings.Contains(item.ItemName, "📖 Страница 1") && item.Quantity > 0 {
			availablePages = append(availablePages, "📖 Страница 1 «Забытая тишина»")
			pageTexts["📖 Страница 1 «Забытая тишина»"] = `📖 Страница 1 «Забытая тишина»

"Мир не был уничтожен в битве. Он просто... забыл сам себя.
Годы прошли — может, столетия, может, тысячелетия. Никто не знает точно. От былых королевств остались лишь заросшие руины, поросшие мхом камни и полустёртые знаки, выгравированные на обломках."`
		}
		if strings.Contains(item.ItemName, "📖 Страница 2") && item.Quantity > 0 {
			availablePages = append(availablePages, "📖 Страница 2 «Пыль веков»")
			pageTexts["📖 Страница 2 «Пыль веков»"] = `📖 Страница 2 «Пыль веков»

"Люди исчезли. Не все, возможно, но память о них — точно.
Земля забыла их шаги. Знания рассыпались, будто песок в ветре. Остались лишь сны, смутные образы, и тихий зов из глубин мира."`
		}
		if strings.Contains(item.ItemName, "📖 Страница 3") && item.Quantity > 0 {
			availablePages = append(availablePages, "📖 Страница 3 «Первый шаг»")
			pageTexts["📖 Страница 3 «Первый шаг»"] = `📖 Страница 3 «Первый шаг»

"Ты — один из тех, кто откликнулся.
Никто не сказал тебе, зачем ты проснулся. В этом нет наставников, богов или проводников. Только ты, дикая земля — и чувство, что всё это уже было. Что ты здесь не впервые."`
		}
	}

	if len(availablePages) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет страниц для чтения.")
		h.sendMessage(msg)
		return
	}

	// Если есть только одна страница, читаем её
	if len(availablePages) == 1 {
		pageText := pageTexts[availablePages[0]]
		msg := tgbotapi.NewMessage(message.Chat.ID, pageText)
		h.sendMessage(msg)
		return
	}

	// Если страниц несколько, показываем все
	var allTexts []string
	for _, pageName := range availablePages {
		allTexts = append(allTexts, pageTexts[pageName])
	}

	fullText := strings.Join(allTexts, "\n\n---\n\n")
	msg := tgbotapi.NewMessage(message.Chat.ID, fullText)
	h.sendMessage(msg)
}

func (h *BotHandlers) handleReadPage1(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Проверяем наличие страницы 1 в инвентаре
	pageQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "📖 Страница 1 «Забытая тишина»")
	if err != nil {
		log.Printf("Error checking page in inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	if pageQuantity == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет первой страницы.")
		h.sendMessage(msg)
		return
	}

	// Показываем текст первой страницы
	pageText := `📖 Страница 1 «Забытая тишина»

"Мир не был уничтожен в битве. Он просто... забыл сам себя.
Годы прошли — может, столетия, может, тысячелетия. Никто не знает точно. От былых королевств остались лишь заросшие руины, поросшие мхом камни и полустёртые знаки, выгравированные на обломках."`

	msg := tgbotapi.NewMessage(message.Chat.ID, pageText)
	h.sendMessage(msg)
}

func (h *BotHandlers) handleReadPage2(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Проверяем наличие страницы 2 в инвентаре
	pageQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "📖 Страница 2 «Пыль веков»")
	if err != nil {
		log.Printf("Error checking page in inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	if pageQuantity == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет второй страницы.")
		h.sendMessage(msg)
		return
	}

	// Показываем текст второй страницы
	pageText := `📖 Страница 2 «Пыль веков»

"Люди исчезли. Не все, возможно, но память о них — точно.
Земля забыла их шаги. Знания рассыпались, будто песок в ветре. Остались лишь сны, смутные образы, и тихий зов из глубин мира."`

	msg := tgbotapi.NewMessage(message.Chat.ID, pageText)
	h.sendMessage(msg)
}

func (h *BotHandlers) handleReadPage3(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Проверяем наличие страницы 3 в инвентаре
	pageQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "📖 Страница 3 «Первый шаг»")
	if err != nil {
		log.Printf("Error checking page in inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	if pageQuantity == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет третьей страницы.")
		h.sendMessage(msg)
		return
	}

	// Показываем текст третьей страницы
	pageText := `📖 Страница 3 «Первый шаг»

"Ты — один из тех, кто откликнулся.
Никто не сказал тебе, зачем ты проснулся. В этом нет наставников, богов или проводников. Только ты, дикая земля — и чувство, что всё это уже было. Что ты здесь не впервые."`

	msg := tgbotapi.NewMessage(message.Chat.ID, pageText)
	h.sendMessage(msg)
}

// Функции для сбора в лесу
func (h *BotHandlers) startGatheringAtPosition(userID int64, chatID int64, resourceName string, duration int, callbackID string, rowStr, colStr string) {
	row, _ := strconv.Atoi(rowStr)
	col, _ := strconv.Atoi(colStr)

	h.startGathering(userID, chatID, resourceName, duration, callbackID, row, col)
}

func (h *BotHandlers) startGathering(userID int64, chatID int64, resourceName string, duration int, callbackID string, row, col int) {
	// Проверяем, идет ли уже сбор, добыча/рубка или крафт
	if _, exists := h.gatheringTimers[userID]; exists {
		msg := tgbotapi.NewMessage(chatID, "Нельзя начинать новую добычу, пока не закончена текущая.")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}
	if _, exists := h.miningTimers[userID]; exists {
		msg := tgbotapi.NewMessage(chatID, "Нельзя начинать новую добычу, пока не закончена текущая.")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}
	if _, exists := h.choppingTimers[userID]; exists {
		msg := tgbotapi.NewMessage(chatID, "Нельзя начинать новую добычу, пока не закончена текущая.")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}
	if _, exists := h.craftingTimers[userID]; exists {
		msg := tgbotapi.NewMessage(chatID, "Нельзя совершать действия пока идет создание предметов.")
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		return
	}

	// Проверяем наличие ножа
	hasTool, durability, err := h.db.HasToolInInventory(player.ID, "Простой нож")
	if err != nil {
		log.Printf("Error checking tool: %v", err)
		return
	}

	if !hasTool {
		msg := tgbotapi.NewMessage(chatID, `В инвентаре нет предмета "Простой нож".`)
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}

	// Отвечаем на callback
	callbackConfig := tgbotapi.NewCallback(callbackID, "")
	h.requestAPI(callbackConfig)

	// Удаляем предыдущее сообщение о результате сбора, если оно существует
	if session, exists := h.gatheringSessions[userID]; exists && session.ResultMessageID != 0 {
		deleteResultMsg := tgbotapi.NewDeleteMessage(chatID, session.ResultMessageID)
		h.requestAPI(deleteResultMsg)
		session.ResultMessageID = 0 // Сбрасываем ID
	}

	// Отправляем сообщение о начале сбора
	initialText := fmt.Sprintf(`Идет сбор "%s". Время сбора %d сек.
		
%s 0%%`, resourceName, duration, h.createProgressBar(0, 100))

	gatheringMsg := tgbotapi.NewMessage(chatID, initialText)
	sentMsg, _ := h.sendMessageWithResponse(gatheringMsg)

	// Запускаем горутину для обновления прогресс бара
	go h.updateGatheringProgress(userID, chatID, sentMsg.MessageID, resourceName, duration, durability, row, col)

	// Создаем заглушку таймера
	timer := time.NewTimer(time.Duration(duration) * time.Second)
	h.gatheringTimers[userID] = timer
}

func (h *BotHandlers) updateGatheringProgress(userID int64, chatID int64, messageID int, resourceName string, totalDuration int, durability int, row, col int) {
	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime).Seconds()
			progress := int(elapsed)

			if progress >= totalDuration {
				// Сбор завершен
				h.completeGathering(userID, chatID, resourceName, durability, messageID, row, col)
				return
			}

			// Обновляем прогресс бар
			percentage := int((elapsed / float64(totalDuration)) * 100)
			progressBar := h.createProgressBar(progress, totalDuration)

			newText := fmt.Sprintf(`Начался сбор "%s". Время сбора %d сек.
			
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

func (h *BotHandlers) completeGathering(userID int64, chatID int64, resourceName string, oldDurability int, messageID int, row, col int) {
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

	// Обновляем прочность ножа и сытость
	if err := h.db.UpdateItemDurability(player.ID, "Простой нож", 1); err != nil {
		log.Printf("Error updating item durability: %v", err)
	}
	if err := h.db.UpdatePlayerSatiety(player.ID, -1); err != nil {
		log.Printf("Error updating player satiety: %v", err)
	}

	// Добавляем опыт за сбор
	levelUp, newLevel, err := h.db.UpdateGatheringExperience(player.ID, 2)
	if err != nil {
		log.Printf("Error updating gathering experience: %v", err)
	}

	// Получаем обновленные данные
	updatedPlayer, _ := h.db.GetPlayer(userID)
	updatedGathering, _ := h.db.GetOrCreateGathering(player.ID)

	// Удаляем сообщение о сборе
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)

	// Показываем результат с системой уровней для сбора
	resultText := fmt.Sprintf(`✅ Ты собрал "%s"!
Получено опыта: 2
Сытость: %d/100
Прочность ножа: %d/100
До следующего уровня: %d опыта`,
		resourceName,
		updatedPlayer.Satiety,
		oldDurability-1,
		(updatedGathering.Level*100)-updatedGathering.Experience)

	msg := tgbotapi.NewMessage(chatID, resultText)
	resultResponse, _ := h.sendMessageWithResponse(msg)

	// Если уровень повысился, показываем сообщение о повышении
	if levelUp {
		levelUpText := fmt.Sprintf("🎉 Поздравляем! Уровень сбора повышен до %d уровня!", newLevel)
		levelUpMsg := tgbotapi.NewMessage(chatID, levelUpText)
		h.sendMessage(levelUpMsg)
	}

	// Сохраняем ID сообщения с результатом в сессии
	if session, exists := h.gatheringSessions[userID]; exists {
		session.ResultMessageID = resultResponse.MessageID
	}

	// Убираем таймер
	delete(h.gatheringTimers, userID)

	// Обновляем поле - убираем собранный ресурс
	if session, exists := h.gatheringSessions[userID]; exists {
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
			h.updateGatheringField(chatID, session.Resources, session.FieldMessageID)
			// Обновляем информационное сообщение с актуальными данными
			h.updateGatheringInfoMessage(userID, chatID, updatedGathering, session.InfoMessageID)
		} else {
			// Поле истощено, устанавливаем кулдаун
			if err := h.db.ExhaustGathering(userID); err != nil {
				log.Printf("Error exhausting gathering: %v", err)
			}

			// Устанавливаем таймер кулдауна на 60 секунд
			h.gatheringCooldowns[userID] = time.Now().Add(60 * time.Second)

			// Удаляем сообщение с полем сбора
			deleteFieldMsg := tgbotapi.NewDeleteMessage(chatID, session.FieldMessageID)
			h.requestAPI(deleteFieldMsg)

			// Удаляем сообщение с информацией о сборе
			deleteInfoMsg := tgbotapi.NewDeleteMessage(chatID, session.InfoMessageID)
			h.requestAPI(deleteInfoMsg)

			exhaustMsg := tgbotapi.NewMessage(chatID, `⚠️ Ягодные кусты истощены! Необходимо подождать 1 минуту до восстановления.
Нажми кнопку "🌿 Сбор" чтобы проверить готовность.`)
			h.sendForestKeyboard(exhaustMsg)

			// Удаляем сессию
			delete(h.gatheringSessions, userID)
		}
	}
}

func (h *BotHandlers) updateGatheringField(chatID int64, field [][]string, messageID int) {
	text := "Выберите ресурс для сбора:"

	// Создаем инлайн клавиатуру на основе переданного поля
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < 3; i++ {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3; j++ {
			cell := field[i][j]
			var callbackData string

			switch cell {
			case "🍇":
				callbackData = fmt.Sprintf("gathering_berry_%d_%d", i, j)
			default:
				callbackData = fmt.Sprintf("gathering_empty_%d_%d", i, j)
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

func (h *BotHandlers) updateGatheringInfoMessage(userID int64, chatID int64, gathering *models.Gathering, messageID int) {
	// Вычисляем опыт до следующего уровня
	expToNext := (gathering.Level * 100) - gathering.Experience

	infoText := fmt.Sprintf(`🌿 Сбор (Уровень %d)
До следующего уровня: %d опыта

Доступные ресурсы:
🍇 Ягоды`, gathering.Level, expToNext)

	// Удаляем старое сообщение
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)

	// Создаем новое сообщение с клавиатурой
	gatheringKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	gatheringKeyboard.ResizeKeyboard = true

	newMsg := tgbotapi.NewMessage(chatID, infoText)
	newMsg.ReplyMarkup = gatheringKeyboard
	newResponse, _ := h.sendMessageWithResponse(newMsg)

	// Обновляем ID сообщения в сессии
	if session, exists := h.gatheringSessions[userID]; exists {
		session.InfoMessageID = newResponse.MessageID
	}
}

func (h *BotHandlers) startCrafting(userID int64, chatID int64, itemName string, quantity int) {
	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		return
	}

	// Для березового бруса: потребляем березу (2 березы за 1 брус)
	if itemName == "Березовый брус" {
		requiredBirch := quantity * 2
		err := h.db.ConsumeItem(player.ID, "Береза", requiredBirch)
		if err != nil {
			log.Printf("Error consuming birch: %v", err)
			msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при потреблении ресурсов.")
			h.sendMessage(msg)
			return
		}
	}

	// Вычисляем общее время крафта (20 секунд за один предмет)
	totalDuration := quantity * 20

	// Отправляем сообщение о начале крафта
	craftText := fmt.Sprintf(`Идет создание предмета "%s". Время создания %d сек.

⏳ 0%%`, itemName, totalDuration)

	msg := tgbotapi.NewMessage(chatID, craftText)
	response, err := h.sendMessageWithResponse(msg)
	if err != nil {
		log.Printf("Error sending craft message: %v", err)
		return
	}

	// Запускаем таймер крафта
	h.craftingTimers[userID] = time.NewTimer(time.Duration(totalDuration) * time.Second)

	// Запускаем горутину для обновления прогресса
	go h.updateCraftingProgress(userID, chatID, response.MessageID, itemName, quantity, totalDuration)
}

func (h *BotHandlers) updateCraftingProgress(userID int64, chatID int64, messageID int, itemName string, quantity int, totalDuration int) {
	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime).Seconds()
			progress := int(elapsed)

			if progress >= totalDuration {
				// Крафт завершен
				h.completeCrafting(userID, chatID, itemName, quantity, messageID)
				return
			}

			// Обновляем прогресс бар
			percentage := int((elapsed / float64(totalDuration)) * 100)
			progressBar := h.createProgressBar(progress, totalDuration)

			newText := fmt.Sprintf(`Идет создание предмета "%s". Время создания %d сек.

%s %d%%`, itemName, totalDuration, progressBar, percentage)

			// Редактируем сообщение
			editMsg := tgbotapi.NewEditMessageText(chatID, messageID, newText)
			h.editMessage(editMsg)

		case <-time.After(time.Duration(totalDuration+1) * time.Second):
			// Таймаут на случай, если что-то пошло не так
			return
		}
	}
}

func (h *BotHandlers) completeCrafting(userID int64, chatID int64, itemName string, quantity int, messageID int) {
	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		return
	}

	// Добавляем созданные предметы в инвентарь
	if err := h.db.AddItemToInventory(player.ID, itemName, quantity); err != nil {
		log.Printf("Error adding crafted items to inventory: %v", err)
	}

	// Отнимаем сытость (1 единица за каждый созданный предмет)
	if err := h.db.UpdatePlayerSatiety(player.ID, -quantity); err != nil {
		log.Printf("Error updating player satiety: %v", err)
	}

	// Получаем обновленные данные игрока для отображения сытости
	updatedPlayer, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting updated player: %v", err)
		// Если не удалось получить обновленные данные, используем старые
		updatedPlayer = player
	}

	// Удаляем сообщение о крафте
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)

	// Показываем результат с сытостью
	resultText := fmt.Sprintf(`✅ Создание завершено!
Получено: "%s" x%d
Сытость: %d/100`, itemName, quantity, updatedPlayer.Satiety)

	msg := tgbotapi.NewMessage(chatID, resultText)
	h.sendMessage(msg)

	// Проверяем прогресс квеста 3 (создание березовых брусов)
	if itemName == "Березовый брус" {
		h.checkBirchPlankQuestProgress(userID, chatID, player.ID, quantity)
	}

	// Убираем таймер
	delete(h.craftingTimers, userID)
}

func (h *BotHandlers) checkBirchQuestProgress(userID int64, chatID int64, playerID int) {
	// Проверяем активный квест 1 (рубка березы)
	quest, err := h.db.GetPlayerQuest(playerID, 1)
	if err != nil {
		log.Printf("Error getting quest: %v", err)
		return
	}

	// Если квест не активен, ничего не делаем
	if quest == nil || quest.Status != "active" {
		return
	}

	// Увеличиваем прогресс квеста
	newProgress := quest.Progress + 1
	err = h.db.UpdateQuestProgress(playerID, 1, newProgress)
	if err != nil {
		log.Printf("Error updating quest progress: %v", err)
		return
	}

	// Проверяем, выполнен ли квест
	if newProgress >= quest.Target {
		// Квест выполнен!
		err = h.db.UpdateQuestStatus(playerID, 1, "completed")
		if err != nil {
			log.Printf("Error completing quest: %v", err)
			return
		}

		// Добавляем награды
		// 10 опыта игроку
		err = h.db.UpdatePlayerExperience(playerID, 10)
		if err != nil {
			log.Printf("Error updating player experience: %v", err)
		}

		// Добавляем страницу в инвентарь
		err = h.db.AddItemToInventory(playerID, "📖 Страница 1 «Забытая тишина»", 1)
		if err != nil {
			log.Printf("Error adding quest item to inventory: %v", err)
		}

		// Отправляем сообщение о выполнении квеста
		questCompleteText := `🪓 Квест 1: Дерево под топор ВЫПОЛНЕН!
Получена награда:
🎖 10 опыта
📖 Страница 1 «Забытая тишина»`

		msg := tgbotapi.NewMessage(chatID, questCompleteText)
		h.sendMessage(msg)
	}
}

func (h *BotHandlers) checkBirchPlankQuestProgress(userID int64, chatID int64, playerID int, quantity int) {
	// Проверяем активный квест 3 (создание березовых брусов)
	quest, err := h.db.GetPlayerQuest(playerID, 3)
	if err != nil {
		log.Printf("Error getting quest 3: %v", err)
		return
	}

	// Если квест не активен, ничего не делаем
	if quest == nil || quest.Status != "active" {
		return
	}

	// Увеличиваем прогресс квеста на количество созданных брусов
	newProgress := quest.Progress + quantity
	err = h.db.UpdateQuestProgress(playerID, 3, newProgress)
	if err != nil {
		log.Printf("Error updating quest 3 progress: %v", err)
		return
	}

	// Проверяем, выполнен ли квест
	if newProgress >= quest.Target {
		// Квест выполнен!
		err = h.db.UpdateQuestStatus(playerID, 3, "completed")
		if err != nil {
			log.Printf("Error completing quest 3: %v", err)
			return
		}

		// Добавляем награды
		// 10 опыта игроку
		err = h.db.UpdatePlayerExperience(playerID, 10)
		if err != nil {
			log.Printf("Error updating player experience: %v", err)
		}

		// Добавляем страницу в инвентарь
		err = h.db.AddItemToInventory(playerID, "📖 Страница 3 «Первый шаг»", 1)
		if err != nil {
			log.Printf("Error adding quest item to inventory: %v", err)
		}

		// Отправляем сообщение о выполнении квеста
		questCompleteText := `🪚 Квест 3: Руки мастера ВЫПОЛНЕН!
Получена награда:
🎖 10 опыта
📖 Страница 3 «Первый шаг»`

		msg := tgbotapi.NewMessage(chatID, questCompleteText)
		h.sendMessage(msg)
	}
}

func (h *BotHandlers) checkStoneQuestProgress(userID int64, chatID int64, playerID int) {
	// Проверяем активный квест 2 (добыча камня)
	quest, err := h.db.GetPlayerQuest(playerID, 2)
	if err != nil {
		log.Printf("Error getting quest 2: %v", err)
		return
	}

	// Если квест не активен, ничего не делаем
	if quest == nil || quest.Status != "active" {
		return
	}

	// Увеличиваем прогресс квеста
	newProgress := quest.Progress + 1
	err = h.db.UpdateQuestProgress(playerID, 2, newProgress)
	if err != nil {
		log.Printf("Error updating quest 2 progress: %v", err)
		return
	}

	// Проверяем, выполнен ли квест
	if newProgress >= quest.Target {
		// Квест выполнен!
		err = h.db.UpdateQuestStatus(playerID, 2, "completed")
		if err != nil {
			log.Printf("Error completing quest 2: %v", err)
			return
		}

		// Добавляем награды
		// 10 опыта игроку
		err = h.db.UpdatePlayerExperience(playerID, 10)
		if err != nil {
			log.Printf("Error updating player experience: %v", err)
		}

		// Добавляем страницу в инвентарь
		err = h.db.AddItemToInventory(playerID, "📖 Страница 2 «Пыль веков»", 1)
		if err != nil {
			log.Printf("Error adding quest item to inventory: %v", err)
		}

		// Отправляем сообщение о выполнении квеста
		questCompleteText := `⛏ Квест 2: Вглубь ВЫПОЛНЕН!
Получена награда:
🎖 10 опыта
📖 Страница 2 «Пыль веков»`

		msg := tgbotapi.NewMessage(chatID, questCompleteText)
		h.sendMessage(msg)
	}
}
