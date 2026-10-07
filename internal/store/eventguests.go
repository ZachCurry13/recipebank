package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// An event's guest link: guests open it without an account to say they're
// coming, with their allergies and diets, and what they're bringing.

// EventLink is an event's link for guests, until it runs out.
type EventLink struct {
	Token     string `db:"token" json:"token"`
	EventID   int64  `db:"event_id" json:"event_id"`
	ExpiresAt string `db:"expires_at" json:"expires_at"` // RFC 3339, UTC
	CreatedBy string `db:"created_by" json:"-"`
}

// EventGuest is someone coming who added themselves through the link.
type EventGuest struct {
	ID            int64             `db:"id" json:"id"`
	EventID       int64             `db:"event_id" json:"-"`
	Name          string            `db:"name" json:"name"`
	AllergiesJSON string            `db:"allergies" json:"-"`
	Allergies     map[string]string `db:"-" json:"allergies"` // allergen key → severity
	DietsText     string            `db:"diets" json:"-"`
	Diets         []string          `db:"-" json:"diets"`
	Note          string            `db:"note" json:"note"`
	Key           string            `db:"edit_key" json:"-"` // their browser keeps it, to change what they said
}

func newToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// EventLinkFor is an event's link that hasn't run out, or nil.
func (s *Store) EventLinkFor(eventID int64) (*EventLink, error) {
	var out []EventLink
	err := s.DB.Select(&out, `SELECT token, event_id, expires_at, created_by FROM event_links WHERE event_id = ? AND expires_at > ?
		ORDER BY expires_at DESC LIMIT 1`, eventID, now())
	if err != nil || len(out) == 0 {
		return nil, err
	}
	return &out[0], nil
}

// CreateEventLink makes a new link for an event, open until expires.
func (s *Store) CreateEventLink(eventID int64, by string, expires time.Time) (*EventLink, error) {
	tok, err := newToken()
	if err != nil {
		return nil, err
	}
	l := &EventLink{Token: tok, EventID: eventID, ExpiresAt: expires.UTC().Format(time.RFC3339), CreatedBy: by}
	_, err = s.DB.Exec(`INSERT INTO event_links (token, event_id, expires_at, created_by) VALUES (?, ?, ?, ?)`, l.Token, l.EventID, l.ExpiresAt, l.CreatedBy)
	return l, err
}

// EventLinkByToken finds a link that hasn't run out; nil when it's unknown or over.
func (s *Store) EventLinkByToken(token string) (*EventLink, error) {
	var out []EventLink
	err := s.DB.Select(&out, `SELECT token, event_id, expires_at, created_by FROM event_links WHERE token = ? AND expires_at > ?`, token, now())
	if err != nil || len(out) == 0 {
		return nil, err
	}
	return &out[0], nil
}

// StopEventLink ends every link to an event (the guests who came through it stay).
func (s *Store) StopEventLink(eventID int64) error {
	_, err := s.DB.Exec(`DELETE FROM event_links WHERE event_id = ?`, eventID)
	return err
}

func (g *EventGuest) decode() {
	g.Allergies = map[string]string{}
	_ = json.Unmarshal([]byte(g.AllergiesJSON), &g.Allergies)
	g.Diets = SplitList(g.DietsText)
}

// EventGuests are the guests who added themselves to an event.
func (s *Store) EventGuests(eventID int64) ([]EventGuest, error) {
	out := []EventGuest{}
	err := s.DB.Select(&out, `SELECT id, event_id, name, allergies, diets, note, edit_key FROM event_guests WHERE event_id = ? ORDER BY id`, eventID)
	for i := range out {
		out[i].decode()
	}
	return out, err
}

// GuestByKey is the guest whose browser holds key, or ErrNotFound.
func (s *Store) GuestByKey(eventID int64, key string) (*EventGuest, error) {
	var g EventGuest
	err := s.DB.Get(&g, `SELECT id, event_id, name, allergies, diets, note, edit_key FROM event_guests WHERE event_id = ? AND edit_key = ?`, eventID, key)
	if errors.Is(err, sql.ErrNoRows) || key == "" {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	g.decode()
	return &g, nil
}

// SaveGuest adds a guest (ID 0, given a new key) or changes one.
func (s *Store) SaveGuest(g *EventGuest) error {
	al, _ := json.Marshal(g.Allergies)
	if g.ID == 0 {
		key, err := newToken()
		if err != nil {
			return err
		}
		res, err := s.DB.Exec(`INSERT INTO event_guests (event_id, name, allergies, diets, note, edit_key) VALUES (?, ?, ?, ?, ?, ?)`,
			g.EventID, strings.TrimSpace(g.Name), string(al), strings.Join(g.Diets, ","), strings.TrimSpace(g.Note), key)
		if err != nil {
			return err
		}
		g.ID, _ = res.LastInsertId()
		g.Key = key
		return nil
	}
	_, err := s.DB.Exec(`UPDATE event_guests SET name = ?, allergies = ?, diets = ?, note = ? WHERE id = ? AND event_id = ?`,
		strings.TrimSpace(g.Name), string(al), strings.Join(g.Diets, ","), strings.TrimSpace(g.Note), g.ID, g.EventID)
	if err == nil {
		// What they bring says their name.
		_, err = s.DB.Exec(`UPDATE event_dishes SET brings = ? WHERE guest_id = ?`, strings.TrimSpace(g.Name), g.ID)
	}
	return err
}

// DeleteGuest removes a guest and the dishes they added.
func (s *Store) DeleteGuest(eventID, id int64) error {
	_, err := s.DB.Exec(`DELETE FROM event_guests WHERE id = ? AND event_id = ?`, id, eventID)
	return err
}

// CountGuestDishes counts the dishes guests added to an event (to keep a link from being flooded).
func (s *Store) CountGuestDishes(eventID int64) (int, error) {
	var n int
	err := s.DB.Get(&n, `SELECT COUNT(*) FROM event_dishes WHERE event_id = ? AND guest_id IS NOT NULL`, eventID)
	return n, err
}
