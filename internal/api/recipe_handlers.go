package api

import (
	"github.com/zachcurry13/recipebank/internal/budget"
	"net/http"
	"strconv"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
)

// handleGetRecipe sends one recipe with verdicts, swaps and (for Home &
// Care) hazards, for ?who= or everyone.
func (s *Server) handleGetRecipe(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	rc, err := s.Store.Recipe(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	s.writeChecked(w, r, rc)
}

// writeChecked sends a recipe (saved or a draft) with everything the checks found.
func (s *Server) writeChecked(w http.ResponseWriter, r *http.Request, rc *recipe.Recipe) {
	diners, err := s.diners(r)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	verdicts := make([]safety.Verdict, 0, len(diners))
	for _, p := range diners {
		verdicts = append(verdicts, safety.Check(rc, p))
	}
	out := map[string]any{"recipe": rc, "verdicts": verdicts, "swaps": safety.Swaps(rc, diners, verdicts),
		"heat": heatOf(rc), "hazards": []safety.Hazard{}, "pantry": s.pantryHints(rc, verdicts), "level": rc.Level()}
	if rc.Area == recipe.AreaHome {
		out["hazards"] = safety.Hazards(rc, s.Store.Pets())
	} else {
		out["needs"], out["kids"] = rc.Needs(), rc.KidsCanHelp()
	}
	if rc.NeedsReview && rc.SourceKind == "photo" {
		out["card_check"] = safety.CrossCheck(rc)
	}
	if rc.ID > 0 {
		out["cooks"], _ = s.Store.CooksFor(rc.ID)
	}
	if stock, err := s.Store.ListStock(rc.Area); err == nil {
		if c := budget.Recipe(rc, stock, 1); c.Priced > 0 {
			out["cost"] = c
		}
	}
	if rc.VersionOf != nil {
		if orig, err := s.Store.Recipe(*rc.VersionOf); err == nil {
			out["original"] = map[string]any{"id": orig.ID, "title": orig.Title}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// handleCheckDraft checks an unsaved recipe (the import preview).
func (s *Server) handleCheckDraft(w http.ResponseWriter, r *http.Request) {
	var rc recipe.Recipe
	if !readJSON(w, r, &rc, 1<<20) {
		return
	}
	rc.Clean()
	s.writeChecked(w, r, &rc)
}

// handleSaveRecipe saves a new recipe (POST) or changes one (PUT).
func (s *Server) handleSaveRecipe(w http.ResponseWriter, r *http.Request) {
	var rc recipe.Recipe
	if !readJSON(w, r, &rc, 1<<20) {
		return
	}
	rc.ID = 0
	var aiLines []recipe.Ingredient
	if id, ok := pathID(r, "id"); ok {
		old, err := s.Store.Recipe(id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		rc.ID, rc.CreatedBy, rc.VersionOf = id, old.CreatedBy, old.VersionOf
		aiLines = s.learnFrom(old, &rc)
	} else {
		aiLines = s.learnFrom(nil, &rc)
		rc.CreatedBy = auth.UserFrom(r).Username
		if rc.VersionOf != nil {
			if _, err := s.Store.Recipe(*rc.VersionOf); err != nil {
				rc.VersionOf = nil
			}
		}
	}
	if !s.ownPhotos(&rc) {
		writeErr(w, http.StatusBadRequest, "unknown photo")
		return
	}
	if rc.Photo == "" && rc.ImageURL != "" {
		rc.Photo, _ = s.downloadPhoto(r.Context(), rc.ImageURL) // a missing photo never stops the save
	}
	id, err := s.Store.SaveRecipe(&rc)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	s.learnMisreads(aiLines, rc.Ingredients)
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (s *Server) handleDeleteRecipe(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.DeleteRecipe(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.cleanPhotos()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleVersion copies a recipe as "our version" of it, keeping the original.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	rc, err := s.Store.Recipe(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	orig := rc.ID
	rc.ID, rc.VersionOf, rc.Rating = 0, &orig, 0
	rc.Title = "Our " + rc.Title
	rc.CreatedBy = auth.UserFrom(r).Username
	newID, err := s.Store.SaveRecipe(rc)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": newID})
}

// handleLabelChecked records that a parent read an ingredient's label and it
// has none of an allergen ({"ingredient": 2, "allergen": "milk", "checked": true}).
func (s *Server) handleLabelChecked(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var body struct {
		Ingredient int    `json:"ingredient"`
		Allergen   string `json:"allergen"`
		Checked    bool   `json:"checked"`
	}
	if !ok || !readJSON(w, r, &body, 1<<10) {
		return
	}
	if _, known := safety.AllergenByKey(body.Allergen); !known {
		writeErr(w, http.StatusBadRequest, "unknown allergen")
		return
	}
	rc, err := s.Store.Recipe(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	if body.Ingredient < 0 || body.Ingredient >= len(rc.Ingredients) {
		writeErr(w, http.StatusBadRequest, "no such ingredient")
		return
	}
	in := &rc.Ingredients[body.Ingredient]
	kept := []string{}
	for _, a := range in.Checked {
		if a != body.Allergen {
			kept = append(kept, a)
		}
	}
	if body.Checked {
		kept = append(kept, body.Allergen)
	}
	in.Checked = kept
	if _, err := s.Store.SaveRecipe(rc); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.writeChecked(w, r, rc)
}

// handleCardChecked records that someone compared a photo recipe with its
// card: the "not sure" lines and the caution for allergies go away.
func (s *Server) handleCardChecked(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	rc, err := s.Store.Recipe(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	rc.NeedsReview = false
	for i := range rc.Ingredients {
		rc.Ingredients[i].Unsure = false
	}
	for i := range rc.Steps {
		rc.Steps[i].Unsure = false
	}
	if _, err := s.Store.SaveRecipe(rc); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.writeChecked(w, r, rc)
}

func (s *Server) handleRating(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var body struct {
		Stars int `json:"stars"`
	}
	if !ok || !readJSON(w, r, &body, 1<<10) {
		return
	}
	if err := s.Store.SetRating(id, body.Stars); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"stars": body.Stars})
}

// heatOf is the recipe's heat, or the heat its ingredients suggest.
func heatOf(rc *recipe.Recipe) int {
	if rc.Heat >= 0 {
		return rc.Heat
	}
	lines := make([]string, len(rc.Ingredients))
	for i, in := range rc.Ingredients {
		lines[i] = in.Line
	}
	h, _ := safety.EstimateHeat(lines)
	return h
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
