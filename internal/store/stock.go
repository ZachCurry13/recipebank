package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
)

// StockItem is something in the house: food in the pantry (area "kitchen")
// or a supply in the closet (area "home").
type StockItem struct {
	ID              int64    `db:"id" json:"id"`
	Area            string   `db:"area" json:"area"`
	Name            string   `db:"name" json:"name"`
	Brand           string   `db:"brand" json:"brand"`
	Barcode         string   `db:"barcode" json:"barcode"`
	Qty             float64  `db:"qty" json:"qty"`
	Unit            string   `db:"unit" json:"unit"`
	Location        string   `db:"location" json:"location"`
	UseBy           string   `db:"use_by" json:"use_by"`
	LowAt           float64  `db:"low_at" json:"low_at"`
	AllergensJSON   string   `db:"allergens" json:"-"`
	TracesJSON      string   `db:"traces" json:"-"`
	Allergens       []string `db:"-" json:"allergens"`
	Traces          []string `db:"-" json:"traces"`
	IngredientsText string   `db:"ingredients_text" json:"ingredients_text"`
	LabelSource     string   `db:"label_source" json:"label_source"`
	CreatedAt       string   `db:"created_at" json:"created_at"`
	UpdatedAt       string   `db:"updated_at" json:"updated_at"`
}

// Low reports whether the item is at or under its running-low level.
func (it *StockItem) Low() bool { return it.LowAt > 0 && it.Qty <= it.LowAt }

const stockCols = `id, area, name, brand, barcode, qty, unit, location, use_by, low_at, allergens, traces,
	ingredients_text, label_source, created_at, updated_at`

func decodeStock(items []StockItem) []StockItem {
	for i := range items {
		it := &items[i]
		_ = json.Unmarshal([]byte(it.AllergensJSON), &it.Allergens)
		_ = json.Unmarshal([]byte(it.TracesJSON), &it.Traces)
		if it.Allergens == nil {
			it.Allergens = []string{}
		}
		if it.Traces == nil {
			it.Traces = []string{}
		}
	}
	return items
}

// ListStock returns an area's items ("" = both), by name.
func (s *Store) ListStock(area string) ([]StockItem, error) {
	items := []StockItem{}
	q := `SELECT ` + stockCols + ` FROM stock`
	var args []any
	if area != "" {
		q += ` WHERE area = ?`
		args = append(args, area)
	}
	err := s.DB.Select(&items, q+` ORDER BY name COLLATE NOCASE`, args...)
	return decodeStock(items), err
}

func (s *Store) StockItem(id int64) (*StockItem, error) {
	var items []StockItem
	if err := s.DB.Select(&items, `SELECT `+stockCols+` FROM stock WHERE id = ?`, id); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return &decodeStock(items)[0], nil
}

// SaveStock inserts (ID 0) or updates an item and returns its ID.
func (s *Store) SaveStock(it *StockItem) (int64, error) {
	it.Name = strings.TrimSpace(it.Name)
	if it.Area != "home" {
		it.Area = "kitchen"
	}
	al, _ := json.Marshal(nonNilStrings(it.Allergens))
	tr, _ := json.Marshal(nonNilStrings(it.Traces))
	args := []any{it.Area, it.Name, strings.TrimSpace(it.Brand), strings.TrimSpace(it.Barcode), max(0, it.Qty),
		strings.TrimSpace(it.Unit), strings.TrimSpace(it.Location), it.UseBy, max(0, it.LowAt), string(al), string(tr),
		it.IngredientsText, it.LabelSource}
	if it.ID == 0 {
		res, err := s.DB.Exec(`INSERT INTO stock (area, name, brand, barcode, qty, unit, location, use_by, low_at,
			allergens, traces, ingredients_text, label_source) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, args...)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	res, err := s.DB.Exec(`UPDATE stock SET area = ?, name = ?, brand = ?, barcode = ?, qty = ?, unit = ?,
		location = ?, use_by = ?, low_at = ?, allergens = ?, traces = ?, ingredients_text = ?, label_source = ?,
		updated_at = CURRENT_TIMESTAMP WHERE id = ?`, append(args, it.ID)...)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, ErrNotFound
	}
	return it.ID, nil
}

// AdjustStock changes an item's amount by delta (never below 0).
func (s *Store) AdjustStock(id int64, delta float64) (*StockItem, error) {
	res, err := s.DB.Exec(`UPDATE stock SET qty = MAX(0, qty + ?), updated_at = CURRENT_TIMESTAMP WHERE id = ?`, delta, id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.StockItem(id)
}

func (s *Store) DeleteStock(id int64) error {
	res, err := s.DB.Exec(`DELETE FROM stock WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// StockByBarcode finds an item already in the house with this barcode.
func (s *Store) StockByBarcode(area, barcode string) (*StockItem, error) {
	var items []StockItem
	err := s.DB.Select(&items, `SELECT `+stockCols+` FROM stock WHERE area = ? AND barcode = ? LIMIT 1`, area, barcode)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && len(items) == 0) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &decodeStock(items)[0], nil
}

func nonNilStrings(l []string) []string {
	if l == nil {
		return []string{}
	}
	return l
}
