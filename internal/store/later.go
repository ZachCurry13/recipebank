package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// "Read it later": card photos and pasted recipes no AI could read when they
// were added wait here, are read in the background, and are kept as a draft
// until someone checks and saves them.

// Later is one card or pasted recipe waiting for an AI.
type Later struct {
	ID         int64    `db:"id" json:"id"`
	Kind       string   `db:"kind" json:"kind"` // photo | text
	Area       string   `db:"area" json:"area"`
	PhotosJSON string   `db:"photos" json:"-"`
	Photos     []string `db:"-" json:"photos"`
	Text       string   `db:"text" json:"-"`
	Source     string   `db:"source" json:"source"` // JSON the web app keeps (a cookbook page, a card from the pile)
	Status     string   `db:"status" json:"status"`
	Draft      string   `db:"draft" json:"-"` // the recipe read, as JSON
	Title      string   `db:"title" json:"title"`
	Error      string   `db:"error" json:"error"`
	Tries      int      `db:"tries" json:"tries"`
	NextTry    string   `db:"next_try" json:"next_try"`
	GiveUp     string   `db:"give_up" json:"-"`
	UserID     int64    `db:"user_id" json:"-"`
	AddedBy    string   `db:"added_by" json:"added_by"`
	CreatedAt  string   `db:"created_at" json:"created_at"`
}

const laterCols = `id, kind, area, photos, text, source, status, draft, title, error, tries, next_try, give_up, user_id, added_by, created_at`

func (l *Later) unpack() {
	l.Photos = []string{}
	_ = json.Unmarshal([]byte(l.PhotosJSON), &l.Photos)
}

// AddLater puts a card or pasted recipe on the list, to be read first at next.
func (s *Store) AddLater(l Later, next time.Time, keep time.Duration) (int64, error) {
	photos, _ := json.Marshal(l.Photos)
	if l.Photos == nil {
		photos = []byte("[]")
	}
	res, err := s.DB.Exec(`INSERT INTO later (kind, area, photos, text, source, next_try, give_up, user_id, added_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, l.Kind, l.Area, string(photos), l.Text, l.Source,
		next.UTC().Format(time.RFC3339), next.Add(keep).UTC().Format(time.RFC3339), l.UserID, l.AddedBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Laters lists everything on the list, oldest first.
func (s *Store) Laters() ([]Later, error) {
	var out []Later
	if err := s.DB.Select(&out, `SELECT `+laterCols+` FROM later ORDER BY id`); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].unpack()
	}
	return out, nil
}

// LaterByID finds one item.
func (s *Store) LaterByID(id int64) (Later, error) {
	var l Later
	err := s.DB.Get(&l, `SELECT `+laterCols+` FROM later WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return l, ErrNotFound
	}
	l.unpack()
	return l, err
}

// LaterDue is the waiting item whose turn came first, if any is due by now.
func (s *Store) LaterDue(now time.Time) (Later, bool, error) {
	var l Later
	err := s.DB.Get(&l, `SELECT `+laterCols+` FROM later WHERE status = 'waiting' AND next_try <= ?
		ORDER BY next_try, id LIMIT 1`, now.UTC().Format(time.RFC3339))
	if errors.Is(err, sql.ErrNoRows) {
		return l, false, nil
	}
	l.unpack()
	return l, err == nil, err
}

// LaterRead keeps what the AI read: the item is ready to be checked.
func (s *Store) LaterRead(id int64, title, draft string) error {
	_, err := s.DB.Exec(`UPDATE later SET status = 'ready', title = ?, draft = ?, error = '', tries = tries + 1 WHERE id = ?`,
		title, draft, id)
	return err
}

// LaterRetry notes a try that didn't work; the item waits until next.
func (s *Store) LaterRetry(id int64, msg string, next time.Time) error {
	_, err := s.DB.Exec(`UPDATE later SET error = ?, tries = tries + 1, next_try = ? WHERE id = ?`,
		msg, next.UTC().Format(time.RFC3339), id)
	return err
}

// LaterFailed stops trying: msg says why.
func (s *Store) LaterFailed(id int64, msg string) error {
	_, err := s.DB.Exec(`UPDATE later SET status = 'failed', error = ?, tries = tries + 1 WHERE id = ?`, msg, id)
	return err
}

// LaterNow puts an item back in line to be read at once, for two more days.
func (s *Store) LaterNow(id int64, now time.Time, keep time.Duration) error {
	res, err := s.DB.Exec(`UPDATE later SET status = 'waiting', next_try = ?, give_up = ?, draft = '', title = '' WHERE id = ?`,
		now.UTC().Format(time.RFC3339), now.Add(keep).UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteLater takes an item off the list (its photos go with the daily clean-up
// unless a recipe kept them).
func (s *Store) DeleteLater(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM later WHERE id = ?`, id)
	return err
}

// laterPhotos are the photos still waiting, so the clean-up keeps them.
func (s *Store) laterPhotos(used map[string]bool) error {
	var lists []string
	if err := s.DB.Select(&lists, `SELECT photos FROM later`); err != nil {
		return err
	}
	for _, j := range lists {
		var names []string
		_ = json.Unmarshal([]byte(j), &names)
		for _, n := range names {
			used[n] = true
		}
	}
	return nil
}
