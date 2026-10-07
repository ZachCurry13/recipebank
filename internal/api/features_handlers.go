package api

import (
	"net/http"
	"slices"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/store"
)

// featureNames say what a feature is in "… is turned off" answers.
var featureNames = map[string]string{"email": "Email", "share": "Sharing by link", "nutrition": "Nutrition estimates"}

// feature refuses a request for something the admin turned off. The web app
// hides it anyway; this keeps a shared link or an old page from using it.
func (s *Server) feature(f string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !s.Store.FeatureOn(f) {
				writeErr(w, http.StatusForbidden, featureNames[f]+" is turned off (Admin → Features)")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// handleSimpler saves the signed-in person's simpler view: the meals their
// plan shows (none chosen = all) and the pages left out of their menu.
func (s *Server) handleSimpler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PlanMeals   []string `json:"plan_meals"`
		HiddenPages []string `json:"hidden_pages"`
		MenuOrder   []string `json:"menu_order"`
	}
	if !readJSON(w, r, &body, 2<<10) {
		return
	}
	for _, m := range body.PlanMeals {
		if !slices.Contains(store.PlanMeals, m) {
			writeErr(w, http.StatusBadRequest, "unknown meal "+m)
			return
		}
	}
	for _, p := range body.HiddenPages {
		if !slices.Contains(store.HidePages, p) {
			writeErr(w, http.StatusBadRequest, "that page can't be hidden: "+p)
			return
		}
	}
	seen := map[string]bool{}
	for _, p := range body.MenuOrder {
		if !slices.Contains(store.MenuPages, p) || seen[p] {
			writeErr(w, http.StatusBadRequest, "that isn't a page for the menu: "+p)
			return
		}
		seen[p] = true
	}
	if len(body.PlanMeals) == len(store.PlanMeals) {
		body.PlanMeals = nil // every meal is the same as the default
	}
	if err := s.Store.SetSimpler(auth.UserFrom(r).ID, body.PlanMeals, body.HiddenPages, body.MenuOrder); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
