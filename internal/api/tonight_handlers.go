package api

import (
	"hash/fnv"
	"math/rand"
	"net/http"
	"time"

	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

// tonightMeal is a meal planned today, with everyone's full verdicts.
type tonightMeal struct {
	planMeal
	Full []safety.Verdict `json:"full"`
}

// handleTonight sends today (?date= the phone's date): what's planned with
// who's eating and why, the next planned meal, food to use soon, and ideas
// everyone home can eat when nothing's planned for dinner.
func (s *Server) handleTonight(w http.ResponseWriter, r *http.Request) {
	today, err := time.Parse(day, r.URL.Query().Get("date"))
	if err != nil {
		today = time.Now()
	}
	date := today.Format(day)
	pc, err := s.loadPlanContext(date, today.AddDate(0, 0, 7).Format(day))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	ids, set := pc.who(date)
	diners := pc.diners(ids)
	entries, err := s.Store.PlanBetween(date, today.AddDate(0, 0, 7).Format(day))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	meals := []tonightMeal{}
	var next *planMeal
	dinner := false
	for _, e := range entries {
		if e.Date != date {
			if next == nil {
				nids, _ := pc.who(e.Date)
				m := s.buildMeal(pc, e, pc.diners(nids))
				next = &m
			}
			continue
		}
		m := tonightMeal{planMeal: s.buildMeal(pc, e, diners), Full: []safety.Verdict{}}
		if e.RecipeID != nil {
			if rc := s.recipeFor(pc, *e.RecipeID); rc != nil {
				for _, p := range diners {
					m.Full = append(m.Full, safety.Check(rc, p))
				}
			}
		}
		dinner = dinner || e.Meal == "dinner"
		meals = append(meals, m)
	}
	out := map[string]any{"date": date, "who": ids, "who_set": set, "people": pc.people, "meals": meals,
		"next": next, "use_soon": s.useSoon(today), "ideas": []planRecipe{}}
	if !dinner {
		out["ideas"] = s.ideas(date, diners, 4)
	}
	writeJSON(w, http.StatusOK, out)
}

// useSoon lists pantry food past or within two days of its use-by date.
func (s *Server) useSoon(today time.Time) []store.StockItem {
	items, err := s.Store.ListStock("kitchen")
	out := []store.StockItem{}
	if err != nil {
		return out
	}
	limit := today.AddDate(0, 0, 2).Format(day)
	for _, it := range items {
		if it.UseBy != "" && it.UseBy <= limit && it.Qty > 0 {
			out = append(out, it)
		}
	}
	return out
}

// ideas picks up to n kitchen recipes OK for everyone eating, a different
// handful each day.
func (s *Server) ideas(date string, diners []safety.Person, n int) []planRecipe {
	all, err := s.Store.ListRecipes("kitchen")
	out := []planRecipe{}
	if err != nil {
		return out
	}
	var ok []planRecipe
	for i := range all {
		rc := &all[i]
		good := true
		for _, p := range diners {
			if safety.Check(rc, p).Status != safety.OK {
				good = false
				break
			}
		}
		if good {
			ok = append(ok, planRecipe{ID: rc.ID, Title: rc.Title, Photo: rc.Photo, TotalMin: rc.TotalMin, Servings: rc.Servings})
		}
	}
	h := fnv.New64a()
	h.Write([]byte(date))
	rand.New(rand.NewSource(int64(h.Sum64()))).Shuffle(len(ok), func(i, j int) { ok[i], ok[j] = ok[j], ok[i] })
	if len(ok) > n {
		ok = ok[:n]
	}
	return append(out, ok...)
}
