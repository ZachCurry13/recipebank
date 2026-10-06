package api

import (
	"fmt"
	"testing"
)

// "We cooked it" with thumbs: kept on the recipe, and Tonight's ideas put
// what the people eating liked first, then untried, then just cooked, then 👎.
func TestCookedItAndLikes(t *testing.T) {
	c, _ := setup(t)
	var a, b struct{ ID int64 }
	c.do("POST", "/api/people", map[string]any{"name": "A", "heat_max": -1}, &a)
	c.do("POST", "/api/people", map[string]any{"name": "B", "heat_max": -1}, &b)
	add := func(title string) int64 {
		var rc struct{ ID int64 }
		c.do("POST", "/api/recipes", map[string]any{"title": title, "ingredients": []map[string]string{{"line": "1 cup rice"}}}, &rc)
		return rc.ID
	}
	tacos, soup, salad, stew := add("Tacos"), add("Soup"), add("Salad"), add("Stew")
	cook := func(id int64, on string, thumbs map[string]int) int {
		return c.do("POST", fmt.Sprintf("/api/recipes/%d/cooks", id), map[string]any{"cooked_on": on, "thumbs": thumbs}, nil)
	}
	cook(tacos, "2026-09-01", map[string]int{fmt.Sprint(a.ID): 1, fmt.Sprint(b.ID): 1, "999": 1})
	cook(soup, "2026-09-02", map[string]int{fmt.Sprint(a.ID): -1})
	cook(salad, "2026-10-04", map[string]int{fmt.Sprint(a.ID): 1})
	if code := cook(stew, "yesterday", nil); code != 400 {
		t.Fatalf("a bad date: %d", code)
	}

	var page struct {
		Cooks []struct {
			ID     int64
			Thumbs map[string]int
		}
	}
	c.do("GET", fmt.Sprintf("/api/recipes/%d", tacos), nil, &page)
	if len(page.Cooks) != 1 || len(page.Cooks[0].Thumbs) != 2 {
		t.Fatalf("tacos' cooks (an unknown person is dropped): %+v", page.Cooks)
	}

	var tonight struct {
		Ideas []struct {
			ID    int64
			Liked bool
		}
	}
	c.do("GET", "/api/tonight?date=2026-10-06", nil, &tonight)
	var order []int64
	for _, i := range tonight.Ideas {
		order = append(order, i.ID)
	}
	if fmt.Sprint(order) != fmt.Sprint([]int64{tacos, stew, salad, soup}) || !tonight.Ideas[0].Liked || tonight.Ideas[1].Liked {
		t.Fatalf("ideas %v (tacos %d, stew %d, salad %d, soup %d): %+v", order, tacos, stew, salad, soup, tonight.Ideas)
	}

	if code := c.do("DELETE", fmt.Sprintf("/api/cooks/%d", page.Cooks[0].ID), nil, nil); code != 200 {
		t.Fatalf("delete: %d", code)
	}
	c.do("GET", fmt.Sprintf("/api/recipes/%d/cooks", tacos), nil, &page)
	if len(page.Cooks) != 0 {
		t.Fatalf("after delete: %+v", page.Cooks)
	}
}
