package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakePickAI answers every question with ids (as a careless AI might).
func fakePickAI(answer string, prompts *[]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Messages) > 1 {
			*prompts = append(*prompts, req.Messages[1].Content)
		}
		resp, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": answer}}}})
		w.Write(resp)
	}))
}

func TestCollectionsAndSeasons(t *testing.T) {
	c, srv := setup(t)
	var soup, mac, cookies struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Lentil soup", "total_min": 40, "ingredients": []map[string]string{{"line": "1 cup lentils"}, {"line": "1 onion"}}}, &soup)
	c.do("POST", "/api/recipes", map[string]any{"title": "Mac and cheese", "total_min": 25, "ingredients": []map[string]string{{"line": "8 oz macaroni"}, {"line": "2 cups cheddar cheese"}}}, &mac)
	c.do("POST", "/api/recipes", map[string]any{"title": "Gingerbread cookies", "ingredients": []map[string]string{{"line": "3 cups flour"}}}, &cookies)

	var col struct{ ID int64 }
	c.do("POST", "/api/collections", map[string]string{"name": "Cozy dinners", "description": "warm dairy-free dinners"}, &col)

	var prompts []string
	ai := fakePickAI(fmt.Sprintf(`{"ids": [%d, %d, 999], "why": {"%d": "warm and filling"}}`, soup.ID, mac.ID, soup.ID), &prompts)
	defer ai.Close()
	useAI(srv.Store, ai.URL)

	var sug suggestion
	if code := c.do("POST", "/api/collections/"+itoa(col.ID)+"/suggest", nil, &sug); code != 200 || !sug.UsedAI {
		t.Fatalf("suggest: %d %+v", code, sug)
	}
	if len(sug.Picks) != 1 || sug.Picks[0].ID != soup.ID || sug.Picks[0].Why != "warm and filling" {
		t.Fatalf("picks: %+v", sug.Picks)
	}
	if len(sug.LeftOut) != 1 || sug.LeftOut[0].ID != mac.ID || !strings.Contains(sug.LeftOut[0].Reason, "dairy") {
		t.Fatalf("mac and cheese must be left out as not dairy-free: %+v", sug.LeftOut)
	}
	if len(prompts) != 1 || !strings.Contains(prompts[0], "Lentil soup") {
		t.Fatalf("the AI didn't get the list: %v", prompts)
	}

	c.do("POST", "/api/collections/"+itoa(col.ID)+"/recipes", map[string][]int64{"ids": {soup.ID}}, nil)
	var page struct {
		Recipes []recipeCard `json:"recipes"`
	}
	c.do("GET", "/api/collections/"+itoa(col.ID), nil, &page)
	if len(page.Recipes) != 1 || page.Recipes[0].ID != soup.ID {
		t.Fatalf("collection: %+v", page.Recipes)
	}

	var advent struct {
		Recipes []recipeCard `json:"recipes"`
	}
	c.do("GET", "/api/seasons/advent", nil, &advent)
	if len(advent.Recipes) != 1 || advent.Recipes[0].ID != cookies.ID {
		t.Fatalf("Advent baking: %+v", advent.Recipes)
	}
	var lists struct {
		Seasons []seasonShelf `json:"seasons"`
	}
	c.do("GET", "/api/collections?date=2026-12-10&area=kitchen", nil, &lists)
	found := false
	for _, s := range lists.Seasons {
		found = found || s.Key == "advent"
	}
	if !found {
		t.Fatalf("Advent baking should be in season on Dec 10: %+v", lists.Seasons)
	}
}
