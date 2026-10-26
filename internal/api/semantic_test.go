package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeEmbedder answers like an embeddings service whose "meaning" has two
// directions: warm food and cold food. Every text also shares a small base.
func fakeEmbedder(t *testing.T, calls *int) *httptest.Server {
	count := func(text string, words ...string) float32 {
		var n float32
		for _, w := range words {
			n += float32(strings.Count(strings.ToLower(text), w))
		}
		return n
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/embeddings") {
			http.NotFound(w, r)
			return
		}
		*calls++
		var req struct {
			Model string
			Input []string
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var data []map[string]any
		for i, text := range req.Input {
			data = append(data, map[string]any{"index": i, "embedding": []float32{
				count(text, "soup", "stew", "hot", "warm", "winter", "bowl"), count(text, "salad", "fruit", "cold", "summer", "fresh"), 1}})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
}

// With an embedding model chosen, recipes are indexed in the background and
// searches also find recipes close in meaning, after the word matches.
func TestSearchByMeaning(t *testing.T) {
	c, srv := setup(t)
	calls := 0
	emb := fakeEmbedder(t, &calls)
	defer emb.Close()
	useAI(srv.Store, emb.URL)
	for _, rc := range []map[string]any{
		{"title": "Tomato soup", "ingredients": []map[string]string{{"line": "4 tomatoes"}}},
		{"title": "Fruit salad", "ingredients": []map[string]string{{"line": "2 apples"}}},
		{"title": "Winter bowl", "ingredients": []map[string]string{{"line": "1 cup rice"}}},
	} {
		c.do("POST", "/api/recipes", rc, nil)
	}
	titles := func(path string) []string {
		var out struct{ Recipes []struct{ Title string } }
		c.do("GET", path, nil, &out)
		var ts []string
		for _, r := range out.Recipes {
			ts = append(ts, r.Title)
		}
		return ts
	}
	if got := titles("/api/recipes?q=hot+bowl+for+tonight"); len(got) != 0 {
		t.Fatalf("off: words only: %v", got)
	}

	var st struct {
		Model          string
		Ready          bool
		Indexed, Total int
	}
	if code := c.do("PUT", "/api/admin/embed", map[string]string{"model": "nomic-embed-text"}, &st); code != 200 || !st.Ready || st.Indexed != 0 {
		t.Fatalf("turn on: %d %+v", code, st)
	}
	if code := c.do("PUT", "/api/admin/embed", map[string]string{"model": "bad name"}, nil); code != 400 {
		t.Fatalf("bad model name: %d", code)
	}
	srv.indexRecipes(context.Background())
	c.do("GET", "/api/admin/embed", nil, &st)
	if st.Indexed != 3 || st.Total != 3 {
		t.Fatalf("indexed: %+v", st)
	}
	before := calls
	srv.indexRecipes(context.Background())
	if calls != before {
		t.Fatal("nothing changed: nothing is embedded again")
	}

	// "hot bowl for tonight" shares no words with "Tomato soup", but its meaning is close.
	got := titles("/api/recipes?q=hot+bowl+for+tonight")
	if len(got) != 2 || got[0] != "Tomato soup" && got[0] != "Winter bowl" || got[0] == "Fruit salad" || got[1] == "Fruit salad" {
		t.Fatalf("by meaning: %v", got)
	}
	// A word match comes first, then the meaning matches.
	if got := titles("/api/recipes?q=soup"); len(got) < 1 || got[0] != "Tomato soup" {
		t.Fatalf("word first: %v", got)
	}
	var found struct {
		Groups []struct {
			Key  string
			Hits []struct{ Title string }
		}
	}
	c.do("GET", "/api/find?q=hot+bowl+for+tonight", nil, &found)
	if len(found.Groups) == 0 || found.Groups[0].Key != "recipes" || len(found.Groups[0].Hits) != 2 {
		t.Fatalf("search everything: %+v", found)
	}

	// An edited recipe is embedded again; a new model drops the old vectors.
	var id struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Cold summer fruit cup", "ingredients": []map[string]string{{"line": "1 peach"}}}, &id)
	srv.indexRecipes(context.Background())
	c.do("GET", "/api/admin/embed", nil, &st)
	if st.Indexed != 4 {
		t.Fatalf("a new recipe is indexed: %+v", st)
	}
	c.do("PUT", "/api/admin/embed", map[string]string{"model": "other-model"}, &st)
	if st.Indexed != 0 {
		t.Fatalf("a new model starts over: %+v", st)
	}

	// The AI being off never breaks the search.
	emb.Close()
	if got := titles("/api/recipes?q=soup"); len(got) != 1 {
		t.Fatalf("AI off: %v", got)
	}
}
