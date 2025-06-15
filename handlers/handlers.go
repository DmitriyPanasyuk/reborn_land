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
	huntingSessions         map[int64]*models.HuntingSession
	miningTimers            map[int64]*time.Timer
	choppingTimers          map[int64]*time.Timer
	gatheringTimers         map[int64]*time.Timer
	huntingTimers           map[int64]*time.Timer
	craftingTimers          map[int64]*time.Timer // Таймеры для крафта
	mineCooldowns           map[int64]time.Time   // Время окончания кулдауна шахты
	forestCooldowns         map[int64]time.Time   // Время окончания кулдауна леса
	gatheringCooldowns      map[int64]time.Time   // Время окончания кулдауна сбора
	huntingCooldowns        map[int64]time.Time   // Время окончания кулдауна охоты
	playerLocation          map[int64]string      // Текущее местоположение игрока
	restingTimers           map[int64]*time.Timer // Таймеры для отдыха
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
		huntingSessions:         make(map[int64]*models.HuntingSession),
		miningTimers:            make(map[int64]*time.Timer),
		choppingTimers:          make(map[int64]*time.Timer),
		gatheringTimers:         make(map[int64]*time.Timer),
		huntingTimers:           make(map[int64]*time.Timer),
		craftingTimers:          make(map[int64]*time.Timer),
		mineCooldowns:           make(map[int64]time.Time),
		forestCooldowns:         make(map[int64]time.Time),
		gatheringCooldowns:      make(map[int64]time.Time),
		huntingCooldowns:        make(map[int64]time.Time),
		playerLocation:          make(map[int64]string),
		restingTimers:           make(map[int64]*time.Timer),
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

	// Проверяем, не отдыхает ли игрок
	if _, exists := h.restingTimers[userID]; exists {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Нельзя совершить действие пока не завершен отдых.")
		h.sendMessage(msg)
		return
	}

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
	case "/read4":
		h.handleReadPage4(message)
	case "/read5":
		h.handleReadPage5(message)
	case "/read6":
		h.handleReadPage6(message)
	case "/read7":
		h.handleReadPage7(message)
	case "/read8":
		h.handleReadPage8(message)
	case "/open":
		h.handleOpenHut(message)
	case "/rest":
		h.handleRest(message)
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

	// Добавляем стартовые предметы новому игроку
	err = h.db.AddItemToInventoryWithDurability(player.ID, "Простой лук", 1, 100)
	if err != nil {
		log.Printf("Error adding bow to new player: %v", err)
	}

	err = h.db.AddItemToInventory(player.ID, "Стрелы", 100)
	if err != nil {
		log.Printf("Error adding arrows to new player: %v", err)
	}

	// Убираем флаг ожидания имени
	delete(h.waitingForName, userID)

	// Отправляем сообщение об успешной регистрации
	successText := fmt.Sprintf(`✅ Регистрация прошла успешно!

Добро пожаловать, %s! 👋

Твой уровень: %d
Опыт: %d/100
Сытость: %d/100

🎁 Стартовые предметы добавлены в инвентарь:
• Простой лук - 1 шт. (Прочность: 100/100)
• Простой нож - 1 шт. (Прочность: 100/100)
• Простой кирка - 1 шт. (Прочность: 100/100)
• Простой топор - 1 шт. (Прочность: 100/100)
• Стрелы - 100 шт.
• Лесная ягода - 10 шт.`, player.Name, player.Level, player.Experience, player.Satiety)

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
	// Получаем информацию об игроке
	player, err := h.db.GetPlayer(message.From.ID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		return
	}

	// Получаем инвентарь
	inventory, err := h.db.GetPlayerInventory(player.ID)
	if err != nil {
		log.Printf("Error getting inventory: %v", err)
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
		msg := tgbotapi.NewMessage(message.Chat.ID, "У тебя нет ягод для еды!")
		h.sendMessage(msg)
		return
	}

	// Уменьшаем количество ягод
	err = h.db.ConsumeItem(player.ID, "Лесная ягода", 1)
	if err != nil {
		log.Printf("Error consuming berries: %v", err)
		return
	}

	// Увеличиваем сытость
	err = h.db.UpdatePlayerSatiety(player.ID, 5)
	if err != nil {
		log.Printf("Error updating satiety: %v", err)
		return
	}

	// Получаем обновленное значение сытости
	updatedPlayer, err := h.db.GetPlayer(message.From.ID)
	if err != nil {
		log.Printf("Error getting updated player: %v", err)
		return
	}

	// Отправляем сообщение о съеденных ягодах
	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Ты съел ягоды! Сытость: %d/100", updatedPlayer.Satiety))
	h.sendMessage(msg)

	// Проверяем прогресс квеста 7
	h.checkBerryEatingQuestProgress(message.From.ID, message.Chat.ID, player.ID)
}

