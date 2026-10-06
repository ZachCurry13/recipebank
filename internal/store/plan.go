package store

import (
	"encoding/json"
	"strings"
)

// Meals, in the order of a day.
var Meals = []string{"breakfast", "lunch", "dinner", "snack"}

// ValidMeal reports whether m is a known meal.
func ValidMeal(m string) bool {
	for _, x := range Meals {
		if x == m {
			return true
		}
	}
	return false
}

// PlanEntry is one planned meal: a recipe, or just a note.
type PlanEntry struct {
	ID       int64   `db:"id" json:"id"`
	Date     string  `db:"date" json:"date"`
	Meal     string  `db:"meal" json:"meal"`
	RecipeID *int64  `db:"recipe_id" json:"recipe_id"`
	Title    string  `db:"title" json:"title"`
	Servings float64 `db:"servings" json:"servings"`
	Note     string  `db:"note" json:"note"`
	// LeftoversOf is the planned meal this one eats the leftovers of.
	LeftoversOf *int64 `db:"leftovers_of" json:"leftovers_of"`
	CreatedBy   string `db:"created_by" json:"created_by"`
}

const planCols = `id, date, meal, recipe_id, title, servings, note, leftovers_of, created_by`

// PlanBetween returns the entries from one date to another (inclusive).
func (s *Store) PlanBetween(from, to string) ([]PlanEntry, error) {
	out := []PlanEntry{}
	err := s.DB.Select(&out, `SELECT `+planCols+` FROM meal_plan WHERE date >= ? AND date <= ?
		ORDER BY date, CASE meal WHEN 'breakfast' THEN 0 WHEN 'lunch' THEN 1 WHEN 'dinner' THEN 2 ELSE 3 END, id`, from, to)
	return out, err
}

func (s *Store) PlanEntry(id int64) (*PlanEntry, error) {
	var out []PlanEntry
	if err := s.DB.Select(&out, `SELECT `+planCols+` FROM meal_plan WHERE id = ?`, id); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return &out[0], nil
}

// SavePlan adds (ID 0) or changes a planned meal.
func (s *Store) SavePlan(e *PlanEntry) (int64, error) {
	e.Title = strings.TrimSpace(e.Title)
	if e.ID == 0 {
		res, err := s.DB.Exec(`INSERT INTO meal_plan (date, meal, recipe_id, title, servings, note, leftovers_of, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, e.Date, e.Meal, e.RecipeID, e.Title, max(0, e.Servings), e.Note, e.LeftoversOf, e.CreatedBy)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	res, err := s.DB.Exec(`UPDATE meal_plan SET date = ?, meal = ?, recipe_id = ?, title = ?, servings = ?, note = ?,
		leftovers_of = ? WHERE id = ?`, e.Date, e.Meal, e.RecipeID, e.Title, max(0, e.Servings), e.Note, e.LeftoversOf, e.ID)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, ErrNotFound
	}
	return e.ID, nil
}

func (s *Store) DeletePlan(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM meal_plan WHERE id = ?`, id)
	return err
}

// PlanDays returns who's home on each date that was set (date → people ids).
func (s *Store) PlanDays(from, to string) (map[string][]int64, error) {
	var rows []struct {
		Date string `db:"date"`
		Who  string `db:"who"`
	}
	if err := s.DB.Select(&rows, `SELECT date, who FROM plan_days WHERE date >= ? AND date <= ?`, from, to); err != nil {
		return nil, err
	}
	out := map[string][]int64{}
	for _, r := range rows {
		var ids []int64
		_ = json.Unmarshal([]byte(r.Who), &ids)
		out[r.Date] = ids
	}
	return out, nil
}

// SetPlanDay saves who's home on a date; nil goes back to everyone but guests.
func (s *Store) SetPlanDay(date string, who []int64) error {
	if who == nil {
		_, err := s.DB.Exec(`DELETE FROM plan_days WHERE date = ?`, date)
		return err
	}
	b, _ := json.Marshal(who)
	_, err := s.DB.Exec(`INSERT INTO plan_days (date, who) VALUES (?, ?)
		ON CONFLICT(date) DO UPDATE SET who = excluded.who`, date, string(b))
	return err
}

// LeftoversFor lists the planned meals that eat a meal's leftovers.
func (s *Store) LeftoversFor(id int64) ([]PlanEntry, error) {
	out := []PlanEntry{}
	err := s.DB.Select(&out, `SELECT `+planCols+` FROM meal_plan WHERE leftovers_of = ? ORDER BY date, id`, id)
	return out, err
}
