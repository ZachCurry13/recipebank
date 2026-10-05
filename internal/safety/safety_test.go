package safety

import (
	"strings"
	"testing"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

func dish(lines ...string) *recipe.Recipe {
	r := &recipe.Recipe{Title: "Test", Heat: -1}
	for _, l := range lines {
		r.Ingredients = append(r.Ingredients, recipe.ParseLine(l))
	}
	return r
}

func allergic(key, sev string) Person {
	return Person{ID: 1, Name: "Kid", HeatMax: -1, Allergies: map[string]string{key: sev}}
}

func TestHiddenNames(t *testing.T) {
	cases := []struct{ line, allergen string }{
		{"2 tbsp whey protein", "milk"},
		{"1 tsp ghee", "milk"},
		{"1/4 cup tahini", "sesame"},
		{"1 tbsp Worcestershire sauce", "fish"},
		{"2 tbsp soy sauce", "wheat"},
		{"2 tbsp soy sauce", "soy"},
		{"1 cup semolina", "wheat"},
		{"1/2 cup pesto", "treenut"},
		{"3 large eggs", "egg"},
		{"1 cup mayonnaise", "egg"},
		{"1 cup crème fraîche", "milk"},
		{"1/2 cup rolled oats", "gluten"},
	}
	for _, c := range cases {
		v := Check(dish(c.line), allergic(c.allergen, Allergic))
		if v.Status != No {
			t.Errorf("%q with %s allergy: got %s, want no", c.line, c.allergen, v.Status)
		}
	}
}

func TestLookAlikes(t *testing.T) {
	cases := []struct{ line, allergen string }{
		{"1 can coconut milk", "milk"},
		{"1/4 tsp cream of tartar", "milk"},
		{"1 cup almond flour", "wheat"},
		{"1 cup buckwheat groats", "wheat"},
		{"1 tsp nutmeg", "treenut"},
		{"1 eggplant, cubed", "egg"},
		{"1 cup water chestnuts", "treenut"},
		{"2 cups butternut squash", "milk"},
	}
	for _, c := range cases {
		v := Check(dish(c.line), allergic(c.allergen, Avoid))
		if v.Status == No {
			t.Errorf("%q with %s: got no (%v)", c.line, c.allergen, v.Reasons)
		}
	}
	// Almond flour is still a tree nut.
	if v := Check(dish("1 cup almond flour"), allergic("treenut", Allergic)); v.Status != No {
		t.Errorf("almond flour for a tree-nut allergy: got %s", v.Status)
	}
}

func TestUnknownIsNeverSafeForSevere(t *testing.T) {
	r := dish("2 cups chicken stock", "1 lb chicken breast", "1 tsp salt")
	v := Check(r, allergic("milk", Severe))
	if v.Status != Unsure {
		t.Fatalf("chicken stock, severe milk allergy: got %s, want unsure", v.Status)
	}
	// A mystery ingredient is unsure in strict mode...
	if v := Check(dish("1 cup Grandma's topping"), allergic("milk", Severe)); v.Status != Unsure {
		t.Errorf("unknown ingredient, severe: got %s, want unsure", v.Status)
	}
	// ...but plain foods pass.
	plain := dish("1 lb boneless skinless chicken breasts", "2 cloves garlic, minced", "Salt and pepper to taste", "1 tbsp olive oil")
	if v := Check(plain, allergic("milk", Severe)); v.Status != OK {
		t.Errorf("plain chicken, severe milk: got %s %+v", v.Status, v.Reasons)
	}
	// A parent checked the label: the stock no longer blocks.
	r.Ingredients[0].Checked = []string{"milk"}
	if v := Check(r, allergic("milk", Severe)); v.Status != OK {
		t.Errorf("label checked: got %s %+v", v.Status, v.Reasons)
	}
}

func TestAllergicPackagedNeedsLabel(t *testing.T) {
	if v := Check(dish("1 cup chocolate chips"), allergic("milk", Allergic)); v.Status != Unsure {
		t.Errorf("chocolate chips, milk allergy: got %s", v.Status)
	}
	if v := Check(dish("1 cup chocolate chips"), allergic("milk", Avoid)); v.Status != OK {
		t.Errorf("chocolate chips, milk avoid: got %s", v.Status)
	}
}

func TestDiets(t *testing.T) {
	veg := Person{Name: "V", HeatMax: -1, Diets: []string{"vegetarian"}}
	if v := Check(dish("1 lb ground beef"), veg); v.Status != No {
		t.Errorf("beef, vegetarian: %s", v.Status)
	}
	if v := Check(dish("4 cups vegetable broth", "1 onion"), veg); v.Status != OK {
		t.Errorf("vegetable broth, vegetarian: %s %+v", v.Status, v.Reasons)
	}
	if v := Check(dish("4 cups broth"), veg); v.Status != Unsure {
		t.Errorf("plain broth, vegetarian: %s", v.Status)
	}
	vegan := Person{Name: "V", HeatMax: -1, Diets: []string{"vegan"}}
	if v := Check(dish("2 tbsp honey"), vegan); v.Status != No {
		t.Errorf("honey, vegan: %s", v.Status)
	}
	kosher := Person{Name: "K", HeatMax: -1, Diets: []string{"kosher"}}
	if v := Check(dish("1 lb ground beef", "1 cup shredded cheddar cheese"), kosher); v.Status != No {
		t.Errorf("cheeseburger, kosher-style: %s", v.Status)
	}
}

func TestHeatAndDislikes(t *testing.T) {
	mild := Person{Name: "M", HeatMax: 1}
	if v := Check(dish("2 jalapeños, minced", "1 red bell pepper"), mild); v.Status != No {
		t.Errorf("jalapeños, heat max 1: %s", v.Status)
	}
	if v := Check(dish("1 red bell pepper", "1 tsp black pepper"), mild); v.Status != OK {
		t.Errorf("bell pepper, heat max 1: %s %+v", v.Status, v.Reasons)
	}
	picky := Person{Name: "P", HeatMax: -1, Dislikes: []string{"mushroom"}}
	if v := Check(dish("8 oz cremini mushrooms"), picky); v.Status != No {
		t.Errorf("mushrooms, dislike: %s", v.Status)
	}
}

func TestSwapsRespectEveryone(t *testing.T) {
	r := dish("1 cup milk", "2 cups flour")
	milkKid := allergic("milk", Allergic)
	oatKid := Person{ID: 2, Name: "Oat", HeatMax: -1, Allergies: map[string]string{"gluten": Allergic}}
	diners := []Person{milkKid, oatKid}
	vs := []Verdict{Check(r, milkKid), Check(r, oatKid)}
	for _, s := range Swaps(r, diners, vs) {
		if s.To == "oat milk" {
			t.Errorf("oat milk offered to someone with a gluten (oats) allergy")
		}
	}
	found := false
	for _, s := range Swaps(r, []Person{milkKid}, vs[:1]) {
		if s.To == "oat milk" || s.To == "rice milk" {
			found = true
		}
	}
	if !found {
		t.Errorf("no milk swap offered for a milk allergy alone")
	}
}

func TestHomeHazards(t *testing.T) {
	r := &recipe.Recipe{Area: recipe.AreaHome, Title: "Bathroom cleaner", Ingredients: []recipe.Ingredient{
		recipe.ParseLine("1 cup bleach"), recipe.ParseLine("1 cup white vinegar")}}
	hz := Hazards(r, nil)
	if len(hz) == 0 || hz[0].Level != "danger" || !strings.Contains(hz[0].Text, "chlorine gas") {
		t.Fatalf("bleach + vinegar: %+v", hz)
	}
	paste := &recipe.Recipe{Area: recipe.AreaHome, Title: "Mint toothpaste", Ingredients: []recipe.Ingredient{
		recipe.ParseLine("2 tbsp baking soda"), recipe.ParseLine("1 tsp xylitol"), recipe.ParseLine("5 drops peppermint oil")}}
	text := ""
	for _, h := range Hazards(paste, []string{"dog", "cat"}) {
		text += h.Level + ":" + h.Text + "\n"
	}
	for _, want := range []string{"poisonous to dogs", "toxic to cats", "no fluoride", "pea-sized"} {
		if !strings.Contains(text, want) {
			t.Errorf("toothpaste hazards missing %q:\n%s", want, text)
		}
	}
}

func TestStrictKnowsPlainFoods(t *testing.T) {
	plain := dish("2 cups plain flour", "2 large eggs", "1 red bell pepper, seeded and diced", "caster sugar to serve",
		"1 tbsp sunflower or vegetable oil plus a little extra for frying", "10 drops peppermint essential oil", "1 tsp xylitol")
	if v := Check(plain, allergic("milk", Severe)); v.Status != OK {
		t.Errorf("plain foods, severe milk: %s %+v", v.Status, v.Reasons)
	}
	for _, line := range []string{"1 packet ranch seasoning mix", "2 cups white bread cubes", "1 cup mystery topping blend"} {
		if v := Check(dish(line), allergic("egg", Severe)); v.Status == OK {
			t.Errorf("%q, severe egg: got OK", line)
		}
	}
}

func TestSameFood(t *testing.T) {
	yes := [][2]string{{"chicken broth", "Brand Chicken Broth 32 oz"}, {"flour", "King Arthur All-Purpose Flour"},
		{"large eggs", "Eggs, large, dozen"}, {"olive oil", "Kirkland Extra Virgin Olive Oil"},
		{"ground beef", "Ground Beef 80/20 1 lb"}, {"salt", "Morton Kosher Salt"}, {"butter", "Unsalted Butter"},
		{"chicken broth", "Chicken Broth, Low Sodium"}, {"chicken broth", "Organic Chicken Broth With Herbs"},
		{"tuna", "Tuna in Water"}}
	no := [][2]string{{"butter", "Peanut Butter"}, {"chicken", "Chicken Broth"}, {"vinegar", "Rice Vinegar"},
		{"syrup", "Corn Syrup"}, {"milk", "Oat Milk"}, {"flour", "Almond Flour"}, {"oil", "Olive Oil"}}
	for _, c := range yes {
		if !SameFood(c[0], c[1]) {
			t.Errorf("%q should match %q", c[0], c[1])
		}
	}
	for _, c := range no {
		if SameFood(c[0], c[1]) {
			t.Errorf("%q should not match %q", c[0], c[1])
		}
	}
}

func TestBrothSwapIsABroth(t *testing.T) {
	r := dish("4 cups chicken broth")
	veg := Person{ID: 1, Name: "V", HeatMax: -1, Diets: []string{"vegetarian"}}
	swaps := Swaps(r, []Person{veg}, []Verdict{Check(r, veg)})
	if len(swaps) == 0 || swaps[0].To != "vegetable broth" {
		t.Fatalf("swaps for chicken broth: %+v", swaps)
	}
	for _, s := range swaps {
		if s.To == "chickpeas" || s.To == "lentils" {
			t.Errorf("a meat swap was offered for broth: %+v", s)
		}
	}
}