func (h *BotHandlers) checkBerryEatingQuestProgress(userID int64, chatID int64, playerID int) {
	// Проверяем активный квест 7 (съесть 3 ягоды)
	quest, err := h.db.GetPlayerQuest(playerID, 7)
	if err != nil {
		log.Printf("Error getting quest 7: %v", err)
		return
	}

	// Если квест не активен, ничего не делаем
	if quest == nil || quest.Status != "active" {
		return
	}

	// Увеличиваем прогресс квеста
	newProgress := quest.Progress + 1
	err = h.db.UpdateQuestProgress(playerID, 7, newProgress)
	if err != nil {
		log.Printf("Error updating quest 7 progress: %v", err)
		return
	}

	// Проверяем, выполнен ли квест
	if newProgress >= quest.Target {
		// Квест выполнен!
		err = h.db.UpdateQuestStatus(playerID, 7, "completed")
		if err != nil {
			log.Printf("Error completing quest 7: %v", err)
			return
		}

		// Добавляем награды
		// 10 опыта игроку
		err = h.db.UpdatePlayerExperience(playerID, 10)
		if err != nil {
			log.Printf("Error updating player experience: %v", err)
		}

		// Добавляем страницу в инвентарь
		err = h.db.AddItemToInventory(playerID, "📖 Страница 7 «Шёпот ветра»", 1)
		if err != nil {
			log.Printf("Error adding quest item to inventory: %v", err)
		}

		// Отправляем сообщение о выполнении квеста
		questCompleteText := `🍇 Квест 7: Перекус ВЫПОЛНЕН!
Получена награда:
🎖 10 опыта
📖 Страница 7 «Шёпот ветра»`

		msg := tgbotapi.NewMessage(chatID, questCompleteText)
		h.sendMessage(msg)
	}
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

	if _, isHunting := h.huntingTimers[userID]; isHunting {
		// Если идет охота, не позволяем выйти
		msg := tgbotapi.NewMessage(chatID, "Идет охота.")
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

	// Проверяем, есть ли активная сессия охоты
	if session, exists := h.huntingSessions[userID]; exists {
		// Удаляем сообщение с полем охоты
		deleteFieldMsg := tgbotapi.NewDeleteMessage(chatID, session.FieldMessageID)
		h.requestAPI(deleteFieldMsg)

		// Удаляем сообщение с информацией об охоте
		deleteInfoMsg := tgbotapi.NewDeleteMessage(chatID, session.InfoMessageID)
		h.requestAPI(deleteInfoMsg)

		// Удаляем сессию охоты
		delete(h.huntingSessions, userID)

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
	userID := message.From.ID

	// Проверяем, активен ли кулдаун охоты
	if cooldownEnd, exists := h.huntingCooldowns[userID]; exists {
		if time.Now().Before(cooldownEnd) {
			// Кулдаун еще активен
			remainingTime := time.Until(cooldownEnd)
			msg := tgbotapi.NewMessage(message.Chat.ID,
				fmt.Sprintf("До обновления охотничьих угодий осталось %d сек.", int(remainingTime.Seconds())))
			h.sendMessage(msg)
			return
		} else {
			// Кулдаун истек, удаляем его
			delete(h.huntingCooldowns, userID)
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

	// Получаем или создаем охоту
	hunting, err := h.db.GetOrCreateHunting(player.ID)
	if err != nil {
		log.Printf("Error getting hunting: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при работе с охотой.")
		h.sendMessage(msg)
		return
	}

	// Если охота была истощена в базе данных, восстанавливаем её
	if hunting.IsExhausted {
		if err := h.db.SetHuntingExhausted(player.ID, false); err != nil {
			log.Printf("Error setting hunting exhausted: %v", err)
		}
		hunting.IsExhausted = false
	}

	// Создаем новую сессию охоты
	h.createNewHuntingSession(userID, message.Chat.ID, hunting)
}

func (h *BotHandlers) createNewHuntingSession(userID int64, chatID int64, hunting *models.Hunting) {
	// Генерируем случайное поле
	field := h.generateRandomHuntingField()

	// Показываем поле и получаем MessageID
	fieldMessageID, infoMessageID := h.showHuntingField(chatID, hunting, field)

	// Создаем сессию
	session := &models.HuntingSession{
		PlayerID:       userID,
		Resources:      field,
		IsActive:       true,
		IsHunting:      false,
		StartedAt:      time.Now(),
		FieldMessageID: fieldMessageID,
		InfoMessageID:  infoMessageID,
	}

	h.huntingSessions[userID] = session
}

func (h *BotHandlers) generateRandomHuntingField() [][]string {
	// Создаем пустое поле 3x3
	field := make([][]string, 3)
	for i := range field {
		field[i] = make([]string, 3)
	}

	// Доступные ресурсы для охоты
	availableResources := []string{"🐰", "🐦"}

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

func (h *BotHandlers) showHuntingField(chatID int64, hunting *models.Hunting, field [][]string) (int, int) {
	// Создаем инлайн клавиатуру на основе переданного поля
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < 3; i++ {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3; j++ {
			cell := field[i][j]
			var callbackData string

			switch cell {
			case "🐰":
				callbackData = fmt.Sprintf("hunt_rabbit_%d_%d", i, j)
			case "🐦":
				callbackData = fmt.Sprintf("hunt_bird_%d_%d", i, j)
			default:
				callbackData = fmt.Sprintf("hunt_empty_%d_%d", i, j)
				cell = " "
			}

			button := tgbotapi.NewInlineKeyboardButtonData(cell, callbackData)
			row = append(row, button)
		}
		keyboard = append(keyboard, row)
	}

	// Сначала отправляем поле охоты с инлайн кнопками
	fieldMsg := tgbotapi.NewMessage(chatID, "Выберите цель для охоты:")
	fieldMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	fieldResponse, _ := h.sendChattableWithResponse(fieldMsg)

	// Затем отправляем информационное сообщение с клавиатурой
	// Вычисляем опыт до следующего уровня
	expToNext := (hunting.Level * 100) - hunting.Experience

	infoText := fmt.Sprintf(`🎯 Охота (Уровень %d)
До следующего уровня: %d опыта

Доступные ресурсы:
🐰 Кролик
🐦 Куропатка`, hunting.Level, expToNext)

	huntingKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	huntingKeyboard.ResizeKeyboard = true

	infoMsg := tgbotapi.NewMessage(chatID, infoText)
	infoMsg.ReplyMarkup = huntingKeyboard
	infoResponse, _ := h.sendChattableWithResponse(infoMsg)

	// Возвращаем ID поля и ID информационного сообщения
	return fieldResponse.MessageID, infoResponse.MessageID
}

func (h *BotHandlers) startHuntingAtPosition(userID int64, chatID int64, resourceName string, duration int, callbackID string, rowStr, colStr string) {
	row, _ := strconv.Atoi(rowStr)
	col, _ := strconv.Atoi(colStr)

	h.startHunting(userID, chatID, resourceName, duration, callbackID, row, col)
}

func (h *BotHandlers) startHunting(userID int64, chatID int64, resourceName string, duration int, callbackID string, row, col int) {
	// Проверяем, идет ли уже охота, добыча в шахте, рубка в лесу или крафт
	if _, exists := h.huntingTimers[userID]; exists {
		msg := tgbotapi.NewMessage(chatID, "Нельзя начинать новую охоту, пока не закончена текущая.")
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
	if _, exists := h.gatheringTimers[userID]; exists {
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

	// Проверяем наличие лука
	hasBow, bowDurability, err := h.db.HasToolInInventory(player.ID, "Простой лук")
	if err != nil {
		log.Printf("Error checking bow: %v", err)
		return
	}

	if !hasBow {
		msg := tgbotapi.NewMessage(chatID, `В инвентаре нет предмета "Простой лук".`)
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}

	// Проверяем наличие стрел
	arrowQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "Стрелы")
	if err != nil {
		log.Printf("Error getting arrow quantity: %v", err)
		arrowQuantity = 0
	}

	if arrowQuantity < 1 {
		msg := tgbotapi.NewMessage(chatID, `В инвентаре нет предмета "Стрелы".`)
		h.sendMessage(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "")
		h.requestAPI(callbackConfig)
		return
	}

	// Отвечаем на callback
	callbackConfig := tgbotapi.NewCallback(callbackID, "")
	h.requestAPI(callbackConfig)

	// Удаляем предыдущее сообщение о результате охоты, если оно существует
	if session, exists := h.huntingSessions[userID]; exists && session.ResultMessageID != 0 {
		deleteResultMsg := tgbotapi.NewDeleteMessage(chatID, session.ResultMessageID)
		h.requestAPI(deleteResultMsg)
		session.ResultMessageID = 0 // Сбрасываем ID
	}

	// Отправляем сообщение о начале охоты
	initialText := fmt.Sprintf(`Идет охота на "%s". Время охоты %d сек.
		
%s 0%%`, resourceName, duration, h.createProgressBar(0, 10))

	huntingMsg := tgbotapi.NewMessage(chatID, initialText)
	sentMsg, _ := h.sendMessageWithResponse(huntingMsg)

	// Запускаем горутину для обновления прогресс бара
	go h.updateHuntingProgress(userID, chatID, sentMsg.MessageID, resourceName, duration, bowDurability, row, col)

	// Создаем заглушку таймера
	timer := time.NewTimer(time.Duration(duration) * time.Second)
	h.huntingTimers[userID] = timer
}

func (h *BotHandlers) updateHuntingProgress(userID int64, chatID int64, messageID int, resourceName string, totalDuration int, durability int, row, col int) {
	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime).Seconds()
			progress := int(elapsed)

			if progress >= totalDuration {
				// Охота завершена
				h.completeHunting(userID, chatID, resourceName, durability, messageID, row, col)
				return
			}

			// Обновляем прогресс бар
			percentage := int((elapsed / float64(totalDuration)) * 100)
			progressBar := h.createProgressBar(progress, totalDuration)

			newText := fmt.Sprintf(`Началась охота на "%s". Время охоты %d сек.
			
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

func (h *BotHandlers) completeHunting(userID int64, chatID int64, resourceName string, oldDurability int, messageID int, row, col int) {
	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		return
	}

	// Уменьшаем прочность лука на 1
	newDurability := oldDurability - 1
	if newDurability <= 0 {
		// Лук сломался, удаляем его
		err = h.db.RemoveItemFromInventory(player.ID, "Простой лук", 1)
		if err != nil {
			log.Printf("Error removing broken bow: %v", err)
		}
	} else {
		// Обновляем прочность лука
		err = h.db.UpdateToolDurability(player.ID, "Простой лук", newDurability)
		if err != nil {
			log.Printf("Error updating bow durability: %v", err)
		}
	}

	// Уменьшаем количество стрел на 1
	err = h.db.RemoveItemFromInventory(player.ID, "Стрелы", 1)
	if err != nil {
		log.Printf("Error removing arrow: %v", err)
	}

	// Добавляем добытый ресурс в инвентарь
	err = h.db.AddItemToInventory(player.ID, resourceName, 1)
	if err != nil {
		log.Printf("Error adding hunted resource: %v", err)
	}

	// Добавляем опыт охоты
	expGained := 2
	levelUp, newLevel, err := h.db.UpdateHuntingExperience(player.ID, expGained)
	if err != nil {
		log.Printf("Error updating hunting experience: %v", err)
	}

	// Получаем количество стрел после охоты
	arrowsLeft, err := h.db.GetItemQuantityInInventory(player.ID, "Стрелы")
	if err != nil {
		log.Printf("Error getting arrows quantity: %v", err)
		arrowsLeft = 0
	}

	// Получаем обновленные данные
	updatedPlayer, _ := h.db.GetPlayer(userID)
	updatedHunting, _ := h.db.GetOrCreateHunting(player.ID)

	// Удаляем таймер
	if timer, exists := h.huntingTimers[userID]; exists {
		timer.Stop()
		delete(h.huntingTimers, userID)
	}

	// Обновляем поле охоты (убираем добытый ресурс)
	if session, exists := h.huntingSessions[userID]; exists {
		session.Resources[row][col] = ""
		h.updateHuntingField(chatID, session.Resources, session.FieldMessageID)

		// Проверяем, остались ли ресурсы
		hasResources := false
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if session.Resources[i][j] != "" {
					hasResources = true
					break
				}
			}
			if hasResources {
				break
			}
		}

		if !hasResources {
			// Все ресурсы добыты, истощаем охоту
			err = h.db.ExhaustHunting(userID)
			if err != nil {
				log.Printf("Error exhausting hunting: %v", err)
			}

			// Устанавливаем кулдаун на 1 минуту
			h.huntingCooldowns[userID] = time.Now().Add(1 * time.Minute)

			// Удаляем сообщение с полем охоты
			deleteFieldMsg := tgbotapi.NewDeleteMessage(chatID, session.FieldMessageID)
			h.requestAPI(deleteFieldMsg)

			// Удаляем сообщение с информацией об охоте
			deleteInfoMsg := tgbotapi.NewDeleteMessage(chatID, session.InfoMessageID)
			h.requestAPI(deleteInfoMsg)

			// Отправляем сообщение об истощении с клавиатурой леса
			exhaustMsg := tgbotapi.NewMessage(chatID, `⚠️ Охотничьи угодья истощены! Необходимо подождать 1 минуту до восстановления ресурсов.
Нажми кнопку "🎯 Охота" чтобы проверить готовность.`)
			h.sendForestKeyboard(exhaustMsg)

			// Удаляем сессию
			delete(h.huntingSessions, userID)
		} else {
			// Обновляем информационное сообщение с актуальными данными охоты
			h.updateHuntingInfoMessage(userID, chatID, updatedHunting, session.InfoMessageID)
		}
	}

	// Формируем сообщение о результате
	satietyText := ""
	if updatedPlayer != nil {
		satietyText = fmt.Sprintf(`
🍖 Сытость: %d`, updatedPlayer.Satiety)
	}

	// Вычисляем опыт до следующего уровня
	expToNext := 0
	if updatedHunting != nil {
		expToNext = (updatedHunting.Level * 100) - updatedHunting.Experience
	}

	resultText := fmt.Sprintf(`✅ Охота завершена!

Добыто: %s x1
Опыт охоты: +%d
До следующего уровня: %d опыта%s`, resourceName, expGained, expToNext, satietyText)

	if levelUp {
		resultText += fmt.Sprintf(`
🎉 Уровень охоты повышен до %d!`, newLevel)
	}

	if newDurability <= 0 {
		resultText += fmt.Sprintf(`
💔 Простой лук сломался!
🏹 Стрел осталось: %d`, arrowsLeft)
	} else {
		resultText += fmt.Sprintf(`
🏹 Прочность лука: %d
🏹 Стрел осталось: %d`, newDurability, arrowsLeft)
	}

	// Редактируем сообщение с прогрессом на результат
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, resultText)
	h.editMessage(editMsg)

	// Сохраняем ID сообщения с результатом в сессии
	if session, exists := h.huntingSessions[userID]; exists {
		session.ResultMessageID = messageID
	}

	// Проверяем прогресс квеста 5 (первая охота)
	h.checkHuntingQuestProgress(userID, chatID, player.ID)
}

func (h *BotHandlers) updateHuntingField(chatID int64, field [][]string, messageID int) {
	// Создаем инлайн клавиатуру на основе обновленного поля
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < 3; i++ {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3; j++ {
			cell := field[i][j]
			var callbackData string

			switch cell {
			case "🐰":
				callbackData = fmt.Sprintf("hunt_rabbit_%d_%d", i, j)
			case "🐦":
				callbackData = fmt.Sprintf("hunt_bird_%d_%d", i, j)
			default:
				callbackData = fmt.Sprintf("hunt_empty_%d_%d", i, j)
				cell = " "
			}

			button := tgbotapi.NewInlineKeyboardButtonData(cell, callbackData)
			row = append(row, button)
		}
		keyboard = append(keyboard, row)
	}

	// Редактируем сообщение с полем
	editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.NewInlineKeyboardMarkup(keyboard...))
	h.editMessage(editMsg)
}

func (h *BotHandlers) updateHuntingInfoMessage(userID int64, chatID int64, hunting *models.Hunting, messageID int) {
	// Вычисляем опыт до следующего уровня
	expToNext := (hunting.Level * 100) - hunting.Experience

	infoText := fmt.Sprintf(`🎯 Охота (Уровень %d)
До следующего уровня: %d опыта

Доступные ресурсы:
🐰 Кролик
🐦 Куропатка`, hunting.Level, expToNext)

	// Удаляем старое сообщение
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)

	// Создаем новое сообщение с клавиатурой
	huntingKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	huntingKeyboard.ResizeKeyboard = true

	newMsg := tgbotapi.NewMessage(chatID, infoText)
	newMsg.ReplyMarkup = huntingKeyboard
	newResponse, _ := h.sendMessageWithResponse(newMsg)

	// Обновляем ID сообщения в сессии
	if session, exists := h.huntingSessions[userID]; exists {
		session.InfoMessageID = newResponse.MessageID
	}
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
	var buttonText, callbackData string
	if canBuild {
		buttonText = "Создать ✅"
		callbackData = "craft_Простая хижина"
	} else {
		buttonText = "Создать ❌"
		callbackData = "no_craft_Простая хижина"
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, recipeText)

	// Создаем инлайн клавиатуру с кнопкой создать
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData),
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

	// Обрабатываем навигацию по страницам
	if strings.HasPrefix(data, "page_") {
		parts := strings.Split(data, "_")
		if len(parts) != 3 {
			return
		}

		direction := parts[1]
		currentPage, _ := strconv.Atoi(parts[2])

		// Получаем игрока
		player, err := h.db.GetPlayer(userID)
		if err != nil {
			log.Printf("Error getting player: %v", err)
			return
		}

		// Получаем инвентарь игрока
		inventory, err := h.db.GetPlayerInventory(player.ID)
		if err != nil {
			log.Printf("Error getting inventory: %v", err)
			return
		}

		// Создаем карту доступных страниц
		pageMap := make(map[int]struct {
			title string
			text  string
		})

		// Определяем доступные страницы и их тексты
		for _, item := range inventory {
			if strings.Contains(item.ItemName, "📖 Страница 1") && item.Quantity > 0 {
				pageMap[1] = struct {
					title string
					text  string
				}{
					title: "📖 Страница 1 «Забытая тишина»",
					text:  "Мир не был уничтожен в битве. Он просто... забыл сам себя.\nГоды прошли — может, столетия, может, тысячелетия. Никто не знает точно. От былых королевств остались лишь заросшие руины, поросшие мхом камни и полустёртые знаки, выгравированные на обломках.",
				}
			}
			if strings.Contains(item.ItemName, "📖 Страница 2") && item.Quantity > 0 {
				pageMap[2] = struct {
					title string
					text  string
				}{
					title: "📖 Страница 2 «Пепел памяти»",
					text:  "Люди исчезли. Не все, возможно, но память о них — точно.\nЗемля забыла их шаги. Знания рассыпались, будто песок в ветре. Остались лишь сны, смутные образы, и тихий зов из глубин мира.",
				}
			}
			if strings.Contains(item.ItemName, "📖 Страница 3") && item.Quantity > 0 {
				pageMap[3] = struct {
					title string
					text  string
				}{
					title: "📖 Страница 3 «Пробуждение»",
					text:  "Ты — один из тех, кто откликнулся.\nНикто не сказал тебе, зачем ты проснулся. В этом нет наставников, богов или проводников. Только ты, дикая земля — и чувство, что всё это уже было. Что ты здесь не впервые.",
				}
			}
			if strings.Contains(item.ItemName, "📖 Страница 4") && item.Quantity > 0 {
				pageMap[4] = struct {
					title string
					text  string
				}{
					title: "📖 Страница 4 «Без имени»",
					text:  "У тебя ничего нет. Ни дома, ни имени, ни цели. Только старая кирка, тёплый свет солнца и бескрайняя, живая земля, что будто наблюдает за каждым твоим шагом.",
				}
			}
			if strings.Contains(item.ItemName, "📖 Страница 5") && item.Quantity > 0 {
				pageMap[5] = struct {
					title string
					text  string
				}{
					title: "📖 Страница 5 «Искра перемен»",
					text:  "Но ты чувствуешь — если построить хижину, зажечь огонь, добыть первый камень… что-то изменится.\nВ тебе. В этом месте. В самой памяти мира.\nВозможно, ты не просто выживший. Возможно, ты — начало нового.",
				}
			}
			if strings.Contains(item.ItemName, "📖 Страница 6") && item.Quantity > 0 {
				pageMap[6] = struct {
					title string
					text  string
				}{
					title: "📖 Страница 6 «Наблюдающий лес»",
					text:  "Поначалу земля молчала. Ты копал, строил, охотился — и всё было, как будто в пустоте.\nНо с каждым ударом по камню, с каждым дымком над костром ты чувствовал, что что-то наблюдает. Не враждебное. Но древнее.",
				}
			}
			if strings.Contains(item.ItemName, "📖 Страница 7") && item.Quantity > 0 {
				pageMap[7] = struct {
					title string
					text  string
				}{
					title: "📖 Страница 7 «Шёпот ветра»",
					text:  "Иногда по ночам ты слышал, как шелестят листья без ветра.\nКак в костре трескается не дрова, а слова. Неслышные, шепчущие.\nЗемля словно пыталась заговорить с тобой, но ещё не решалась.",
				}
			}
			if strings.Contains(item.ItemName, "📖 Страница 8") && item.Quantity > 0 {
				pageMap[8] = struct {
					title string
					text  string
				}{
					title: "📖 Страница 8 «След древних»",
					text:  "Ты начал находить странные вещи. Камень с гладкой гранью, словно вырезанной руками.\nОбломок кости с выжженным символом. Одинокую статую, стоящую посреди леса, покрытую мхом, но не разрушенную.",
				}
			}
		}

		// Определяем следующую или предыдущую страницу
		var targetPage int
		if direction == "next" {
			for i := currentPage + 1; i <= 8; i++ {
				if _, exists := pageMap[i]; exists {
					targetPage = i
					break
				}
			}
		} else if direction == "prev" {
			for i := currentPage - 1; i >= 1; i-- {
				if _, exists := pageMap[i]; exists {
					targetPage = i
					break
				}
			}
		}

		if targetPage == 0 {
			// Если страница не найдена, отвечаем на callback
			callbackConfig := tgbotapi.NewCallback(callback.ID, "")
			h.requestAPI(callbackConfig)
			return
		}

		// Формируем текст сообщения
		page := pageMap[targetPage]
		text := fmt.Sprintf("%s\n\n%s\n\nСтраница %d из %d", page.title, page.text, targetPage, len(pageMap))

		// Создаем кнопки навигации
		var keyboard tgbotapi.InlineKeyboardMarkup
		var row []tgbotapi.InlineKeyboardButton

		// Кнопка "Назад" с callback_data
		prevBtn := tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", fmt.Sprintf("page_prev_%d", targetPage))
		row = append(row, prevBtn)

		// Кнопка "Дальше" с callback_data
		nextBtn := tgbotapi.NewInlineKeyboardButtonData("Дальше ▶️", fmt.Sprintf("page_next_%d", targetPage))
		row = append(row, nextBtn)

		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)

		// Обновляем сообщение
		editMsg := tgbotapi.NewEditMessageTextAndMarkup(
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			text,
			keyboard,
		)
		h.editMessage(editMsg)

		// Отвечаем на callback
		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		h.requestAPI(callbackConfig)

		// Проверяем прогресс квеста 6 после успешного чтения страницы
		h.checkLorePagesQuestProgressSequential(userID, callback.Message.Chat.ID, player.ID, targetPage)

		return
	}

	// Обрабатываем остальные callback'и
	if strings.HasPrefix(data, "mine_") {
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
	} else if strings.HasPrefix(data, "hunt_rabbit_") {
		// Обрабатываем callback'и от охоты на кролика
		parts := strings.Split(data, "_")
		if len(parts) == 4 {
			row, col := parts[2], parts[3]
			h.startHuntingAtPosition(userID, callback.Message.Chat.ID, "Кролик", 20, callback.ID, row, col)
		}
	} else if strings.HasPrefix(data, "hunt_bird_") {
		// Обрабатываем callback'и от охоты на куропатку
		parts := strings.Split(data, "_")
		if len(parts) == 4 {
			row, col := parts[2], parts[3]
			h.startHuntingAtPosition(userID, callback.Message.Chat.ID, "Куропатка", 20, callback.ID, row, col)
		}
	} else if strings.HasPrefix(data, "hunt_empty_") {
		// Пустая ячейка
		callbackConfig := tgbotapi.NewCallback(callback.ID, "Здесь нет добычи!")
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
	} else if strings.HasPrefix(data, "quest_accept_6") {
		// Принятие квеста 6
		questIDStr := strings.TrimPrefix(data, "quest_accept_")
		questID, _ := strconv.Atoi(questIDStr)
		h.handleQuestAccept(userID, callback.Message.Chat.ID, questID, callback.ID, callback.Message.MessageID)
	} else if strings.HasPrefix(data, "quest_decline_6") {
		// Отказ от квеста 6
		h.handleQuestDecline(callback.Message.Chat.ID, callback.ID, callback.Message.MessageID)
	} else if strings.HasPrefix(data, "quest_accept_7") {
		// Принятие квеста 7
		questIDStr := strings.TrimPrefix(data, "quest_accept_")
		questID, _ := strconv.Atoi(questIDStr)
		h.handleQuestAccept(userID, callback.Message.Chat.ID, questID, callback.ID, callback.Message.MessageID)
	} else if strings.HasPrefix(data, "quest_decline_7") {
		// Отказ от квеста 7
		h.handleQuestDecline(callback.Message.Chat.ID, callback.ID, callback.Message.MessageID)
	} else if strings.HasPrefix(data, "quest_accept_8") {
		// Принятие квеста 8
		questIDStr := strings.TrimPrefix(data, "quest_accept_")
		questID, _ := strconv.Atoi(questIDStr)
		h.handleQuestAccept(userID, callback.Message.Chat.ID, questID, callback.ID, callback.Message.MessageID)
	} else if strings.HasPrefix(data, "quest_decline_8") {
		// Отказ от квеста 8
		h.handleQuestDecline(callback.Message.Chat.ID, callback.ID, callback.Message.MessageID)
	} else {
		// Остальные callback
		// msg := tgbotapi.NewMessage(callback.Message.Chat.ID, "🔨 Функция пока в разработке...")
		// h.sendMessage(msg)
		// callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		// h.requestAPI(callbackConfig)
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
Задание: Наруби 5 берёзы.
Для выполнения задания необходимо проследовать в Добыча/Лес/Рубка.
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
Задание: Наруби 5 берёзы. (%d/5)
Для выполнения задания необходимо проследовать в Добыча/Лес/Рубка.
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
Задание: Добудь 3 камня.
Для выполнения задания необходимо проследовать в Добыча/Шахта.
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
Для выполнения задания необходимо проследовать в Добыча/Шахта.
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
Задание: Создай 3 берёзовых бруса.
Для выполнения задания необходимо проследовать в Рабочее место/Верстак и выполнить команду создания бруса.
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
Для выполнения задания необходимо проследовать в Рабочее место/Верстак и выполнить команду создания бруса.
Награда: 🎖 10 опыта + 📖 Страница 3 «Первый шаг»`, quest3.Progress)

				msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
				h.sendMessage(msg)
				return
			}

			if quest3.Status == "completed" {
				// Квест 3 выполнен, проверяем квест 4
				quest4, err := h.db.GetPlayerQuest(player.ID, 4)
				if err != nil {
					log.Printf("Error getting quest 4: %v", err)
					msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении квеста.")
					h.sendMessage(msg)
					return
				}

				if quest4 == nil || quest4.Status == "available" {
					// Квест 4 еще не создан или доступен для принятия
					if quest4 == nil {
						// Создаем квест 4, если его нет
						err := h.db.CreateQuest(player.ID, 4, 5) // Квест 4: собрать 5 лесных ягод
						if err != nil {
							log.Printf("Error creating quest 4: %v", err)
							msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при создании квеста.")
							h.sendMessage(msg)
							return
						}
					}

					// Показываем предложение квеста 4
					questText := `🍇 Квест 4: Дар леса
Задание: Собери 5 лесных ягод.
Для выполнения задания необходимо проследовать в Добыча/Лес/Сбор.
Награда: 🎖 10 опыта + 📖 Страница 4 «Голос земли»`

					msg := tgbotapi.NewMessage(message.Chat.ID, questText)

					// Создаем инлайн кнопки
					acceptBtn := tgbotapi.NewInlineKeyboardButtonData("Принять", "quest_accept_4")
					declineBtn := tgbotapi.NewInlineKeyboardButtonData("Отказ", "quest_decline_4")
					keyboard := tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(acceptBtn, declineBtn),
					)
					msg.ReplyMarkup = keyboard
					h.sendMessage(msg)
					return
				}

				if quest4.Status == "active" {
					// Квест 4 активен, показываем прогресс
					activeText := fmt.Sprintf(`Активный квест: 🍇 Квест 4: Дар леса
Задание: Собери 5 лесных ягод (%d/5)
Для выполнения задания необходимо проследовать в Добыча/Лес/Сбор.
Награда: 🎖 10 опыта + 📖 Страница 4 «Голос земли»`, quest4.Progress)

					msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
					h.sendMessage(msg)
					return
				}

				if quest4.Status == "completed" {
					// Квест 4 выполнен, проверяем квест 5
					quest5, err := h.db.GetPlayerQuest(player.ID, 5)
					if err != nil {
						log.Printf("Error getting quest 5: %v", err)
						msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении квеста.")
						h.sendMessage(msg)
						return
					}
					if quest5 == nil || quest5.Status == "available" {
						// Квест 5 еще не создан или доступен для принятия
						if quest5 == nil {
							// Создаем квест 5, если его нет
							err := h.db.CreateQuest(player.ID, 5, 1) // Квест 5: совершить первую охоту
							if err != nil {
								log.Printf("Error creating quest 5: %v", err)
								msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при создании квеста.")
								h.sendMessage(msg)
								return
							}
						}
						// Показываем предложение квеста 5
						questText := `🎯 Квест 5: Звериный взгляд
Задание: Соверши первую охоту.
Для выполнения задания необходимо проследовать в Добыча/Лес/Охота.
Награда: 🎖 10 опыта + 📖 Страница 5 «След древних»`
						msg := tgbotapi.NewMessage(message.Chat.ID, questText)
						acceptBtn := tgbotapi.NewInlineKeyboardButtonData("Принять", "quest_accept_5")
						declineBtn := tgbotapi.NewInlineKeyboardButtonData("Отказ", "quest_decline_5")
						keyboard := tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(acceptBtn, declineBtn),
						)
						msg.ReplyMarkup = keyboard
						h.sendMessage(msg)
						return
					}
					if quest5.Status == "active" {
						activeText := fmt.Sprintf(`Активный квест: 🎯 Квест 5: Звериный взгляд
Задание: Соверши первую охоту (%d/1)
Для выполнения задания необходимо проследовать в Добыча/Лес/Охота.
Награда: 🎖 10 опыта + 📖 Страница 5 «След древних»`, quest5.Progress)
						msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
						h.sendMessage(msg)
						return
					}
					if quest5.Status == "completed" {
						// Квест 5 выполнен, проверяем квест 6
						quest6, err := h.db.GetPlayerQuest(player.ID, 6)
						if err != nil {
							log.Printf("Error getting quest 6: %v", err)
							msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении квеста.")
							h.sendMessage(msg)
							return
						}
						if quest6 == nil || quest6.Status == "available" {
							// Квест 6 еще не создан или доступен для принятия
							if quest6 == nil {
								// Создаем квест 6, если его нет
								err := h.db.CreateQuest(player.ID, 6, 5) // Квест 6: прочитать 5 страниц
								if err != nil {
									log.Printf("Error creating quest 6: %v", err)
									msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при создании квеста.")
									h.sendMessage(msg)
									return
								}
							}
							// Показываем предложение квеста 6
							questText := `📘 Квест 6: Живое Хранилище
Задание: Открой 5 страниц лора
Для выполнения задания необходимо в инвентаре в разделе "Страницы"
вызвать команду look и затем прочитать 5 страниц ЛОРа от 1-й до 5-й.
Награда: 🎖 10 опыта + 📖 Страница 6 «Сон о башне»`
							msg := tgbotapi.NewMessage(message.Chat.ID, questText)
							acceptBtn := tgbotapi.NewInlineKeyboardButtonData("Принять", "quest_accept_6")
							declineBtn := tgbotapi.NewInlineKeyboardButtonData("Отказ", "quest_decline_6")
							keyboard := tgbotapi.NewInlineKeyboardMarkup(
								tgbotapi.NewInlineKeyboardRow(acceptBtn, declineBtn),
							)
							msg.ReplyMarkup = keyboard
							h.sendMessage(msg)
							return
						}
						if quest6.Status == "active" {
							activeText := fmt.Sprintf(`Активный квест: 📘 Квест 6: Живое Хранилище
Задание: Открой 5 страниц лора (%d/5)
Награда: 🎖 10 опыта + 📖 Страница 6 «Сон о башне»`, quest6.Progress)
							msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
							h.sendMessage(msg)
							return
						}
						if quest6.Status == "completed" {
							// Квест 6 выполнен, проверяем квест 7
							quest7, err := h.db.GetPlayerQuest(player.ID, 7)
							if err != nil {
								log.Printf("Error getting quest 7: %v", err)
								msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении квеста.")
								h.sendMessage(msg)
								return
							}

							if quest7 == nil || quest7.Status == "available" {
								// Квест 7 еще не создан или доступен для принятия
								if quest7 == nil {
									// Создаем квест 7, если его нет
									err := h.db.CreateQuest(player.ID, 7, 3) // Квест 7: съесть 3 ягоды
									if err != nil {
										log.Printf("Error creating quest 7: %v", err)
										msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при создании квеста.")
										h.sendMessage(msg)
										return
									}
								}

								// Показываем предложение квеста 7
								questText := `🍇 Квест 7: Перекус
Задание: Съешь 3 лесные ягоды.
Для выполнения квеста необходимо в инвентаре напротив предмета "Лесная ягода" выполнить команду eat.
Награда: 🎖 10 опыта + 📖 Страница 7 «Шёпот ветра»`

								msg := tgbotapi.NewMessage(message.Chat.ID, questText)

								// Создаем инлайн кнопки
								acceptBtn := tgbotapi.NewInlineKeyboardButtonData("Принять", "quest_accept_7")
								declineBtn := tgbotapi.NewInlineKeyboardButtonData("Отказ", "quest_decline_7")
								keyboard := tgbotapi.NewInlineKeyboardMarkup(
									tgbotapi.NewInlineKeyboardRow(acceptBtn, declineBtn),
								)
								msg.ReplyMarkup = keyboard
								h.sendMessage(msg)
								return
							}

							if quest7.Status == "active" {
								// Квест 7 активен, показываем прогресс
								activeText := fmt.Sprintf(`Активный квест: 🍇 Квест 7: Перекус
Задание: Съешь 3 лесные ягоды (%d/3)
Для выполнения квеста необходимо в инвентаре напротив предмета "Лесная ягода" выполнить команду eat.
Награда: 🎖 10 опыта + 📖 Страница 7 «Шёпот ветра»`, quest7.Progress)

								msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
								h.sendMessage(msg)
								return
							}

							if quest7.Status == "completed" {
								// Квест 7 выполнен, проверяем квест 8
								quest8, err := h.db.GetPlayerQuest(player.ID, 8)
								if err != nil {
									log.Printf("Error getting quest 8: %v", err)
									msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении квеста.")
									h.sendMessage(msg)
									return
								}

								if quest8 == nil || quest8.Status == "available" {
									// Квест 8 еще не создан или доступен для принятия
									if quest8 == nil {
										// Создаем квест 8, если его нет
										err := h.db.CreateQuest(player.ID, 8, 1) // Квест 8: построить простую хижину
										if err != nil {
											log.Printf("Error creating quest 8: %v", err)
											msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при создании квеста.")
											h.sendMessage(msg)
											return
										}
									}

									// Проверяем, построена ли уже хижина
									if player.SimpleHutBuilt {
										// Если хижина уже построена, сразу завершаем квест
										err = h.db.UpdateQuestStatus(player.ID, 8, "completed")
										if err != nil {
											log.Printf("Error completing quest 8: %v", err)
											msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при выполнении квеста.")
											h.sendMessage(msg)
											return
										}

										// Добавляем награды
										err = h.db.UpdatePlayerExperience(player.ID, 10)
										if err != nil {
											log.Printf("Error updating player experience: %v", err)
										}

										err = h.db.AddItemToInventory(player.ID, "📖 Страница 8 «След древних»", 1)
										if err != nil {
											log.Printf("Error adding quest item to inventory: %v", err)
										}

										// Отправляем сообщение о выполнении квеста
										questCompleteText := `🛖 Квест 8: Под крышей ВЫПОЛНЕН!
Получена награда:
🎖 10 опыта
📖 Страница 8 «След древних»`

										msg := tgbotapi.NewMessage(message.Chat.ID, questCompleteText)
										h.sendMessage(msg)
										return
									}

									// Показываем предложение квеста 8
									questText := `🛖 Квест 8: Под крышей
Задание: Построй простую хижину.
Для выполнения задания необходимо проследовать в Постройки и выполнить команду создания простой хижины.
Награда: 🎖 10 опыта + 📖 Страница 8 «След древних»`

									msg := tgbotapi.NewMessage(message.Chat.ID, questText)
									acceptBtn := tgbotapi.NewInlineKeyboardButtonData("Принять", "quest_accept_8")
									declineBtn := tgbotapi.NewInlineKeyboardButtonData("Отказ", "quest_decline_8")
									keyboard := tgbotapi.NewInlineKeyboardMarkup(
										tgbotapi.NewInlineKeyboardRow(acceptBtn, declineBtn),
									)
									msg.ReplyMarkup = keyboard
									h.sendMessage(msg)
									return
								}

								if quest8.Status == "active" {
									activeText := fmt.Sprintf(`Активный квест: 🛖 Квест 8: Под крышей
Задание: Построй простую хижину (%d/1)
Для выполнения задания необходимо проследовать в Постройки и выполнить команду создания простой хижины.
Награда: 🎖 10 опыта + 📖 Страница 8 «След древних»`, quest8.Progress)
									msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
									h.sendMessage(msg)
									return
								}

								if quest8.Status == "completed" {
									msg := tgbotapi.NewMessage(message.Chat.ID, "🎉 Ты завершил всю цепочку ЛОР-квестов! Продолжение следует...")
									h.sendMessage(msg)
									return
								}
							}
						}
					}
				}
			}
		}
	}

	// Квест 4 выполнен, проверяем квест 5
	quest5, err := h.db.GetPlayerQuest(player.ID, 5)
	if err != nil {
		log.Printf("Error getting quest 5: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении квеста.")
		h.sendMessage(msg)
		return
	}
	if quest5 == nil || quest5.Status == "available" {
		// Квест 5 еще не создан или доступен для принятия
		if quest5 == nil {
			// Создаем квест 5, если его нет
			err := h.db.CreateQuest(player.ID, 5, 1) // Квест 5: совершить первую охоту
			if err != nil {
				log.Printf("Error creating quest 5: %v", err)
				msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при создании квеста.")
				h.sendMessage(msg)
				return
			}
		}
		// Показываем предложение квеста 5
		questText := `🎯 Квест 5: Звериный взгляд
Задание: Соверши первую охоту.
Для выполнения задания необходимо проследовать в Добыча/Лес/Охота.
Награда: 🎖 10 опыта + 📖 Страница 5 «След древних»`
		msg := tgbotapi.NewMessage(message.Chat.ID, questText)
		acceptBtn := tgbotapi.NewInlineKeyboardButtonData("Принять", "quest_accept_5")
		declineBtn := tgbotapi.NewInlineKeyboardButtonData("Отказ", "quest_decline_5")
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(acceptBtn, declineBtn),
		)
		msg.ReplyMarkup = keyboard
		h.sendMessage(msg)
		return
	}
	if quest5.Status == "active" {
		activeText := fmt.Sprintf(`Активный квест: 🎯 Квест 5: Звериный взгляд
Задание: Соверши первую охоту (%d/1)
Для выполнения задания необходимо проследовать в Добыча/Лес/Охота.
Награда: 🎖 10 опыта + 📖 Страница 5 «След древних»`, quest5.Progress)
		msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
		h.sendMessage(msg)
		return
	}
	if quest5.Status == "completed" {
		// Квест 5 выполнен, проверяем квест 6
		quest6, err := h.db.GetPlayerQuest(player.ID, 6)
		if err != nil {
			log.Printf("Error getting quest 6: %v", err)
			msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении квеста.")
			h.sendMessage(msg)
			return
		}
		if quest6 == nil || quest6.Status == "available" {
			// Квест 6 еще не создан или доступен для принятия
			if quest6 == nil {
				// Создаем квест 6, если его нет
				err := h.db.CreateQuest(player.ID, 6, 5) // Квест 6: прочитать 5 страниц
				if err != nil {
					log.Printf("Error creating quest 6: %v", err)
					msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при создании квеста.")
					h.sendMessage(msg)
					return
				}
			}
			// Показываем предложение квеста 6
			questText := `📘 Квест 6: Живое Хранилище
Задание: Открой 5 страниц лора.
Для выполнения задания необходимо в инвентаре в разделе "Страницы"
вызвать команду look и затем прочитать 5 страниц ЛОРа от 1-й до 5-й.
Награда: 🎖 10 опыта + 📖 Страница 6 «Сон о башне»`
			msg := tgbotapi.NewMessage(message.Chat.ID, questText)
			acceptBtn := tgbotapi.NewInlineKeyboardButtonData("Принять", "quest_accept_6")
			declineBtn := tgbotapi.NewInlineKeyboardButtonData("Отказ", "quest_decline_6")
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(acceptBtn, declineBtn),
			)
			msg.ReplyMarkup = keyboard
			h.sendMessage(msg)
			return
		}
		if quest6.Status == "active" {
			activeText := fmt.Sprintf(`Активный квест: 📘 Квест 6: Живое Хранилище
Задание: Открой 5 страниц лора (%d/5)
Для выполнения задания необходимо в инвентаре в разделе "Страницы"
вызвать команду look и затем прочитать 5 страниц ЛОРа от 1-й до 5-й.
Награда: 🎖 10 опыта + 📖 Страница 6 «Сон о башне»`, quest6.Progress)
			msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
			h.sendMessage(msg)
			return
		}
		if quest6.Status == "completed" {
			msg := tgbotapi.NewMessage(message.Chat.ID, "🎉 Ты завершил всю цепочку ЛОР-квестов! Продолжение следует...")
			h.sendMessage(msg)
			return
		}
	}

	// Квест 6 выполнен, проверяем квест 7
	quest7, err := h.db.GetPlayerQuest(player.ID, 7)
	if err != nil {
		log.Printf("Error getting quest 7: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при получении квеста.")
		h.sendMessage(msg)
		return
	}
	if quest7 == nil || quest7.Status == "available" {
		// Квест 7 еще не создан или доступен для принятия
		if quest7 == nil {
			// Создаем квест 7, если его нет
			err := h.db.CreateQuest(player.ID, 7, 3) // Квест 7: съесть 3 ягоды
			if err != nil {
				log.Printf("Error creating quest 7: %v", err)
				msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка при создании квеста.")
				h.sendMessage(msg)
				return
			}
		}
		// Показываем предложение квеста 7
		questText := `🍔 Квест 7: Перекус
Задание: Съешь 3 ягоды
Для выполнения квеста в инвентаре напротив Лесной ягоды выполнить команду eat.
Награда: 🎖 10 опыта + 📖 Страница 7 «Шёпот ветра»`
		msg := tgbotapi.NewMessage(message.Chat.ID, questText)
		acceptBtn := tgbotapi.NewInlineKeyboardButtonData("Принять", "quest_accept_7")
		declineBtn := tgbotapi.NewInlineKeyboardButtonData("Отказ", "quest_decline_7")
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(acceptBtn, declineBtn),
		)
		msg.ReplyMarkup = keyboard
		h.sendMessage(msg)
		return
	}
	if quest7.Status == "active" {
		activeText := fmt.Sprintf(`Активный квест: 🍔 Квест 7: Перекус
Задание: Съешь 3 ягоды (%d/3)
Для выполнения квеста необходимо в инвентаре напротив предмета "Лесная ягода" выполнить команду eat.
Награда: 🎖 10 опыта + 📖 Страница 7 «Шёпот ветра»`, quest7.Progress)
		msg := tgbotapi.NewMessage(message.Chat.ID, activeText)
		h.sendMessage(msg)
		return
	}
	if quest7.Status == "completed" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "🎉 Ты завершил всю цепочку ЛОР-квестов! Продолжение следует...")
		h.sendMessage(msg)
		return
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
	userID := message.From.ID
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	buildingsText := "🏘️ Доступные постройки:\n"
	builtText := ""

	if player.SimpleHutBuilt {
		builtText += "🏠 Простая хижина /open\n"
	} else {
		buildingsText += "Простая хижина /create_simple_hut\n"
	}

	if builtText != "" {
		buildingsText += "\nПостроено:\n" + builtText
	}

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

	// Создаем карту доступных страниц
	pageMap := make(map[int]struct {
		title string
		text  string
	})

	// Определяем доступные страницы и их тексты
	for _, item := range inventory {
		if strings.Contains(item.ItemName, "📖 Страница 1") && item.Quantity > 0 {
			pageMap[1] = struct {
				title string
				text  string
			}{
				title: "📖 Страница 1 «Забытая тишина»",
				text:  "Мир не был уничтожен в битве. Он просто... забыл сам себя.\nГоды прошли — может, столетия, может, тысячелетия. Никто не знает точно. От былых королевств остались лишь заросшие руины, поросшие мхом камни и полустёртые знаки, выгравированные на обломках.",
			}
		}
		if strings.Contains(item.ItemName, "📖 Страница 2") && item.Quantity > 0 {
			pageMap[2] = struct {
				title string
				text  string
			}{
				title: "📖 Страница 2 «Пепел памяти»",
				text:  "Люди исчезли. Не все, возможно, но память о них — точно.\nЗемля забыла их шаги. Знания рассыпались, будто песок в ветре. Остались лишь сны, смутные образы, и тихий зов из глубин мира.",
			}
		}
		if strings.Contains(item.ItemName, "📖 Страница 3") && item.Quantity > 0 {
			pageMap[3] = struct {
				title string
				text  string
			}{
				title: "📖 Страница 3 «Пробуждение»",
				text:  "Ты — один из тех, кто откликнулся.\nНикто не сказал тебе, зачем ты проснулся. В этом нет наставников, богов или проводников. Только ты, дикая земля — и чувство, что всё это уже было. Что ты здесь не впервые.",
			}
		}
		if strings.Contains(item.ItemName, "📖 Страница 4") && item.Quantity > 0 {
			pageMap[4] = struct {
				title string
				text  string
			}{
				title: "📖 Страница 4 «Без имени»",
				text:  "У тебя ничего нет. Ни дома, ни имени, ни цели. Только старая кирка, тёплый свет солнца и бескрайняя, живая земля, что будто наблюдает за каждым твоим шагом.",
			}
		}
		if strings.Contains(item.ItemName, "📖 Страница 5") && item.Quantity > 0 {
			pageMap[5] = struct {
				title string
				text  string
			}{
				title: "📖 Страница 5 «Искра перемен»",
				text:  "Но ты чувствуешь — если построить хижину, зажечь огонь, добыть первый камень… что-то изменится.\nВ тебе. В этом месте. В самой памяти мира.\nВозможно, ты не просто выживший. Возможно, ты — начало нового.",
			}
		}
		if strings.Contains(item.ItemName, "📖 Страница 6") && item.Quantity > 0 {
			pageMap[6] = struct {
				title string
				text  string
			}{
				title: "📖 Страница 6 «Наблюдающий лес»",
				text:  "Поначалу земля молчала. Ты копал, строил, охотился — и всё было, как будто в пустоте.\nНо с каждым ударом по камню, с каждым дымком над костром ты чувствовал, что что-то наблюдает. Не враждебное. Но древнее.",
			}
		}
		if strings.Contains(item.ItemName, "📖 Страница 7") && item.Quantity > 0 {
			pageMap[7] = struct {
				title string
				text  string
			}{
				title: "📖 Страница 7 «Шёпот ветра»",
				text:  "Иногда по ночам ты слышал, как шелестят листья без ветра.\nКак в костре трескается не дрова, а слова. Неслышные, шепчущие.\nЗемля словно пыталась заговорить с тобой, но ещё не решалась.",
			}
		}
		if strings.Contains(item.ItemName, "📖 Страница 8") && item.Quantity > 0 {
			pageMap[8] = struct {
				title string
				text  string
			}{
				title: "📖 Страница 8 «След древних»",
				text:  "Ты начал находить странные вещи. Камень с гладкой гранью, словно вырезанной руками.\nОбломок кости с выжженным символом. Одинокую статую, стоящую посреди леса, покрытую мхом, но не разрушенную.",
			}
		}
	}

	if len(pageMap) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "📖 У вас пока нет страниц.")
		h.sendMessage(msg)
		return
	}

	// Находим первую доступную страницу
	var firstPage int
	for i := 1; i <= 8; i++ {
		if _, exists := pageMap[i]; exists {
			firstPage = i
			break
		}
	}

	// Формируем текст сообщения
	page := pageMap[firstPage]
	text := fmt.Sprintf("%s\n\n%s\n\nСтраница %d из %d", page.title, page.text, firstPage, len(pageMap))

	// Создаем кнопки навигации
	var keyboard tgbotapi.InlineKeyboardMarkup
	var row []tgbotapi.InlineKeyboardButton

	// Кнопка "Назад" с callback_data
	prevBtn := tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", fmt.Sprintf("page_prev_%d", firstPage))
	row = append(row, prevBtn)

	// Кнопка "Дальше" с callback_data
	// Кнопка "Дальше" активна, если есть следующая страница
	nextBtn := tgbotapi.NewInlineKeyboardButtonData("Дальше ▶️", fmt.Sprintf("page_next_%d", firstPage))
	row = append(row, nextBtn)

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)

	// Отправляем сообщение с первой страницей
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	h.sendMessage(msg)

	// Проверяем прогресс квеста 6 после успешного чтения страницы
	h.checkLorePagesQuestProgressSequential(message.From.ID, message.Chat.ID, player.ID, firstPage)
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

	// Находим все страницы в инвентаре и сортируем их
	pageTexts := make(map[string]string)
	pageMap := make(map[int]string)

	// Определяем доступные страницы и их тексты
	for _, item := range inventory {
		if strings.Contains(item.ItemName, "📖 Страница 1") && item.Quantity > 0 {
			pageMap[1] = "📖 Страница 1 «Забытая тишина»"
			pageTexts["📖 Страница 1 «Забытая тишина»"] = `📖 Страница 1 «Забытая тишина»

"Мир не был уничтожен в битве. Он просто... забыл сам себя.
Годы прошли — может, столетия, может, тысячелетия. Никто не знает точно. От былых королевств остались лишь заросшие руины, поросшие мхом камни и полустёртые знаки, выгравированные на обломках."`
		}
		if strings.Contains(item.ItemName, "📖 Страница 2") && item.Quantity > 0 {
			pageMap[2] = "📖 Страница 2 «Пепел памяти»"
			pageTexts["📖 Страница 2 «Пепел памяти»"] = `📖 Страница 2 «Пепел памяти»

"Люди исчезли. Не все, возможно, но память о них — точно.
Земля забыла их шаги. Знания рассыпались, будто песок в ветре. Остались лишь сны, смутные образы, и тихий зов из глубин мира."`
		}
		if strings.Contains(item.ItemName, "📖 Страница 3") && item.Quantity > 0 {
			pageMap[3] = "📖 Страница 3 «Пробуждение»"
			pageTexts["📖 Страница 3 «Пробуждение»"] = `📖 Страница 3 «Пробуждение»

"Ты — один из тех, кто откликнулся.
Никто не сказал тебе, зачем ты проснулся. В этом нет наставников, богов или проводников. Только ты, дикая земля — и чувство, что всё это уже было. Что ты здесь не впервые."`
		}
		if strings.Contains(item.ItemName, "📖 Страница 4") && item.Quantity > 0 {
			pageMap[4] = "📖 Страница 4 «Без имени»"
			pageTexts["📖 Страница 4 «Без имени»"] = `📖 Страница 4 «Без имени»

"У тебя ничего нет. Ни дома, ни имени, ни цели. Только старая кирка, тёплый свет солнца и бескрайняя, живая земля, что будто наблюдает за каждым твоим шагом."`
		}
		if strings.Contains(item.ItemName, "📖 Страница 5") && item.Quantity > 0 {
			pageMap[5] = "📖 Страница 5 «Искра перемен»"
			pageTexts["📖 Страница 5 «Искра перемен»"] = `📖 Страница 5 «Искра перемен»

"Но ты чувствуешь — если построить хижину, зажечь огонь, добыть первый камень… что-то изменится.
В тебе. В этом месте. В самой памяти мира.
Возможно, ты не просто выживший. Возможно, ты — начало нового."`
		}
		if strings.Contains(item.ItemName, "📖 Страница 6") && item.Quantity > 0 {
			pageMap[6] = "📖 Страница 6 «Наблюдающий лес»"
			pageTexts["📖 Страница 6 «Наблюдающий лес»"] = `📖 Страница 6 «Наблюдающий лес»

"Поначалу земля молчала. Ты копал, строил, охотился — и всё было, как будто в пустоте.
Но с каждым ударом по камню, с каждым дымком над костром ты чувствовал, что что-то наблюдает. Не враждебное. Но древнее."`
		}
		if strings.Contains(item.ItemName, "📖 Страница 7") && item.Quantity > 0 {
			pageMap[7] = "📖 Страница 7 «Шёпот ветра»"
			pageTexts["📖 Страница 7 «Шёпот ветра»"] = `📖 Страница 7 «Шёпот ветра»

"Иногда по ночам ты слышал, как шелестят листья без ветра.
Как в костре трескается не дрова, а слова. Неслышные, шепчущие.
Земля словно пыталась заговорить с тобой, но ещё не решалась."`
		}
		if strings.Contains(item.ItemName, "📖 Страница 8") && item.Quantity > 0 {
			pageMap[8] = "📖 Страница 8 «След древних»"
			pageTexts["📖 Страница 8 «След древних»"] = `📖 Страница 8 «След древних»

"Ты начал находить странные вещи. Камень с гладкой гранью, словно вырезанной руками.
Обломок кости с выжженным символом. Одинокую статую, стоящую посреди леса, покрытую мхом, но не разрушенную."`
		}
	}

	// Создаем отсортированный список страниц
	var availablePages []string
	for i := 1; i <= 8; i++ {
		if page, exists := pageMap[i]; exists {
			availablePages = append(availablePages, page)
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

	// Проверяем прогресс квеста 6 после успешного чтения страницы
	h.checkLorePagesQuestProgressSequential(message.From.ID, message.Chat.ID, player.ID, 6)

	return
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

	// Проверяем прогресс квеста 6 после успешного чтения страницы
	h.checkLorePagesQuestProgressSequential(message.From.ID, message.Chat.ID, player.ID, 1)
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
	pageQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "📖 Страница 2 «Пепел памяти»")
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
	pageText := `📖 Страница 2 «Пепел памяти»

"Люди исчезли. Не все, возможно, но память о них — точно.
Земля забыла их шаги. Знания рассыпались, будто песок в ветре. Остались лишь сны, смутные образы, и тихий зов из глубин мира."`

	msg := tgbotapi.NewMessage(message.Chat.ID, pageText)
	h.sendMessage(msg)

	// Проверяем прогресс квеста 6 после успешного чтения страницы
	h.checkLorePagesQuestProgressSequential(message.From.ID, message.Chat.ID, player.ID, 2)
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
	pageQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "📖 Страница 3 «Пробуждение»")
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
	pageText := `📖 Страница 3 «Пробуждение»

"Ты — один из тех, кто откликнулся.
Никто не сказал тебе, зачем ты проснулся. В этом нет наставников, богов или проводников. Только ты, дикая земля — и чувство, что всё это уже было. Что ты здесь не впервые."`

	msg := tgbotapi.NewMessage(message.Chat.ID, pageText)
	h.sendMessage(msg)

	// Проверяем прогресс квеста 6 после успешного чтения страницы
	h.checkLorePagesQuestProgressSequential(message.From.ID, message.Chat.ID, player.ID, 3)
}

