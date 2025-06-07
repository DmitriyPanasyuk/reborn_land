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
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
		{"Деревянный брусок", "material", 0},
		{"Камень", "material", 0},
		{"Уголь", "material", 0},
		{"Сухожилие", "material", 0},
		{"Перо", "material", 0},
		{"Кость", "material", 0},
		{"Веревка", "material", 0},
		{"Крючок", "material", 0},
		{"Береза", "material", 0},
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

func (db *DB) CreatePlayer(telegramID int64, name string) (*models.Player, error) {
	var player models.Player
	err := db.conn.QueryRow(`
		INSERT INTO players (telegram_id, name, level, experience, satiety) 
		VALUES ($1, $2, 1, 0, 100) 
		RETURNING id, telegram_id, name, level, experience, satiety, created_at`,
		telegramID, name,
	).Scan(&player.ID, &player.TelegramID, &player.Name, &player.Level, &player.Experience, &player.Satiety, &player.CreatedAt)

	if err != nil {
		return nil, err
	}

	// Создаем начальный инвентарь
	if err := db.createStarterInventory(player.ID); err != nil {
		return nil, err
	}

	return &player, nil
}

func (db *DB) createStarterInventory(playerID int) error {
	// Получаем ID предметов
	var axeID, knifeID, berryID, pickaxeID int

	err := db.conn.QueryRow("SELECT id FROM items WHERE name = 'Простой топор'").Scan(&axeID)
	if err != nil {
		return err
	}

	err = db.conn.QueryRow("SELECT id FROM items WHERE name = 'Простой нож'").Scan(&knifeID)
	if err != nil {
		return err
	}

	err = db.conn.QueryRow("SELECT id FROM items WHERE name = 'Лесная ягода'").Scan(&berryID)
	if err != nil {
		return err
	}

	err = db.conn.QueryRow("SELECT id FROM items WHERE name = 'Простая кирка'").Scan(&pickaxeID)
	if err != nil {
		return err
	}

	// Добавляем в инвентарь
	items := []struct {
		itemID     int
		quantity   int
		durability int
	}{
		{axeID, 1, 100},
		{knifeID, 1, 100},
		{pickaxeID, 1, 100},
		{berryID, 10, 0},
	}

	for _, item := range items {
		_, err := db.conn.Exec(
			"INSERT INTO inventory (player_id, item_id, quantity, durability) VALUES ($1, $2, $3, $4)",
			playerID, item.itemID, item.quantity, item.durability,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) GetPlayer(telegramID int64) (*models.Player, error) {
	var player models.Player
	err := db.conn.QueryRow(`
		SELECT id, telegram_id, name, level, experience, satiety, created_at 
		FROM players WHERE telegram_id = $1`,
		telegramID,
	).Scan(&player.ID, &player.TelegramID, &player.Name, &player.Level, &player.Experience, &player.Satiety, &player.CreatedAt)

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
			{ItemName: "Деревянный брусок", Quantity: 1},
			{ItemName: "Камень", Quantity: 1},
		},
		"Простая кирка": {
			{ItemName: "Деревянный брусок", Quantity: 1},
			{ItemName: "Камень", Quantity: 1},
		},
		"Простой лук": {
			{ItemName: "Деревянный брусок", Quantity: 1},
			{ItemName: "Сухожилие", Quantity: 1},
		},
		"Стрелы": {
			{ItemName: "Деревянный брусок", Quantity: 1},
			{ItemName: "Камень", Quantity: 1},
			{ItemName: "Перо", Quantity: 1},
		},
		"Простой нож": {
			{ItemName: "Деревянный брусок", Quantity: 1},
			{ItemName: "Кость", Quantity: 1},
		},
		"Простая удочка": {
			{ItemName: "Деревянный брусок", Quantity: 1},
			{ItemName: "Веревка", Quantity: 1},
			{ItemName: "Крючок", Quantity: 1},
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

func (db *DB) Close() error {
	return db.conn.Close()
}
