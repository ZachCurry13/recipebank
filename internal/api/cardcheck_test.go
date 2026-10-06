package api

import (
	"fmt"
	"strings"
	"testing"
)

// A photo recipe nobody has checked is "not sure" for an allergy, says what
// to look at, and "Checked against the card" clears it.
func TestCheckedAgainstTheCard(t *testing.T) {
	c, _ := setup(t)
	c.do("POST", "/api/people", map[string]any{"name": "Kid", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "allergic"}}}, nil)
	var saved struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Rice bowl", "source_kind": "photo", "needs_review": true,
		"ingredients": []map[string]any{{"line": "1 cup rice"}, {"line": "1 cup peas", "unsure": true}},
		"steps":       []map[string]string{{"text": "Cook the rice with the carrots."}}}, &saved)
	type page struct {
		Recipe struct {
			NeedsReview bool `json:"needs_review"`
			Ingredients []struct {
				Unsure bool `json:"unsure"`
			} `json:"ingredients"`
		} `json:"recipe"`
		Verdicts  []struct{ Status string } `json:"verdicts"`
		CardCheck []string                  `json:"card_check"`
	}
	var before page
	c.do("GET", fmt.Sprintf("/api/recipes/%d", saved.ID), nil, &before)
	if len(before.Verdicts) != 1 || before.Verdicts[0].Status != "unsure" {
		t.Fatalf("before: %+v", before)
	}
	if len(before.CardCheck) != 1 || !strings.Contains(before.CardCheck[0], "carrot") {
		t.Fatalf("card check: %q", before.CardCheck)
	}
	var after page
	if code := c.do("POST", fmt.Sprintf("/api/recipes/%d/card-checked", saved.ID), map[string]any{}, &after); code != 200 {
		t.Fatalf("card-checked: %d", code)
	}
	if after.Recipe.NeedsReview || after.Recipe.Ingredients[1].Unsure || after.Verdicts[0].Status != "ok" || after.CardCheck != nil {
		t.Fatalf("after: %+v", after)
	}
}
