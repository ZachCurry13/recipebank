package api

import (
	"fmt"
	"testing"
)

// From a recipe the person ticks which lines go on the list: the check says
// what the house has, and ticked lines are added even if it has them.
func TestPickLinesForTheList(t *testing.T) {
	c, _ := setup(t)
	var rc struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Pancakes", "ingredients": []map[string]string{
		{"line": "2 cups flour"}, {"line": "2 eggs"}, {"line": "1 cup milk"}}}, &rc)
	c.do("POST", "/api/stock", map[string]any{"area": "kitchen", "name": "Eggs", "qty": 12}, nil)

	var check struct {
		Have []*struct{ Stock string }
	}
	c.do("GET", fmt.Sprintf("/api/shopping/recipe/%d", rc.ID), nil, &check)
	if len(check.Have) != 3 || check.Have[0] != nil || check.Have[1] == nil || check.Have[1].Stock != "Eggs" || check.Have[2] != nil {
		t.Fatalf("check: %+v", check.Have)
	}
	var res struct{ Added int }
	c.do("POST", "/api/shopping/recipe", map[string]any{"id": rc.ID, "factor": 2, "only": []int{0, 1}}, &res)
	var list struct {
		Items []struct {
			Name string
			Qty  float64
		}
	}
	c.do("GET", "/api/shopping", nil, &list)
	if res.Added != 2 || len(list.Items) != 2 {
		t.Fatalf("added %d: %+v", res.Added, list.Items)
	}
	for _, it := range list.Items {
		if it.Name == "flour" && it.Qty != 4 {
			t.Fatalf("doubled: %+v", it)
		}
	}
}
