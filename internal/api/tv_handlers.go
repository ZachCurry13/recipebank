package api

import (
	"crypto/rand"
	"crypto/subtle"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/zachcurry13/recipebank/internal/auth"
)

// Kitchen screen: a TV or tablet opens /tv (no sign-in) and shows a short
// code. In cook mode, a phone enters the code and sends each step to it, big.
// The screen sees only what the phone sends (the step, its ingredients and the
// timers), never the account. Screens live in memory and end after 12 hours.

const (
	tvLife   = 12 * time.Hour
	tvMax    = 100
	tvLetter = "ACDEFGHJKLMNPQRTUVWXY34679" // no look-alikes (0/O, 1/I, 5/S, 2/Z, 8/B)
)

type tvItem struct {
	Amount string `json:"amount"`
	Name   string `json:"name"`
}

type tvTimer struct {
	Label string `json:"label"`
	Left  int    `json:"left"` // seconds left when sent
	Done  bool   `json:"done"`
}

// tvView is what the phone sends: all the screen ever shows.
type tvView struct {
	Title   string    `json:"title"`
	At      int       `json:"at"`
	Total   int       `json:"total"`
	Section string    `json:"section"`
	Text    string    `json:"text"`
	Items   []tvItem  `json:"items"`
	Timers  []tvTimer `json:"timers"`
}

type tvScreen struct {
	secret  string
	born    time.Time
	paired  bool
	view    *tvView
	version int // changes with each thing sent, so the screen redraws only when needed
}

type tvHub struct {
	mu      sync.Mutex
	screens map[string]*tvScreen
}

func (h *tvHub) sweep() {
	for code, sc := range h.screens {
		if time.Since(sc.born) > tvLife {
			delete(h.screens, code)
		}
	}
}

func randomFrom(alphabet string, n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

func (s *Server) handleTVPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	page, err := fs.ReadFile(s.Web, "tv.html")
	if err != nil {
		http.Error(w, "not available", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(page)
}

// handleTVNew starts a screen: a code to type on the phone, and a secret only
// this screen keeps, so nobody else can read it by guessing the code.
func (s *Server) handleTVNew(w http.ResponseWriter, r *http.Request) {
	if !s.tvNew.allow(clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "too many screens: wait a few minutes")
		return
	}
	s.tv.mu.Lock()
	defer s.tv.mu.Unlock()
	if s.tv.screens == nil {
		s.tv.screens = map[string]*tvScreen{}
	}
	s.tv.sweep()
	if len(s.tv.screens) >= tvMax {
		writeErr(w, http.StatusServiceUnavailable, "too many screens are open right now")
		return
	}
	code := randomFrom(tvLetter, 6)
	for s.tv.screens[code] != nil {
		code = randomFrom(tvLetter, 6)
	}
	secret := randomFrom("abcdefghijklmnopqrstuvwxyz0123456789", 24)
	s.tv.screens[code] = &tvScreen{secret: secret, born: time.Now()}
	writeJSON(w, http.StatusOK, map[string]string{"code": code, "secret": secret})
}

// handleTVScreen is what the screen asks every second or two.
func (s *Server) handleTVScreen(w http.ResponseWriter, r *http.Request) {
	code, secret := strings.ToUpper(r.URL.Query().Get("code")), r.URL.Query().Get("secret")
	s.tv.mu.Lock()
	defer s.tv.mu.Unlock()
	sc := s.tv.screens[code]
	if sc == nil || time.Since(sc.born) > tvLife || subtle.ConstantTimeCompare([]byte(sc.secret), []byte(secret)) != 1 {
		writeErr(w, http.StatusNotFound, "this screen ended: reload to get a new code")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"paired": sc.paired, "view": sc.view, "version": sc.version})
}

// tvFind is the phone's screen for a code typed in.
func (s *Server) tvFind(w http.ResponseWriter, code string) *tvScreen {
	sc := s.tv.screens[strings.ToUpper(strings.TrimSpace(code))]
	if sc == nil || time.Since(sc.born) > tvLife {
		writeErr(w, http.StatusNotFound, "no screen has that code: check the code on the TV")
		return nil
	}
	return sc
}

func (s *Server) handleTVPair(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &body, 1<<10) {
		return
	}
	if !s.tvTries.allow(auth.UserFrom(r).Username) { // guessing codes
		writeErr(w, http.StatusTooManyRequests, "too many tries: wait a few minutes")
		return
	}
	s.tv.mu.Lock()
	defer s.tv.mu.Unlock()
	sc := s.tvFind(w, body.Code)
	if sc == nil {
		return
	}
	sc.paired, sc.version = true, sc.version+1
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func clip(s string, n int) string {
	if r := []rune(strings.TrimSpace(s)); len(r) > n {
		return string(r[:n])
	}
	return strings.TrimSpace(s)
}

func (s *Server) handleTVShow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
		View tvView `json:"view"`
	}
	if !readJSON(w, r, &body, 32<<10) {
		return
	}
	v := body.View
	v.Title, v.Section, v.Text = clip(v.Title, 120), clip(v.Section, 80), clip(v.Text, 2000)
	v.At, v.Total = max(v.At, 0), min(max(v.Total, 0), 500)
	if len(v.Items) > 20 {
		v.Items = v.Items[:20]
	}
	for i := range v.Items {
		v.Items[i].Amount, v.Items[i].Name = clip(v.Items[i].Amount, 40), clip(v.Items[i].Name, 120)
	}
	if len(v.Timers) > 8 {
		v.Timers = v.Timers[:8]
	}
	for i := range v.Timers {
		v.Timers[i].Label, v.Timers[i].Left = clip(v.Timers[i].Label, 60), min(max(v.Timers[i].Left, 0), 24*3600)
	}
	s.tv.mu.Lock()
	defer s.tv.mu.Unlock()
	sc := s.tvFind(w, body.Code)
	if sc == nil {
		return
	}
	if !sc.paired {
		writeErr(w, http.StatusConflict, "connect the screen first")
		return
	}
	sc.view, sc.version = &v, sc.version+1
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleTVStop clears the screen (the phone left cook mode).
func (s *Server) handleTVStop(w http.ResponseWriter, r *http.Request) {
	s.tv.mu.Lock()
	defer s.tv.mu.Unlock()
	if sc := s.tv.screens[strings.ToUpper(chi.URLParam(r, "code"))]; sc != nil {
		sc.view, sc.version = nil, sc.version+1
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
