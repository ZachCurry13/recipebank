package nutrition

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

var foods = map[string]Food{
	"flour": {Name: "Wheat flour, white, all-purpose", Per100g: Nutrients{Kcal: 364, Protein: 10, Carbs: 76},
		Portions: []Portion{{"1 cup", 125}}},
	"eggs":           {Name: "Egg, whole, raw", Per100g: Nutrients{Kcal: 143, Protein: 12.6, Fat: 9.5}, Portions: []Portion{{"1 large", 50}, {"1 cup", 243}}},
	"milk":           {Name: "Milk, whole", Per100g: Nutrients{Kcal: 61, Protein: 3.2}},
	"butter":         {Name: "Butter, salted", Per100g: Nutrients{Kcal: 717, Fat: 81}, Portions: []Portion{{"1 tbsp", 14.2}}},
	"diced tomatoes": {Name: "Tomatoes, canned", Per100g: Nutrients{Kcal: 18}},
}

func lookup(f string) (Food, bool) { food, ok := foods[f]; return food, ok }

func near(a, b float64) bool { return math.Abs(a-b) < 0.5 }

func TestCompute(t *testing.T) {
	var ings []recipe.Ingredient
	for _, l := range []string{"2 cups flour", "2 eggs", "1 cup milk", "2 T. butter", "1 (14 oz) can diced tomatoes",
		"salt to taste", "1 cup mystery spice"} {
		ings = append(ings, recipe.ParseLine(l))
	}
	e := Compute(ings, 4, lookup)
	// flour 250 g = 910 kcal; eggs 100 g = 143; milk ~236 ml = 144; butter 28.4 g = 204; tomatoes 397 g = 71.
	want := (910 + 143 + 144.3 + 203.6 + 71.4) / 4
	if !near(e.PerServing.Kcal, want) || e.Counted != 5 || e.Servings != 4 {
		t.Fatalf("kcal %.1f (want %.1f), counted %d", e.PerServing.Kcal, want, e.Counted)
	}
	if strings.Join(e.NotCounted, "|") != "salt to taste|1 cup mystery spice" {
		t.Fatalf("not counted: %q", e.NotCounted)
	}
	if whole := Compute(ings[:1], 0, lookup); !near(whole.PerServing.Kcal, 910) || whole.Servings != 0 {
		t.Fatalf("no servings means the whole recipe: %+v", whole)
	}
}

// A fake FoodData Central: the key goes in a header, nutrients and portions
// come from the search, or from the food's own page when the search has none.
func TestLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "k" || r.URL.Query().Get("api_key") != "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// The real service refuses food types joined with commas.
		if dt := r.URL.Query()["dataType"]; r.URL.Path == "/foods/search" && (len(dt) < 2 || strings.Contains(strings.Join(dt, "|"), ",")) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch {
		case r.URL.Path == "/foods/search" && strings.Contains(r.URL.Query().Get("query"), "flour"):
			w.Write([]byte(`{"foods":[{"fdcId":1,"description":"Wheat flour","foodNutrients":[{"nutrientId":1008,"value":364},{"nutrientId":1003,"value":10}],
				"foodMeasures":[{"disseminationText":"1 cup","gramWeight":125},{"disseminationText":"Quantity not specified","gramWeight":1}]}]}`))
		case r.URL.Path == "/foods/search" && strings.Contains(r.URL.Query().Get("query"), "egg"):
			w.Write([]byte(`{"foods":[{"fdcId":2,"description":"Egg","foodNutrients":[{"nutrientId":2047,"value":143}]}]}`))
		case r.URL.Path == "/food/2":
			w.Write([]byte(`{"foodPortions":[{"amount":1,"modifier":"large","gramWeight":50,"measureUnit":{"name":"undetermined"}}]}`))
		default:
			w.Write([]byte(`{"foods":[]}`))
		}
	}))
	defer srv.Close()
	c := Client{HTTP: srv.Client(), Base: srv.URL, Key: "k", Agent: "RecipeBank-test"}
	flour, err := c.Lookup(context.Background(), "flour")
	if err != nil || flour.Per100g.Kcal != 364 || len(flour.Portions) != 1 || flour.Portions[0].Grams != 125 {
		t.Fatalf("flour: %+v %v", flour, err)
	}
	egg, err := c.Lookup(context.Background(), "egg")
	if err != nil || egg.Per100g.Kcal != 143 || len(egg.Portions) != 1 || egg.Portions[0].Text != "1 large" {
		t.Fatalf("egg: %+v %v", egg, err)
	}
	if _, err := c.Lookup(context.Background(), "nothing"); err != ErrNotFound {
		t.Fatalf("unknown: %v", err)
	}
	if _, err := (Client{HTTP: srv.Client(), Base: srv.URL, Key: "wrong"}).Lookup(context.Background(), "flour"); err == nil || !strings.Contains(err.Error(), "key") {
		t.Fatalf("a wrong key: %v", err)
	}
}

func TestSearchTermsAndScore(t *testing.T) {
	if searchTerms("Eggs") != "egg whole raw fresh" || searchTerms("flour") == "flour" || searchTerms("kale") != "kale" {
		t.Fatal("search terms")
	}
	q := searchTerms("flour")
	if score(q, "Arrowroot flour") >= score(q, "Wheat flour, white, all-purpose, enriched, bleached") {
		t.Fatal("arrowroot flour shouldn't win for flour")
	}
	if score("onions raw", "Onions, frozen, chopped, cooked, boiled, drained, without salt") >= score("onions raw", "Onions, raw") {
		t.Fatal("the plain food should win")
	}
}