func (h *BotHandlers) handleReadPage4(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Проверяем наличие страницы 4 в инвентаре
	pageQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "📖 Страница 4 «Без имени»")
	if err != nil {
		log.Printf("Error checking page in inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	if pageQuantity == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет четвертой страницы.")
		h.sendMessage(msg)
		return
	}

	// Показываем текст четвертой страницы
	pageText := `📖 Страница 4 «Без имени»

"У тебя ничего нет. Ни дома, ни имени, ни цели. Только старая кирка, тёплый свет солнца и бескрайняя, живая земля, что будто наблюдает за каждым твоим шагом."`

	msg := tgbotapi.NewMessage(message.Chat.ID, pageText)
	h.sendMessage(msg)

	// Проверяем прогресс квеста 6 после успешного чтения страницы
	h.checkLorePagesQuestProgressSequential(message.From.ID, message.Chat.ID, player.ID, 4)
}

func (h *BotHandlers) handleReadPage5(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Проверяем наличие страницы 5 в инвентаре
	pageQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "📖 Страница 5 «Искра перемен»")
	if err != nil {
		log.Printf("Error checking page in inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	if pageQuantity == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет пятой страницы.")
		h.sendMessage(msg)
		return
	}

	// Показываем текст пятой страницы
	pageText := `📖 Страница 5 «Искра перемен»

"Но ты чувствуешь — если построить хижину, зажечь огонь, добыть первый камень… что-то изменится.
В тебе. В этом месте. В самой памяти мира.
Возможно, ты не просто выживший. Возможно, ты — начало нового."`

	msg := tgbotapi.NewMessage(message.Chat.ID, pageText)
	h.sendMessage(msg)

	// Проверяем прогресс квеста 6 после успешного чтения страницы
	h.checkLorePagesQuestProgressSequential(message.From.ID, message.Chat.ID, player.ID, 5)
}

