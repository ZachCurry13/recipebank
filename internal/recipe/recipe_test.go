package recipe

import "testing"

func TestParseLine(t *testing.T) {
	cases := []struct {
		line       string
		qty        float64
		unit, food string
		note       string
	}{
		{"1 ½ cups flour, sifted", 1.5, "cup", "flour", "sifted"},
		{"1½ cups sugar", 1.5, "cup", "sugar", ""},
		{"2 1/2 tbsp olive oil", 2.5, "tbsp", "olive oil", ""},
		{"3 large eggs", 3, "", "large eggs", ""},
		{"1 (14 oz) can diced tomatoes", 1, "can", "diced tomatoes", "14 oz"},
		{"1 T butter", 1, "tbsp", "butter", ""},
		{"1 t salt", 1, "tsp", "salt", ""},
		{"1 T. butter", 1, "tbsp", "butter", ""},
		{"1 t. cumin", 1, "tsp", "cumin", ""},
		{"2 pt. milk", 2, "pint", "milk", ""},
		{"½ tsp vanilla extract", 0.5, "tsp", "vanilla extract", ""},
		{"10 drops tea tree oil", 10, "drop", "tea tree oil", ""},
		{"Salt and pepper to taste", 0, "", "Salt and pepper to taste", ""},
	}
	for _, c := range cases {
		in := ParseLine(c.line)
		q := 0.0
		if in.Qty != nil {
			q = *in.Qty
		}
		if q != c.qty || in.Unit != c.unit || in.Food != c.food || in.Note != c.note {
			t.Errorf("%q → qty %v unit %q food %q note %q", c.line, q, in.Unit, in.Food, in.Note)
		}
	}
	if in := ParseLine("2-3 cloves garlic"); in.QtyMax == nil || *in.QtyMax != 3 || in.Unit != "clove" {
		t.Errorf("range: %+v", in)
	}
}

const page = `<html><head><title>Best Cookies</title>
<script type="application/ld+json">{"@context":"https://schema.org","@graph":[{"@type":"WebPage"},
{"@type":["Recipe"],"name":"Chewy Cookies &amp; Cream","description":"<p>So good</p>",
"recipeYield":["24","24 cookies"],"prepTime":"PT15M","cookTime":"PT1H10M",
"image":{"@type":"ImageObject","url":"https://example.com/c.jpg"},
"recipeIngredient":["2 cups flour","1 cup butter, softened"],
"recipeInstructions":[{"@type":"HowToSection","name":"Dough","itemListElement":[{"@type":"HowToStep","text":"Mix it."}]},
{"@type":"HowToStep","text":"Bake at 350°F for 10 minutes."}]}]}</script></head><body>hi</body></html>`

func TestFromHTML(t *testing.T) {
	r, ok := FromHTML([]byte(page))
	if !ok {
		t.Fatal("no recipe found")
	}
	if r.Title != "Chewy Cookies & Cream" || r.Summary != "So good" || r.Servings != 24 {
		t.Errorf("title/summary/servings: %q %q %v", r.Title, r.Summary, r.Servings)
	}
	if r.PrepMin != 15 || r.CookMin != 70 || r.ImageURL != "https://example.com/c.jpg" {
		t.Errorf("times/image: %d %d %q", r.PrepMin, r.CookMin, r.ImageURL)
	}
	if len(r.Ingredients) != 2 || r.Ingredients[1].Food != "butter" {
		t.Errorf("ingredients: %+v", r.Ingredients)
	}
	if len(r.Steps) != 2 || r.Steps[0].Section != "Dough" {
		t.Errorf("steps: %+v", r.Steps)
	}
	if _, ok := FromHTML([]byte("<html><body>No recipe here</body></html>")); ok {
		t.Error("found a recipe on a page without one")
	}
}

func TestFromText(t *testing.T) {
	r, ok := FromText("Grandma's Pancakes\nFluffy and quick.\n\nIngredients\n- 1 cup flour\n- 1 egg\n\nDirections\n1. Mix.\n2. Fry 2 minutes a side.\n")
	if !ok || r.Title != "Grandma's Pancakes" || len(r.Ingredients) != 2 || len(r.Steps) != 2 || r.Steps[0].Text != "Mix." {
		t.Fatalf("%v %+v", ok, r)
	}
	if _, ok := FromText("just some words"); ok {
		t.Error("read a recipe from plain words")
	}
}

// A card copied by the AI as plain text: two columns already flattened,
// amounts with abbreviations, a [?] where the writing was unclear.
func TestFromTextCard(t *testing.T) {
	card := "Unstuffed Peppers (4 serv.)\nIngredients:\n- 3 tbsp. olive oil\n- 1 1/2 c. diced onion\n- 1/8 tsp. red pepper flakes [?]\n- 1 lb. ground beef\n- salt & pepper\nInstructions:\nHeat 2 tbsp. olive oil in pan over medium-high heat.\n"
	r, ok := FromText(card)
	if !ok || r.Title != "Unstuffed Peppers" || r.Servings != 4 || len(r.Ingredients) != 5 || !r.NeedsReview {
		t.Fatalf("%v %+v", ok, r)
	}
	oil, onion, flakes := r.Ingredients[0], r.Ingredients[1], r.Ingredients[2]
	if oil.Qty == nil || *oil.Qty != 3 || oil.Unit != "tbsp" || oil.Food != "olive oil" {
		t.Errorf("oil: %+v", oil)
	}
	if onion.Qty == nil || *onion.Qty != 1.5 || onion.Unit != "cup" {
		t.Errorf("onion: %+v", onion)
	}
	if !flakes.Unsure || flakes.Food != "red pepper flakes" {
		t.Errorf("flakes: %+v", flakes)
	}
}
