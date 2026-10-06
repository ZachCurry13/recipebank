package api

import (
	"fmt"
	"strings"
	"testing"
)

// Corrections made while checking a card teach the reader; later edits
// (after "Checked against the card") are recipe changes and don't.
func TestTeachTheReaderFromCorrections(t *testing.T) {
	c, srv := setup(t)
	var sides []int
	ai := fakeVisionAI(t, 2000, &sides)
	defer ai.Close()
	useAI(srv.Store, ai.URL)

	var draft struct {
		Recipe map[string]any `json:"recipe"`
	}
	c.do("POST", "/api/import/photo", photoBody(800, 600), &draft)
	ings := draft.Recipe["ingredients"].([]any)
	ings[2].(map[string]any)["line"] = "1 T. baking soda" // the card says soda; the AI read powder
	ings[2].(map[string]any)["food"] = ""
	var saved struct{ ID int64 }
	if code := c.do("POST", "/api/recipes", draft.Recipe, &saved); code != 200 {
		t.Fatalf("save: %d", code)
	}
	type hintList struct {
		Hints []struct {
			ID           int64
			Wrong, Right string
			Times        int
		}
	}
	var hints hintList
	c.do("GET", "/api/admin/hints", nil, &hints)
	if len(hints.Hints) != 1 || hints.Hints[0].Wrong != "baking powder" || hints.Hints[0].Right != "baking soda" {
		t.Fatalf("learned: %+v", hints)
	}
	if got := strings.Join(srv.readingHints(), "; "); got != `"baking powder" was really "baking soda"` {
		t.Fatalf("hints for the reader: %s", got)
	}

	// After checking, changing soda back to powder is a recipe change.
	c.do("POST", fmt.Sprintf("/api/recipes/%d/card-checked", saved.ID), map[string]any{}, nil)
	var page struct {
		Recipe map[string]any `json:"recipe"`
	}
	c.do("GET", fmt.Sprintf("/api/recipes/%d", saved.ID), nil, &page)
	page.Recipe["ingredients"].([]any)[2].(map[string]any)["line"] = "1 T. baking powder"
	page.Recipe["ingredients"].([]any)[2].(map[string]any)["food"] = ""
	c.do("PUT", fmt.Sprintf("/api/recipes/%d", saved.ID), page.Recipe, nil)
	c.do("GET", "/api/admin/hints", nil, &hints)
	if len(hints.Hints) != 1 {
		t.Fatalf("learned from a later edit: %+v", hints)
	}

	if code := c.do("POST", "/api/admin/hints", map[string]string{"wrong": "T.", "right": "tablespoon"}, &hints); code != 200 || len(hints.Hints) != 2 {
		t.Fatalf("add: %d %+v", code, hints)
	}
	c.do("DELETE", fmt.Sprintf("/api/admin/hints/%d", hints.Hints[0].ID), nil, &hints)
	if len(hints.Hints) != 1 {
		t.Fatalf("delete: %+v", hints)
	}
}