func (h *BotHandlers) handleReadPage6(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Проверяем наличие страницы 6 в инвентаре
	pageQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "📖 Страница 6 «Наблюдающий лес»")
	if err != nil {
		log.Printf("Error checking page in inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	if pageQuantity == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет шестой страницы.")
		h.sendMessage(msg)
		return
	}

	// Показываем текст шестой страницы
	pageText := `📖 Страница 6 «Наблюдающий лес»

"Поначалу земля молчала. Ты копал, строил, охотился — и всё было, как будто в пустоте.
Но с каждым ударом по камню, с каждым дымком над костром ты чувствовал, что что-то наблюдает. Не враждебное. Но древнее."`

	msg := tgbotapi.NewMessage(message.Chat.ID, pageText)
	h.sendMessage(msg)

	// Проверяем прогресс квеста 6 после успешного чтения страницы
	h.checkLorePagesQuestProgressSequential(message.From.ID, message.Chat.ID, player.ID, 6)
}

func (h *BotHandlers) handleReadPage7(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Проверяем наличие страницы 7 в инвентаре
	pageQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "📖 Страница 7 «Шёпот ветра»")
	if err != nil {
		log.Printf("Error checking page in inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	if pageQuantity == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет седьмой страницы.")
		h.sendMessage(msg)
		return
	}

	// Показываем текст седьмой страницы
	pageText := `📖 Страница 7 «Шёпот ветра»

"Иногда по ночам ты слышал, как шелестят листья без ветра.
Как в костре трескается не дрова, а слова. Неслышные, шепчущие.
Земля словно пыталась заговорить с тобой, но ещё не решалась."`

	msg := tgbotapi.NewMessage(message.Chat.ID, pageText)
	h.sendMessage(msg)
}

