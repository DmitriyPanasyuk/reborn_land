package database

import (
	"database/sql"
	"fmt"
	"log"
	"reborn_land/models"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

// Item представляет предмет в инвентаре игрока
type Item struct {
	ItemName string
	ItemType string
	Quantity int
}

func New(databaseURL string) (*DB, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	db := &DB{conn: conn}

	// Создаем таблицы при инициализации
	if err := db.createTables(); err != nil {
		return nil, err
	}

	// Добавляем начальные предметы
	if err := db.seedItems(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS players (
			id SERIAL PRIMARY KEY,
			telegram_id BIGINT UNIQUE NOT NULL,
			name VARCHAR(30) NOT NULL,
			level INTEGER DEFAULT 1,
			experience INTEGER DEFAULT 0,
			satiety INTEGER DEFAULT 100,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			simple_hut_built BOOLEAN DEFAULT false
		)`,
		`CREATE TABLE IF NOT EXISTS items (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			type VARCHAR(50) NOT NULL,
			durability_max INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS inventory (
			id SERIAL PRIMARY KEY,
			player_id INTEGER REFERENCES players(id),
			item_id INTEGER REFERENCES items(id),
			quantity INTEGER DEFAULT 1,
			durability INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS mines (
			id SERIAL PRIMARY KEY,
			player_id INTEGER REFERENCES players(id),
			level INTEGER DEFAULT 1,
			experience INTEGER DEFAULT 0,
			last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_exhausted BOOLEAN DEFAULT false
		)`,
		`CREATE TABLE IF NOT EXISTS forests (
			id SERIAL PRIMARY KEY,
			player_id INTEGER REFERENCES players(id),
			level INTEGER DEFAULT 1,
			experience INTEGER DEFAULT 0,
			last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_exhausted BOOLEAN DEFAULT false
		)`,
		`CREATE TABLE IF NOT EXISTS gathering (
			id SERIAL PRIMARY KEY,
			player_id INTEGER REFERENCES players(id),
			level INTEGER DEFAULT 1,
			experience INTEGER DEFAULT 0,
			last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_exhausted BOOLEAN DEFAULT false
		)`,
		`CREATE TABLE IF NOT EXISTS quests (
			id SERIAL PRIMARY KEY,
			player_id INTEGER REFERENCES players(id),
			quest_id INTEGER NOT NULL,
			status VARCHAR(20) DEFAULT 'available',
			progress INTEGER DEFAULT 0,
			target INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP NULL
		)`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) seedItems() error {
	// Список всех необходимых предметов
	items := []struct {
		name          string
		itemType      string
		durabilityMax int
	}{
		{"Простой топор", "tool", 100},
		{"Простой нож", "tool", 100},
		{"Лесная ягода", "food", 0},
		{"Простая кирка", "tool", 100},
		{"Простой лук", "tool", 100},
		{"Стрелы", "ammunition", 0},
		{"Простая удочка", "tool", 100},
		{"Березовый брус", "material", 0},
		{"Камень", "material", 0},
		{"Уголь", "material", 0},
		{"Сухожилие", "material", 0},
		{"Перо", "material", 0},
		{"Кость", "material", 0},
		{"Веревка", "material", 0},
		{"Крючок", "material", 0},
		{"Береза", "material", 0},
		{"📖 Страница 1 «Забытая тишина»", "quest_item", 0},
		{"📖 Страница 2 «Пепел памяти»", "quest_item", 0},
		{"📖 Страница 3 «Пробуждение»", "quest_item", 0},
		{"📖 Страница 4 «Без имени»", "quest_item", 0},
		{"📖 Страница 5 «Искра перемен»", "quest_item", 0},
		{"📖 Страница 6 «Наблюдающий лес»", "quest_item", 0},
	}

	// Добавляем каждый предмет, если его нет
	for _, item := range items {
		var exists bool
		err := db.conn.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM items WHERE name = $1)",
			item.name,
		).Scan(&exists)

		if err != nil {
			return err
		}

		if !exists {
			_, err := db.conn.Exec(
				"INSERT INTO items (name, type, durability_max) VALUES ($1, $2, $3)",
				item.name, item.itemType, item.durabilityMax,
			)
			if err != nil {
				return err
			}
			log.Printf("Added item: %s", item.name)
		}
	}

	return nil
}

func (db *DB) PlayerExists(telegramID int64) (bool, error) {
	var exists bool
	err := db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE telegram_id = $1)", telegramID).Scan(&exists)
	return exists, err
}

func (db *DB) GetPlayer(telegramID int64) (*models.Player, error) {
	var player models.Player
	err := db.conn.QueryRow(`
		SELECT id, telegram_id, name, level, experience, satiety, created_at, simple_hut_built
		FROM players WHERE telegram_id = $1`,
		telegramID,
	).Scan(&player.ID, &player.TelegramID, &player.Name, &player.Level, &player.Experience, &player.Satiety, &player.CreatedAt, &player.SimpleHutBuilt)

	if err != nil {
		return nil, err
	}

	return &player, nil
}

func (db *DB) GetPlayerInventory(playerID int) ([]models.InventoryItem, error) {
	rows, err := db.conn.Query(`
		SELECT i.id, i.player_id, i.item_id, it.name, i.quantity, i.durability, it.type
		FROM inventory i
		JOIN items it ON i.item_id = it.id
		WHERE i.player_id = $1 AND i.quantity > 0
		ORDER BY it.type, it.name`,
		playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.InventoryItem
	for rows.Next() {
		var item models.InventoryItem
		err := rows.Scan(&item.ID, &item.PlayerID, &item.ItemID, &item.ItemName, &item.Quantity, &item.Durability, &item.Type)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (db *DB) GetItemQuantityInInventory(playerID int, itemName string) (int, error) {
	var quantity int
	err := db.conn.QueryRow(`
		SELECT COALESCE(SUM(i.quantity), 0)
		FROM inventory i
		JOIN items it ON i.item_id = it.id
		WHERE i.player_id = $1 AND it.name = $2`,
		playerID, itemName,
	).Scan(&quantity)

	return quantity, err
}

func (db *DB) GetRecipeRequirements(itemName string) ([]models.RecipeIngredient, error) {
	// Возвращаем статические рецепты для демонстрации
	// В будущем это можно вынести в отдельную таблицу
	recipes := map[string][]models.RecipeIngredient{
		"Простой топор": {
			{ItemName: "Березовый брус", Quantity: 1},
			{ItemName: "Камень", Quantity: 1},
		},
		"Простая кирка": {
			{ItemName: "Березовый брус", Quantity: 1},
			{ItemName: "Камень", Quantity: 1},
		},
		"Простой лук": {
			{ItemName: "Березовый брус", Quantity: 1},
			{ItemName: "Сухожилие", Quantity: 1},
		},
		"Стрелы": {
			{ItemName: "Березовый брус", Quantity: 1},
			{ItemName: "Камень", Quantity: 1},
			{ItemName: "Перо", Quantity: 1},
		},
		"Простой нож": {
			{ItemName: "Березовый брус", Quantity: 1},
			{ItemName: "Кость", Quantity: 1},
		},
		"Простая удочка": {
			{ItemName: "Березовый брус", Quantity: 1},
			{ItemName: "Веревка", Quantity: 1},
			{ItemName: "Крючок", Quantity: 1},
		},
		"Березовый брус": {
			{ItemName: "Береза", Quantity: 2},
		},
	}

	if recipe, exists := recipes[itemName]; exists {
		return recipe, nil
	}

	return nil, fmt.Errorf("recipe not found for item: %s", itemName)
}

func (db *DB) GetOrCreateMine(playerID int) (*models.Mine, error) {
	var mine models.Mine

	// Пытаемся найти существующую шахту
	err := db.conn.QueryRow(`
		SELECT id, player_id, level, experience, last_used, is_exhausted 
		FROM mines WHERE player_id = $1`, playerID,
	).Scan(&mine.ID, &mine.PlayerID, &mine.Level, &mine.Experience, &mine.LastUsed, &mine.IsExhausted)

	if err == sql.ErrNoRows {
		// Создаем новую шахту
		err = db.conn.QueryRow(`
			INSERT INTO mines (player_id, level, experience, is_exhausted) 
			VALUES ($1, 1, 0, false) 
			RETURNING id, player_id, level, experience, last_used, is_exhausted`,
			playerID,
		).Scan(&mine.ID, &mine.PlayerID, &mine.Level, &mine.Experience, &mine.LastUsed, &mine.IsExhausted)
	}

	return &mine, err
}

func (db *DB) UpdateMineExperience(playerID int, expGained int) (bool, int, error) {
	// Получаем текущий уровень и опыт
	var currentLevel, currentExp int
	err := db.conn.QueryRow(`
		SELECT level, experience 
		FROM mines WHERE player_id = $1`,
		playerID,
	).Scan(&currentLevel, &currentExp)
	if err != nil {
		return false, 0, err
	}

	// Вычисляем новый опыт
	newExp := currentExp + expGained

	// Вычисляем новый уровень
	newLevel := currentLevel
	for newExp >= newLevel*100 {
		newLevel++
	}

	// Обновляем данные в базе
	_, err = db.conn.Exec(`
		UPDATE mines 
		SET experience = $1, level = $2
		WHERE player_id = $3`,
		newExp, newLevel, playerID,
	)
	if err != nil {
		return false, 0, err
	}

	// Возвращаем информацию о повышении уровня
	levelUp := newLevel > currentLevel
	return levelUp, newLevel, nil
}

func (db *DB) SetMineExhausted(playerID int, exhausted bool) error {
	_, err := db.conn.Exec(`
		UPDATE mines 
		SET is_exhausted = $1, last_used = CURRENT_TIMESTAMP
		WHERE player_id = $2`,
		exhausted, playerID,
	)
	return err
}

func (db *DB) UpdateItemDurability(playerID int, itemName string, durabilityLoss int) error {
	_, err := db.conn.Exec(`
		UPDATE inventory 
		SET durability = durability - $1
		FROM items 
		WHERE inventory.item_id = items.id 
		AND inventory.player_id = $2 
		AND items.name = $3 
		AND inventory.durability > 0`,
		durabilityLoss, playerID, itemName,
	)
	return err
}

func (db *DB) AddItemToInventory(playerID int, itemName string, quantity int) error {
	// Сначала получаем ID предмета
	var itemID int
	err := db.conn.QueryRow("SELECT id FROM items WHERE name = $1", itemName).Scan(&itemID)
	if err != nil {
		return err
	}

	// Проверяем, есть ли уже этот предмет в инвентаре
	var existingQuantity int
	err = db.conn.QueryRow(`
		SELECT quantity FROM inventory 
		WHERE player_id = $1 AND item_id = $2`,
		playerID, itemID,
	).Scan(&existingQuantity)

	if err == sql.ErrNoRows {
		// Добавляем новый предмет
		_, err = db.conn.Exec(`
			INSERT INTO inventory (player_id, item_id, quantity, durability) 
			VALUES ($1, $2, $3, 0)`,
			playerID, itemID, quantity,
		)
	} else if err == nil {
		// Увеличиваем количество существующего предмета
		_, err = db.conn.Exec(`
			UPDATE inventory 
			SET quantity = quantity + $1
			WHERE player_id = $2 AND item_id = $3`,
			quantity, playerID, itemID,
		)
	}

	return err
}

func (db *DB) AddItemToInventoryWithDurability(playerID int, itemName string, quantity int, durability int) error {
	// Сначала получаем ID предмета
	var itemID int
	err := db.conn.QueryRow("SELECT id FROM items WHERE name = $1", itemName).Scan(&itemID)
	if err != nil {
		return err
	}

	// Проверяем, есть ли уже этот предмет в инвентаре
	var existingQuantity int
	err = db.conn.QueryRow(`
		SELECT quantity FROM inventory 
		WHERE player_id = $1 AND item_id = $2`,
		playerID, itemID,
	).Scan(&existingQuantity)

	if err == sql.ErrNoRows {
		// Добавляем новый предмет с указанной прочностью
		_, err = db.conn.Exec(`
			INSERT INTO inventory (player_id, item_id, quantity, durability) 
			VALUES ($1, $2, $3, $4)`,
			playerID, itemID, quantity, durability,
		)
	} else if err == nil {
		// Увеличиваем количество существующего предмета
		_, err = db.conn.Exec(`
			UPDATE inventory 
			SET quantity = quantity + $1
			WHERE player_id = $2 AND item_id = $3`,
			quantity, playerID, itemID,
		)
	}

	return err
}

func (db *DB) UpdatePlayerSatiety(playerID int, satietyChange int) error {
	_, err := db.conn.Exec(`
		UPDATE players 
		SET satiety = LEAST(GREATEST(satiety + $1, 0), 100)
		WHERE id = $2`,
		satietyChange, playerID,
	)
	return err
}

func (db *DB) ConsumeItem(playerID int, itemName string, quantity int) error {
	// Получаем ID предмета
	var itemID int
	err := db.conn.QueryRow("SELECT id FROM items WHERE name = $1", itemName).Scan(&itemID)
	if err != nil {
		return err
	}

	// Уменьшаем количество предмета в инвентаре
	result, err := db.conn.Exec(`
		UPDATE inventory 
		SET quantity = quantity - $1
		WHERE player_id = $2 AND item_id = $3 AND quantity >= $1`,
		quantity, playerID, itemID,
	)
	if err != nil {
		return err
	}

	// Проверяем, был ли обновлен хотя бы один ряд
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("insufficient quantity of item %s", itemName)
	}

	// Удаляем предметы с количеством 0
	_, err = db.conn.Exec(`
		DELETE FROM inventory 
		WHERE player_id = $1 AND item_id = $2 AND quantity <= 0`,
		playerID, itemID,
	)

	return err
}

func (db *DB) HasToolInInventory(playerID int, toolName string) (bool, int, error) {
	var durability int
	err := db.conn.QueryRow(`
		SELECT i.durability
		FROM inventory i
		JOIN items it ON i.item_id = it.id
		WHERE i.player_id = $1 AND it.name = $2 AND i.quantity > 0 AND i.durability > 0`,
		playerID, toolName,
	).Scan(&durability)

	if err == sql.ErrNoRows {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	return true, durability, nil
}

func (db *DB) ExhaustMine(playerID int64) error {
	_, err := db.conn.Exec(`
		UPDATE mines 
		SET is_exhausted = true, last_used = CURRENT_TIMESTAMP
		WHERE player_id = $1`,
		playerID,
	)
	return err
}

func (db *DB) GetOrCreateForest(playerID int) (*models.Forest, error) {
	var forest models.Forest

	// Пытаемся найти существующий лес
	err := db.conn.QueryRow(`
		SELECT id, player_id, level, experience, last_used, is_exhausted 
		FROM forests WHERE player_id = $1`, playerID,
	).Scan(&forest.ID, &forest.PlayerID, &forest.Level, &forest.Experience, &forest.LastUsed, &forest.IsExhausted)

	if err == sql.ErrNoRows {
		// Создаем новый лес
		err = db.conn.QueryRow(`
			INSERT INTO forests (player_id, level, experience, is_exhausted) 
			VALUES ($1, 1, 0, false) 
			RETURNING id, player_id, level, experience, last_used, is_exhausted`,
			playerID,
		).Scan(&forest.ID, &forest.PlayerID, &forest.Level, &forest.Experience, &forest.LastUsed, &forest.IsExhausted)
	}

	return &forest, err
}

func (db *DB) UpdateForestExperience(playerID int, expGained int) (bool, int, error) {
	// Получаем текущий уровень и опыт
	var currentLevel, currentExp int
	err := db.conn.QueryRow(`
		SELECT level, experience 
		FROM forests WHERE player_id = $1`,
		playerID,
	).Scan(&currentLevel, &currentExp)
	if err != nil {
		return false, 0, err
	}

	// Вычисляем новый опыт
	newExp := currentExp + expGained

	// Вычисляем новый уровень
	newLevel := currentLevel
	for newExp >= newLevel*100 {
		newLevel++
	}

	// Обновляем данные в базе
	_, err = db.conn.Exec(`
		UPDATE forests 
		SET experience = $1, level = $2
		WHERE player_id = $3`,
		newExp, newLevel, playerID,
	)
	if err != nil {
		return false, 0, err
	}

	// Возвращаем информацию о повышении уровня
	levelUp := newLevel > currentLevel
	return levelUp, newLevel, nil
}

func (db *DB) SetForestExhausted(playerID int, exhausted bool) error {
	_, err := db.conn.Exec(`
		UPDATE forests 
		SET is_exhausted = $1, last_used = CURRENT_TIMESTAMP
		WHERE player_id = $2`,
		exhausted, playerID,
	)
	return err
}

func (db *DB) ExhaustForest(playerID int64) error {
	_, err := db.conn.Exec(`
		UPDATE forests 
		SET is_exhausted = true, last_used = CURRENT_TIMESTAMP
		WHERE player_id = $1`,
		playerID,
	)
	return err
}

func (db *DB) GetOrCreateGathering(playerID int) (*models.Gathering, error) {
	var gathering models.Gathering

	// Пытаемся найти существующий сбор
	err := db.conn.QueryRow(`
		SELECT id, player_id, level, experience, last_used, is_exhausted 
		FROM gathering WHERE player_id = $1`, playerID,
	).Scan(&gathering.ID, &gathering.PlayerID, &gathering.Level, &gathering.Experience, &gathering.LastUsed, &gathering.IsExhausted)

	if err == sql.ErrNoRows {
		// Создаем новый сбор
		err = db.conn.QueryRow(`
			INSERT INTO gathering (player_id, level, experience, is_exhausted) 
			VALUES ($1, 1, 0, false) 
			RETURNING id, player_id, level, experience, last_used, is_exhausted`,
			playerID,
		).Scan(&gathering.ID, &gathering.PlayerID, &gathering.Level, &gathering.Experience, &gathering.LastUsed, &gathering.IsExhausted)
	}

	return &gathering, err
}

func (db *DB) UpdateGatheringExperience(playerID int, expGained int) (bool, int, error) {
	// Получаем текущий уровень и опыт
	var currentLevel, currentExp int
	err := db.conn.QueryRow(`
		SELECT level, experience 
		FROM gathering WHERE player_id = $1`,
		playerID,
	).Scan(&currentLevel, &currentExp)
	if err != nil {
		return false, 0, err
	}

	// Вычисляем новый опыт
	newExp := currentExp + expGained

	// Вычисляем новый уровень (для сбора: 1 уровень = 100 опыта, 2 уровень = 200 опыта и т.д.)
	newLevel := (newExp / 100) + 1

	// Обновляем данные в базе
	_, err = db.conn.Exec(`
		UPDATE gathering 
		SET experience = $1, level = $2
		WHERE player_id = $3`,
		newExp, newLevel, playerID,
	)
	if err != nil {
		return false, 0, err
	}

	// Возвращаем информацию о повышении уровня
	levelUp := newLevel > currentLevel
	return levelUp, newLevel, nil
}

func (db *DB) SetGatheringExhausted(playerID int, exhausted bool) error {
	_, err := db.conn.Exec(`
		UPDATE gathering 
		SET is_exhausted = $1, last_used = CURRENT_TIMESTAMP
		WHERE player_id = $2`,
		exhausted, playerID,
	)
	return err
}

func (db *DB) ExhaustGathering(playerID int64) error {
	_, err := db.conn.Exec(`
		UPDATE gathering 
		SET is_exhausted = true, last_used = CURRENT_TIMESTAMP
		WHERE player_id = $1`,
		playerID,
	)
	return err
}

// Функции для работы с квестами
func (db *DB) GetPlayerQuest(playerID int, questID int) (*models.Quest, error) {
	var quest models.Quest
	err := db.conn.QueryRow(`
		SELECT id, player_id, quest_id, status, progress, target, created_at, completed_at
		FROM quests 
		WHERE player_id = $1 AND quest_id = $2`,
		playerID, questID,
	).Scan(&quest.ID, &quest.PlayerID, &quest.QuestID, &quest.Status, &quest.Progress, &quest.Target, &quest.CreatedAt, &quest.CompletedAt)

	if err == sql.ErrNoRows {
		return nil, nil // Квест не найден
	}
	if err != nil {
		return nil, err
	}

	return &quest, nil
}

func (db *DB) CreateQuest(playerID int, questID int, target int) error {
	_, err := db.conn.Exec(`
		INSERT INTO quests (player_id, quest_id, status, progress, target)
		VALUES ($1, $2, 'available', 0, $3)`,
		playerID, questID, target,
	)
	return err
}

func (db *DB) UpdateQuestStatus(playerID int, questID int, status string) error {
	if status == "completed" {
		// Если квест завершается, обновляем статус и время завершения
		_, err := db.conn.Exec(`
			UPDATE quests 
			SET status = $3, completed_at = CURRENT_TIMESTAMP
			WHERE player_id = $1 AND quest_id = $2`,
			playerID, questID, status,
		)
		return err
	} else {
		// Если квест не завершается, обновляем только статус
		_, err := db.conn.Exec(`
			UPDATE quests 
			SET status = $3
			WHERE player_id = $1 AND quest_id = $2`,
			playerID, questID, status,
		)
		return err
	}
}

func (db *DB) UpdateQuestProgress(playerID int, questID int, progress int) error {
	_, err := db.conn.Exec(`
		UPDATE quests 
		SET progress = $3
		WHERE player_id = $1 AND quest_id = $2`,
		playerID, questID, progress,
	)
	return err
}

func (db *DB) UpdatePlayerExperience(playerID int, expGained int) error {
	_, err := db.conn.Exec(`
		UPDATE players 
		SET experience = experience + $2
		WHERE id = $1`,
		playerID, expGained,
	)
	return err
}

func (db *DB) GetOrCreateHunting(playerID int) (*models.Hunting, error) {
	var hunting models.Hunting

	// Пытаемся найти существующую охоту
	err := db.conn.QueryRow(`
		SELECT id, player_id, level, experience, last_used, is_exhausted 
		FROM hunting WHERE player_id = $1`, playerID,
	).Scan(&hunting.ID, &hunting.PlayerID, &hunting.Level, &hunting.Experience, &hunting.LastUsed, &hunting.IsExhausted)

	if err == sql.ErrNoRows {
		// Создаем новую охоту
		err = db.conn.QueryRow(`
			INSERT INTO hunting (player_id, level, experience, is_exhausted) 
			VALUES ($1, 1, 0, false) 
			RETURNING id, player_id, level, experience, last_used, is_exhausted`,
			playerID,
		).Scan(&hunting.ID, &hunting.PlayerID, &hunting.Level, &hunting.Experience, &hunting.LastUsed, &hunting.IsExhausted)
	}

	return &hunting, err
}

func (db *DB) UpdateHuntingExperience(playerID int, expGained int) (bool, int, error) {
	// Получаем текущий уровень и опыт
	var currentLevel, currentExp int
	err := db.conn.QueryRow(`
		SELECT level, experience 
		FROM hunting WHERE player_id = $1`,
		playerID,
	).Scan(&currentLevel, &currentExp)
	if err != nil {
		return false, 0, err
	}

	// Вычисляем новый опыт
	newExp := currentExp + expGained

	// Вычисляем новый уровень
	newLevel := currentLevel
	for newExp >= newLevel*100 {
		newLevel++
	}

	// Обновляем данные в базе
	_, err = db.conn.Exec(`
		UPDATE hunting 
		SET experience = $1, level = $2
		WHERE player_id = $3`,
		newExp, newLevel, playerID,
	)
	if err != nil {
		return false, 0, err
	}

	// Возвращаем информацию о повышении уровня
	levelUp := newLevel > currentLevel
	return levelUp, newLevel, nil
}

func (db *DB) SetHuntingExhausted(playerID int, exhausted bool) error {
	_, err := db.conn.Exec(`
		UPDATE hunting 
		SET is_exhausted = $1, last_used = CURRENT_TIMESTAMP
		WHERE player_id = $2`,
		exhausted, playerID,
	)
	return err
}

func (db *DB) ExhaustHunting(playerID int64) error {
	_, err := db.conn.Exec(`
		UPDATE hunting 
		SET is_exhausted = true, last_used = CURRENT_TIMESTAMP
		WHERE player_id = $1`,
		playerID,
	)
	return err
}

func (db *DB) RemoveItemFromInventory(playerID int, itemName string, quantity int) error {
	// Обновляем количество через JOIN с таблицей items
	_, err := db.conn.Exec(`
		UPDATE inventory 
		SET quantity = quantity - $3
		FROM items 
		WHERE inventory.item_id = items.id 
		AND inventory.player_id = $1 
		AND items.name = $2 
		AND inventory.quantity >= $3`,
		playerID, itemName, quantity,
	)
	if err != nil {
		return err
	}

	// Удаляем записи с нулевым количеством
	_, err = db.conn.Exec(`
		DELETE FROM inventory 
		USING items 
		WHERE inventory.item_id = items.id 
		AND inventory.player_id = $1 
		AND items.name = $2 
		AND inventory.quantity <= 0`,
		playerID, itemName,
	)
	return err
}

func (db *DB) UpdateToolDurability(playerID int, toolName string, newDurability int) error {
	_, err := db.conn.Exec(`
		UPDATE inventory 
		SET durability = $3
		FROM items 
		WHERE inventory.item_id = items.id 
		AND inventory.player_id = $1 
		AND items.name = $2`,
		playerID, toolName, newDurability,
	)
	return err
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// GetPlayerItems возвращает все предметы игрока
func (db *DB) GetPlayerItems(playerID int) ([]Item, error) {
	var items []Item
	query := `SELECT item_name, item_type, quantity FROM player_items WHERE player_id = $1`
	rows, err := db.conn.Query(query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item Item
		err := rows.Scan(&item.ItemName, &item.ItemType, &item.Quantity)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// AddPlayerItem добавляет предмет игроку
func (db *DB) AddPlayerItem(playerID int, itemName string, quantity int) error {
	query := `INSERT INTO player_items (player_id, item_name, item_type, quantity)
			  VALUES ($1, $2, (SELECT item_type FROM seed_items WHERE item_name = $2), $3)
			  ON CONFLICT (player_id, item_name) 
			  DO UPDATE SET quantity = player_items.quantity + $3`
	_, err := db.conn.Exec(query, playerID, itemName, quantity)
	return err
}

// AddPlayerExperience добавляет опыт игроку
func (db *DB) AddPlayerExperience(playerID int, amount int) error {
	query := `UPDATE players SET experience = experience + $1 WHERE id = $2`
	_, err := db.conn.Exec(query, amount, playerID)
	return err
}

// Устанавливает simple_hut_built = true для игрока
func (db *DB) UpdateSimpleHutBuilt(playerID int, built bool) error {
	_, err := db.conn.Exec(`UPDATE players SET simple_hut_built = $1 WHERE id = $2`, built, playerID)
	return err
}

func (db *DB) CreatePlayer(telegramID int64, name string) (*models.Player, error) {
	var player models.Player
	err := db.conn.QueryRow(`
		INSERT INTO players (telegram_id, name, level, experience, satiety, simple_hut_built)
		VALUES ($1, $2, 1, 0, 100, false)
		RETURNING id, telegram_id, name, level, experience, satiety, created_at, simple_hut_built`,
		telegramID, name,
	).Scan(&player.ID, &player.TelegramID, &player.Name, &player.Level, &player.Experience, &player.Satiety, &player.CreatedAt, &player.SimpleHutBuilt)

	if err != nil {
		return nil, err
	}

	// Создаем начальный инвентарь
	if err := db.createStarterInventory(player.ID); err != nil {
		return nil, err
	}

	return &player, nil
}

// createStarterInventory выдает стартовые предметы новому игроку
func (db *DB) createStarterInventory(playerID int) error {
	// Добавляем стартовые инструменты и ресурсы
	if err := db.AddItemToInventoryWithDurability(playerID, "Простой лук", 1, 100); err != nil {
		return err
	}
	if err := db.AddItemToInventoryWithDurability(playerID, "Простой нож", 1, 100); err != nil {
		return err
	}
	if err := db.AddItemToInventoryWithDurability(playerID, "Простая кирка", 1, 100); err != nil {
		return err
	}
	if err := db.AddItemToInventoryWithDurability(playerID, "Простой топор", 1, 100); err != nil {
		return err
	}
	if err := db.AddItemToInventory(playerID, "Стрелы", 100); err != nil {
		return err
	}
	if err := db.AddItemToInventory(playerID, "Лесная ягода", 10); err != nil {
		return err
	}
	return nil
}
