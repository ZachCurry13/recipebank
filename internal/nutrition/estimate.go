package nutrition

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

// Estimate is a recipe's nutrition per serving. Lines whose food or amount
// it couldn't work out are listed, so it says what it leaves out.
type Estimate struct {
	PerServing Nutrients `json:"per_serving"`
	Servings   float64   `json:"servings"` // 0: unknown, the numbers are for the whole recipe
	Counted    int       `json:"counted"`
	NotCounted []string  `json:"not_counted"`
	Lines      []Line    `json:"lines"` // how each counted line was worked out
}

// Line is a counted ingredient line: the food it matched and its weight.
type Line struct {
	Text  string  `json:"text"`
	Food  string  `json:"food"`
	Grams float64 `json:"grams"`
}

var (
	weights = map[string]float64{"g": 1, "kg": 1000, "oz": 28.3495, "lb": 453.592}
	volumes = map[string]float64{"ml": 1, "l": 1000, "tsp": 4.929, "tbsp": 14.787, "cup": 236.588, "fl oz": 29.574,
		"pint": 473.176, "quart": 946.353, "gallon": 3785.41}
	tiny = map[string]float64{"pinch": 0.36, "dash": 0.6, "drop": 0.05} // grams, roughly
	// A household measure's own amount and unit: "1 cup, chopped", "2 tbsp".
	measureRE = regexp.MustCompile(`^(\d+(?:\.\d+)?|\d+/\d+)?\s*(cup|tbsp|tablespoon|tsp|teaspoon|fl oz)\b`)
	// A package's size in its note: "1 (14 oz) can".
	sizeRE    = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*-?\s*(oz|ounce|ounces|g|grams|lb|pound|pounds)\b`)
	measureML = map[string]float64{"cup": 236.588, "tbsp": 14.787, "tablespoon": 14.787, "tsp": 4.929, "teaspoon": 4.929, "fl oz": 29.574}
	liquids   = []string{"water", "milk", "broth", "stock", "juice", "wine", "vinegar", "cream", "sauce", "beer", "coffee", "tea"}
)

// Compute adds up the ingredients. lookup finds a food (false when unknown).
func Compute(ings []recipe.Ingredient, servings float64, lookup func(food string) (Food, bool)) Estimate {
	e := Estimate{Servings: servings, NotCounted: []string{}, Lines: []Line{}}
	var total Nutrients
	for _, in := range ings {
		if strings.TrimSpace(in.Food) == "" {
			continue
		}
		f, ok := lookup(in.Food)
		if !ok {
			e.NotCounted = append(e.NotCounted, in.Line)
			continue
		}
		g, ok := grams(in, f)
		if !ok {
			e.NotCounted = append(e.NotCounted, in.Line)
			continue
		}
		total = total.add(f.Per100g, g/100)
		e.Counted++
		e.Lines = append(e.Lines, Line{Text: in.Line, Food: f.Name, Grams: g})
	}
	if servings <= 0 {
		servings = 1
	}
	e.PerServing = total.scale(1 / servings)
	return e
}

// grams turns a line's amount into grams of its food.
func grams(in recipe.Ingredient, f Food) (float64, bool) {
	if in.Qty == nil {
		return 0, false
	}
	q := *in.Qty
	if in.QtyMax != nil {
		q = (q + *in.QtyMax) / 2
	}
	unit := in.Unit
	switch {
	case weights[unit] > 0:
		return q * weights[unit], true
	case tiny[unit] > 0:
		return q * tiny[unit], true
	case volumes[unit] > 0:
		if d, ok := density(f); ok {
			return q * volumes[unit] * d, true
		}
		return 0, false
	case unit == "can" || unit == "package" || unit == "jar" || unit == "bottle":
		if m := sizeRE.FindStringSubmatch(strings.ToLower(in.Note)); m != nil {
			n, _ := strconv.ParseFloat(m[1], 64)
			w := map[string]float64{"oz": 28.3495, "ounce": 28.3495, "ounces": 28.3495, "g": 1, "grams": 1, "lb": 453.592, "pound": 453.592, "pounds": 453.592}[m[2]]
			return q * n * w, true
		}
		return 0, false
	}
	if p, ok := piece(f, unit); ok {
		return q * p, true
	}
	return 0, false
}

// density is grams per ml, from the food's cup or spoon weight; liquids
// without one count as water.
func density(f Food) (float64, bool) {
	for _, p := range f.Portions {
		m := measureRE.FindStringSubmatch(p.Text)
		if m == nil {
			continue
		}
		amount := 1.0
		if m[1] != "" {
			amount = fraction(m[1])
		}
		if amount > 0 {
			return p.Grams / (amount * measureML[m[2]]), true
		}
	}
	name := strings.ToLower(f.Name)
	for _, l := range liquids {
		if strings.Contains(name, l) {
			return 1, true
		}
	}
	return 0, false
}

// piece is the weight of one: "1 clove", "1 slice", or for "2 eggs" a
// medium (or large, whole, small) one.
func piece(f Food, unit string) (float64, bool) {
	want := []string{unit}
	if unit == "" {
		want = []string{"medium", "large", "whole", "small", "fruit", "piece", "item"}
		if strings.HasPrefix(strings.ToLower(f.Name), "egg") {
			want = []string{"large", "medium", "whole"} // recipes mean large eggs
		}
	}
	for _, w := range want {
		for _, p := range f.Portions {
			if w != "" && strings.Contains(p.Text, w) && measureRE.FindString(p.Text) == "" {
				return p.Grams, true
			}
		}
	}
	return 0, false
}

func fraction(s string) float64 {
	if a, b, ok := strings.Cut(s, "/"); ok {
		x, _ := strconv.ParseFloat(a, 64)
		y, _ := strconv.ParseFloat(b, 64)
		if y == 0 {
			return 0
		}
		return x / y
	}
	x, _ := strconv.ParseFloat(s, 64)
	return x
}

func (n Nutrients) add(m Nutrients, times float64) Nutrients {
	return Nutrients{Kcal: n.Kcal + m.Kcal*times, Protein: n.Protein + m.Protein*times, Fat: n.Fat + m.Fat*times,
		Carbs: n.Carbs + m.Carbs*times, Fiber: n.Fiber + m.Fiber*times, Sugar: n.Sugar + m.Sugar*times,
		SodiumMg: n.SodiumMg + m.SodiumMg*times}
}

func (n Nutrients) scale(by float64) Nutrients { return Nutrients{}.add(n, by) }
