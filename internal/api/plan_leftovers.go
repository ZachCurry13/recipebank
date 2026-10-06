package api

import (
	"github.com/zachcurry13/recipebank/internal/budget"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/store"
)

// Planned leftovers: Tuesday's dinner can be the leftovers of Monday's roast.
// Monday is then made bigger (and bought for), and Tuesday buys nothing.

// planRef points at a planned meal.
type planRef struct {
	ID   int64  `json:"id"`
	Date string `json:"date"`
	Meal string `json:"meal"`
}

// addLeftovers fills in where a leftovers meal is from, or how much extra a
// meal makes for the leftovers planned from it.
func (s *Server) addLeftovers(m *planMeal, rc *recipe.Recipe) {
	if m.LeftoversOf != nil {
		if src, err := s.Store.PlanEntry(*m.LeftoversOf); err == nil {
			m.From = &planRef{src.ID, src.Date, src.Meal}
		}
		return
	}
	m.Extra = 0
	later, err := s.Store.LeftoversFor(m.ID)
	if err != nil {
		return
	}
	for _, l := range later {
		m.For = append(m.For, planRef{l.ID, l.Date, l.Meal})
		m.Extra += servingsOf(l, rc)
	}
}

// servingsOf is how many a planned meal is for: as planned, or the recipe's own.
func servingsOf(e store.PlanEntry, rc *recipe.Recipe) float64 {
	if e.Servings > 0 {
		return e.Servings
	}
	if rc.Servings > 0 {
		return rc.Servings
	}
	return 1
}

// mealFactor scales a planned recipe for its servings plus the leftovers
// planned from it.
func (s *Server) mealFactor(e store.PlanEntry, rc *recipe.Recipe) float64 {
	m := planMeal{PlanEntry: e}
	s.addLeftovers(&m, rc)
	if rc.Servings <= 0 {
		return 1 + float64(len(m.For)) // unknown servings: one more batch per leftovers meal
	}
	return (servingsOf(e, rc) + m.Extra) / rc.Servings
}

// planCost estimates the planned recipes (leftovers included in the meal
// they come from) from the pantry's prices.
func (s *Server) planCost(entries []store.PlanEntry) budget.Cost {
	var total budget.Cost
	stock, err := s.Store.ListStock("kitchen")
	if err != nil {
		return total
	}
	for _, e := range entries {
		if e.RecipeID == nil || e.LeftoversOf != nil {
			continue
		}
		rc, err := s.Store.Recipe(*e.RecipeID)
		if err != nil {
			continue
		}
		c := budget.Recipe(rc, stock, s.mealFactor(e, rc))
		total.Total += c.Total
		total.Priced += c.Priced
		total.Lines += c.Lines
	}
	return total
}
