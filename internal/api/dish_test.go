package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeDishAI answers a photo with a dish, a request to write a recipe with
// plain text, and the organizing step with recipe JSON.
func fakeDishAI(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		answer := `{"title":"Chicken tikka masala","ingredients":[{"line":"1 lb chicken thighs"},{"line":"1 cup heavy cream"},{"line":"1 can tomato sauce"}],"steps":[{"text":"Brown the chicken."},{"text":"Simmer in the sauce."}]}`
		switch {
		case bytes.Contains(raw, []byte("image_url")):
			answer = `{"name":"Chicken tikka masala","description":"Chicken in a creamy tomato curry sauce.","ingredients":["chicken","cream","tomato"],"cuisine":"Indian"}`
		case bytes.Contains(raw, []byte("Write a simple home recipe")):
			answer = "Chicken Tikka Masala (4 serv.)\nIngredients:\n1 lb chicken thighs\n1 cup heavy cream\n1 can tomato sauce\nInstructions:\nBrown the chicken.\nSimmer in the sauce."
		}
		resp, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": answer}}}})
		w.Write(resp)
	}))
}

// A photo of a dish finds the family's similar recipes, and the AI's best
// guess at it is a draft, checked for everyone like any other.
func TestDishPhoto(t *testing.T) {
	c, srv := setup(t)
	ai := fakeDishAI(t)
	defer ai.Close()
	useAI(srv.Store, ai.URL)
	c.do("POST", "/api/people", map[string]any{"name": "Kid", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "allergic"}}}, nil)
	var curry, cake struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Grandma's chicken curry", "ingredients": []map[string]string{{"line": "2 lb chicken"}, {"line": "1 can tomatoes"}}}, &curry)
	c.do("POST", "/api/recipes", map[string]any{"title": "Lemon cake", "ingredients": []map[string]string{{"line": "2 cups flour"}}}, &cake)

	var seen struct {
		Dish struct {
			Name string `json:"name"`
		} `json:"dish"`
		Matches []struct {
			ID int64 `json:"id"`
		} `json:"matches"`
	}
	if code := c.do("POST", "/api/import/dish", photoBody(800, 600), &seen); code != 200 || seen.Dish.Name != "Chicken tikka masala" {
		t.Fatalf("dish: %d %+v", code, seen)
	}
	if len(seen.Matches) != 1 || seen.Matches[0].ID != curry.ID {
		t.Fatalf("matches: %+v", seen.Matches)
	}

	var draft struct {
		Recipe   map[string]any   `json:"recipe"`
		Verdicts []map[string]any `json:"verdicts"`
	}
	if code := c.do("POST", "/api/import/dish/draft", map[string]any{"name": "Chicken tikka masala", "ingredients": []string{"chicken"}}, &draft); code != 200 {
		t.Fatalf("draft: %d", code)
	}
	if draft.Recipe["source_kind"] != "ai" || !strings.Contains(draft.Recipe["source_note"].(string), "best guess") ||
		draft.Recipe["servings"] != 4.0 || len(draft.Verdicts) != 1 || draft.Verdicts[0]["status"] != "no" {
		t.Fatalf("draft: %+v %+v", draft.Recipe, draft.Verdicts)
	}
	if code := c.do("POST", "/api/import/dish/draft", map[string]any{"name": "  "}, nil); code != 400 {
		t.Fatalf("no name: %d", code)
	}
	// The draft saves like any recipe (0.3 and 0.4 refused "ai" recipes).
	var saved struct{ ID int64 }
	if code := c.do("POST", "/api/recipes", draft.Recipe, &saved); code != 200 || saved.ID == 0 {
		t.Fatalf("save the draft: %d", code)
	}
}
