package api

import (
	"testing"

	"github.com/zachcurry13/recipebank/internal/store"
)

type shopList struct {
	Items []store.ShopItem  `json:"items"`
	Low   []store.StockItem `json:"low"`
}

func (l shopList) find(name string) *store.ShopItem {
	for i := range l.Items {
		if l.Items[i].Name == name {
			return &l.Items[i]
		}
	}
	return nil
}

func TestShoppingList(t *testing.T) {
	c, _ := setup(t)
	var a, b struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Pancakes", "ingredients": []map[string]string{
		{"line": "2 cups flour"}, {"line": "2 large eggs"}, {"line": "1 cup milk"}}}, &a)
	c.do("POST", "/api/recipes", map[string]any{"title": "Crepes", "ingredients": []map[string]string{
		{"line": "4 tbsp flour"}, {"line": "1 egg"}, {"line": "1 tsp salt"}}}, &b)
	c.do("POST", "/api/stock", map[string]any{"area": "kitchen", "name": "Morton Kosher Salt", "qty": 1}, nil)

	var res struct {
		Added int        `json:"added"`
		Have  []haveItem `json:"have"`
	}
	c.do("POST", "/api/shopping/recipe", map[string]any{"id": a.ID, "factor": 2}, &res)
	c.do("POST", "/api/shopping/recipe", map[string]any{"id": b.ID}, &res)
	if res.Added != 2 || len(res.Have) != 1 || res.Have[0].Food != "salt" {
		t.Fatalf("crepes: added %d, have %+v", res.Added, res.Have)
	}
	var list shopList
	c.do("GET", "/api/shopping", nil, &list)
	flour, eggs := list.find("flour"), list.find("large eggs")
	if flour == nil || flour.Qty != 4.25 || flour.Unit != "cup" || flour.Note != "for Pancakes, Crepes" {
		t.Fatalf("flour: %+v", flour)
	}
	if eggs == nil || eggs.Qty != 5 || len(list.Items) != 3 {
		t.Fatalf("eggs: %+v (items %d)", eggs, len(list.Items))
	}

	// Using the last toothpaste puts it on the list; buying it restocks it.
	var paste store.StockItem
	c.do("POST", "/api/stock", map[string]any{"area": "home", "name": "Toothpaste", "qty": 2, "low_at": 1}, &paste)
	c.do("POST", "/api/stock/"+itoa(paste.ID)+"/adjust", map[string]float64{"delta": -1}, nil)
	c.do("POST", "/api/stock/"+itoa(paste.ID)+"/adjust", map[string]float64{"delta": -1}, nil)
	c.do("GET", "/api/shopping", nil, &list)
	tp := list.find("Toothpaste")
	if tp == nil || tp.Section != "Personal care" || len(list.Low) != 0 {
		t.Fatalf("toothpaste on the list once: %+v, low %d", tp, len(list.Low))
	}
	checked := true
	c.do("PUT", "/api/shopping/"+itoa(tp.ID), store.ShopChange{Checked: &checked}, nil)
	var cleared struct {
		Cleared   int      `json:"cleared"`
		Restocked []string `json:"restocked"`
	}
	c.do("POST", "/api/shopping/clear", nil, &cleared)
	var after store.StockItem
	c.do("POST", "/api/stock/"+itoa(paste.ID)+"/adjust", map[string]float64{"delta": 0}, &after)
	if cleared.Cleared != 1 || len(cleared.Restocked) != 1 || after.Qty != 1 {
		t.Fatalf("cleared %+v, toothpaste now %v", cleared, after.Qty)
	}

	// Typed items are understood too.
	c.do("POST", "/api/shopping", map[string]string{"text": "2 lb apples"}, &list)
	if ap := list.find("apples"); ap == nil || ap.Qty != 2 || ap.Unit != "lb" || ap.Section != "Produce" {
		t.Fatalf("apples: %+v", ap)
	}
}
