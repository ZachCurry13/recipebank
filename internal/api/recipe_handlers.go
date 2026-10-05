package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
)

// recipeCard is a recipe in the Library: the facts on its card and each
// diner's verdict.
type recipeCard struct {
	ID          int64    `json:"id"`
	Area        string   `json:"area"`
	Title       string   `json:"title"`
	Photo       string   `json:"photo"`
	TotalMin    int      `json:"total_min"`
	Heat        int      `json:"heat"`
	Course      string   `json:"course"`
	Cuisine     string   `json:"cuisine"`
	Protein     string   `json:"protein"`
	Rating      int      `json:"rating"`
	NeedsReview bool     `json:"needs_review"`
	Verdicts    []status `json:"verdicts"`
}

type status struct {
	PersonID int64  `json:"person_id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
}

// handleListRecipes lists an area's recipes, filtered, with verdicts for
// the chosen diners. ?ok=1 keeps only recipes OK for all of them
// (?ok=unsure also keeps "not sure").
func (s *Server) handleListRecipes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	all, err := s.Store.ListRecipes(q.Get("area"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	diners, err := s.diners(r)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	search := strings.ToLower(strings.TrimSpace(q.Get("q")))
	maxMin, _ := strconv.Atoi(q.Get("max_min"))
	maxHeat, heatErr := strconv.Atoi(q.Get("heat"))
	diet := q.Get("diet")
	cards := []recipeCard{}
	facets := map[string]map[string]int{"course": {}, "cuisine": {}, "protein": {}}
	for i := range all {
		rc := &all[i]
		heat := heatOf(rc)
		if search != "" && !strings.Contains(rc.Text(), search) ||
			maxMin > 0 && (rc.TotalMin == 0 || rc.TotalMin > maxMin) ||
			heatErr == nil && heat > maxHeat ||
			!matches(q.Get("course"), rc.Course) || !matches(q.Get("cuisine"), rc.Cuisine) || !matches(q.Get("protein"), rc.Protein) {
			continue
		}
		if diet != "" && safety.Check(rc, safety.Person{HeatMax: -1, Diets: []string{diet}}).Status != safety.OK {
			continue
		}
		card := recipeCard{ID: rc.ID, Area: rc.Area, Title: rc.Title, Photo: rc.Photo, TotalMin: rc.TotalMin, Heat: heat,
			Course: rc.Course, Cuisine: rc.Cuisine, Protein: rc.Protein, Rating: rc.Rating, NeedsReview: rc.NeedsReview,
			Verdicts: []status{}}
		worst := safety.OK
		for _, p := range diners {
			v := safety.Check(rc, p)
			card.Verdicts = append(card.Verdicts, status{p.ID, p.Name, v.Status})
			worst = worse(worst, v.Status)
		}
		if ok := q.Get("ok"); ok == "1" && worst != safety.OK || ok == "unsure" && worst == safety.No {
			continue
		}
		cards = append(cards, card)
		for k, v := range map[string]string{"course": rc.Course, "cuisine": rc.Cuisine, "protein": rc.Protein} {
			if v != "" {
				facets[k][strings.ToLower(v)]++
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"recipes": cards, "facets": facets})
}

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
		"heat": heatOf(rc), "hazards": []safety.Hazard{}, "pantry": s.pantryHints(rc, verdicts)}
	if rc.Area == recipe.AreaHome {
		out["hazards"] = safety.Hazards(rc, s.Store.Pets())
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
	if id, ok := pathID(r, "id"); ok {
		old, err := s.Store.Recipe(id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		rc.ID, rc.CreatedBy, rc.VersionOf = id, old.CreatedBy, old.VersionOf
	} else {
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

func matches(want, have string) bool { return want == "" || strings.EqualFold(want, have) }

func worse(a, b string) string {
	rank := map[string]int{safety.OK: 0, safety.Unsure: 1, safety.No: 2}
	if rank[b] > rank[a] {
		return b
	}
	return a
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
