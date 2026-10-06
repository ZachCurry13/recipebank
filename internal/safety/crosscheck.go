package safety

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

// Reading a card can drop a line or misread an amount. CrossCheck compares
// the recipe with itself (no AI) and says what to look at on the card.

var (
	// Pans and tools named after food: "grease the muffin tin" isn't muffins.
	toolWords = compile("muffin pan", "muffin tin", "muffin cup", "cake pan", "cookie sheet", "cookie cutter",
		"bread pan", "loaf pan", "pie plate", "pie dish", "pie pan", "biscuit cutter", "pastry brush", "pastry bag",
		"pastry blender", "cake tester", "butter knife", "egg timer")
	// Words for the finished dish rather than an ingredient.
	dishWords = compile("cake", "cookie", "muffin", "cupcake", "pancake", "waffle", "biscuit", "pastry", "dumpling",
		"date", "orange", "herb")
	// "11/2" on a card is almost always "1 1/2".
	runTogether = regexp.MustCompile(`(^|\s)([1-9])([1-9]/[2348])(\s|$)`)
)

// foodNames is every known food and allergen word, built on first use
// (after the lists' own setup has run).
var foodNames = sync.OnceValue(func() phrases {
	ps := phrases{byFirst: map[string][][]string{}}
	for f := range knownFoods {
		w := strings.Fields(f)
		ps.byFirst[w[0]] = append(ps.byFirst[w[0]], w)
	}
	for _, a := range Allergens {
		for first, list := range a.terms.byFirst {
			ps.byFirst[first] = append(ps.byFirst[first], list...)
		}
	}
	return ps
})

// CrossCheck lists what looks missed or misread in a recipe read from a
// photo, as plain sentences; none means nothing stood out.
func CrossCheck(r *recipe.Recipe) []string {
	var out []string
	if len(r.Ingredients) == 0 {
		out = append(out, "No ingredients were found.")
	}
	if len(r.Steps) == 0 {
		out = append(out, "No steps were found.")
	}
	if missing := stepFoodsMissing(r); len(missing) > 0 {
		it := "it"
		if len(missing) > 1 {
			it = "them"
		}
		out = append(out, fmt.Sprintf("The steps use %s, but no ingredient line has %s.", joinAnd(missing), it))
	}
	var noAmount []string
	for _, in := range r.Ingredients {
		if m := runTogether.FindStringSubmatch(in.Line); m != nil {
			out = append(out, fmt.Sprintf("\"%s%s\" in \"%s\" may mean %s %s.", m[2], m[3], in.Line, m[2], m[3]))
		}
		if amountMissing(in) {
			noAmount = append(noAmount, in.Line)
		}
		if odd := oddAmount(in); odd != "" {
			out = append(out, odd)
		}
	}
	if len(noAmount) > 0 {
		if len(noAmount) > 3 {
			noAmount = append(noAmount[:3], "…")
		}
		out = append(out, "No amount on: "+strings.Join(noAmount, "; ")+".")
	}
	return out
}

// stepFoodsMissing finds foods the steps name that no ingredient line has
// (staples, tools and the dish itself aside; serving ideas don't count).
func stepFoodsMissing(r *recipe.Recipe) []string {
	ings := ingredientText(r)
	title := newText(r.Title)
	var missing []string
	seen := map[string]bool{}
	for _, t := range stepSentences(r) {
		for _, food := range foodsIn(t) {
			ps := compile(food)
			if seen[food] || Staple(food) || ings.has(ps) || title.has(ps) || sameAllergenIn(food, ings) {
				continue
			}
			seen[food] = true
			missing = append(missing, food)
		}
	}
	if len(missing) > 4 {
		missing = missing[:4]
	}
	return missing
}

// StepsMention returns a word for allergen key that the steps use ("stir in
// the butter") when no ingredient line has that allergen, or "".
func StepsMention(r *recipe.Recipe, key string) string {
	a, ok := AllergenByKey(key)
	if !ok || len(r.Steps) == 0 || isAllergen(ingredientText(r), key) {
		return ""
	}
	for _, t := range stepSentences(r) {
		t.mask(a.not)
		if hit := t.first(a.terms); hit != "" {
			return hit
		}
	}
	return ""
}

func ingredientText(r *recipe.Recipe) text {
	var lines []string
	for _, in := range r.Ingredients {
		lines = append(lines, in.Line, in.Food)
	}
	return newText(strings.Join(lines, " \n "))
}

// stepSentences splits the steps into sentences, leaving out serving ideas
// ("Serve with sour cream") and masking tools and the dish itself.
func stepSentences(r *recipe.Recipe) []text {
	var out []text
	for _, st := range r.Steps {
		for _, s := range strings.FieldsFunc(st.Text, func(c rune) bool { return c == '.' || c == ';' || c == '!' }) {
			t := newText(s)
			if len(t) == 0 || t[0] == "serve" {
				continue
			}
			t.mask(toolWords)
			t.mask(dishWords)
			out = append(out, t)
		}
	}
	return out
}

// foodsIn lists the foods named in t, longest names first ("bell pepper",
// not "pepper").
func foodsIn(t text) []string {
	var out []string
	for i := 0; i < len(t); {
		best := []string(nil)
		for _, p := range foodNames().byFirst[t[i]] {
			if len(p) > len(best) && t.at(i, p) {
				best = p
			}
		}
		if best == nil {
			i++
			continue
		}
		out = append(out, strings.Join(best, " "))
		i += len(best)
	}
	return out
}

// sameAllergenIn: the steps say "cheese" and the lines have "cheddar".
func sameAllergenIn(food string, ings text) bool {
	t := newText(food)
	for _, a := range Allergens {
		if isAllergen(t, a.Key) && isAllergen(ings, a.Key) {
			return true
		}
	}
	return false
}

func amountMissing(in recipe.Ingredient) bool {
	if in.Qty != nil || in.Unit != "" || in.Food == "" || Staple(in.Food) {
		return false
	}
	l := strings.ToLower(in.Line + " " + in.Note)
	for _, w := range []string{"to taste", "optional", "for serving", "garnish", "as needed", "pinch", "dash", "for greasing"} {
		if strings.Contains(l, w) {
			return false
		}
	}
	return true
}

// oddAmount flags amounts far bigger than a home recipe uses.
func oddAmount(in recipe.Ingredient) string {
	if in.Qty == nil {
		return ""
	}
	limit := map[string]float64{"cup": 16, "tbsp": 16, "tsp": 12, "lb": 20, "oz": 64, "g": 5000, "kg": 10, "ml": 5000, "l": 10}
	q := *in.Qty
	if l, ok := limit[in.Unit]; (ok && q > l) || (in.Unit == "" && q > 60) || q <= 0 {
		return fmt.Sprintf("Check the amount in \"%s\".", in.Line)
	}
	return ""
}

func joinAnd(list []string) string {
	if len(list) < 2 {
		return strings.Join(list, "")
	}
	return strings.Join(list[:len(list)-1], ", ") + " and " + list[len(list)-1]
}
