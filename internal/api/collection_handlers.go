package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/seasons"
	"github.com/zachcurry13/recipebank/internal/store"
)

type seasonShelf struct {
	Key   string `json:"key"`
	Icon  string `json:"icon"`
	Name  string `json:"name"`
	Area  string `json:"area"`
	Count int    `json:"count"`
}

// currentShelves are the seasons in season now (?date= the phone's date) that have recipes.
func (s *Server) currentShelves(r *http.Request, area string) ([]seasonShelf, error) {
	now, err := time.Parse(day, r.URL.Query().Get("date"))
	if err != nil {
		now = time.Now()
	}
	all, err := s.Store.ListRecipes(area)
	if err != nil {
		return nil, err
	}
	out := []seasonShelf{}
	for _, se := range seasons.Current(now) {
		n := 0
		for i := range all {
			if se.Fits(&all[i]) {
				n++
			}
		}
		if n > 0 {
			out = append(out, seasonShelf{se.Key, se.Icon, se.Name, se.Area, n})
		}
	}
	return out, nil
}

func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	cs, err := s.Store.ListCollections()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	shelves, err := s.currentShelves(r, r.URL.Query().Get("area"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"collections": cs, "seasons": shelves})
}

// shelfCards turns recipes into cards for ?who= (or everyone).
func (s *Server) shelfCards(r *http.Request, list []*recipe.Recipe) ([]recipeCard, error) {
	diners, err := s.diners(r)
	if err != nil {
		return nil, err
	}
	cards := []recipeCard{}
	for _, rc := range list {
		c, _ := cardFor(rc, diners)
		cards = append(cards, c)
	}
	return cards, nil
}

func (s *Server) handleCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	c, err := s.Store.Collection(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	ids, err := s.Store.CollectionRecipeIDs(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	var list []*recipe.Recipe
	for _, rid := range ids {
		if rc, err := s.Store.Recipe(rid); err == nil {
			list = append(list, rc)
		}
	}
	cards, err := s.shelfCards(r, list)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"collection": c, "recipes": cards})
}

// handleSeason sends a seasonal shelf's recipes (whether or not it's in season now).
func (s *Server) handleSeason(w http.ResponseWriter, r *http.Request) {
	se, ok := seasons.Find(chi.URLParam(r, "key"))
	if !ok {
		writeErr(w, http.StatusNotFound, "no such season")
		return
	}
	all, err := s.Store.ListRecipes(se.Area)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	var list []*recipe.Recipe
	for i := range all {
		if se.Fits(&all[i]) {
			list = append(list, &all[i])
		}
	}
	cards, err := s.shelfCards(r, list)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"season": seasonShelf{se.Key, se.Icon, se.Name, se.Area, len(cards)},
		"theme": se.Theme, "recipes": cards})
}

func (s *Server) handleSaveCollection(w http.ResponseWriter, r *http.Request) {
	var c store.Collection
	if !readJSON(w, r, &c, 8<<10) {
		return
	}
	c.ID = 0
	if id, ok := pathID(r, "id"); ok {
		c.ID = id
	}
	if n := len(strings.TrimSpace(c.Name)); n == 0 || n > 80 || len(c.Description) > 500 || len(c.Icon) > 16 {
		writeErr(w, http.StatusBadRequest, "a name (up to 80 letters) is needed")
		return
	}
	c.CreatedBy = auth.UserFrom(r).Username
	id, err := s.Store.SaveCollection(&c)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (s *Server) handleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.DeleteCollection(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleCollectionAdd puts recipes on a collection ({"ids": [1, 2]}).
func (s *Server) handleCollectionAdd(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if !ok || !readJSON(w, r, &body, 16<<10) {
		return
	}
	if _, err := s.Store.Collection(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	if err := s.Store.AddToCollection(id, body.IDs); err != nil {
		writeErr(w, http.StatusBadRequest, "one of those recipes isn't in RecipeBank")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleCollectionRemove(w http.ResponseWriter, r *http.Request) {
	id, ok1 := pathID(r, "id")
	rid, ok2 := pathID(r, "rid")
	if !ok1 || !ok2 {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.RemoveFromCollection(id, rid); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// keepIf filters recipes by the sure rules (diets, time, everyone eating).
func keepIf(rc *recipe.Recipe, diets []string, maxMin int, everyone []safety.Person) (bool, string) {
	for _, d := range diets {
		if v := safety.Check(rc, safety.Person{HeatMax: -1, Diets: []string{d}}); v.Status != safety.OK {
			dt, _ := safety.DietByKey(d)
			return false, "not " + strings.ToLower(dt.Label)
		}
	}
	if maxMin > 0 && (rc.TotalMin == 0 || rc.TotalMin > maxMin) {
		return false, "takes longer"
	}
	for _, p := range everyone {
		if v := safety.Check(rc, p); v.Status != safety.OK {
			return false, "not OK for " + p.Name
		}
	}
	return true, ""
}
