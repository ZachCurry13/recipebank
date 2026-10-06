package api

import (
	"fmt"
	"math"
	"testing"

	"github.com/zachcurry13/recipebank/internal/store"
)

func TestBudget(t *testing.T) {
	c, srv := setup(t)
	near := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }
	c.do("POST", "/api/stock", map[string]any{"area": "kitchen", "name": "Flour", "qty": 1, "price": 4, "size": "5 lb"}, nil)
	c.do("POST", "/api/stock", map[string]any{"area": "kitchen", "name": "Eggs", "qty": 12, "price": 3.6, "size": "12 count"}, nil)
	var rc struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Pasta dough", "servings": 4,
		"ingredients": []map[string]string{{"line": "2 cups flour"}, {"line": "3 eggs"}, {"line": "pinch of salt"}, {"line": "1 tbsp olive oil"}}}, &rc)

	var page struct {
		Cost *struct {
			Total         float64
			Priced, Lines int
		}
	}
	c.do("GET", fmt.Sprintf("/api/recipes/%d", rc.ID), nil, &page)
	// 250 g of a 5 lb bag ≈ $0.44, 3 of 12 eggs = $0.90; salt and oil are staples.
	if page.Cost == nil || !near(page.Cost.Total, 1.34) || page.Cost.Priced != 2 || page.Cost.Lines != 2 {
		t.Fatalf("recipe cost: %+v", page.Cost)
	}

	c.do("POST", "/api/shopping", map[string]string{"text": "2 cups flour"}, nil)
	c.do("POST", "/api/shopping", map[string]string{"text": "18 eggs"}, nil)
	c.do("POST", "/api/shopping", map[string]string{"text": "basil"}, nil)
	_ = srv.Store.SetSetting(store.KeyBudgetWeekly, "150")
	var list struct {
		Costs  map[string]float64
		Budget struct {
			Cost struct {
				Total         float64
				Priced, Lines int
			}
			Weekly float64
		}
	}
	c.do("GET", "/api/shopping", nil, &list)
	if !near(list.Budget.Cost.Total, 4+7.2) || list.Budget.Cost.Priced != 2 || list.Budget.Cost.Lines != 3 || list.Budget.Weekly != 150 || len(list.Costs) != 2 {
		t.Fatalf("shopping: %+v", list)
	}

	c.do("POST", "/api/plan", map[string]any{"date": "2026-10-07", "meal": "dinner", "recipe_id": rc.ID, "servings": 8}, nil)
	var plan struct {
		Cost struct {
			Total  float64
			Priced int
		}
	}
	c.do("GET", "/api/plan?from=2026-10-07&days=1", nil, &plan)
	if !near(plan.Cost.Total, 2.68) || plan.Cost.Priced != 2 {
		t.Fatalf("the plan, doubled: %+v", plan.Cost)
	}
}
