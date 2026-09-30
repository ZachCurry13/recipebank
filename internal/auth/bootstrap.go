package auth

import (
	"log"

	"github.com/zachcurry13/recipebank/internal/store"
)

// Bootstrap handles a brand-new database. Normally it does nothing and the
// first person to open the web page creates the admin account there. If
// RECIPEBANK_ADMIN_PASSWORD is set (optional, for automated installs), that
// admin is created up front instead.
func Bootstrap(s *store.Store, username, password string) error {
	n, err := s.CountUsers()
	if err != nil || n > 0 {
		return err
	}
	if password == "" {
		log.Printf("no accounts yet: open RecipeBank in your browser to create the admin account")
		return nil
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	if _, err := s.CreateUser(username, hash, store.RoleAdmin); err != nil {
		return err
	}
	log.Printf("created admin user %q from RECIPEBANK_ADMIN_PASSWORD", username)
	return nil
}
