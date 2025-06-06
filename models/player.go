package models

import "time"

type Player struct {
	ID         int       `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	Name       string    `json:"name"`
	Level      int       `json:"level"`
	Experience int       `json:"experience"`
	Satiety    int       `json:"satiety"`
	CreatedAt  time.Time `json:"created_at"`
}

type Item struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	DurabilityMax int    `json:"durability_max"`
}

type InventoryItem struct {
	ID         int    `json:"id"`
	PlayerID   int    `json:"player_id"`
	ItemID     int    `json:"item_id"`
	ItemName   string `json:"item_name"`
	Quantity   int    `json:"quantity"`
	Durability int    `json:"durability"`
	Type       string `json:"type"`
}

type Recipe struct {
	ID          int                `json:"id"`
	ItemID      int                `json:"item_id"`
	ItemName    string             `json:"item_name"`
	Ingredients []RecipeIngredient `json:"ingredients"`
}

type RecipeIngredient struct {
	ItemID   int    `json:"item_id"`
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
}