func (h *BotHandlers) handleReadPage8(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Проверяем наличие страницы 8 в инвентаре
	pageQuantity, err := h.db.GetItemQuantityInInventory(player.ID, "📖 Страница 8 «След древних»")
	if err != nil {
		log.Printf("Error checking page in inventory: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	if pageQuantity == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет восьмой страницы.")
		h.sendMessage(msg)
		return
	}

	// Показываем текст восьмой страницы
	pageText := `📖 Страница 8 «След древних»

"Ты начал находить странные вещи. Камень с гладкой гранью, словно вырезанной руками.
Обломок кости с выжженным символом. Одинокую статую, стоящую посреди леса, покрытую мхом, но не разрушенную."`

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

	// Проверяем прогресс квеста 4 (сбор лесных ягод)
	if resourceName == "Лесная ягода" {
		h.checkBerryQuestProgress(userID, chatID, player.ID)
	}

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

	// Вычисляем общее время крафта
	var totalDuration int
	if itemName == "Простая хижина" {
		totalDuration = 120
	} else {
		totalDuration = quantity * 20
	}

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
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		return
	}

	if itemName == "Простая хижина" {
		// Удаляем ресурсы
		requirements := []struct {
			ItemName string
			Quantity int
		}{
			{"Береза", 20},
			{"Березовый брус", 10},
			{"Камень", 15},
			{"Лесная ягода", 10},
		}
		for _, req := range requirements {
			err := h.db.RemoveItemFromInventory(player.ID, req.ItemName, req.Quantity)
			if err != nil {
				log.Printf("Error removing item from inventory: %v", err)
			}
		}
		// Обновляем статус хижины
		err = h.db.UpdateSimpleHutBuilt(player.ID, true)
		if err != nil {
			log.Printf("Error updating simple hut status: %v", err)
		}
		// Проверяем квест 8
		quest8, err := h.db.GetPlayerQuest(player.ID, 8)
		if err != nil {
			log.Printf("Error getting quest 8: %v", err)
		} else if quest8 != nil {
			if quest8.Status == "active" {
				err = h.db.UpdateQuestProgress(player.ID, 8, 1)
				if err != nil {
					log.Printf("Error updating quest progress: %v", err)
				}
				if quest8.Progress+1 >= quest8.Target {
					err = h.db.UpdateQuestStatus(player.ID, 8, "completed")
					if err != nil {
						log.Printf("Error completing quest 8: %v", err)
					}
					err = h.db.UpdatePlayerExperience(player.ID, 10)
					if err != nil {
						log.Printf("Error updating player experience: %v", err)
					}
					h.addPage8IfNotExists(player.ID)
					questCompleteText := `🛖 Квест 8: Под крышей ВЫПОЛНЕН!
Получена награда:
🎖 10 опыта
📖 Страница 8 «След древних»`
					msg := tgbotapi.NewMessage(chatID, questCompleteText)
					h.sendMessage(msg)
				}
			} else if quest8.Status == "completed" {
				// Если квест уже завершён, но страницы нет — добавить её
				h.addPage8IfNotExists(player.ID)
			}
		}
		// Удаляем сообщение о крафте
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
		h.requestAPI(deleteMsg)
		completeText := `✅ Строительство "Простая хижина" завершено!
Теперь у вас есть укрытие от непогоды.`
		msg := tgbotapi.NewMessage(chatID, completeText)
		h.sendMessage(msg)
		delete(h.craftingTimers, userID)
		return
	}

	// Обычный крафт
	if err := h.db.AddItemToInventory(player.ID, itemName, quantity); err != nil {
		log.Printf("Error adding crafted items to inventory: %v", err)
	}
	if err := h.db.UpdatePlayerSatiety(player.ID, -quantity); err != nil {
		log.Printf("Error updating player satiety: %v", err)
	}
	updatedPlayer, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting updated player: %v", err)
		updatedPlayer = player
	}
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)
	resultText := fmt.Sprintf(`✅ Создание завершено!
Получено: "%s" x%d
Сытость: %d/100`, itemName, quantity, updatedPlayer.Satiety)
	msg := tgbotapi.NewMessage(chatID, resultText)
	h.sendMessage(msg)
	if itemName == "Березовый брус" {
		h.checkBirchPlankQuestProgress(userID, chatID, player.ID, quantity)
	}
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
		err = h.db.AddItemToInventory(playerID, "📖 Страница 3 «Пробуждение»", 1)
		if err != nil {
			log.Printf("Error adding quest item to inventory: %v", err)
		}

		// Отправляем сообщение о выполнении квеста
		questCompleteText := `🪚 Квест 3: Руки мастера ВЫПОЛНЕН!
Получена награда:
🎖 10 опыта
📖 Страница 3 «Пробуждение»`

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
		err = h.db.AddItemToInventory(playerID, "📖 Страница 2 «Пепел памяти»", 1)
		if err != nil {
			log.Printf("Error adding quest item to inventory: %v", err)
		}

		// Отправляем сообщение о выполнении квеста
		questCompleteText := `⛏ Квест 2: Вглубь ВЫПОЛНЕН!
Получена награда:
🎖 10 опыта
📖 Страница 2 «Пепел памяти»`

		msg := tgbotapi.NewMessage(chatID, questCompleteText)
		h.sendMessage(msg)
	}
}

