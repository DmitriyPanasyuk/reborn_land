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

type Mine struct {
	ID          int       `json:"id"`
	PlayerID    int       `json:"player_id"`
	Level       int       `json:"level"`
	Experience  int       `json:"experience"`
	LastUsed    time.Time `json:"last_used"`
	IsExhausted bool      `json:"is_exhausted"`
}

type MineSession struct {
	PlayerID        int64      `json:"player_id"`
	Resources       [][]string `json:"resources"` // 3x3 массив ресурсов
	IsActive        bool       `json:"is_active"`
	IsMining        bool       `json:"is_mining"`
	StartedAt       time.Time  `json:"started_at"`
	FieldMessageID  int        `json:"field_message_id"`  // ID сообщения с полем шахты
	InfoMessageID   int        `json:"info_message_id"`   // ID сообщения с информацией о шахте
	ResultMessageID int        `json:"result_message_id"` // ID сообщения с результатом добычи
}

type Forest struct {
	ID          int       `json:"id"`
	PlayerID    int       `json:"player_id"`
	Level       int       `json:"level"`
	Experience  int       `json:"experience"`
	LastUsed    time.Time `json:"last_used"`
	IsExhausted bool      `json:"is_exhausted"`
}

type Gathering struct {
	ID          int       `json:"id"`
	PlayerID    int       `json:"player_id"`
	Level       int       `json:"level"`
	Experience  int       `json:"experience"`
	LastUsed    time.Time `json:"last_used"`
	IsExhausted bool      `json:"is_exhausted"`
}

type ForestSession struct {
	PlayerID        int64      `json:"player_id"`
	Resources       [][]string `json:"resources"` // 3x3 массив ресурсов
	IsActive        bool       `json:"is_active"`
	IsChopping      bool       `json:"is_chopping"`
	StartedAt       time.Time  `json:"started_at"`
	FieldMessageID  int        `json:"field_message_id"`  // ID сообщения с полем леса
	InfoMessageID   int        `json:"info_message_id"`   // ID сообщения с информацией о лесе
	ResultMessageID int        `json:"result_message_id"` // ID сообщения с результатом рубки
}

type GatheringSession struct {
	PlayerID        int64      `json:"player_id"`
	Resources       [][]string `json:"resources"` // 3x3 массив ресурсов
	IsActive        bool       `json:"is_active"`
	IsGathering     bool       `json:"is_gathering"`
	StartedAt       time.Time  `json:"started_at"`
	FieldMessageID  int        `json:"field_message_id"`  // ID сообщения с полем сбора
	InfoMessageID   int        `json:"info_message_id"`   // ID сообщения с информацией о сборе
	ResultMessageID int        `json:"result_message_id"` // ID сообщения с результатом сбора
}

type Quest struct {
	ID          int        `json:"id"`
	PlayerID    int        `json:"player_id"`
	QuestID     int        `json:"quest_id"` // ID квеста (1, 2, 3...)
	Status      string     `json:"status"`   // "available", "active", "completed"
	Progress    int        `json:"progress"` // текущий прогресс
	Target      int        `json:"target"`   // цель квеста
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
}
