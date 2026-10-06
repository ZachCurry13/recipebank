package store

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

const recipeCols = `id, area, title, summary, servings, yield_text, prep_min, cook_min, total_min, heat, course,
	cuisine, protein, ingredients, steps, notes, storage, source_kind, source_url, source_note, photo,
	source_photos, needs_review, read_by, ai_reading, rating, version_of, created_by, created_at, updated_at`

// recipeRow is a recipe as stored, with its lists as JSON.
type recipeRow struct {
	recipe.Recipe
	IngredientsJSON  string `db:"ingredients"`
	StepsJSON        string `db:"steps"`
	SourcePhotosJSON string `db:"source_photos"`
}

func (row *recipeRow) decode() recipe.Recipe {
	r := row.Recipe
	_ = json.Unmarshal([]byte(row.IngredientsJSON), &r.Ingredients)
	_ = json.Unmarshal([]byte(row.StepsJSON), &r.Steps)
	_ = json.Unmarshal([]byte(row.SourcePhotosJSON), &r.SourcePhotos)
	if r.Ingredients == nil {
		r.Ingredients = []recipe.Ingredient{}
	}
	if r.Steps == nil {
		r.Steps = []recipe.Step{}
	}
	if r.SourcePhotos == nil {
		r.SourcePhotos = []string{}
	}
	return r
}

// ListRecipes returns every recipe in an area ("" = both), by title.
func (s *Store) ListRecipes(area string) ([]recipe.Recipe, error) {
	var rows []recipeRow
	q := `SELECT ` + recipeCols + ` FROM recipes`
	var args []any
	if area != "" {
		q += ` WHERE area = ?`
		args = append(args, area)
	}
	if err := s.DB.Select(&rows, q+` ORDER BY title COLLATE NOCASE`, args...); err != nil {
		return nil, err
	}
	out := make([]recipe.Recipe, len(rows))
	for i := range rows {
		out[i] = rows[i].decode()
	}
	return out, nil
}

func (s *Store) Recipe(id int64) (*recipe.Recipe, error) {
	var row recipeRow
	if err := s.DB.Get(&row, `SELECT `+recipeCols+` FROM recipes WHERE id = ?`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	r := row.decode()
	return &r, nil
}

// SaveRecipe inserts r (ID 0) or updates it, and returns its ID.
func (s *Store) SaveRecipe(r *recipe.Recipe) (int64, error) {
	r.Clean()
	ing, _ := json.Marshal(r.Ingredients)
	steps, _ := json.Marshal(r.Steps)
	photos, _ := json.Marshal(r.SourcePhotos)
	args := []any{r.Area, r.Title, r.Summary, r.Servings, r.YieldText, r.PrepMin, r.CookMin, r.TotalMin, r.Heat,
		r.Course, r.Cuisine, r.Protein, string(ing), string(steps), r.Notes, r.Storage, r.SourceKind, r.SourceURL,
		r.SourceNote, r.Photo, string(photos), r.NeedsReview, r.ReadBy, r.AIReading, r.Rating, r.VersionOf}
	if r.ID == 0 {
		res, err := s.DB.Exec(`INSERT INTO recipes (area, title, summary, servings, yield_text, prep_min, cook_min,
			total_min, heat, course, cuisine, protein, ingredients, steps, notes, storage, source_kind, source_url,
			source_note, photo, source_photos, needs_review, read_by, ai_reading, rating, version_of, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, append(args, r.CreatedBy)...)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	res, err := s.DB.Exec(`UPDATE recipes SET area = ?, title = ?, summary = ?, servings = ?, yield_text = ?,
		prep_min = ?, cook_min = ?, total_min = ?, heat = ?, course = ?, cuisine = ?, protein = ?, ingredients = ?,
		steps = ?, notes = ?, storage = ?, source_kind = ?, source_url = ?, source_note = ?, photo = ?,
		source_photos = ?, needs_review = ?, read_by = ?, ai_reading = ?, rating = ?, version_of = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, append(args, r.ID)...)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, ErrNotFound
	}
	return r.ID, nil
}

// SetRating saves a recipe's stars (0-5).
func (s *Store) SetRating(id int64, stars int) error {
	_, err := s.DB.Exec(`UPDATE recipes SET rating = ? WHERE id = ?`, max(0, min(5, stars)), id)
	return err
}

func (s *Store) DeleteRecipe(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM recipes WHERE id = ?`, id)
	return err
}

// PhotosInUse lists every photo file a recipe still points at.
func (s *Store) PhotosInUse() (map[string]bool, error) {
	rs, err := s.ListRecipes("")
	if err != nil {
		return nil, err
	}
	used := map[string]bool{}
	for _, r := range rs {
		if r.Photo != "" {
			used[r.Photo] = true
		}
		for _, p := range r.SourcePhotos {
			used[p] = true
		}
	}
	return used, nil
}

// CountRecipes counts every saved recipe, Kitchen and Home & Care.
func (s *Store) CountRecipes() (int, error) {
	var n int
	err := s.DB.Get(&n, `SELECT COUNT(*) FROM recipes`)
	return n, err
}
