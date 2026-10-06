package api

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPlanMyWeek(t *testing.T) {
	c, _ := setup(t)
	var kid struct{ ID int64 }
	c.do("POST", "/api/people", map[string]any{"name": "Kid", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "allergic"}}}, &kid)
	c.do("POST", "/api/people", map[string]any{"name": "Parent", "heat_max": -1}, nil)
	add := func(title, course string, servings float64, mins int, lines ...string) int64 {
		var ings []map[string]string
		for _, l := range lines {
			ings = append(ings, map[string]string{"line": l})
		}
		var rc struct{ ID int64 }
		c.do("POST", "/api/recipes", map[string]any{"title": title, "course": course, "servings": servings, "total_min": mins, "ingredients": ings}, &rc)
		return rc.ID
	}
	stirfry := add("Chicken stir fry", "main", 2, 25, "1 lb chicken", "1 cup rice")
	mac := add("Mac and cheese", "main", 2, 30, "8 oz macaroni", "2 cups cheddar cheese")
	chili := add("Big pot of chili", "main", 12, 90, "2 lb ground beef", "2 cans kidney beans")
	brownies := add("Brownies", "dessert", 2, 40, "1 cup sugar", "1 cup cocoa")
	tacos := add("Tacos", "main", 2, 30, "1 lb ground turkey", "8 tortillas")
	soup := add("Lentil soup", "", 2, 60, "1 cup lentils", "1 onion")
	spinach := add("Spinach pasta", "main", 2, 20, "8 oz pasta", "4 cups spinach")
	// Liked tacos long ago; the kid disliked the soup.
	c.do("POST", fmt.Sprintf("/api/recipes/%d/cooks", tacos), map[string]any{"cooked_on": "2026-08-01", "thumbs": map[string]int{fmt.Sprint(kid.ID): 1}}, nil)
	c.do("POST", fmt.Sprintf("/api/recipes/%d/cooks", soup), map[string]any{"cooked_on": "2026-08-02", "thumbs": map[string]int{fmt.Sprint(kid.ID): -1}}, nil)
	monday := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	c.do("POST", "/api/stock", map[string]any{"area": "kitchen", "name": "Spinach", "qty": 1, "use_by": monday.AddDate(0, 0, 1).Format(day)}, nil)
	// Wednesday is planned already.
	c.do("POST", "/api/plan", map[string]any{"date": monday.AddDate(0, 0, 2).Format(day), "meal": "dinner", "title": "Pizza night out"}, nil)

	var draft struct {
		Days []struct {
			Date        string
			Recipe      *planRecipe
			Why         []string
			LeftoversOf string `json:"leftovers_of"`
		}
	}
	if code := c.do("POST", "/api/plan/suggest", map[string]any{"from": monday.Format(day), "days": 7}, &draft); code != 200 {
		t.Fatalf("suggest: %d", code)
	}
	seen := map[int64]int{}
	dates := map[string]bool{}
	var leftovers, liked, usesUp bool
	for _, d := range draft.Days {
		dates[d.Date] = true
		if d.Recipe == nil {
			t.Fatalf("a day without a recipe: %+v", d)
		}
		switch d.Recipe.ID {
		case mac, brownies, soup:
			t.Fatalf("%s shouldn't be picked: %+v", d.Recipe.Title, d)
		}
		if d.LeftoversOf != "" {
			leftovers = leftovers || d.Recipe.ID == chili
			continue
		}
		seen[d.Recipe.ID]++
		why := strings.Join(d.Why, ", ")
		liked = liked || (d.Recipe.ID == tacos && strings.Contains(why, "liked"))
		usesUp = usesUp || (d.Recipe.ID == spinach && strings.Contains(why, "uses up Spinach"))
	}
	if dates[monday.AddDate(0, 0, 2).Format(day)] {
		t.Fatal("Wednesday was planned already")
	}
	for id, n := range seen {
		if n > 1 {
			t.Fatalf("recipe %d picked %d times", id, n)
		}
	}
	if !liked || !usesUp || !leftovers || seen[stirfry] != 1 || seen[chili] != 1 {
		t.Fatalf("draft: %+v", draft.Days)
	}
}
