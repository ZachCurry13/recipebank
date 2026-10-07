package recipe

import (
	"strings"
	"testing"
)

// A made-up paste like the ones people send, and what a small AI model
// tends to make of it: lines shortened, a weight split off as its own line.
const pasted = `Oatmeal Raisin Cookies (about 24):
8 T butter, softened
1 Cup brown sugar (dark is best)
2 eggs
1 1/2 Cups flour (more if sticky)
1 Cup raisins (golden or dark, or half milk chocolate chips) 150g
1/2 t salt

Preheat the oven to 350. Cream the butter and sugar, then beat in the eggs.
Stir in the flour, salt and raisins and bake 10 minutes.`

func TestRestoreLines(t *testing.T) {
	ai := []Ingredient{{Line: "8 T butter, softened"}, {Line: "1 cup brown sugar"}, {Line: "2 eggs"},
		{Line: "1 1/2 Cups flour"}, {Line: "1 Cup raisins"}, {Line: "150g"}, {Line: "1/2 t salt"}, {Line: "1 tsp vanilla"}}
	steps := []Step{{Text: "Preheat the oven to 350. Cream the butter and sugar, then beat in the eggs."},
		{Text: "Stir in the flour, salt and raisins and bake 10 minutes."}}
	got := RestoreLines(pasted, ai, steps)
	var lines []string
	for _, in := range got {
		lines = append(lines, in.Line)
	}
	want := []string{"8 T butter, softened", "1 Cup brown sugar (dark is best)", "2 eggs", "1 1/2 Cups flour (more if sticky)",
		"1 Cup raisins (golden or dark, or half milk chocolate chips) 150g", "1/2 t salt", "1 tsp vanilla"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("got\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
	// "1 tsp vanilla" isn't in the paste: kept as the AI gave it (the person checks the draft).
	// A line restored in full is parsed again, so the note that says milk is back.
	r := &Recipe{Ingredients: got}
	r.Clean()
	if !strings.Contains(r.Ingredients[4].Note, "milk chocolate") {
		t.Fatalf("note: %+v", r.Ingredients[4])
	}
}

// Words in the steps never become ingredient lines, and a line the AI kept
// whole stays as it is.
func TestRestoreLinesLeavesStepsAlone(t *testing.T) {
	ai := []Ingredient{{Line: "salt"}, {Line: "2 eggs"}}
	got := RestoreLines("2 eggs\nAdd the salt and stir until smooth and glossy.", ai, []Step{{Text: "Add the salt and stir until smooth and glossy."}})
	if got[0].Line != "salt" || got[1].Line != "2 eggs" {
		t.Fatalf("%+v", got)
	}
}
