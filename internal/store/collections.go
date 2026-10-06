package store

import "strings"

// Collection is a themed shelf of recipes.
type Collection struct {
	ID          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
	Icon        string `db:"icon" json:"icon"`
	Area        string `db:"area" json:"area"`
	CreatedBy   string `db:"created_by" json:"created_by"`
	Count       int    `db:"count" json:"count"`
}

// ListCollections returns every collection with how many recipes it holds.
func (s *Store) ListCollections() ([]Collection, error) {
	out := []Collection{}
	err := s.DB.Select(&out, `SELECT c.id, c.name, c.description, c.icon, c.area, c.created_by,
		(SELECT COUNT(*) FROM collection_recipes cr WHERE cr.collection_id = c.id) AS count
		FROM collections c ORDER BY c.name COLLATE NOCASE`)
	return out, err
}

func (s *Store) Collection(id int64) (*Collection, error) {
	all, err := s.ListCollections()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, ErrNotFound
}

// SaveCollection adds (ID 0) or renames/redescribes a collection.
func (s *Store) SaveCollection(c *Collection) (int64, error) {
	c.Name, c.Description = strings.TrimSpace(c.Name), strings.TrimSpace(c.Description)
	if c.Area != "home" {
		c.Area = "kitchen"
	}
	if strings.TrimSpace(c.Icon) == "" {
		c.Icon = "📚"
	}
	if c.ID == 0 {
		res, err := s.DB.Exec(`INSERT INTO collections (name, description, icon, area, created_by) VALUES (?, ?, ?, ?, ?)`,
			c.Name, c.Description, c.Icon, c.Area, c.CreatedBy)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	res, err := s.DB.Exec(`UPDATE collections SET name = ?, description = ?, icon = ? WHERE id = ?`,
		c.Name, c.Description, c.Icon, c.ID)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, ErrNotFound
	}
	return c.ID, nil
}

func (s *Store) DeleteCollection(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM collections WHERE id = ?`, id)
	return err
}

// CollectionRecipeIDs lists a collection's recipes, newest first.
func (s *Store) CollectionRecipeIDs(id int64) ([]int64, error) {
	ids := []int64{}
	err := s.DB.Select(&ids, `SELECT recipe_id FROM collection_recipes WHERE collection_id = ? ORDER BY added_at DESC`, id)
	return ids, err
}

// AddToCollection puts recipes on a shelf (ones already there are skipped).
func (s *Store) AddToCollection(id int64, recipeIDs []int64) error {
	for _, r := range recipeIDs {
		if _, err := s.DB.Exec(`INSERT OR IGNORE INTO collection_recipes (collection_id, recipe_id) VALUES (?, ?)`, id, r); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) RemoveFromCollection(id, recipeID int64) error {
	_, err := s.DB.Exec(`DELETE FROM collection_recipes WHERE collection_id = ? AND recipe_id = ?`, id, recipeID)
	return err
}

// CollectionsOf lists the collections a recipe is on.
func (s *Store) CollectionsOf(recipeID int64) ([]int64, error) {
	ids := []int64{}
	err := s.DB.Select(&ids, `SELECT collection_id FROM collection_recipes WHERE recipe_id = ?`, recipeID)
	return ids, err
}
