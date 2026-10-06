package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zachcurry13/recipebank/internal/store"
)

func TestNutritionEstimate(t *testing.T) {
	c, srv := setup(t)
	asked := 0
	fdc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked++
		if r.Header.Get("X-Api-Key") != "test-key" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if strings.Contains(r.URL.Query().Get("query"), "flour") {
			w.Write([]byte(`{"foods":[{"fdcId":1,"description":"Wheat flour","foodNutrients":[{"nutrientId":1008,"value":364}],
				"foodMeasures":[{"disseminationText":"1 cup","gramWeight":125}]}]}`))
			return
		}
		w.Write([]byte(`{"foods":[]}`))
	}))
	defer fdc.Close()
	srv.Nutrition = fdc.URL
	var rc struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Flatbread", "servings": 2,
		"ingredients": []map[string]string{{"line": "2 cups flour"}, {"line": "1 cup unicorn dust"}}}, &rc)
	path := fmt.Sprintf("/api/recipes/%d/nutrition", rc.ID)

	type answer struct {
		Estimate struct {
			PerServing struct {
				Kcal float64 `json:"kcal"`
			} `json:"per_serving"`
			NotCounted []string `json:"not_counted"`
			Lines      []struct {
				Food string `json:"food"`
			} `json:"lines"`
		} `json:"estimate"`
		KeySet bool   `json:"key_set"`
		Error  string `json:"error"`
	}
	var a answer
	c.do("GET", path, nil, &a)
	if a.KeySet || asked != 0 || len(a.Estimate.NotCounted) != 2 {
		t.Fatalf("without a key nothing is looked up: %+v (asked %d)", a, asked)
	}

	_ = srv.Store.SetSetting(store.KeyUSDAKey, "test-key")
	c.do("GET", path, nil, &a)
	if !a.KeySet || a.Estimate.PerServing.Kcal != 455 || len(a.Estimate.Lines) != 1 || a.Estimate.Lines[0].Food != "Wheat flour" ||
		len(a.Estimate.NotCounted) != 1 || asked != 2 {
		t.Fatalf("estimate: %+v (asked %d)", a, asked)
	}
	c.do("GET", path, nil, &a)
	if asked != 2 || a.Estimate.PerServing.Kcal != 455 {
		t.Fatalf("a second estimate should use what's kept (asked %d): %+v", asked, a)
	}

	var settings map[string]any
	c.do("GET", "/api/admin/settings", nil, &settings)
	if _, sent := settings[store.KeyUSDAKey]; sent || settings[store.KeyUSDAKey+"_set"] != true {
		t.Fatalf("the USDA key must stay on the server: %v", settings)
	}
}
