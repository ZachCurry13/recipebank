package api

import (
	"strings"
	"testing"
)

// Monday's roast for 4 with leftovers for 2 on Tuesday: Monday makes 6 and
// is bought for once; Tuesday says where it's from and buys nothing.
func TestPlannedLeftovers(t *testing.T) {
	c, _ := setup(t)
	var roast struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Roast chicken", "servings": 4,
		"ingredients": []map[string]string{{"line": "4 lb chicken"}, {"line": "2 lemons"}}}, &roast)
	var mon, tue struct{ ID int64 }
	c.do("POST", "/api/plan", map[string]any{"date": "2026-10-05", "meal": "dinner", "recipe_id": roast.ID}, &mon)
	if code := c.do("POST", "/api/plan", map[string]any{"date": "2026-10-06", "meal": "dinner", "leftovers_of": mon.ID, "servings": 2}, &tue); code != 200 {
		t.Fatalf("leftovers: %d", code)
	}
	if code := c.do("POST", "/api/plan", map[string]any{"date": "2026-10-04", "meal": "dinner", "leftovers_of": mon.ID}, nil); code != 400 {
		t.Fatalf("leftovers before the meal they're from: %d", code)
	}
	if code := c.do("POST", "/api/plan", map[string]any{"date": "2026-10-07", "meal": "dinner", "leftovers_of": tue.ID}, nil); code != 400 {
		t.Fatalf("leftovers of leftovers: %d", code)
	}

	var plan struct {
		Days []struct {
			Date  string
			Meals []planMeal
		}
	}
	c.do("GET", "/api/plan?from=2026-10-05&days=2", nil, &plan)
	m, l := plan.Days[0].Meals[0], plan.Days[1].Meals[0]
	if m.Extra != 2 || len(m.For) != 1 || m.For[0].Date != "2026-10-06" {
		t.Fatalf("Monday: %+v", m)
	}
	if l.From == nil || l.From.ID != mon.ID || l.Recipe == nil || l.Recipe.ID != roast.ID {
		t.Fatalf("Tuesday: %+v", l)
	}

	c.do("POST", "/api/plan/shopping", map[string]any{"from": "2026-10-05", "days": 2}, nil)
	var list struct {
		Items []struct {
			Name string
			Qty  float64
		}
	}
	c.do("GET", "/api/shopping", nil, &list)
	if len(list.Items) != 2 {
		t.Fatalf("the list should have chicken and lemons once: %+v", list.Items)
	}
	for _, it := range list.Items {
		if strings.Contains(it.Name, "chicken") && it.Qty != 6 {
			t.Fatalf("chicken for 6 servings (4 lb × 6/4): %+v", list.Items)
		}
		if strings.Contains(it.Name, "lemon") && it.Qty != 3 {
			t.Fatalf("lemons: %+v", list.Items)
		}
	}

	// Removing Monday removes its leftovers too.
	c.do("DELETE", "/api/plan/"+itoa(mon.ID), nil, nil)
	c.do("GET", "/api/plan?from=2026-10-05&days=2", nil, &plan)
	if len(plan.Days[1].Meals) != 0 {
		t.Fatalf("leftovers of a removed meal stay: %+v", plan.Days[1].Meals)
	}
}
