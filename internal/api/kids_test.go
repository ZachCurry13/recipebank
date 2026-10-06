package api

import (
	"fmt"
	"testing"
)

// A recipe says which steps need a grown-up, and the Library finds the ones
// kids can help with.
func TestKidsCanHelp(t *testing.T) {
	c, _ := setup(t)
	add := func(title string, steps ...string) int64 {
		var st []map[string]string
		for _, s := range steps {
			st = append(st, map[string]string{"text": s})
		}
		var rc struct{ ID int64 }
		c.do("POST", "/api/recipes", map[string]any{"title": title, "ingredients": []map[string]string{{"line": "1 cup flour"}}, "steps": st}, &rc)
		return rc.ID
	}
	cookies := add("Cookies", "Preheat the oven to 350°F.", "Stir the butter and sugar.", "Roll into balls.", "Bake 12 minutes.")
	add("Steak", "Heat a skillet over high heat.", "Sear the steak.", "Slice it.")

	var got struct {
		Needs [][]string `json:"needs"`
		Kids  bool       `json:"kids"`
	}
	c.do("GET", fmt.Sprintf("/api/recipes/%d", cookies), nil, &got)
	if !got.Kids || len(got.Needs) != 4 || len(got.Needs[0]) != 1 || got.Needs[0][0] != "oven" || len(got.Needs[1]) != 0 {
		t.Fatalf("cookies: %+v", got)
	}
	var lib struct {
		Recipes []struct{ ID int64 } `json:"recipes"`
	}
	c.do("GET", "/api/recipes?area=kitchen&kids=1", nil, &lib)
	if len(lib.Recipes) != 1 || lib.Recipes[0].ID != cookies {
		t.Fatalf("kids can help: %+v", lib.Recipes)
	}
}
