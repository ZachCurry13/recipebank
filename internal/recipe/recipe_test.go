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
