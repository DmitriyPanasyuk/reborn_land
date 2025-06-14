package db

import (
	"database/sql"
	"log"
)

// SeedItems добавляет базовые предметы в базу данных
func SeedItems(db *sql.DB) error {
	items := []struct {
		Name        string
		Type        string
		Description string
		Durability  int
	}{
		{Name: "🪓 Топор", Type: "tool", Description: "Инструмент для рубки деревьев", Durability: 100},
		{Name: "⛏️ Кирка", Type: "tool", Description: "Инструмент для добычи камня", Durability: 100},
		{Name: "🌿 Лесная ягода", Type: "food", Description: "Съедобная ягода, восстанавливает 5 единиц сытости", Durability: 0},
		{Name: "🪨 Камень", Type: "resource", Description: "Базовый строительный материал", Durability: 0},
		{Name: "🪵 Дерево", Type: "resource", Description: "Базовый строительный материал", Durability: 0},
		{Name: "🏠 Простая хижина", Type: "building", Description: "Простое жилище", Durability: 0},
		{Name: "🏠 Улучшенная хижина", Type: "building", Description: "Улучшенное жилище", Durability: 0},
		{Name: "🏠 Дом", Type: "building", Description: "Полноценный дом", Durability: 0},
		{Name: "🏠 Особняк", Type: "building", Description: "Роскошное жилище", Durability: 0},
		{Name: "🏰 Замок", Type: "building", Description: "Величественное строение", Durability: 0},
		{Name: "📖 Страница 1 «Забытая тишина»", Type: "lore", Description: "Страница 1 из книги лора", Durability: 0},
		{Name: "📖 Страница 2 «Пепел памяти»", Type: "lore", Description: "Страница 2 из книги лора", Durability: 0},
		{Name: "📖 Страница 3 «Пробуждение»", Type: "lore", Description: "Страница 3 из книги лора", Durability: 0},
		{Name: "📖 Страница 4 «Без имени»", Type: "lore", Description: "Страница 4 из книги лора", Durability: 0},
		{Name: "📖 Страница 5 «Искра перемен»", Type: "lore", Description: "Страница 5 из книги лора", Durability: 0},
		{Name: "📖 Страница 6 «Наблюдающий лес»", Type: "lore", Description: "Страница 6 из книги лора", Durability: 0},
		{Name: "📖 Страница 7 «Шёпот ветра»", Type: "lore", Description: "Страница 7 из книги лора", Durability: 0},
		{Name: "📖 Страница 8 «След древних»", Type: "lore", Description: "Страница 8 из книги лора", Durability: 0},
	}

	for _, item := range items {
		_, err := db.Exec(`
			INSERT INTO items (name, type, description, durability)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (name) DO NOTHING
		`, item.Name, item.Type, item.Description, item.Durability)
		if err != nil {
			log.Printf("Error seeding item %s: %v", item.Name, err)
			return err
		}
	}

	return nil
}

// SeedQuests добавляет базовые квесты в базу данных
func SeedQuests(db *sql.DB) error {
	quests := []struct {
		ID          int
		Name        string
		Description string
		Target      int
		RewardExp   int
		RewardItem  string
	}{
		{ID: 1, Name: "Первые шаги", Description: "Добыть 5 камней", Target: 5, RewardExp: 10, RewardItem: "📖 Страница 1 «Забытая тишина»"},
		{ID: 2, Name: "Древесный путь", Description: "Срубить 5 деревьев", Target: 5, RewardExp: 10, RewardItem: "📖 Страница 2 «Пепел памяти»"},
		{ID: 3, Name: "Ягодный сбор", Description: "Собрать 5 ягод", Target: 5, RewardExp: 10, RewardItem: "📖 Страница 3 «Пробуждение»"},
		{ID: 4, Name: "Строитель", Description: "Построить простую хижину", Target: 1, RewardExp: 10, RewardItem: "📖 Страница 4 «Без имени»"},
		{ID: 5, Name: "Звериный взгляд", Description: "Завершить первую охоту", Target: 1, RewardExp: 10, RewardItem: "📖 Страница 5 «Искра перемен»"},
		{ID: 6, Name: "Живое Хранилище", Description: "Открой 5 страниц лора", Target: 5, RewardExp: 10, RewardItem: "📖 Страница 6 «Наблюдающий лес»"},
		{ID: 7, Name: "Перекус", Description: "Съешь 3 ягоды", Target: 3, RewardExp: 10, RewardItem: "📖 Страница 7 «Шёпот ветра»"},
		{ID: 8, Name: "Под крышей", Description: "Построй простую хижину", Target: 1, RewardExp: 10, RewardItem: "📖 Страница 8 «След древних»"},
	}

	for _, quest := range quests {
		_, err := db.Exec(`
			INSERT INTO quests (id, name, description, target, reward_exp, reward_item)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE
			SET name = EXCLUDED.name,
				description = EXCLUDED.description,
				target = EXCLUDED.target,
				reward_exp = EXCLUDED.reward_exp,
				reward_item = EXCLUDED.reward_item
		`, quest.ID, quest.Name, quest.Description, quest.Target, quest.RewardExp, quest.RewardItem)
		if err != nil {
			log.Printf("Error seeding quest %d: %v", quest.ID, err)
			return err
		}
	}

	return nil
}
