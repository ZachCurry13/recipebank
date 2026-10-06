package store

// KeyUSDAKey is the free data.gov key for USDA FoodData Central (nutrition
// estimates). It is a secret: never sent to the browser.
const KeyUSDAKey = "usda_api_key"

func init() { SecretKeys[KeyUSDAKey] = true }

// NutritionFood returns a food's cached answer ("" = not found); ok is false
// when it was never looked up, or a "not found" is over 30 days old.
func (s *Store) NutritionFood(food string) (data string, ok bool) {
	var row struct {
		Data  string `db:"data"`
		Stale bool   `db:"stale"`
	}
	err := s.DB.Get(&row, `SELECT data, (data = '' AND fetched_at < datetime('now', '-30 days')) AS stale
		FROM nutrition_foods WHERE food = ?`, food)
	if err != nil || row.Stale {
		return "", false
	}
	return row.Data, true
}

func (s *Store) SaveNutritionFood(food, data string) error {
	_, err := s.DB.Exec(`INSERT INTO nutrition_foods (food, data) VALUES (?, ?)
		ON CONFLICT(food) DO UPDATE SET data = excluded.data, fetched_at = CURRENT_TIMESTAMP`, food, data)
	return err
}
