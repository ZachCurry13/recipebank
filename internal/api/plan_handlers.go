package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

const day = "2006-01-02"

// planMeal is a planned meal with its recipe's card facts and verdicts.
type planMeal struct {
	store.PlanEntry
	Recipe   *planRecipe `json:"recipe"`
	Verdicts []status    `json:"verdicts"`
	From     *planRef    `json:"from,omitempty"` // leftovers: the meal they're from
	Extra    float64     `json:"extra"`          // servings to make on top, for planned leftovers
	For      []planRef   `json:"for"`            // the meals those leftovers are for
}

type planRecipe struct {
	ID       int64   `json:"id"`
	Title    string  `json:"title"`
	Photo    string  `json:"photo"`
	TotalMin int     `json:"total_min"`
	Servings float64 `json:"servings"`
	Liked    bool    `json:"liked,omitempty"` // the people eating liked it before
}

type planDay struct {
	Date   string     `json:"date"`
	Who    []int64    `json:"who"`
	WhoSet bool       `json:"who_set"` // false = everyone but guests
	Meals  []planMeal `json:"meals"`
}

// planContext is what building plan days needs, loaded once.
type planContext struct {
	people  []store.Person
	byID    map[int64]*store.Person
	days    map[string][]int64
	recipes map[int64]*recipe.Recipe
}

func (s *Server) loadPlanContext(from, to string) (*planContext, error) {
	ps, err := s.Store.ListPeople()
	if err != nil {
		return nil, err
	}
	days, err := s.Store.PlanDays(from, to)
	if err != nil {
		return nil, err
	}
	pc := &planContext{people: ps, byID: map[int64]*store.Person{}, days: days, recipes: map[int64]*recipe.Recipe{}}
	for i := range ps {
		pc.byID[ps[i].ID] = &ps[i]
	}
	return pc, nil
}

// who is who's eating on a date: as set, else everyone but guests.
func (pc *planContext) who(date string) ([]int64, bool) {
	if ids, ok := pc.days[date]; ok {
		return ids, true
	}
	ids := []int64{}
	for _, p := range pc.people {
		if !p.IsGuest {
			ids = append(ids, p.ID)
		}
	}
	return ids, false
}

func (pc *planContext) diners(ids []int64) []safety.Person {
	out := []safety.Person{}
	for _, id := range ids {
		if p := pc.byID[id]; p != nil {
			out = append(out, p.Checkable())
		}
	}
	return out
}

func (s *Server) recipeFor(pc *planContext, id int64) *recipe.Recipe {
	if rc, ok := pc.recipes[id]; ok {
		return rc
	}
	rc, err := s.Store.Recipe(id)
	if err != nil {
		rc = nil
	}
	pc.recipes[id] = rc
	return rc
}

func (s *Server) buildMeal(pc *planContext, e store.PlanEntry, diners []safety.Person) planMeal {
	m := planMeal{PlanEntry: e, Verdicts: []status{}, For: []planRef{}}
	if e.RecipeID == nil {
		return m
	}
	rc := s.recipeFor(pc, *e.RecipeID)
	if rc == nil {
		return m
	}
	s.addLeftovers(&m, rc)
	m.Recipe = &planRecipe{ID: rc.ID, Title: rc.Title, Photo: rc.Photo, TotalMin: rc.TotalMin, Servings: rc.Servings}
	for _, p := range diners {
		m.Verdicts = append(m.Verdicts, status{p.ID, p.Name, safety.Check(rc, p).Status})
	}
	return m
}