func (h *BotHandlers) checkBerryQuestProgress(userID int64, chatID int64, playerID int) {
	// Проверяем активный квест 4 (сбор лесных ягод)
	quest, err := h.db.GetPlayerQuest(playerID, 4)
	if err != nil {
		log.Printf("Error getting quest 4: %v", err)
		return
	}

	// Если квест не активен, ничего не делаем
	if quest == nil || quest.Status != "active" {
		return
	}

	// Увеличиваем прогресс квеста
	newProgress := quest.Progress + 1
	err = h.db.UpdateQuestProgress(playerID, 4, newProgress)
	if err != nil {
		log.Printf("Error updating quest 4 progress: %v", err)
		return
	}

	// Проверяем, выполнен ли квест
	if newProgress >= quest.Target {
		// Квест выполнен!
		err = h.db.UpdateQuestStatus(playerID, 4, "completed")
		if err != nil {
			log.Printf("Error completing quest 4: %v", err)
			return
		}

		// Добавляем награды
		// 10 опыта игроку
		err = h.db.UpdatePlayerExperience(playerID, 10)
		if err != nil {
			log.Printf("Error updating player experience: %v", err)
		}

		// Добавляем страницу в инвентарь
		err = h.db.AddItemToInventory(playerID, "📖 Страница 4 «Без имени»", 1)
		if err != nil {
			log.Printf("Error adding quest item to inventory: %v", err)
		}

		// Отправляем сообщение о выполнении квеста
		questCompleteText := `🍇 Квест 4: Дар леса ВЫПОЛНЕН!
Получена награда:
🎖 10 опыта
📖 Страница 4 «Без имени»`

		msg := tgbotapi.NewMessage(chatID, questCompleteText)
		h.sendMessage(msg)
	}
}

