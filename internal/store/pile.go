package store

import "time"

// PileItem is a recipe card or clipping still to photograph.
type PileItem struct {
	ID       int64  `db:"id" json:"id"`
	Title    string `db:"title" json:"title"`
	Note     string `db:"note" json:"note"`
	RecipeID *int64 `db:"recipe_id" json:"recipe_id"`
	DoneAt   string `db:"done_at" json:"done_at"` // "" while still to do
}

// ListPile is the pile: still to do first, then done ones, newest first.
func (s *Store) ListPile() ([]PileItem, error) {
	out := []PileItem{}
	err := s.DB.Select(&out, `SELECT id, title, note, recipe_id, done_at FROM pile
		ORDER BY done_at != '', done_at DESC, id`)
	return out, err
}

// AddPile adds cards to the pile.
func (s *Store) AddPile(title, note string) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO pile (title, note) VALUES (?, ?)`, title, note)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdatePile changes a card's name and note, and ticks it off (done) or not;
// recipeID is the recipe saved from it, when there is one.
func (s *Store) UpdatePile(id int64, title, note string, done bool, recipeID *int64) error {
	doneAt := ""
	if done {
		doneAt = time.Now().UTC().Format(time.RFC3339)
	}
	// Ticking off keeps the first time it was done; unticking forgets the recipe.
	res, err := s.DB.Exec(`UPDATE pile SET title = ?, note = ?,
		done_at = CASE WHEN ? = '' THEN '' ELSE COALESCE(NULLIF(done_at, ''), ?) END,
		recipe_id = CASE WHEN ? = '' THEN NULL ELSE COALESCE(?, recipe_id) END WHERE id = ?`,
		title, note, doneAt, doneAt, doneAt, recipeID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeletePile removes a card from the pile.
func (s *Store) DeletePile(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM pile WHERE id = ?`, id)
	return err
}
