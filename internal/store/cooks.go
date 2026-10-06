package store

// Cook is one time a recipe was cooked, with each person's 👍 (1) or 👎 (-1).
type Cook struct {
	ID       int64         `db:"id" json:"id"`
	RecipeID int64         `db:"recipe_id" json:"recipe_id"`
	CookedOn string        `db:"cooked_on" json:"cooked_on"` // YYYY-MM-DD
	Note     string        `db:"note" json:"note"`
	AddedBy  string        `db:"added_by" json:"added_by"`
	Thumbs   map[int64]int `db:"-" json:"thumbs"` // person id → 1 or -1
}

// AddCook saves a cook and its thumbs together.
func (s *Store) AddCook(c Cook) (int64, error) {
	tx, err := s.DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO cooks (recipe_id, cooked_on, note, added_by) VALUES (?, ?, ?, ?)`,
		c.RecipeID, c.CookedOn, c.Note, c.AddedBy)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for person, thumb := range c.Thumbs {
		if _, err := tx.Exec(`INSERT INTO cook_thumbs (cook_id, person_id, thumb) VALUES (?, ?, ?)`, id, person, thumb); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

// CooksFor lists a recipe's cooks, newest first.
func (s *Store) CooksFor(recipeID int64) ([]Cook, error) {
	cooks := []Cook{}
	if err := s.DB.Select(&cooks, `SELECT id, recipe_id, cooked_on, note, added_by FROM cooks
		WHERE recipe_id = ? ORDER BY cooked_on DESC, id DESC`, recipeID); err != nil {
		return nil, err
	}
	var thumbs []struct {
		CookID   int64 `db:"cook_id"`
		PersonID int64 `db:"person_id"`
		Thumb    int   `db:"thumb"`
	}
	if err := s.DB.Select(&thumbs, `SELECT t.cook_id, t.person_id, t.thumb FROM cook_thumbs t
		JOIN cooks c ON c.id = t.cook_id WHERE c.recipe_id = ?`, recipeID); err != nil {
		return nil, err
	}
	byID := map[int64]map[int64]int{}
	for _, t := range thumbs {
		if byID[t.CookID] == nil {
			byID[t.CookID] = map[int64]int{}
		}
		byID[t.CookID][t.PersonID] = t.Thumb
	}
	for i := range cooks {
		cooks[i].Thumbs = byID[cooks[i].ID]
		if cooks[i].Thumbs == nil {
			cooks[i].Thumbs = map[int64]int{}
		}
	}
	return cooks, nil
}

func (s *Store) DeleteCook(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM cooks WHERE id = ?`, id)
	return err
}

// Likes is how each person feels about each recipe (the sum of their
// thumbs) and when each recipe was last cooked.
type Likes struct {
	Net  map[int64]map[int64]int // recipe → person → sum of thumbs
	Last map[int64]string        // recipe → last cooked_on
}

func (s *Store) Likes() (Likes, error) {
	l := Likes{Net: map[int64]map[int64]int{}, Last: map[int64]string{}}
	var sums []struct {
		RecipeID int64 `db:"recipe_id"`
		PersonID int64 `db:"person_id"`
		Net      int   `db:"net"`
	}
	if err := s.DB.Select(&sums, `SELECT c.recipe_id, t.person_id, SUM(t.thumb) AS net FROM cook_thumbs t
		JOIN cooks c ON c.id = t.cook_id GROUP BY c.recipe_id, t.person_id`); err != nil {
		return l, err
	}
	for _, x := range sums {
		if l.Net[x.RecipeID] == nil {
			l.Net[x.RecipeID] = map[int64]int{}
		}
		l.Net[x.RecipeID][x.PersonID] = x.Net
	}
	var last []struct {
		RecipeID int64  `db:"recipe_id"`
		On       string `db:"on_day"`
	}
	if err := s.DB.Select(&last, `SELECT recipe_id, MAX(cooked_on) AS on_day FROM cooks GROUP BY recipe_id`); err != nil {
		return l, err
	}
	for _, x := range last {
		l.Last[x.RecipeID] = x.On
	}
	return l, nil
}
