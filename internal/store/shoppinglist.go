package store

import (
	"strings"

	"github.com/zachcurry13/recipebank/internal/shopping"
)

// ShopItem is one line on the shopping list.
type ShopItem struct {
	ID        int64   `db:"id" json:"id"`
	Key       string  `db:"key" json:"-"`
	Name      string  `db:"name" json:"name"`
	Qty       float64 `db:"qty" json:"qty"`
	Unit      string  `db:"unit" json:"unit"`
	Section   string  `db:"section" json:"section"`
	Area      string  `db:"area" json:"area"`
	Note      string  `db:"note" json:"note"`
	StockID   *int64  `db:"stock_id" json:"stock_id"`
	Checked   bool    `db:"checked" json:"checked"`
	AddedBy   string  `db:"added_by" json:"added_by"`
	CreatedAt string  `db:"created_at" json:"created_at"`
	UpdatedAt string  `db:"updated_at" json:"updated_at"`
}

const shopCols = `id, key, name, qty, unit, section, area, note, stock_id, checked, added_by, created_at, updated_at`

// ListShopping returns the list, unticked first, by section.
func (s *Store) ListShopping() ([]ShopItem, error) {
	items := []ShopItem{}
	err := s.DB.Select(&items, `SELECT `+shopCols+` FROM shopping ORDER BY checked, section, name COLLATE NOCASE`)
	return items, err
}

// AddShopping puts it on the list, adding its amount to an unticked line for
// the same food when the units allow ("2 cups flour" + "1 cup flour").
// forWhat ("Soup") is noted on the line. It returns the line's ID.
func (s *Store) AddShopping(it ShopItem, forWhat string) (int64, error) {
	it.Name = strings.TrimSpace(it.Name)
	if it.Area != "home" {
		it.Area = "kitchen"
	}
	if it.Section == "" {
		it.Section = shopping.Section(it.Name, it.Area)
	}
	var same []ShopItem
	if err := s.DB.Select(&same, `SELECT `+shopCols+` FROM shopping WHERE checked = 0 AND key = ? AND area = ? AND key != ''`,
		it.Key, it.Area); err != nil {
		return 0, err
	}
	for _, row := range same {
		if q, ok := shopping.Add(row.Qty, row.Unit, it.Qty, it.Unit); ok {
			_, err := s.DB.Exec(`UPDATE shopping SET qty = ?, note = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
				q, addNote(row.Note, forWhat), row.ID)
			return row.ID, err
		}
	}
	res, err := s.DB.Exec(`INSERT INTO shopping (key, name, qty, unit, section, area, note, stock_id, added_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, it.Key, it.Name, max(0, it.Qty), it.Unit, it.Section, it.Area,
		addNote("", forWhat), it.StockID, it.AddedBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func addNote(note, forWhat string) string {
	forWhat = strings.TrimSpace(forWhat)
	if forWhat == "" {
		return note
	}
	for _, p := range strings.Split(strings.TrimPrefix(note, "for "), ", ") {
		if p == forWhat {
			return note
		}
	}
	if note == "" {
		return "for " + forWhat
	}
	return note + ", " + forWhat
}

// ShopChange is what a tick or an edit changes (nil = leave as is).
type ShopChange struct {
	Checked *bool    `json:"checked"`
	Name    *string  `json:"name"`
	Qty     *float64 `json:"qty"`
	Unit    *string  `json:"unit"`
	Section *string  `json:"section"`
}

func (s *Store) UpdateShopping(id int64, c ShopChange) error {
	var sets []string
	var args []any
	add := func(col string, v any) { sets = append(sets, col+" = ?"); args = append(args, v) }
	if c.Checked != nil {
		add("checked", *c.Checked)
	}
	if c.Name != nil && strings.TrimSpace(*c.Name) != "" {
		add("name", strings.TrimSpace(*c.Name))
	}
	if c.Qty != nil {
		add("qty", max(0, *c.Qty))
	}
	if c.Unit != nil {
		add("unit", strings.TrimSpace(*c.Unit))
	}
	if c.Section != nil {
		add("section", *c.Section)
	}
	if len(sets) == 0 {
		return nil
	}
	res, err := s.DB.Exec(`UPDATE shopping SET `+strings.Join(sets, ", ")+`, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		append(args, id)...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteShopping(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM shopping WHERE id = ?`, id)
	return err
}

// ClearChecked removes the ticked lines and returns them.
func (s *Store) ClearChecked() ([]ShopItem, error) {
	done := []ShopItem{}
	if err := s.DB.Select(&done, `SELECT `+shopCols+` FROM shopping WHERE checked = 1`); err != nil {
		return nil, err
	}
	_, err := s.DB.Exec(`DELETE FROM shopping WHERE checked = 1`)
	return done, err
}

// OnListForStock reports whether a running-low item is already on the list, unticked.
func (s *Store) OnListForStock(stockID int64) bool {
	var n int
	_ = s.DB.Get(&n, `SELECT COUNT(*) FROM shopping WHERE stock_id = ? AND checked = 0`, stockID)
	return n > 0
}
