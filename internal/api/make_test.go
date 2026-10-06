package api

import (
	"fmt"
	"strings"
	"testing"
)

// "What can I make?": recipes needing least come first, ones with nothing on
// hand are left out, and what's missing is listed by line for "Substitute".
func TestWhatCanIMake(t *testing.T) {
	c, srv := setup(t)
	add := func(title string, lines ...string) int64 {
		var ings []map[string]string
		for _, l := range lines {
			ings = append(ings, map[string]string{"line": l})
		}
		var rc struct{ ID int64 }
		c.do("POST", "/api/recipes", map[string]any{"title": title, "ingredients": ings}, &rc)
		return rc.ID
	}
	omelet := add("Omelet", "3 eggs", "1/4 cup cheddar cheese", "salt and pepper")
	cookies := add("Cookies", "2 cups flour", "1 cup butter", "1 cup sugar", "2 eggs")
	add("Chili", "1 lb ground beef", "1 onion", "2 tsp cumin")

	type result struct {
		Card struct {
			ID int64 `json:"id"`
		} `json:"card"`
		Missing []string `json:"missing"`
		Lines   []int    `json:"missing_lines"`
	}
	var out struct {
		Results []result `json:"results"`
	}
	if code := c.do("POST", "/api/make", map[string]any{"have": []string{"eggs", "cheddar"}}, &out); code != 200 {
		t.Fatalf("make: %d", code)
	}
	if len(out.Results) != 2 || out.Results[0].Card.ID != omelet || len(out.Results[0].Missing) != 0 {
		t.Fatalf("results: %+v", out.Results)
	}
	if r := out.Results[1]; r.Card.ID != cookies || len(r.Missing) != 3 || fmt.Sprint(r.Lines) != "[0 1 2]" {
		t.Fatalf("cookies: %+v", r)
	}
	if code := c.do("POST", "/api/make", map[string]any{"have": []string{}}, nil); code != 400 {
		t.Fatalf("nothing on hand: %d", code)
	}

	// Typed foods count as on hand for "Substitute" too.
	var sub substituteAnswer
	c.do("POST", fmt.Sprintf("/api/recipes/%d/substitute", cookies), map[string]any{"line": 1, "have": []string{"olive oil"}}, &sub)
	haveOil := false
	for _, i := range sub.Ideas {
		haveOil = haveOil || (strings.Contains(i.To, "oil") && i.Have)
	}
	if !haveOil {
		t.Fatalf("butter → oil should show as on hand: %+v", sub.Ideas)
	}

	// A fridge photo becomes a list to tick.
	var prompts []string
	ai := fakePickAI(`{"foods": ["eggs", "Cheddar cheese", "eggs", "carrots"]}`, &prompts)
	defer ai.Close()
	useAI(srv.Store, ai.URL)
	var seen struct {
		Foods []string `json:"foods"`
	}
	if code := c.do("POST", "/api/make/photo", photoBody(800, 600), &seen); code != 200 || strings.Join(seen.Foods, ",") != "eggs,Cheddar cheese,carrots" {
		t.Fatalf("photo: %d %+v", code, seen)
	}
}
