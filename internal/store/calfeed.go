package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
)

// CalFeed is a person's private calendar link for the meal plan.
type CalFeed struct {
	UserID    int64             `db:"user_id" json:"-"`
	Token     string            `db:"token" json:"token"`
	TimesJSON string            `db:"times" json:"-"`
	Times     map[string]string `db:"-" json:"times"` // meal → "18:00"
	Events    bool              `db:"events" json:"events"`
}

// MealTimes are when meals show in the calendar unless a person chooses.
var MealTimes = map[string]string{"breakfast": "08:00", "lunch": "12:00", "snack": "15:00", "dinner": "18:00"}

var clock = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func (f *CalFeed) unpack() {
	f.Times = map[string]string{}
	_ = json.Unmarshal([]byte(f.TimesJSON), &f.Times)
	for meal, t := range MealTimes {
		if !clock.MatchString(f.Times[meal]) {
			f.Times[meal] = t
		}
	}
}

// CalFeedFor is a person's calendar link; ok is false when they have none.
func (s *Store) CalFeedFor(userID int64) (CalFeed, bool, error) {
	var f CalFeed
	err := s.DB.Get(&f, `SELECT user_id, token, times, events FROM cal_feeds WHERE user_id = ?`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return f, false, nil
	}
	f.unpack()
	return f, err == nil, err
}

// CalFeedByToken finds the link a calendar app asks with.
func (s *Store) CalFeedByToken(token string) (CalFeed, error) {
	var f CalFeed
	err := s.DB.Get(&f, `SELECT user_id, token, times, events FROM cal_feeds WHERE token = ?`, token)
	if errors.Is(err, sql.ErrNoRows) {
		return f, ErrNotFound
	}
	f.unpack()
	return f, err
}

// NewCalFeed makes a person's link, or a new one in place of the old (which
// stops working); their times are kept.
func (s *Store) NewCalFeed(userID int64) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	_, err = s.DB.Exec(`INSERT INTO cal_feeds (user_id, token) VALUES (?, ?)
		ON CONFLICT (user_id) DO UPDATE SET token = excluded.token`, userID, token)
	return token, err
}

// SetCalFeed keeps the times meals show at (only real meals and times) and
// whether events show too.
func (s *Store) SetCalFeed(userID int64, times map[string]string, events bool) error {
	clean := map[string]string{}
	for meal := range MealTimes {
		if clock.MatchString(times[meal]) {
			clean[meal] = times[meal]
		}
	}
	data, _ := json.Marshal(clean)
	res, err := s.DB.Exec(`UPDATE cal_feeds SET times = ?, events = ? WHERE user_id = ?`, string(data), events, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCalFeed turns a person's link off.
func (s *Store) DeleteCalFeed(userID int64) error {
	_, err := s.DB.Exec(`DELETE FROM cal_feeds WHERE user_id = ?`, userID)
	return err
}

// RecipeTitles maps each recipe to its title.
func (s *Store) RecipeTitles() (map[int64]string, error) {
	var rows []struct {
		ID    int64  `db:"id"`
		Title string `db:"title"`
	}
	if err := s.DB.Select(&rows, `SELECT id, title FROM recipes`); err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(rows))
	for _, r := range rows {
		out[r.ID] = r.Title
	}
	return out, nil
}
