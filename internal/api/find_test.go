package api

import (
	"fmt"
	"testing"
)

// Search everything finds a word in recipes, the house, the list, the
// cookbooks and events, skips what's turned off, and needs every word.
func TestSearchEverything(t *testing.T) {
	c, _ := setup(t)
	c.do("POST", "/api/recipes", map[string]any{"title": "Lemon bars", "ingredients": []map[string]string{{"line": "2 lemons"}}}, nil)
	c.do("POST", "/api/recipes", map[string]any{"title": "Tuna melt", "ingredients": []map[string]string{{"line": "1 tbsp lemon juice"}}}, nil)
	c.do("POST", "/api/recipes", map[string]any{"title": "Lemon cleaner", "area": "home", "ingredients": []map[string]string{{"line": "1 cup vinegar"}}}, nil)
	c.do("POST", "/api/stock", map[string]any{"area": "kitchen", "name": "Lemons", "qty": 3}, nil)
	c.do("POST", "/api/shopping", map[string]any{"text": "lemon curd"}, nil)
	var book struct{ ID int64 }
	c.do("POST", "/api/books", map[string]string{"title": "Baking Book"}, &book)
	c.do("POST", fmt.Sprintf("/api/books/%d/entries", book.ID), map[string]any{"entries": []map[string]string{{"title": "Lemon meringue pie", "page": "88"}}}, nil)
	c.do("POST", "/api/events", map[string]any{"name": "Lemonade stand"}, nil)

	type res struct {
		Groups []struct {
			Key  string
			Hits []struct{ Title, Sub, Link string }
		}
	}
	var got res
	c.do("GET", "/api/find?q=lemon", nil, &got)
	keys := map[string]int{}
	for _, g := range got.Groups {
		keys[g.Key] = len(g.Hits)
	}
	if keys["recipes"] != 3 || keys["pantry"] != 1 || keys["shopping"] != 1 || keys["books"] != 1 || keys["events"] != 1 {
		t.Fatalf("groups: %v", keys)
	}
	if got.Groups[0].Key != "recipes" || got.Groups[0].Hits[2].Title != "Tuna melt" {
		t.Fatalf("title matches come first: %+v", got.Groups[0].Hits)
	}
	c.do("GET", "/api/find?q=lemon+meringue", nil, &got)
	if len(got.Groups) != 1 || got.Groups[0].Hits[0].Sub != "Baking Book, page 88" {
		t.Fatalf("every word: %+v", got.Groups)
	}
	c.do("PUT", "/api/admin/settings", map[string]string{"feature_home": "false", "feature_books": "false"}, nil)
	c.do("GET", "/api/find?q=lemon", nil, &got)
	for _, g := range got.Groups {
		if g.Key == "books" || g.Key == "recipes" && len(g.Hits) != 2 {
			t.Fatalf("turned-off features aren't searched: %+v", got.Groups)
		}
	}
	if c.do("GET", "/api/find?q=", nil, &got); len(got.Groups) != 0 {
		t.Fatalf("nothing typed: %+v", got)
	}
}
