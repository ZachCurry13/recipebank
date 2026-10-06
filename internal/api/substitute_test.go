package api

import (
	"fmt"
	"strings"
	"testing"
)

type substituteAnswer struct {
	Ideas []struct {
		To   string
		Have bool
		ByAI bool `json:"by_ai"`
	}
	Skipped int
	AIError string `json:"ai_error"`
}

// "No cumin?": ideas from the table, what's in the pantry first, nothing
// that doesn't suit everyone eating, and AI ideas checked the same way.
func TestSubstitute(t *testing.T) {
	c, srv := setup(t)
	c.do("POST", "/api/people", map[string]any{"name": "Kid", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "allergic"}}}, nil)
	var rc struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Chili", "ingredients": []map[string]string{
		{"line": "1 lb ground beef"}, {"line": "2 tsp cumin"}, {"line": "1 cup buttermilk"}}}, &rc)
	c.do("POST", "/api/stock", map[string]any{"area": "kitchen", "name": "Smoked Paprika", "qty": 1}, nil)
	path := fmt.Sprintf("/api/recipes/%d/substitute", rc.ID)

	var cumin substituteAnswer
	if code := c.do("POST", path, map[string]any{"line": 1}, &cumin); code != 200 || len(cumin.Ideas) < 3 {
		t.Fatalf("cumin: %d %+v", code, cumin)
	}
	if cumin.Ideas[0].To != "smoked paprika" || !cumin.Ideas[0].Have || cumin.Ideas[1].Have {
		t.Fatalf("the pantry's smoked paprika should come first: %+v", cumin.Ideas)
	}

	var milk substituteAnswer
	c.do("POST", path, map[string]any{"line": 2}, &milk)
	if milk.Skipped < 2 {
		t.Fatalf("dairy ideas should be left out for a milk allergy: %+v", milk)
	}
	for _, i := range milk.Ideas {
		if strings.Contains(i.To, "yogurt") || i.To == "milk and lemon juice" {
			t.Fatalf("offered %q to someone allergic to milk", i.To)
		}
	}

	var prompts []string
	ai := fakePickAI(`{"ideas": [{"to": "cheddar cheese", "note": "no"}, {"to": "ground coriander and chili powder", "note": "half and half"}]}`, &prompts)
	defer ai.Close()
	useAI(srv.Store, ai.URL)
	var asked substituteAnswer
	c.do("POST", path, map[string]any{"line": 1, "ai": true}, &asked)
	byAI := 0
	for _, i := range asked.Ideas {
		if i.ByAI {
			byAI++
			if strings.Contains(i.To, "cheddar") {
				t.Fatalf("an AI idea with milk got through: %+v", asked)
			}
		}
	}
	if byAI != 1 || asked.Skipped != cumin.Skipped+1 || asked.AIError != "" {
		t.Fatalf("AI ideas: %+v", asked)
	}
	if len(prompts) != 1 || !strings.Contains(prompts[0], `"2 tsp cumin"`) || !strings.Contains(prompts[0], "Smoked Paprika") ||
		strings.Contains(prompts[0], "Kid") || strings.Contains(prompts[0], "allerg") {
		t.Fatalf("what the AI was told: %q", prompts)
	}
}
