package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

// Teaching the photo reader: corrections made while checking a recipe
// against its card become short hints sent with the next cards.

// readingHints are the family's most common corrections, for the reader.
func (s *Server) readingHints() []string {
	list, err := s.Store.ReadingHints(12)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, h := range list {
		out = append(out, fmt.Sprintf("%q was really %q", h.Wrong, h.Right))
	}
	return out
}

// learnFrom returns the lines the AI read that a save may correct: on a
// photo recipe's first save, the AI's copy of the card; later, the saved
// lines while it still needs checking (and the AI hasn't read it again).
func (s *Server) learnFrom(old, rc *recipe.Recipe) []recipe.Ingredient {
	switch {
	case old == nil && rc.SourceKind == "photo" && rc.AIReading != "":
		read, _ := recipe.FromText(rc.AIReading)
		return read.Ingredients
	case old != nil && old.SourceKind == "photo" && old.NeedsReview && old.AIReading == rc.AIReading:
		return old.Ingredients
	}
	return nil
}

func (s *Server) learnMisreads(before, after []recipe.Ingredient) {
	for _, m := range recipe.Misreads(before, after) {
		_ = s.Store.LearnHint(m[0], m[1]) // a hint that can't be saved is no reason to fail the save
	}
}

func (s *Server) handleListHints(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ReadingHints(0)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hints": list})
}

// handleAddHint adds a hint by hand ("T." is a tablespoon, not a teaspoon).
func (s *Server) handleAddHint(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Wrong string `json:"wrong"`
		Right string `json:"right"`
	}
	if !readJSON(w, r, &body, 1<<10) {
		return
	}
	wrong, right := strings.TrimSpace(body.Wrong), strings.TrimSpace(body.Right)
	if wrong == "" || right == "" || wrong == right || len(wrong) > 60 || len(right) > 60 {
		writeErr(w, http.StatusBadRequest, "write what the AI read and what the card really says (up to 60 letters each)")
		return
	}
	if err := s.Store.LearnHint(wrong, right); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.handleListHints(w, r)
}

func (s *Server) handleDeleteHint(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.DeleteHint(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.handleListHints(w, r)
}
