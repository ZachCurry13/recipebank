package api

import (
	"net/http"
	"strings"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

// eventDish is a dish with its recipe's facts and how it suits everyone coming.
type eventDish struct {
	store.Dish
	Recipe   *planRecipe `json:"recipe"`
	Verdicts []status    `json:"verdicts"`
}

// eventPerson is someone coming, and how many dishes they can eat.
type eventPerson struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	IsGuest bool   `json:"is_guest"`
	OK      int    `json:"ok"`
	Unsure  int    `json:"unsure"`
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	e, err := s.Store.Event(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	dishes, err := s.Store.Dishes(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	people, err := s.Store.ListPeople()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	coming, rules := []eventPerson{}, []safety.Person{}
	for _, wid := range e.Who {
		for i := range people {
			if people[i].ID == wid {
				coming = append(coming, eventPerson{ID: wid, Name: people[i].Name, IsGuest: people[i].IsGuest})
				rules = append(rules, people[i].Checkable())
			}
		}
	}
	out := []eventDish{}
	for _, d := range dishes {
		ed := eventDish{Dish: d, Verdicts: []status{}}
		if d.RecipeID != nil {
			if rc, err := s.Store.Recipe(*d.RecipeID); err == nil {
				ed.Recipe = &planRecipe{ID: rc.ID, Title: rc.Title, Photo: rc.Photo, TotalMin: rc.TotalMin, Servings: rc.Servings}
				for i, p := range rules {
					v := safety.Check(rc, p)
					ed.Verdicts = append(ed.Verdicts, status{p.ID, p.Name, v.Status})
					switch v.Status {
					case safety.OK:
						coming[i].OK++
					case safety.Unsure:
						coming[i].Unsure++
					}
				}
			}
		}
		out = append(out, ed)
	}
	writeJSON(w, http.StatusOK, map[string]any{"event": e, "dishes": out, "coming": coming,
		"headcount": len(coming) + e.Extra, "people": people})
}

// handleAddDish adds a recipe ({"recipe_id": 3}) or a dish without one
// ({"title": "Store-bought pie"}) to the menu, and who brings it.
func (s *Server) handleAddDish(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var d store.Dish
	if !ok || !readJSON(w, r, &d, 4<<10) {
		return
	}
	if _, err := s.Store.Event(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	d.EventID, d.Title, d.Brings = id, strings.TrimSpace(d.Title), strings.TrimSpace(d.Brings)
	switch {
	case d.RecipeID != nil:
		if _, err := s.Store.Recipe(*d.RecipeID); err != nil {
			writeErr(w, http.StatusBadRequest, "that recipe isn't in RecipeBank")
			return
		}
		d.Title = ""
	case d.Title == "" || len(d.Title) > 120:
		writeErr(w, http.StatusBadRequest, "pick a recipe or name the dish")
		return
	}
	if len(d.Brings) > 60 {
		writeErr(w, http.StatusBadRequest, "who brings it: up to 60 letters")
		return
	}
	if _, err := s.Store.AddDish(d); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.handleGetEvent(w, r)
}

func (s *Server) handleDishBrings(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	dish, ok2 := pathID(r, "dish")
	var body struct {
		Brings string `json:"brings"`
	}
	if !ok || !ok2 || !readJSON(w, r, &body, 1<<10) {
		return
	}
	if len(strings.TrimSpace(body.Brings)) > 60 {
		writeErr(w, http.StatusBadRequest, "who brings it: up to 60 letters")
		return
	}
	if err := s.Store.SetDishBrings(id, dish, body.Brings); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.handleGetEvent(w, r)
}

func (s *Server) handleDeleteDish(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	dish, ok2 := pathID(r, "dish")
	if !ok || !ok2 {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.DeleteDish(id, dish); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.handleGetEvent(w, r)
}

// handleEventShopping puts the family's own dishes on the shopping list,
// scaled to everyone coming.
func (s *Server) handleEventShopping(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	e, err := s.Store.Event(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	dishes, err := s.Store.Dishes(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	headcount := float64(len(e.Who) + e.Extra)
	added, have, seen := 0, []haveItem{}, map[string]bool{}
	for _, d := range dishes {
		if d.RecipeID == nil || d.Brings != "" {
			continue
		}
		rc, err := s.Store.Recipe(*d.RecipeID)
		if err != nil {
			continue
		}
		factor := 1.0
		if rc.Servings > 0 && headcount > 0 {
			factor = headcount / rc.Servings
		}
		n, h, err := s.shopRecipe(rc, factor, auth.UserFrom(r).Username)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		added += n
		for _, x := range h {
			if !seen[x.Food] {
				seen[x.Food] = true
				have = append(have, x)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"added": added, "have": have})
}
