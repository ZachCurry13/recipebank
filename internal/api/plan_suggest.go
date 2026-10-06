package api

import (
	"hash/fnv"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/budget"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

// "Plan my week": a draft of dinners for the empty days, for the person to
// keep or change. Only recipes everyone home that day can eat; liked ones,
// ones using up food near its date, and quick ones on weeknights first;
// nothing cooked in the last week or already planned; within the weekly
// budget when prices are known; leftovers planned when a recipe makes plenty.

// notDinner are courses that aren't a dinner on their own.
var notDinner = map[string]bool{"breakfast": true, "dessert": true, "snack": true, "drink": true, "sauce": true,
	"side": true, "appetizer": true, "baking": true, "bread": true}

type dinnerPick struct {
	Date        string      `json:"date"`
	Recipe      *planRecipe `json:"recipe"`
	Why         []string    `json:"why"`
	LeftoversOf string      `json:"leftovers_of,omitempty"` // the date whose dinner this eats up
}

func (s *Server) handlePlanSuggest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		From string `json:"from"`
		Days int    `json:"days"`
		Seed int64  `json:"seed"` // "try again" asks for a different draft
	}
	if !readJSON(w, r, &body, 1<<10) {
		return
	}
	from, err := time.Parse(day, body.From)
	if err != nil || body.Days < 1 || body.Days > 14 {
		writeErr(w, http.StatusBadRequest, "a start date and 1-14 days are needed")
		return
	}
	to := from.AddDate(0, 0, body.Days-1)
	pc, err := s.loadPlanContext(from.Format(day), to.Format(day))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	planned, err := s.Store.PlanBetween(from.Format(day), to.Format(day))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	all, err := s.Store.ListRecipes("kitchen")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	likes, err := s.Store.Likes()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	stock, err := s.Store.ListStock("kitchen")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	soon := s.useSoon(from.AddDate(0, 0, 2))

	hasDinner, used := map[string]bool{}, map[int64]bool{}
	var spent float64
	for _, e := range planned {
		if e.Meal == "dinner" {
			hasDinner[e.Date] = true
		}
		if e.RecipeID != nil {
			used[*e.RecipeID] = true
			if rc := s.recipeFor(pc, *e.RecipeID); rc != nil && e.LeftoversOf == nil {
				spent += budget.Recipe(rc, stock, s.mealFactor(e, rc)).Total
			}
		}
	}
	weekly := s.Store.SettingFloat(store.KeyBudgetWeekly)
	recent := from.AddDate(0, 0, -7).Format(day)

	out := []dinnerPick{}
	var cost budget.Cost
	for i := 0; i < body.Days; i++ {
		date := from.AddDate(0, 0, i)
		ds := date.Format(day)
		if hasDinner[ds] {
			continue
		}
		ids, _ := pc.who(ds)
		diners := pc.diners(ids)
		weeknight := date.Weekday() >= time.Monday && date.Weekday() <= time.Thursday
		type cand struct {
			rc    *recipe.Recipe
			score int
			why   []string
			cost  float64
		}
		var cands []cand
		for j := range all {
			rc := &all[j]
			if used[rc.ID] || notDinner[strings.ToLower(rc.Course)] || len(rc.Ingredients) == 0 ||
				(likes.Last[rc.ID] != "" && likes.Last[rc.ID] >= recent) || !okForAll(rc, diners) {
				continue
			}
			c := cand{rc: rc}
			switch tier := likeTier(likes, rc.ID, diners, recent); tier {
			case 3:
				continue // someone eating didn't like it
			case 0:
				c.score += 3
				c.why = append(c.why, "liked before")
			}
			var uses []string
			for _, it := range soon {
				for _, in := range rc.Ingredients {
					if in.Food != "" && safety.Covers(it.Name, in.Food) {
						uses = append(uses, it.Name)
						break
					}
				}
			}
			if len(uses) > 0 {
				c.score += 2 * len(uses)
				c.why = append(c.why, "uses up "+strings.Join(uses, ", "))
			}
			if weeknight && rc.TotalMin > 0 && rc.TotalMin <= 45 {
				c.score++
				c.why = append(c.why, "quick for a weeknight")
			}
			factor := 1.0
			if rc.Servings > 0 && len(diners) > 0 {
				factor = float64(len(diners)) / rc.Servings
			}
			c.cost = budget.Recipe(rc, stock, factor).Total
			if weekly > 0 && spent+cost.Total+c.cost > weekly {
				continue
			}
			cands = append(cands, c)
		}
		if len(cands) == 0 {
			continue
		}
		h := fnv.New64a()
		h.Write([]byte(ds))
		rand.New(rand.NewSource(int64(h.Sum64())+body.Seed)).Shuffle(len(cands), func(a, b int) { cands[a], cands[b] = cands[b], cands[a] })
		sort.SliceStable(cands, func(a, b int) bool { return cands[a].score > cands[b].score })
		best := cands[0]
		used[best.rc.ID] = true
		cost.Total += best.cost
		pr := &planRecipe{ID: best.rc.ID, Title: best.rc.Title, Photo: best.rc.Photo, TotalMin: best.rc.TotalMin, Servings: best.rc.Servings}
		out = append(out, dinnerPick{Date: ds, Recipe: pr, Why: best.why})
		// A recipe that makes twice what's needed covers tomorrow too.
		next := date.AddDate(0, 0, 1).Format(day)
		if i+1 < body.Days && !hasDinner[next] && len(diners) > 0 && best.rc.Servings >= 2*float64(len(diners)) {
			hasDinner[next] = true
			out = append(out, dinnerPick{Date: next, Recipe: pr, Why: []string{"makes enough for two nights"}, LeftoversOf: ds})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"days": out, "cost": cost.Total, "weekly": weekly, "spent": spent})
}

func okForAll(rc *recipe.Recipe, diners []safety.Person) bool {
	for _, p := range diners {
		if safety.Check(rc, p).Status != safety.OK {
			return false
		}
	}
	return true
}
