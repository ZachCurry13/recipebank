package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/nutrition"
	"github.com/zachcurry13/recipebank/internal/store"
)

// maxLookups caps new FoodData Central lookups per estimate (the rest are cached).
const maxLookups = 25

// handleNutrition estimates a recipe's nutrition per serving. Foods are
// looked up once (only their names are sent) and kept; lines it can't work
// out are listed.
func (s *Server) handleNutrition(w http.ResponseWriter, r *http.Request) {
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
	key := s.Store.Setting(store.KeyUSDAKey)
	client := nutrition.Client{HTTP: s.Fetch, Base: s.Nutrition, Key: key, Agent: userAgent}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	lookups := 0
	var problem error
	lookup := func(food string) (nutrition.Food, bool) {
		k := strings.ToLower(strings.TrimSpace(food))
		var f nutrition.Food
		if data, ok := s.Store.NutritionFood(k); ok {
			return f, data != "" && json.Unmarshal([]byte(data), &f) == nil
		}
		if key == "" || problem != nil || lookups >= maxLookups {
			return f, false
		}
		lookups++
		f, err := client.Lookup(ctx, k)
		switch {
		case errors.Is(err, nutrition.ErrNotFound):
			_ = s.Store.SaveNutritionFood(k, "")
			return f, false
		case err != nil:
			problem = err
			return f, false
		}
		if data, err := json.Marshal(f); err == nil {
			_ = s.Store.SaveNutritionFood(k, string(data))
		}
		return f, true
	}
	est := nutrition.Compute(rc.Ingredients, rc.Servings, lookup)
	out := map[string]any{"estimate": est, "key_set": key != ""}
	if problem != nil {
		out["error"] = problem.Error()
	}
	writeJSON(w, http.StatusOK, out)
}
