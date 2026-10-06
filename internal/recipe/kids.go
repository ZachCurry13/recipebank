package recipe

import (
	"regexp"
	"strings"
)

// "Kids can help": each step is marked by what it needs a grown-up for, from
// word lists (never the AI). A step with none of them is one a kid can do.
// The lists lean careful: a step that might need a knife or heat is marked,
// but words used as a description ("the diced onion", "baking soda") aren't.

// What a step can need a grown-up for.
const (
	NeedKnife = "knife" // cutting
	NeedSharp = "sharp" // graters, blenders, can openers
	NeedStove = "stove" // cooking on the stove
	NeedOven  = "oven"  // the oven or broiler
	NeedHot   = "hot"   // hot liquid, oil, steam or dishes
)

// needOrder keeps a step's needs in a steady order.
var needOrder = []string{NeedKnife, NeedSharp, NeedStove, NeedOven, NeedHot}

// needWords are patterns that start at a word: `chop\b` is "chop", not "chopped".
var needWords = map[string][]string{
	NeedKnife: {`knife`, `knives`, `chop\b`, `chopping`, `dice\b`, `dicing`, `slice\b`, `slicing`, `mince\b`, `mincing`, `cut\b`,
		`cutting\b`, `carve`, `carving`, `julienne`, `halve\b`, `halving`, `quarter (the|them|it)\b`, `trim\b`, `trimming`,
		`fillet`, `debone`, `score\b`, `scoring`, `pit\b`, `pitting`, `core\b`, `coring`, `cleaver`, `butterfl`, `spatchcock`,
		`segment`},
	NeedSharp: {`grate\b`, `grating`, `grater`, `zest\b`, `zesting`, `zester`, `microplane`, `mandoline`, `peeler`, `blender`,
		`blend (until|the|on|it|everything|in)\b`, `pur[eé]e\b`, `food processor`, `processor`, `immersion`, `can opener`,
		`open the can`, `skewer`},
	NeedStove: {`stove`, `stovetop`, `burner`, `skillet`, `frying pan`, `saucepan`, `pot\b`, `wok`, `griddle`, `saut[eé]\b`,
		`saut[eé]ing`, `fry\b`, `frying`, `sear\b`, `searing`, `brown the`, `simmer\b`, `simmering`, `boil\b`, `boiling`,
		`heat the`, `heat (oil|butter)`, `(low|medium|high) heat`, `medium-high`, `medium-low`, `over heat`, `reduce the heat`,
		`caramelize`, `toast the`, `poach\b`, `poaching`, `stir-fry`, `deglaze`, `cook\b`, `cooking\b`},
	NeedOven: {`oven`, `bake\b`, `bakes\b`, `baking`, `roast\b`, `roasting`, `broil`, `preheat`, `pre-heat`, `°[fc]`,
		`\d+ degrees`},
	NeedHot: {`hot\b`, `boil\b`, `boiling`, `steam\b`, `steaming`, `drain\b`, `heat the (oil|butter)`, `heat (oil|butter)`,
		`deep[ -]fry`, `microwave`, `melt\b`, `melting`, `scald`, `sizzl`, `caramel\b`, `candy thermometer`,
		`(out of|from) the oven`, `remove from the heat`, `off the heat`, `temper\b`},
}

// harmless are words that look like a need but aren't one.
var harmless = strings.NewReplacer("baking soda", "", "baking powder", "", "baking sheet", "", "baking dish", "",
	"baking pan", "", "baking tray", "", "baking paper", "", "baking parchment", "", "baking cup", "", "baking mat", "",
	"cooking spray", "", "cooking oil", "", "hot sauce", "", "hot dog", "",
	"hot pepper", "", "hot chili", "", "hot chile", "", "cookie cutter", "", "pizza cutter", "", "biscuit cutter", "")

var needRE = func() map[string]*regexp.Regexp {
	out := map[string]*regexp.Regexp{}
	for need, words := range needWords {
		out[need] = regexp.MustCompile(`(^|[^a-z])(` + strings.Join(words, "|") + `)`)
	}
	return out
}()

// StepNeeds is what a step needs a grown-up for; none means a kid can do it.
func StepNeeds(text string) []string {
	t := harmless.Replace(strings.ToLower(text))
	needs := []string{}
	for _, need := range needOrder {
		if needRE[need].MatchString(t) {
			needs = append(needs, need)
		}
	}
	return needs
}

// Needs is what each step needs a grown-up for, in step order.
func (r *Recipe) Needs() [][]string {
	out := make([][]string, len(r.Steps))
	for i, st := range r.Steps {
		out[i] = StepNeeds(st.Text)
	}
	return out
}

// KidsCanHelp says a kitchen recipe has at least two steps a kid can do on
// their own (or all of them, for a short one).
func (r *Recipe) KidsCanHelp() bool {
	if r.Area == AreaHome || len(r.Steps) == 0 {
		return false
	}
	n := 0
	for _, needs := range r.Needs() {
		if len(needs) == 0 {
			n++
		}
	}
	return n >= 2 || n == len(r.Steps)
}
