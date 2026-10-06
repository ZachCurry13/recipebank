package recipe

import "testing"

func TestLevel(t *testing.T) {
	steps := func(texts ...string) []Step {
		var out []Step
		for _, s := range texts {
			out = append(out, Step{Text: s})
		}
		return out
	}
	toast := Recipe{PrepMin: 2, CookMin: 3, Steps: steps("Toast the bread.", "Butter it.")}
	bread := Recipe{PrepMin: 20, CookMin: 35, TotalMin: 180, Steps: steps("Mix.", "Knead 10 minutes.", "Let rise until doubled.", "Shape.", "Bake.")}
	croissants := Recipe{PrepMin: 90, CookMin: 40, Steps: steps("Make the dough and knead.", "Laminate with butter.", "Fold in thirds and chill.",
		"Repeat.", "Repeat.", "Shape.", "Proof 2 hours.", "Bake.")}
	chosen := Recipe{Difficulty: Hard, Steps: steps("Toast the bread.")}
	for _, c := range []struct {
		r    Recipe
		want string
	}{{toast, Easy}, {bread, Medium}, {croissants, Hard}, {chosen, Hard}, {Recipe{}, Easy}} {
		if got := c.r.Level(); got != c.want {
			t.Errorf("%+v: got %s, want %s", c.r.Steps, got, c.want)
		}
	}
	r := Recipe{Difficulty: "impossible"}
	r.Clean()
	if r.Difficulty != "" {
		t.Errorf("an unknown difficulty is kept: %q", r.Difficulty)
	}
}
