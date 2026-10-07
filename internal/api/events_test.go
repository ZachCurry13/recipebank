package api

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// A holiday dinner: who's coming, every dish checked for them, who can eat
// how many dishes, what others bring, and the family's dishes bought for all.
func TestEvents(t *testing.T) {
	c, _ := setup(t)
	var kid, aunt struct{ ID int64 }
	c.do("POST", "/api/people", map[string]any{"name": "Kid", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "allergic"}}}, &kid)
	c.do("POST", "/api/people", map[string]any{"name": "Aunt", "is_guest": true, "heat_max": -1,
		"rules": []map[string]string{{"kind": "diet", "key": "vegetarian"}}}, &aunt)
	var turkey, potatoes struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Roast turkey", "servings": 4,
		"ingredients": []map[string]string{{"line": "12 lb turkey"}, {"line": "2 tbsp olive oil"}}}, &turkey)
	c.do("POST", "/api/recipes", map[string]any{"title": "Mashed potatoes", "servings": 4,
		"ingredients": []map[string]string{{"line": "4 lb potatoes"}, {"line": "1 cup milk"}}}, &potatoes)

	today := time.Now().Format(day)
	var ev struct{ ID int64 }
	if code := c.do("POST", "/api/events", map[string]any{"name": "Thanksgiving", "date": today,
		"who": []int64{kid.ID, aunt.ID, 9999}, "extra": 6}, &ev); code != 200 {
		t.Fatalf("new event: %d", code)
	}
	if code := c.do("POST", "/api/events", map[string]any{"name": " ", "date": today}, nil); code != 400 {
		t.Fatalf("no name: %d", code)
	}
	base := fmt.Sprintf("/api/events/%d", ev.ID)
	c.do("POST", base+"/dishes", map[string]any{"recipe_id": turkey.ID}, nil)
	c.do("POST", base+"/dishes", map[string]any{"recipe_id": potatoes.ID}, nil)
	type eventPage struct {
		Event struct {
			Who []int64
		}
		Dishes []struct {
			ID       int64
			Title    string
			Brings   string
			Verdicts []status
		}
		Coming    []eventPerson
		Headcount int
	}
	var page eventPage
	if code := c.do("POST", base+"/dishes", map[string]any{"title": "Pumpkin pie", "brings": "Aunt"}, &page); code != 200 {
		t.Fatalf("a dish someone brings: %d", code)
	}
	if len(page.Event.Who) != 2 || page.Headcount != 8 || len(page.Dishes) != 3 {
		t.Fatalf("event (an unknown person is dropped): %+v", page)
	}
	ok := map[string]int{}
	for _, p := range page.Coming {
		ok[p.Name] = p.OK
	}
	// Turkey: fine for Kid, not for the vegetarian Aunt; potatoes: milk, so not for Kid; pie has no recipe.
	if ok["Kid"] != 1 || ok["Aunt"] != 1 {
		t.Fatalf("who can eat what: %+v", page.Coming)
	}
	// The pie has no recipe: "not sure" for a real allergy or a diet (unknown is never OK).
	if len(page.Dishes[0].Verdicts) != 2 || page.Dishes[2].Brings != "Aunt" || len(page.Dishes[2].Verdicts) != 2 ||
		page.Dishes[2].Verdicts[0].Status != "unsure" || page.Dishes[2].Verdicts[1].Status != "unsure" {
		t.Fatalf("dishes: %+v", page.Dishes)
	}
	c.do("PUT", fmt.Sprintf("%s/dishes/%d", base, page.Dishes[1].ID), map[string]string{"brings": "Grandma"}, &page)
	if page.Dishes[1].Brings != "Grandma" {
		t.Fatalf("brings: %+v", page.Dishes[1])
	}

	// Only the turkey is the family's: bought for 8 (12 lb × 8/4).
	c.do("POST", base+"/shopping", map[string]any{}, nil)
	var list struct {
		Items []struct {
			Name string
			Qty  float64
		}
	}
	c.do("GET", "/api/shopping", nil, &list)
	turkeyFor8 := false
	for _, it := range list.Items {
		if strings.Contains(it.Name, "potato") {
			t.Fatalf("Grandma brings the potatoes: %+v", list.Items)
		}
		turkeyFor8 = turkeyFor8 || (strings.Contains(it.Name, "turkey") && it.Qty == 24)
	}
	if !turkeyFor8 {
		t.Fatalf("shopping: %+v", list.Items)
	}

	var tonight struct {
		Events []struct{ Name string }
	}
	c.do("GET", "/api/tonight?date="+today, nil, &tonight)
	if len(tonight.Events) != 1 || tonight.Events[0].Name != "Thanksgiving" {
		t.Fatalf("tonight: %+v", tonight.Events)
	}
	var all struct {
		Events []eventCard
	}
	c.do("GET", "/api/events", nil, &all)
	if len(all.Events) != 1 || all.Events[0].Dishes != 3 || all.Events[0].People != 8 {
		t.Fatalf("list: %+v", all.Events)
	}
	c.do("DELETE", base, nil, nil)
	if code := c.do("GET", base, nil, nil); code != 404 {
		t.Fatalf("after delete: %d", code)
	}
}

// Someone taken off the Family page no longer counts as coming.
func TestEventCountsOnlyPeopleStillThere(t *testing.T) {
	c, _ := setup(t)
	var p struct{ ID int64 }
	c.do("POST", "/api/people", map[string]any{"name": "Cousin", "heat_max": -1}, &p)
	c.do("POST", "/api/events", map[string]any{"name": "Picnic", "who": []int64{p.ID}, "extra": 2}, nil)
	c.do("DELETE", fmt.Sprintf("/api/people/%d", p.ID), nil, nil)
	var all struct {
		Events []eventCard
	}
	c.do("GET", "/api/events", nil, &all)
	if len(all.Events) != 1 || all.Events[0].People != 2 {
		t.Fatalf("list: %+v", all.Events)
	}
}
