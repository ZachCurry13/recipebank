package api

import (
	"net/http"
	"sort"
	"strings"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/recipe"
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
	ID      int64  `json:"id"` // negative: a guest who came through the link (-their guest id)
	Name    string `json:"name"`
	IsGuest bool   `json:"is_guest"`
	ByLink  bool   `json:"by_link,omitempty"`
	Rules   string `json:"rules,omitempty"` // a link guest's allergies and diets, in words
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
	// Guests who said they're coming through the event's link.
	guests, err := s.Store.EventGuests(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	for _, g := range guests {
		coming = append(coming, eventPerson{ID: -g.ID, Name: g.Name, IsGuest: true, ByLink: true, Rules: guestRules(g.Allergies, g.Diets)})
		rules = append(rules, safety.Person{ID: -g.ID, Name: g.Name, HeatMax: -1, Allergies: g.Allergies, Diets: g.Diets})
	}
	out := []eventDish{}
	for _, d := range dishes {
		ed := eventDish{Dish: d, Verdicts: []status{}}
		var rc *recipe.Recipe
		if d.RecipeID != nil {
			if got, err := s.Store.Recipe(*d.RecipeID); err == nil {
				rc = got
				ed.Recipe = &planRecipe{ID: rc.ID, Title: rc.Title, Photo: rc.Photo, TotalMin: rc.TotalMin, Servings: rc.Servings}
			}
		}
		for i, p := range rules {
			st := safety.DishStatus(d.Contains, p) // no recipe: only what its cook said is in it
			if rc != nil {
				st = safety.Check(rc, p).Status
			}
			ed.Verdicts = append(ed.Verdicts, status{p.ID, p.Name, st})
			switch st {
			case safety.OK:
				coming[i].OK++
			case safety.Unsure:
				coming[i].Unsure++
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
		n, h, err := s.shopRecipe(rc, factor, auth.UserFrom(r).Username, nil)
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

// guestRules says a link guest's allergies and diets in words, for the host.
func guestRules(allergies map[string]string, diets []string) string {
	var parts []string
	for k, sev := range allergies {
		if a, ok := safety.AllergenByKey(k); ok {
			parts = append(parts, a.Label+" ("+sev+")")
		}
	}
	sort.Strings(parts)
	for _, d := range diets {
		if dt, ok := safety.DietByKey(d); ok {
			parts = append(parts, dt.Label)
		}
	}
	return strings.Join(parts, ", ")
}