// handlePlan sends ?days= days (1-31, default 7) from ?from= (default today).
func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	from, err := time.Parse(day, r.URL.Query().Get("from"))
	if err != nil {
		from = time.Now()
	}
	n := queryInt(r, "days")
	if n < 1 || n > 31 {
		n = 7
	}
	to := from.AddDate(0, 0, n-1)
	pc, err := s.loadPlanContext(from.Format(day), to.Format(day))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	entries, err := s.Store.PlanBetween(from.Format(day), to.Format(day))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	out := []planDay{}
	for i := 0; i < n; i++ {
		date := from.AddDate(0, 0, i).Format(day)
		ids, set := pc.who(date)
		d := planDay{Date: date, Who: ids, WhoSet: set, Meals: []planMeal{}}
		diners := pc.diners(ids)
		for _, e := range entries {
			if e.Date == date {
				d.Meals = append(d.Meals, s.buildMeal(pc, e, diners))
			}
		}
		out = append(out, d)
	}
	writeJSON(w, http.StatusOK, map[string]any{"days": out, "people": pc.people, "cost": s.planCost(entries)})
}

// handleSavePlan adds (POST) or moves/changes (PUT) a planned meal.
func (s *Server) handleSavePlan(w http.ResponseWriter, r *http.Request) {
	var e store.PlanEntry
	if !readJSON(w, r, &e, 8<<10) {
		return
	}
	e.ID = 0
	if id, ok := pathID(r, "id"); ok {
		e.ID = id
	}
	if _, err := time.Parse(day, e.Date); err != nil || !store.ValidMeal(e.Meal) {
		writeErr(w, http.StatusBadRequest, "a date and a meal (breakfast, lunch, dinner or snack) are needed")
		return
	}
	if e.LeftoversOf != nil {
		src, err := s.Store.PlanEntry(*e.LeftoversOf)
		if err != nil || src.RecipeID == nil || src.Date > e.Date || src.LeftoversOf != nil {
			writeErr(w, http.StatusBadRequest, "leftovers come from an earlier planned recipe")
			return
		}
		e.RecipeID, e.Title = src.RecipeID, ""
	}
	if e.RecipeID != nil {
		if _, err := s.Store.Recipe(*e.RecipeID); err != nil {
			writeErr(w, http.StatusBadRequest, "that recipe isn't in RecipeBank")
			return
		}
	} else if strings.TrimSpace(e.Title) == "" {
		writeErr(w, http.StatusBadRequest, "pick a recipe or type what's planned")
		return
	}
	e.CreatedBy = auth.UserFrom(r).Username
	id, err := s.Store.SavePlan(&e)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (s *Server) handleDeletePlan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.DeletePlan(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handlePlanDay sets who's eating at home on a date ({"who": [1, 2]}, or null for everyone but guests).
func (s *Server) handlePlanDay(w http.ResponseWriter, r *http.Request) {
	date := chi.URLParam(r, "date")
	var body struct {
		Who []int64 `json:"who"`
	}
	if _, err := time.Parse(day, date); err != nil || !readJSON(w, r, &body, 4<<10) {
		writeErr(w, http.StatusBadRequest, "bad date")
		return
	}
	if err := s.Store.SetPlanDay(date, body.Who); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handlePlanShopping puts every planned recipe from ?from for ?days on the
// shopping list, scaled to the planned servings.
func (s *Server) handlePlanShopping(w http.ResponseWriter, r *http.Request) {
	var body struct {
		From string `json:"from"`
		Days int    `json:"days"`
	}
	if !readJSON(w, r, &body, 1<<10) {
		return
	}
	from, err := time.Parse(day, body.From)
	if err != nil || body.Days < 1 || body.Days > 31 {
		writeErr(w, http.StatusBadRequest, "a start date and 1-31 days are needed")
		return
	}
	entries, err := s.Store.PlanBetween(from.Format(day), from.AddDate(0, 0, body.Days-1).Format(day))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	added, have, seen := 0, []haveItem{}, map[string]bool{}
	for _, e := range entries {
		if e.RecipeID == nil || e.LeftoversOf != nil { // leftovers come from a meal that's bought for
			continue
		}
		rc, err := s.Store.Recipe(*e.RecipeID)
		if err != nil {
			continue
		}
		factor := s.mealFactor(e, rc)
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