func (h *BotHandlers) checkHuntingQuestProgress(userID int64, chatID int64, playerID int) {
	// Проверяем активный квест 5 (первая охота)
	quest, err := h.db.GetPlayerQuest(playerID, 5)
	if err != nil {
		log.Printf("Error getting quest 5: %v", err)
		return
	}

	// Если квест не активен, ничего не делаем
	if quest == nil || quest.Status != "active" {
		return
	}

	// Увеличиваем прогресс квеста
	newProgress := quest.Progress + 1
	err = h.db.UpdateQuestProgress(playerID, 5, newProgress)
	if err != nil {
		log.Printf("Error updating quest 5 progress: %v", err)
		return
	}

	// Проверяем, выполнен ли квест
	if newProgress >= 1 {
		// Квест выполнен!
		err = h.db.UpdateQuestStatus(playerID, 5, "completed")
		if err != nil {
			log.Printf("Error completing quest 5: %v", err)
			return
		}

		// Добавляем награды
		// 10 опыта игроку
		err = h.db.UpdatePlayerExperience(playerID, 10)
		if err != nil {
			log.Printf("Error updating player experience: %v", err)
		}

		// Добавляем страницу 5 в инвентарь
		err = h.db.AddItemToInventory(playerID, "📖 Страница 5 «Искра перемен»", 1)
		if err != nil {
			log.Printf("Error adding quest item to inventory: %v", err)
		}

		// Отправляем сообщение о выполнении квеста
		questCompleteText := `🎯 Квест 5: Звериный взгляд ВЫПОЛНЕН!
Получена награда:
🎖 10 опыта
📖 Страница 5 «Искра перемен»`
		msg := tgbotapi.NewMessage(chatID, questCompleteText)
		h.sendMessage(msg)
	}
}

