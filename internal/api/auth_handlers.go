package api

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/store"
	"github.com/zachcurry13/recipebank/internal/version"
)

var usernameRE = regexp.MustCompile(`^[A-Za-z0-9._-]{2,40}$`)

// meResponse is the signed-in person as the web page sees them.
type meResponse struct {
	*store.User
	Version string `json:"version"`
}

func (s *Server) me(u *store.User) meResponse { return meResponse{User: u, Version: version.Version} }

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.logins.allow(remoteHost(r)) {
		writeErr(w, http.StatusTooManyRequests, "too many login attempts; try again in a few minutes")
		return
	}
	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Remember *bool  `json:"remember"` // default true
	}{}
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	remember := body.Remember == nil || *body.Remember
	u, err := s.Auth.Login(w, r, strings.TrimSpace(body.Username), body.Password, remember)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	writeJSON(w, http.StatusOK, s.me(u))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.Auth.Logout(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.me(auth.UserFrom(r)))
}

// handlePrefs saves someone's own units and theme.
func (s *Server) handlePrefs(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Units string `json:"units"`
		Theme string `json:"theme"`
	}
	if !readJSON(w, r, &body, 1<<10) {
		return
	}
	if (body.Units != "" && body.Units != "us" && body.Units != "metric") ||
		(body.Theme != "" && body.Theme != "dark" && body.Theme != "light") {
		writeErr(w, http.StatusBadRequest, "units must be us or metric; theme dark or light")
		return
	}
	u := auth.UserFrom(r)
	if err := s.Store.SetPrefs(u.ID, body.Units, body.Theme); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFrom(r)
	var body struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	if !auth.CheckPassword(u.PasswordHash, body.Current) {
		writeErr(w, http.StatusForbidden, "current password is incorrect")
		return
	}
	hash, err := auth.HashPassword(body.New)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Store.SetPassword(u.ID, hash); err != nil {
		writeStoreErr(w, err)
		return
	}
	// Sign out everywhere else, then start a fresh session here.
	_ = s.Store.DeleteUserSessions(u.ID)
	if _, err := s.Auth.Login(w, r, u.Username, body.New, true); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleSetupStatus tells the web page whether to show "Create your admin
// account" (true only while no accounts exist).
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	n, err := s.Store.CountUsers()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"needed": n == 0})
}

// handleSetup creates the first admin from the web page and signs them in.
// It stops working as soon as any account exists.
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if !s.logins.allow(remoteHost(r)) {
		writeErr(w, http.StatusTooManyRequests, "too many attempts; try again in a few minutes")
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if !usernameRE.MatchString(body.Username) {
		writeErr(w, http.StatusBadRequest, "username must be 2-40 letters, digits, '.', '_' or '-'")
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := s.Store.CreateFirstAdmin(body.Username, hash); err != nil {
		if errors.Is(err, store.ErrSetupDone) {
			writeErr(w, http.StatusConflict, "setup is already done; please sign in")
			return
		}
		writeStoreErr(w, err)
		return
	}
	u, err := s.Auth.Login(w, r, body.Username, body.Password, true)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.me(u))
}
