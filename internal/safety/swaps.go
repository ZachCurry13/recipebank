package safety

import (
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

// Swap is a replacement for one ingredient that works for everyone eating.
type Swap struct {
	Ingredient int    `json:"ingredient"`
	From       string `json:"from"`
	To         string `json:"to"`
	Note       string `json:"note,omitempty"`
}

type swapIdea struct {
	to, note string
}

// swapTable: what a food can be replaced by. Every idea is checked against
// every diner before it's offered.
var swapTable = []struct {
	when  phrases
	ideas []swapIdea
}{
	{compile("butter"), []swapIdea{{"olive oil", "for cooking; use ¾ as much"}, {"coconut oil", "for baking, same amount"}}},
	{compile("milk", "buttermilk", "half and half"), []swapIdea{{"oat milk", ""}, {"rice milk", ""}, {"coconut milk", "richer"}}},
	{compile("cream", "heavy cream", "whipping cream"), []swapIdea{{"coconut cream", ""}}},
	{compile("cheese", "parmesan"), []swapIdea{{"nutritional yeast", "for a cheesy taste"}}},
	{compile("egg"), []swapIdea{{"flax egg (1 tbsp ground flaxseed + 3 tbsp water per egg)", "for baking; let it sit 5 minutes"},
		{"chia egg (1 tbsp chia seeds + 3 tbsp water per egg)", "for baking"}}},
	{compile("flour", "all purpose flour", "bread flour", "wheat flour"), []swapIdea{{"rice flour", "best mixed with a little cornstarch"}}},
	{compile("breadcrumb", "bread crumb", "panko"), []swapIdea{{"crushed corn tortilla chips", "check the label"}, {"cornmeal", ""}}},
	{compile("pasta", "spaghetti", "noodle", "macaroni", "penne"), []swapIdea{{"rice", ""}, {"zucchini noodles", ""}}},
	{compile("soy sauce", "tamari", "shoyu"), []swapIdea{{"coconut aminos", "check the label"}}},
	{compile("peanut butter", "almond butter", "nut butter"), []swapIdea{{"sunflower seed butter", "check the label"}}},
	{compile("almond", "walnut", "pecan", "cashew", "hazelnut", "pistachio", "nut", "peanut"), []swapIdea{{"sunflower seeds", ""}, {"pumpkin seeds", ""}}},
	{compile("tahini"), []swapIdea{{"sunflower seed butter", "check the label"}}},
	{compile("fish sauce", "worcestershire"), []swapIdea{{"coconut aminos", "add a pinch of salt; check the label"}}},
	{compile("shrimp", "crab", "prawn", "lobster", "scallop"), []swapIdea{{"chicken", ""}, {"white beans", ""}}},
	{compile("chicken", "beef", "pork", "turkey", "lamb", "sausage", "bacon", "meat"), []swapIdea{{"mushrooms", ""}, {"chickpeas", ""}, {"lentils", ""}}},
	{compile("chicken broth", "beef broth", "chicken stock", "beef stock", "broth", "stock"), []swapIdea{{"water, salt and herbs", "less flavor"}}},
	{compile("honey"), []swapIdea{{"maple syrup", ""}}},
	{compile("gelatin"), []swapIdea{{"agar", "use about half as much"}}},
	{compile("wine", "beer", "sherry"), []swapIdea{{"water and apple cider vinegar", "mostly water, a splash of vinegar"}}},
	{compile("sesame oil"), []swapIdea{{"sunflower oil", ""}}},
	{compile("mayonnaise", "mayo"), []swapIdea{{"mashed avocado", ""}}},
	{compile("yogurt", "sour cream"), []swapIdea{{"coconut cream and lemon juice", "a squeeze of lemon"}}},
	{compile("jalapeno", "serrano", "cayenne", "chili", "chile", "habanero", "hot sauce", "red pepper flake", "chili flake", "sriracha"),
		[]swapIdea{{"sweet paprika", "the color without the heat"}, {"bell pepper", ""}}},
}

// Swaps finds replacements for the lines that give anyone a "no" or
// "not sure", keeping only swaps that are OK for every diner.
func Swaps(r *recipe.Recipe, diners []Person, verdicts []Verdict) []Swap {
	problem := map[int]bool{}
	for _, v := range verdicts {
		for _, x := range v.Reasons {
			for _, i := range x.Ingredients {
				problem[i] = true
			}
		}
	}
	out := []Swap{}
	for i, in := range r.Ingredients {
		if !problem[i] {
			continue
		}
		t := newText(in.Line)
		for _, row := range swapTable {
			if !t.has(row.when) {
				continue
			}
			for _, idea := range row.ideas {
				if okForAll(r.Area, idea.to, diners) {
					out = append(out, Swap{Ingredient: i, From: in.Food, To: idea.to, Note: idea.note})
				}
			}
			break
		}
	}
	return out
}

func okForAll(area, food string, diners []Person) bool {
	// Only the name counts: "(1 tbsp … per egg)" explains how to make it.
	name := strings.TrimSpace(strings.SplitN(food, "(", 2)[0])
	probe := &recipe.Recipe{Area: area, Heat: 0, Ingredients: []recipe.Ingredient{{Line: name, Food: name}}}
	for _, p := range diners {
		if Check(probe, p).Status != OK {
			return false
		}
	}
	return true
}
