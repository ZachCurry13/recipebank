package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

func (s *Server) handleListCooks(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	cooks, err := s.Store.CooksFor(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cooks": cooks})
}

// handleAddCook records that a recipe was cooked ({"cooked_on": "2026-10-06",
// "note": "", "thumbs": {"3": 1, "4": -1}}); anyone may log it.
func (s *Server) handleAddCook(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var body struct {
		CookedOn string        `json:"cooked_on"`
		Note     string        `json:"note"`
		Thumbs   map[int64]int `json:"thumbs"`
	}
	if !ok || !readJSON(w, r, &body, 8<<10) {
		return
	}
	if _, err := s.Store.Recipe(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	if _, err := time.Parse(day, body.CookedOn); err != nil {
		writeErr(w, http.StatusBadRequest, "pick the day it was cooked")
		return
	}
	people, err := s.Store.ListPeople()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	known := map[int64]bool{}
	for _, p := range people {
		known[p.ID] = true
	}
	thumbs := map[int64]int{}
	for p, t := range body.Thumbs {
		if known[p] && (t == 1 || t == -1) {
			thumbs[p] = t
		}
	}
	note := strings.TrimSpace(body.Note)
	if len(note) > 500 {
		note = note[:500]
	}
	if _, err := s.Store.AddCook(store.Cook{RecipeID: id, CookedOn: body.CookedOn, Note: note,
		AddedBy: auth.UserFrom(r).Username, Thumbs: thumbs}); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.handleListCooks(w, r)
}

func (s *Server) handleDeleteCook(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.DeleteCook(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// likeTier sorts ideas: liked by the people eating (and not just cooked)
// first, then untried, then cooked in the last 6 days, then anything someone
// eating gave a 👎 overall.
func likeTier(l store.Likes, id int64, diners []safety.Person, cutoff string) int {
	net := 0
	for _, p := range diners {
		v := l.Net[id][p.ID]
		if v < 0 {
			return 3
		}
		net += v
	}
	if last := l.Last[id]; last != "" && last >= cutoff {
		return 2
	}
	if net > 0 {
		return 0
	}
	return 1
}
