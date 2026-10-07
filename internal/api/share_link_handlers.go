package api

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/zachcurry13/recipebank/internal/auth"
)

// Share links: a page showing one recipe (title, photo, ingredients, steps,
// notes) to someone without an account, for a week. No names, allergies,
// ratings or card photos are on it.

const shareDays = 7

// handleGetShare reports a recipe's open link, if any.
func (s *Server) handleGetShare(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	sh, err := s.Store.ActiveShare(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"share": sh})
}

// handleShare makes a link (or returns the open one).
func (s *Server) handleShareRecipe(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.Store.Recipe(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	sh, err := s.Store.ActiveShare(id)
	if err == nil && sh == nil {
		sh, err = s.Store.CreateShare(id, auth.UserFrom(r).Username, shareDays)
	}
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"share": sh})
}

func (s *Server) handleStopSharing(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.StopSharing(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type sharePage struct {
	Gone                                bool
	Title, Summary, Facts, Photo, Notes string
	Until                               string
	Ingredients, Steps                  []string
}

// handleSharePage shows the shared recipe (no sign-in needed).
func (s *Server) handleSharePage(w http.ResponseWriter, r *http.Request) {
	s.shareOnce.Do(func() {
		s.shareTmpl, _ = template.ParseFS(s.Web, "share.tmpl.html")
	})
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	if s.shareTmpl == nil {
		http.Error(w, "not available", http.StatusInternalServerError)
		return
	}
	token := chi.URLParam(r, "token")
	page := sharePage{Gone: true}
	status := http.StatusGone
	// While sharing is turned off (Admin → Features), every link has run out.
	if sh, err := s.Store.ShareByToken(token); err == nil && sh != nil && s.Store.FeatureOn("share") {
		if rc, err := s.Store.Recipe(sh.RecipeID); err == nil {
			status = http.StatusOK
			page = sharePage{Title: rc.Title, Summary: rc.Summary, Notes: rc.Notes}
			var facts []string
			if rc.Servings > 0 {
				facts = append(facts, "Serves "+strconv.FormatFloat(rc.Servings, 'f', -1, 64))
			}
			if rc.TotalMin > 0 {
				facts = append(facts, fmt.Sprintf("%d min", rc.TotalMin))
			}
			page.Facts = strings.Join(facts, " · ")
			if rc.Photo != "" {
				page.Photo = "/s/" + token + "/photo"
			}
			for _, in := range rc.Ingredients {
				page.Ingredients = append(page.Ingredients, in.Line)
			}
			for _, st := range rc.Steps {
				page.Steps = append(page.Steps, st.Text)
			}
			if t, err := time.Parse(time.RFC3339, sh.ExpiresAt); err == nil {
				page.Until = t.Local().Format("Monday, January 2")
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = s.shareTmpl.Execute(w, page)
}

// handleSharePhoto sends the shared recipe's own photo (never card photos).
func (s *Server) handleSharePhoto(w http.ResponseWriter, r *http.Request) {
	sh, err := s.Store.ShareByToken(chi.URLParam(r, "token"))
	if err != nil || sh == nil || !s.Store.FeatureOn("share") {
		http.NotFound(w, r)
		return
	}
	rc, err := s.Store.Recipe(sh.RecipeID)
	if err != nil || rc.Photo == "" {
		http.NotFound(w, r)
		return
	}
	p, ok := s.photoPath(rc.Photo)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex")
	http.ServeFile(w, r, p)
}
