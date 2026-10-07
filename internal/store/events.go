package store

import (
	"encoding/json"
	"strings"
)

// Event is a holiday dinner or a potluck.
type Event struct {
	ID        int64   `db:"id" json:"id"`
	Name      string  `db:"name" json:"name"`
	Date      string  `db:"date" json:"date"` // YYYY-MM-DD, or "" when not set
	WhoJSON   string  `db:"who" json:"-"`
	Who       []int64 `db:"-" json:"who"` // people coming, guests included
	Extra     int     `db:"extra" json:"extra"`
	Notes     string  `db:"notes" json:"notes"`
	CreatedBy string  `db:"created_by" json:"created_by"`
}

// Dish is one thing on an event's menu.
type Dish struct {
	ID       int64  `db:"id" json:"id"`
	EventID  int64  `db:"event_id" json:"event_id"`
	RecipeID *int64 `db:"recipe_id" json:"recipe_id"`
	Title    string `db:"title" json:"title"`
	Brings   string `db:"brings" json:"brings"` // "" = the family
	// For a dish without a recipe: the allergens whoever brings it says it has.
	ContainsText string   `db:"contains" json:"-"`
	Contains     []string `db:"-" json:"contains"`
	GuestID      *int64   `db:"guest_id" json:"guest_id"` // added by a guest through the event's link
}

const eventCols = `id, name, date, who, extra, notes, created_by`

func (e *Event) decode() {
	e.Who = []int64{}
	_ = json.Unmarshal([]byte(e.WhoJSON), &e.Who)
}

// ListEvents lists events, the next ones first, then the past, newest first.
func (s *Store) ListEvents(today string) ([]Event, error) {
	out := []Event{}
	err := s.DB.Select(&out, `SELECT `+eventCols+` FROM events ORDER BY
		CASE WHEN date >= ? OR date = '' THEN 0 ELSE 1 END,
		CASE WHEN date >= ? OR date = '' THEN date END ASC, date DESC, id`, today, today)
	for i := range out {
		out[i].decode()
	}
	return out, err
}

func (s *Store) Event(id int64) (*Event, error) {
	var out []Event
	if err := s.DB.Select(&out, `SELECT `+eventCols+` FROM events WHERE id = ?`, id); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	out[0].decode()
	return &out[0], nil
}

// SaveEvent adds (ID 0) or changes an event.
func (s *Store) SaveEvent(e *Event) (int64, error) {
	who, _ := json.Marshal(nonNilIDs(e.Who))
	e.Name = strings.TrimSpace(e.Name)
	if e.ID == 0 {
		res, err := s.DB.Exec(`INSERT INTO events (name, date, who, extra, notes, created_by) VALUES (?, ?, ?, ?, ?, ?)`,
			e.Name, e.Date, string(who), max(0, e.Extra), e.Notes, e.CreatedBy)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	res, err := s.DB.Exec(`UPDATE events SET name = ?, date = ?, who = ?, extra = ?, notes = ? WHERE id = ?`,
		e.Name, e.Date, string(who), max(0, e.Extra), e.Notes, e.ID)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, ErrNotFound
	}
	return e.ID, nil
}

func (s *Store) DeleteEvent(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM events WHERE id = ?`, id)
	return err
}

// Dishes lists an event's menu in the order it was added.
func (s *Store) Dishes(eventID int64) ([]Dish, error) {
	out := []Dish{}
	err := s.DB.Select(&out, `SELECT id, event_id, recipe_id, title, brings, contains, guest_id FROM event_dishes
		WHERE event_id = ? ORDER BY id`, eventID)
	for i := range out {
		out[i].Contains = SplitList(out[i].ContainsText)
	}
	return out, err
}

func (s *Store) AddDish(d Dish) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO event_dishes (event_id, recipe_id, title, brings, contains, guest_id) VALUES (?, ?, ?, ?, ?, ?)`,
		d.EventID, d.RecipeID, strings.TrimSpace(d.Title), strings.TrimSpace(d.Brings), strings.Join(d.Contains, ","), d.GuestID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// SetDishBrings changes who brings a dish of an event.
func (s *Store) SetDishBrings(eventID, id int64, brings string) error {
	res, err := s.DB.Exec(`UPDATE event_dishes SET brings = ? WHERE id = ? AND event_id = ?`, strings.TrimSpace(brings), id, eventID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteDish(eventID, id int64) error {
	_, err := s.DB.Exec(`DELETE FROM event_dishes WHERE id = ? AND event_id = ?`, id, eventID)
	return err
}

func nonNilIDs(ids []int64) []int64 {
	if ids == nil {
		return []int64{}
	}
	return ids
}
