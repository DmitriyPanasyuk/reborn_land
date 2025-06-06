package database

import (
	"database/sql"
	"fmt"
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
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) seedItems() error {
	// Проверяем, есть ли уже предметы в базе
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Предметы уже есть
	}

	// Добавляем начальные предметы
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
		{"Сухожилие", "material", 0},
		{"Перо", "material", 0},
		{"Кость", "material", 0},
		{"Веревка", "material", 0},
		{"Крючок", "material", 0},
	}

	for _, item := range items {
		_, err := db.conn.Exec(
			"INSERT INTO items (name, type, durability_max) VALUES ($1, $2, $3)",
			item.name, item.itemType, item.durabilityMax,
		)
		if err != nil {
			return err
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
	var axeID, knifeID, berryID int

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

	// Добавляем в инвентарь
	items := []struct {
		itemID     int
		quantity   int
		durability int
	}{
		{axeID, 1, 100},
		{knifeID, 1, 100},
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

func (db *DB) Close() error {
	return db.conn.Close()
}
