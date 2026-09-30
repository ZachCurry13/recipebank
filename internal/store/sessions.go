package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// --- sessions ---

// CreateSession stores a new session that expires after lifetime seconds.
// remember=false marks a "this browser session only" sign-in.
func (s *Store) CreateSession(tokenHash string, userID int64, lifetime int, remember bool) error {
	_, err := s.DB.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at, remember)
		VALUES (?, ?, datetime('now', ?), ?)`, tokenHash, userID, fmt.Sprintf("+%d seconds", lifetime), remember)
	return err
}

// Session is a live session's user plus what's needed to renew it.
type Session struct {
	User        *User
	SecondsLeft int
	Remember    bool
}

// SessionByToken resolves a live (unexpired) session.
func (s *Store) SessionByToken(tokenHash string) (*Session, error) {
	var row struct {
		UserID   int64 `db:"user_id"`
		Left     int   `db:"secs_left"`
		Remember bool  `db:"remember"`
	}
	err := s.DB.Get(&row, `SELECT user_id, remember,
		CAST(strftime('%s', expires_at) - strftime('%s', 'now') AS INTEGER) AS secs_left
		FROM sessions WHERE token_hash = ? AND expires_at > datetime('now')`, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u, err := s.UserByID(row.UserID)
	if err != nil {
		return nil, err
	}
	return &Session{User: u, SecondsLeft: row.Left, Remember: row.Remember}, nil
}

// ExtendSession pushes a session's expiry to lifetime seconds from now.
func (s *Store) ExtendSession(tokenHash string, lifetime int) error {
	_, err := s.DB.Exec(`UPDATE sessions SET expires_at = datetime('now', ?) WHERE token_hash = ?`,
		fmt.Sprintf("+%d seconds", lifetime), tokenHash)
	return err
}

func (s *Store) DeleteSession(tokenHash string) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (s *Store) DeleteUserSessions(userID int64) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func (s *Store) PurgeExpiredSessions() error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE expires_at <= datetime('now')`)
	return err
}
