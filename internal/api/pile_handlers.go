package api

import (
	"net/http"
	"strings"

	"github.com/zachcurry13/recipebank/internal/llm"
)

// handleBookIndex reads photos of a cookbook's index into a list of recipes
// by page, for the person to check before adding them.
func (s *Server) handleBookIndex(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Images []string `json:"images"`
	}
	if !readJSON(w, r, &body, 30<<20) {
		return
	}
	images, ok := readImages(w, body.Images, 3)
	if !ok {
		return
	}
	var entries []llm.IndexEntry
	if _, err := s.askPhotos(r.Context(), llm.IndexPrompt, images, func(out string) (perr error) {
		entries, perr = llm.ParseIndex(out)
		return perr
	}); err != nil {
		writeErr(w, http.StatusBadGateway, importErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

// "Still to photograph": the recipe cards and clippings waiting to be
// scanned, ticked off when a recipe is saved from one.

func (s *Server) handleListPile(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListPile()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pile": list})
}

type pileBody struct {
	Title    string `json:"title"`
	Note     string `json:"note"`
	Done     bool   `json:"done"`
	RecipeID *int64 `json:"recipe_id"`
}

func (b *pileBody) ok(w http.ResponseWriter) bool {
	b.Title, b.Note = strings.TrimSpace(b.Title), strings.TrimSpace(b.Note)
	if b.Title == "" || len(b.Title) > 150 || len(b.Note) > 300 {
		writeErr(w, http.StatusBadRequest, "a name under 150 characters (and a note under 300) is needed")
		return false
	}
	return true
}

func (s *Server) handleAddPile(w http.ResponseWriter, r *http.Request) {
	var body pileBody
	if !readJSON(w, r, &body, 4<<10) || !body.ok(w) {
		return
	}
	id, err := s.Store.AddPile(body.Title, body.Note)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (s *Server) handleUpdatePile(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	var body pileBody
	if !readJSON(w, r, &body, 4<<10) || !body.ok(w) {
		return
	}
	if body.RecipeID != nil {
		if _, err := s.Store.Recipe(*body.RecipeID); err != nil {
			writeStoreErr(w, err)
			return
		}
	}
	if err := s.Store.UpdatePile(id, body.Title, body.Note, body.Done, body.RecipeID); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeletePile(w http.ResponseWriter, r *http.Request) {
	id, _ := pathID(r, "id")
	if err := s.Store.DeletePile(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
