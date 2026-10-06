package api

import (
	"net/http"
	"sort"

	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/search"
)

// handleDish looks at a photo of a meal ({"images": [data: URL]}): what it
// most likely is, and the family's recipes most like it.
func (s *Server) handleDish(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Images []string `json:"images"`
	}
	if !readJSON(w, r, &body, 20<<20) {
		return
	}
	images, ok := readImages(w, body.Images, 2)
	if !ok {
		return
	}
	var dish llm.Dish
	if _, err := s.askPhotos(r.Context(), llm.DishPrompt, images, func(out string) (perr error) {
		dish, perr = llm.ParseDish(out)
		return perr
	}); err != nil {
		writeErr(w, http.StatusBadGateway, importErr(err))
		return
	}
	diners, err := s.diners(r)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	all, err := s.Store.ListRecipes("kitchen")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	// Words of the dish's name count most, then its usual foods.
	name := safety.NewWords(search.Keywords(dish.Name), nil)
	foods := safety.NewWords(dish.Ingredients, nil)
	type scored struct {
		i, score int
	}
	var found []scored
	for i := range all {
		score := 3*name.Count(all[i].Title) + name.Count(all[i].Text()) + foods.Count(all[i].Text())
		if score > 1 {
			found = append(found, scored{i, score})
		}
	}
	sort.SliceStable(found, func(a, b int) bool { return found[a].score > found[b].score })
	matches := []recipeCard{}
	for _, f := range found {
		if len(matches) == 6 {
			break
		}
		card, _ := cardFor(&all[f.i], diners)
		matches = append(matches, card)
	}
	writeJSON(w, http.StatusOK, map[string]any{"dish": dish, "matches": matches})
}

// handleDishDraft has the AI write a recipe for a dish it described: a best
// guess, read and checked like any other draft before anyone saves it.
func (s *Server) handleDishDraft(w http.ResponseWriter, r *http.Request) {
	var dish llm.Dish
	if !readJSON(w, r, &dish, 8<<10) {
		return
	}
	dish, err := dish.Clean()
	if err != nil {
		writeErr(w, http.StatusBadRequest, "say what the dish is")
		return
	}
	var text string
	if err := llm.Ask(r.Context(), s.Store, llm.DishDraftSystem, llm.DishDraftPrompt(dish), func(out string) (perr error) {
		text, perr = llm.ParseTranscript(out)
		return perr
	}); err != nil {
		writeErr(w, http.StatusBadGateway, importErr(err))
		return
	}
	rc, err := s.organize(r.Context(), text)
	if err != nil {
		writeErr(w, http.StatusBadGateway, importErr(err))
		return
	}
	rc.SourceKind = "ai"
	rc.SourceNote = "The AI's best guess at " + dish.Name + " from a photo. Check the amounts and steps before cooking."
	s.finishDraft(w, r, &rc, "kitchen")
}
