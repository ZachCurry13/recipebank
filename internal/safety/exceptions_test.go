package safety

import (
	"testing"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

func except(key, sev string, allowed ...string) Person {
	p := allergic(key, sev)
	p.Allowed = map[string][]string{key: allowed}
	return p
}

// Someone allergic to soy who may have highly refined soybean oil: only the
// oil gets through. Everything else made from soy still counts, and so does
// oil that keeps its protein.
func TestAllergyExceptions(t *testing.T) {
	cases := []struct {
		line   string
		person Person
		want   string
	}{
		{"2 tbsp soybean oil", except("soy", Allergic, "soybean oil"), OK},
		{"2 tbsp soya oil", except("soy", Allergic, "soybean oil"), OK},
		{"2 tbsp soybean oil", allergic("soy", Allergic), No}, // without the exception
		{"1 tsp soy lecithin", except("soy", Allergic, "soybean oil"), No},
		{"1 tbsp soybean oil and 1 tbsp soy sauce", except("soy", Allergic, "soybean oil"), No},
		{"2 tbsp cold-pressed soybean oil", except("soy", Allergic, "soybean oil"), No},
		{"1 block tofu", except("soy", Allergic, "soybean oil"), No},
		{"1 tsp soy lecithin", except("soy", Allergic, "soy lecithin"), OK},
		{"1 tsp lecithin", except("soy", Allergic, "soy lecithin"), Unsure}, // could be any lecithin: check the label
		{"1 tbsp soybean oil", except("soy", Allergic, "soy lecithin"), No},
		{"2 cups peanut oil, for frying", except("peanut", Allergic, "peanut oil"), OK},
		{"1 tbsp roasted peanut oil", except("peanut", Allergic, "peanut oil"), No},
		{"1/2 cup peanuts", except("peanut", Allergic, "peanut oil"), No},
		{"1 cup sliced almonds", except("treenut", Allergic, "almond"), OK}, // a parent typed "almond"
		{"1/2 cup cashews", except("treenut", Allergic, "almond"), No},
		{"1 cup mixed nuts", except("treenut", Allergic, "almond"), No},
		{"1 can tuna", except("fish", Allergic, "tuna"), OK},
		{"1 lb salmon", except("fish", Allergic, "tuna"), No},
		// An exception for one allergen doesn't touch another.
		{"2 tbsp soybean oil", Person{ID: 1, HeatMax: -1, Allergies: map[string]string{"soy": Allergic, "peanut": Allergic},
			Allowed: map[string][]string{"peanut": {"peanut oil"}}}, No},
	}
	for _, c := range cases {
		if v := Check(dish(c.line), c.person); v.Status != c.want {
			t.Errorf("%q, allowed %v: got %s, want %s (%+v)", c.line, c.person.Allowed, v.Status, c.want, v.Reasons)
		}
	}
}

// The steps naming an allowed food ("drizzle with soybean oil") don't make a
// recipe "not sure" for that person.
func TestExceptionsInSteps(t *testing.T) {
	r := dish("2 cups rice", "1 cup peas")
	r.Steps = []recipe.Step{{Text: "Fry the rice in soybean oil, then add the peas."}}
	if v := Check(r, allergic("soy", Allergic)); v.Status != Unsure {
		t.Fatalf("without the exception the steps make it unsure: %+v", v)
	}
	if v := Check(r, except("soy", Allergic, "soybean oil")); v.Status != OK {
		t.Fatalf("with it: %+v", v)
	}
	r.Steps = []recipe.Step{{Text: "Fry the rice, then stir in the soy sauce."}}
	if v := Check(r, except("soy", Allergic, "soybean oil")); v.Status != Unsure {
		t.Fatalf("soy sauce in the steps still counts: %+v", v)
	}
}

// Strict mode still doesn't let unknown foods through; a line that's only
// the allowed food is known.
func TestExceptionsInStrictMode(t *testing.T) {
	if v := Check(dish("2 tbsp soybean oil"), except("soy", Severe, "soybean oil")); v.Status != OK {
		t.Fatalf("the allowed oil: %+v", v)
	}
	if v := Check(dish("1 jar mystery sauce"), except("soy", Severe, "soybean oil")); v.Status != Unsure {
		t.Fatalf("unknown is never safe: %+v", v)
	}
}

func TestSameAsAllergen(t *testing.T) {
	for _, c := range []struct {
		key, name string
		same      bool
	}{
		{"soy", "soy", true}, {"soy", "Soya", true}, {"soy", "soybeans", true}, {"treenut", "tree nuts", true},
		{"treenut", "nuts", true}, {"milk", "dairy", true}, {"shellfish", "Shellfish", true},
		{"soy", "soybean oil", false}, {"treenut", "almond", false}, {"fish", "tuna", false}, {"milk", "ghee", false},
	} {
		if got := SameAsAllergen(c.key, c.name); got != c.same {
			t.Errorf("%s / %q: got %v", c.key, c.name, got)
		}
	}
}

func TestOfferedExceptions(t *testing.T) {
	soy, _ := AllergenByKey("soy")
	if len(soy.Exceptions) != 2 || soy.Exceptions[0].Key != "soybean oil" || soy.Exceptions[0].Note == "" {
		t.Fatalf("soy: %+v", soy.Exceptions)
	}
	if milk, _ := AllergenByKey("milk"); len(milk.Exceptions) != 0 {
		t.Fatalf("milk has none offered: %+v", milk.Exceptions)
	}
}