func (h *BotHandlers) checkLorePagesQuestProgressSequential(userID int64, chatID int64, playerID int, pageNumber int) {
	// Проверяем активный квест 6 (чтение страниц)
	quest, err := h.db.GetPlayerQuest(playerID, 6)
	if err != nil {
		log.Printf("Error getting quest 6: %v", err)
		return
	}

	if quest == nil || quest.Status != "active" {
		return
	}

	// Прогресс квеста — номер последней прочитанной страницы (0..5)
	if quest.Progress+1 == pageNumber {
		newProgress := quest.Progress + 1
		err = h.db.UpdateQuestProgress(playerID, 6, newProgress)
		if err != nil {
			log.Printf("Error updating quest 6 progress: %v", err)
			return
		}

		if newProgress == 5 {
			// Квест выполнен!
			err = h.db.UpdateQuestStatus(playerID, 6, "completed")
			if err != nil {
				log.Printf("Error completing quest 6: %v", err)
				return
			}

			// Выдаем награду
			err = h.db.AddItemToInventory(playerID, "📖 Страница 6 «Наблюдающий лес»", 1)
			if err != nil {
				log.Printf("Error adding quest 6 reward: %v", err)
				return
			}

			// Добавляем опыт
			err = h.db.UpdatePlayerExperience(playerID, 10)
			if err != nil {
				log.Printf("Error adding quest 6 experience: %v", err)
				return
			}

			// Отправляем сообщение о выполнении
			msg := tgbotapi.NewMessage(chatID, `🎉 Поздравляем! Ты выполнил квест "Живое Хранилище"!
Награда:
🎖 +10 опыта
📖 Страница 6 «Наблюдающий лес»`)
			h.sendMessage(msg)
		}
	}
}

func (h *BotHandlers) finishSimpleHutBuilding(userID int64, chatID int64, messageID int) {
	totalSeconds := 120
	steps := 12 // обновлять каждые 10 секунд

	// Сразу показываем прогресс-бар 0%
	bar := h.createProgressBar(0, 100)
	progressText := fmt.Sprintf("Идет строительство объекта \"Простая хижина\". Время строительства 120 секунд.\n\n%s 0%%", bar)
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, progressText)
	h.requestAPI(editMsg)

	for i := 1; i <= steps; i++ {
		time.Sleep(time.Duration(totalSeconds/steps) * time.Second)
		progress := i * 100 / steps
		bar := h.createProgressBar(progress, 100)
		progressText := fmt.Sprintf("Идет строительство объекта \"Простая хижина\". Время строительства 120 секунд.\n\n%s %d%%", bar, progress)
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, progressText)
		h.requestAPI(editMsg)
	}

	player, err := h.db.GetPlayer(userID)
	if err != nil {
		return
	}
	// Снимаем ресурсы
	h.db.ConsumeItem(player.ID, "Береза", 20)
	h.db.ConsumeItem(player.ID, "Березовый брус", 10)
	h.db.ConsumeItem(player.ID, "Камень", 15)
	h.db.ConsumeItem(player.ID, "Лесная ягода", 10)
	// Отнимаем 5 сытости
	h.db.UpdatePlayerSatiety(player.ID, -5)
	h.db.UpdateSimpleHutBuilt(player.ID, true)
	updatedPlayer, _ := h.db.GetPlayer(userID)
	// Удаляем сообщение о прогрессе
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	h.requestAPI(deleteMsg)
	// Выводим финальное сообщение
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Объект \"Простая хижина\" успешно построен!\nСытость %d/100", updatedPlayer.Satiety))
	h.sendMessage(msg)
}

// Добавить вспомогательную функцию:
func (h *BotHandlers) addPage8IfNotExists(playerID int) {
	qty, err := h.db.GetItemQuantityInInventory(playerID, "📖 Страница 8 «След древних»")
	if err != nil {
		log.Printf("Error checking for page 8: %v", err)
		return
	}
	if qty == 0 {
		err = h.db.AddItemToInventory(playerID, "📖 Страница 8 «След древних»", 1)
		if err != nil {
			log.Printf("Error adding page 8 to inventory: %v", err)
		}
	}
}

func (h *BotHandlers) handleOpenHut(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Проверяем, построена ли хижина
	if !player.SimpleHutBuilt {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У тебя еще нет простой хижины. Построй ее в разделе Постройки.")
		h.sendMessage(msg)
		return
	}

	// Отправляем сообщение о входе в хижину
	hutText := `🛖 Ты заходишь в свою простую хижину.

Деревянные стены скрипят на ветру, но внутри — тепло и спокойно.  
Костёр ещё тлеет в углу, а рядом лежит твоя нехитрая утварь.  
Это твоё первое убежище в этом мире.

Здесь ты можешь:

😴 Отдохнуть — восстановить 50 ед. сытости за 30 минут отдыха  /rest`

	msg := tgbotapi.NewMessage(message.Chat.ID, hutText)
	h.sendMessage(msg)
}

func (h *BotHandlers) handleRest(message *tgbotapi.Message) {
	userID := message.From.ID

	// Получаем игрока
	player, err := h.db.GetPlayer(userID)
	if err != nil {
		log.Printf("Error getting player: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		h.sendMessage(msg)
		return
	}

	// Проверяем, не отдыхает ли уже игрок
	if _, exists := h.restingTimers[userID]; exists {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Ты уже отдыхаешь. Дождись окончания отдыха.")
		h.sendMessage(msg)
		return
	}

	// Отправляем начальное сообщение с прогресс-баром
	bar := h.createProgressBar(0, 100)
	progressText := fmt.Sprintf("Отдых начался. Время отдыха 30 минут.\n\n%s 0%%", bar)
	msg := tgbotapi.NewMessage(message.Chat.ID, progressText)
	progressMsg, _ := h.sendMessageWithResponse(msg)

	// Запускаем таймер отдыха
	totalSeconds := 1800 // 30 минут
	steps := 30          // обновлять каждую минуту
	messageID := progressMsg.MessageID

	// Сохраняем ID сообщения в таймере
	h.restingTimers[userID] = time.NewTimer(time.Duration(totalSeconds) * time.Second)

	go func() {
		for i := 1; i <= steps; i++ {
			time.Sleep(time.Duration(totalSeconds/steps) * time.Second)
			progress := i * 100 / steps
			bar := h.createProgressBar(progress, 100)
			progressText := fmt.Sprintf("Отдых начался. Время отдыха 30 минут.\n\n%s %d%%", bar, progress)
			editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, messageID, progressText)
			h.requestAPI(editMsg)
		}

		// По истечении времени
		<-h.restingTimers[userID].C
		delete(h.restingTimers, userID)

		// Удаляем сообщение с прогресс-баром
		deleteMsg := tgbotapi.NewDeleteMessage(message.Chat.ID, messageID)
		h.requestAPI(deleteMsg)

		// Восстанавливаем сытость
		err = h.db.UpdatePlayerSatiety(player.ID, 50)
		if err != nil {
			log.Printf("Error updating player satiety: %v", err)
			return
		}

		// Получаем обновленные данные игрока
		updatedPlayer, _ := h.db.GetPlayer(userID)

		// Отправляем сообщение о завершении отдыха
		resultText := fmt.Sprintf("Отдых завершен. Восстановлено 50 ед. сытости.\nСытость %d/100", updatedPlayer.Satiety)
		resultMsg := tgbotapi.NewMessage(message.Chat.ID, resultText)
		h.sendMessage(resultMsg)
	}()
}
