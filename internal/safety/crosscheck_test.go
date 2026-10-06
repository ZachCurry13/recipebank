package safety

import (
	"strings"
	"testing"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

func TestCrossCheckFindsFoodsMissingFromLines(t *testing.T) {
	r := dish("1 lb. ground beef", "1 1/2 c. diced onion", "1 c. rice", "salt and pepper")
	r.Title = "Unstuffed Peppers"
	r.Steps = []recipe.Step{
		{Text: "Brown the beef with the onion. Grease a muffin tin with butter."},
		{Text: "Add the bell peppers and rice; top with cheddar."},
		{Text: "Serve with sour cream."},
	}
	got := CrossCheck(r)
	want := "The steps use butter, bell pepper and cheddar, but no ingredient line has them."
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestCrossCheckAmounts(t *testing.T) {
	r := dish("11/2 c. flour", "shredded cheese", "40 c. sugar", "salt to taste")
	r.Steps = []recipe.Step{{Text: "Mix the flour, cheese and sugar."}}
	got := strings.Join(CrossCheck(r), "\n")
	for _, w := range []string{`"11/2" in "11/2 c. flour" may mean 1` + " " + `1/2.`, `Check the amount in "40 c. sugar".`,
		"No amount on: shredded cheese."} {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q in:\n%s", w, got)
		}
	}
	if strings.Contains(got, "steps use") || strings.Contains(got, "salt") {
		t.Errorf("flagged something that's fine:\n%s", got)
	}
}

func TestCrossCheckCleanCard(t *testing.T) {
	r := dish("2 c. flour", "1 c. buttermilk", "1 T. baking powder", "pinch of salt")
	r.Title = "Grandma's Biscuits"
	r.Steps = []recipe.Step{{Text: "Mix the flour and baking powder, stir in the buttermilk."},
		{Text: "Cut out the biscuits with a biscuit cutter and bake on a cookie sheet."}}
	if got := CrossCheck(r); len(got) != 0 {
		t.Fatalf("a clean card got %q", got)
	}
	if got := CrossCheck(&recipe.Recipe{Title: "Empty"}); len(got) != 2 {
		t.Fatalf("an empty read got %q", got)
	}
}

// A recipe nobody has checked against its card can't be OK for a real
// allergy: a missed line could be the allergen.
func TestUncheckedRecipeIsNeverOKForAllergy(t *testing.T) {
	r := dish("1 c. rice")
	r.NeedsReview, r.SourceKind = true, "photo"
	for _, sev := range []string{Allergic, Severe} {
		v := Check(r, allergic("milk", sev))
		if v.Status != Unsure || len(v.Reasons) != 1 || v.Reasons[0].Rule != "Not checked yet" ||
			!strings.Contains(v.Reasons[0].Text, "card") {
			t.Fatalf("%s: %+v", sev, v)
		}
	}
	if v := Check(r, allergic("milk", Avoid)); v.Status != OK {
		t.Fatalf("a preference: %+v", v)
	}
	if v := Check(r, Person{Name: "No rules", HeatMax: -1}); v.Status != OK {
		t.Fatalf("no allergy: %+v", v)
	}
	if v := Check(dishReviewed("1 c. milk"), allergic("milk", Allergic)); v.Status != No {
		t.Fatalf("a clear no stays no: %+v", v)
	}
	r.NeedsReview = false
	if v := Check(r, allergic("milk", Severe)); v.Status != OK {
		t.Fatalf("after checking: %+v", v)
	}
}

func dishReviewed(lines ...string) *recipe.Recipe {
	r := dish(lines...)
	r.NeedsReview = true
	return r
}

// An allergen only the steps name ("stir in the butter") means a line may be
// missing: never OK for a real allergy.
func TestStepsNamingAnAllergen(t *testing.T) {
	r := dish("1 c. rice")
	r.Steps = []recipe.Step{{Text: "Cook the rice. Stir in the butter."}}
	v := Check(r, allergic("milk", Allergic))
	if v.Status != Unsure || len(v.Reasons) != 1 || !strings.Contains(v.Reasons[0].Text, "steps mention butter") {
		t.Fatalf("allergic: %+v", v)
	}
	if v := Check(r, allergic("milk", Avoid)); v.Status != OK {
		t.Fatalf("a preference: %+v", v)
	}
	r.Steps = []recipe.Step{{Text: "Grease a muffin tin with oil; bake the muffins. Serve with sour cream."}}
	if v := Check(r, allergic("wheat", Severe)); v.Status != OK {
		t.Fatalf("tools, the dish and serving ideas: %+v", v)
	}
	if v := Check(r, allergic("milk", Severe)); v.Status != OK {
		t.Fatalf("serving idea: %+v", v)
	}
	r = dish("2 T. butter", "1 c. rice")
	r.Steps = []recipe.Step{{Text: "Stir the butter into the rice."}}
	if v := Check(r, allergic("milk", Allergic)); v.Status != No || len(v.Reasons) != 1 {
		t.Fatalf("the line already says it: %+v", v)
	}
}
