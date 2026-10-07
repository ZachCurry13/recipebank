package safety

import (
	"strings"
	"sync"
)

// Exception is something made from an allergen that a person with that
// allergy may still have, because their doctor says so: most people with a
// soy allergy can have highly refined soybean oil (refining takes the
// protein out), but not soy lecithin or soy sauce. A parent ticks it, or
// types one ("almond" for someone allergic only to some tree nuts).
//
// Only that exact food is let through: the rest of the line is still checked
// ("soybean oil and soy sauce" is still soy), and words that keep the protein
// in ("cold-pressed peanut oil") keep it counted.
type Exception struct {
	Key    string  `json:"key"`   // what's saved for the person: "soybean oil"
	Label  string  `json:"label"` // "Soybean oil (highly refined)"
	Note   string  `json:"note"`
	words  phrases // what it's called on a recipe or label
	unless phrases // words on the line that keep it counted
}

// pressed are words for oils that keep their protein.
var pressed = []string{"cold pressed", "cold-pressed", "expeller pressed", "expeller-pressed", "unrefined", "virgin",
	"extra virgin", "roasted", "toasted", "gourmet", "aromatic", "artisan"}

// exceptions are the ones offered for each allergen.
var exceptions = map[string][]Exception{
	"soy": {
		{Key: "soybean oil", Label: "Soybean oil (highly refined)",
			Note:  "Most people with a soy allergy can have it: refining takes the protein out. Cold-pressed soybean oil still counts.",
			words: compile("soybean oil", "soy bean oil", "soy oil", "soya oil", "soya bean oil", "soyabean oil"), unless: compile(pressed...)},
		{Key: "soy lecithin", Label: "Soy lecithin", Note: "Some people with a soy allergy can have it, some can't. Ask their doctor.",
			words: compile("soy lecithin", "soya lecithin", "lecithin soy", "lecithin from soy", "soybean lecithin")},
	},
	"peanut": {
		{Key: "peanut oil", Label: "Peanut oil (highly refined)",
			Note:  "Highly refined peanut oil usually has no peanut protein left. Cold-pressed, roasted or gourmet peanut oil still counts.",
			words: compile("peanut oil", "groundnut oil", "arachis oil"), unless: compile(pressed...)},
	},
}

func init() {
	for i := range Allergens {
		Allergens[i].Exceptions = exceptions[Allergens[i].Key]
	}
}

// typed are phrases compiled for exceptions a parent typed, kept for speed.
var typed sync.Map // name → phrases

// exception is the offered exception with that key, or the typed name itself.
func (a Allergen) exception(name string) Exception {
	for _, e := range a.Exceptions {
		if e.Key == name {
			return e
		}
	}
	p, ok := typed.Load(name)
	if !ok {
		p, _ = typed.LoadOrStore(name, compile(name))
	}
	return Exception{Key: name, Label: name, words: p.(phrases)}
}

// maskAllowed blanks in m what the person may have despite the allergy; t is
// the whole line, where words like "cold-pressed" are looked for.
func maskAllowed(m, t text, a Allergen, allowed []string) {
	for _, name := range allowed {
		ex := a.exception(name)
		if ex.unless.byFirst != nil && t.has(ex.unless) {
			continue
		}
		m.mask(ex.words)
	}
}

// onlyAllowed reports whether a line's food is nothing but what the person
// may have ("soybean oil"), so strict mode knows what it is.
func onlyAllowed(food string, t text, a Allergen, allowed []string) bool {
	f := newText(food)
	if len(f) == 0 || len(allowed) == 0 {
		return false
	}
	maskAllowed(f, t, a, allowed)
	for _, w := range f {
		if w != "" {
			return false
		}
	}
	return true
}

// generalNames are an allergen's names for itself: typing one as an
// exception would let the whole allergen through, which isn't an exception.
// Specific foods ("almond" for tree nuts, "tuna" for fish) are fine.
var generalNames = map[string][]string{"soy": {"soya", "soybean", "soy bean"}, "milk": {"dairy", "lactose"},
	"peanut": {"groundnut"}, "treenut": {"nut", "tree nut", "tree"}, "egg": {}, "wheat": {"gluten"}, "fish": {},
	"shellfish": {"crustacean", "seafood"}, "sesame": {"sesame seed"}}

// SameAsAllergen reports whether a typed exception is just the allergen
// itself ("soy" for a soy allergy).
func SameAsAllergen(key, name string) bool {
	a, ok := AllergenByKey(key)
	if !ok {
		return false
	}
	n := strings.Join(words(name), " ")
	if n == "" {
		return true
	}
	for _, g := range append([]string{a.Key, a.Label}, generalNames[key]...) {
		if n == strings.Join(words(g), " ") {
			return true
		}
	}
	return false
}
