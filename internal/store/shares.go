package store

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

// Share is a link showing one recipe to anyone who has it, until it expires.
type Share struct {
	Token     string `db:"token" json:"token"`
	RecipeID  int64  `db:"recipe_id" json:"recipe_id"`
	ExpiresAt string `db:"expires_at" json:"expires_at"` // RFC 3339, UTC
	CreatedBy string `db:"created_by" json:"-"`
}

const shareCols = `token, recipe_id, expires_at, created_by`

// ActiveShare returns a recipe's link that hasn't expired, if any.
func (s *Store) ActiveShare(recipeID int64) (*Share, error) {
	var out []Share
	err := s.DB.Select(&out, `SELECT `+shareCols+` FROM shares WHERE recipe_id = ? AND expires_at > ?
		ORDER BY expires_at DESC LIMIT 1`, recipeID, time.Now().UTC().Format(time.RFC3339))
	if err != nil || len(out) == 0 {
		return nil, err
	}
	return &out[0], nil
}

// CreateShare makes a new link for a recipe, good for days.
func (s *Store) CreateShare(recipeID int64, by string, days int) (*Share, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	sh := &Share{Token: base64.RawURLEncoding.EncodeToString(b), RecipeID: recipeID, CreatedBy: by,
		ExpiresAt: time.Now().UTC().AddDate(0, 0, days).Format(time.RFC3339)}
	_, err := s.DB.Exec(`INSERT INTO shares (token, recipe_id, expires_at, created_by) VALUES (?, ?, ?, ?)`,
		sh.Token, sh.RecipeID, sh.ExpiresAt, sh.CreatedBy)
	return sh, err
}

// ShareByToken finds a link that hasn't expired; nil when it's unknown or over.
func (s *Store) ShareByToken(token string) (*Share, error) {
	var out []Share
	err := s.DB.Select(&out, `SELECT `+shareCols+` FROM shares WHERE token = ? AND expires_at > ?`,
		token, time.Now().UTC().Format(time.RFC3339))
	if err != nil || len(out) == 0 {
		return nil, err
	}
	return &out[0], nil
}

// StopSharing removes every link to a recipe.
func (s *Store) StopSharing(recipeID int64) error {
	_, err := s.DB.Exec(`DELETE FROM shares WHERE recipe_id = ?`, recipeID)
	return err
}
