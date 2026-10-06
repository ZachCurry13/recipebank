// Package store is RecipeBank's database layer. SQLite runs with one
// connection: read rows into slices first, never query inside a rows loop,
// and never call another store method while a transaction is open.
package store

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

type Store struct {
	DB *sqlx.DB
}

func New(db *sqlx.DB) *Store { return &Store{DB: db} }

var ErrNotFound = errors.New("not found")

// Roles: admins run everything, editors (the other parent) manage recipes and
// people, kids read recipes and cook.
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleKid    = "kid"
)

// ValidRole reports whether r is a known role.
func ValidRole(r string) bool { return r == RoleAdmin || r == RoleEditor || r == RoleKid }

type User struct {
	ID           int64  `db:"id" json:"id"`
	Username     string `db:"username" json:"username"`
	PasswordHash string `db:"password_hash" json:"-"`
	Role         string `db:"role" json:"role"`
	PersonID     *int64 `db:"person_id" json:"person_id"`       // their own eating profile
	Units        string `db:"units" json:"units"`               // "" = the house default, "us", "metric"
	Theme        string `db:"theme" json:"theme"`               // "" = match the device, "dark", "light"
	SeenVersion  string `db:"seen_version" json:"seen_version"` // "" = hasn't seen the welcome yet
	CreatedAt    string `db:"created_at" json:"created_at"`
}

func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// CanManage is true for admins and editors.
func (u *User) CanManage() bool { return u.Role == RoleAdmin || u.Role == RoleEditor }
