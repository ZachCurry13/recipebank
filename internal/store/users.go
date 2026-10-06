package store

import (
	"database/sql"
	"errors"
	"fmt"
)

const userCols = `id, username, password_hash, role, person_id, units, theme, seen_version, created_at`

func (s *Store) CountUsers() (int, error) {
	var n int
	err := s.DB.Get(&n, `SELECT COUNT(*) FROM users`)
	return n, err
}

func (s *Store) ListUsers() ([]User, error) {
	us := []User{}
	err := s.DB.Select(&us, `SELECT `+userCols+` FROM users ORDER BY username`)
	return us, err
}

func (s *Store) UserByID(id int64) (*User, error) {
	return s.getUser(`SELECT `+userCols+` FROM users WHERE id = ?`, id)
}

func (s *Store) UserByName(name string) (*User, error) {
	return s.getUser(`SELECT `+userCols+` FROM users WHERE username = ?`, name)
}

func (s *Store) getUser(q string, arg any) (*User, error) {
	var u User
	if err := s.DB.Get(&u, q, arg); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// CreateUser inserts an account.
func (s *Store) CreateUser(username, hash, role string) (*User, error) {
	if !ValidRole(role) {
		return nil, fmt.Errorf("invalid role %q", role)
	}
	res, err := s.DB.Exec(`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`, username, hash, role)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.UserByID(id)
}

// ErrSetupDone is returned when first-run setup is attempted but an account
// already exists.
var ErrSetupDone = errors.New("setup has already been completed")

// CreateFirstAdmin creates an admin only while the users table is empty. The
// check and insert are one statement, so two browsers racing through the
// setup page can't both create an admin.
func (s *Store) CreateFirstAdmin(username, hash string) (*User, error) {
	res, err := s.DB.Exec(`INSERT INTO users (username, password_hash, role)
		SELECT ?, ?, 'admin' WHERE NOT EXISTS (SELECT 1 FROM users)`, username, hash)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrSetupDone
	}
	return s.UserByName(username)
}

// UpdateUser saves an account's role and linked eating profile.
func (s *Store) UpdateUser(id int64, role string, personID *int64) error {
	if !ValidRole(role) {
		return fmt.Errorf("invalid role %q", role)
	}
	_, err := s.DB.Exec(`UPDATE users SET role = ?, person_id = ? WHERE id = ?`, role, personID, id)
	return err
}

// SetPrefs saves someone's own units and theme.
func (s *Store) SetPrefs(id int64, units, theme string) error {
	_, err := s.DB.Exec(`UPDATE users SET units = ?, theme = ? WHERE id = ?`, units, theme, id)
	return err
}

func (s *Store) SetPassword(id int64, hash string) error {
	_, err := s.DB.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, id)
	return err
}

func (s *Store) DeleteUser(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *Store) CountAdmins() (int, error) {
	var n int
	err := s.DB.Get(&n, `SELECT COUNT(*) FROM users WHERE role = 'admin'`)
	return n, err
}

// SetSeen records the version whose welcome or "What's new" someone has seen.
func (s *Store) SetSeen(id int64, version string) error {
	_, err := s.DB.Exec(`UPDATE users SET seen_version = ? WHERE id = ?`, version, id)
	return err
}
