package recipe

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// unitNames maps how units are written to one canonical name.
var unitNames = map[string]string{
	"cup": "cup", "cups": "cup", "c": "cup", "c.": "cup",
	"tablespoon": "tbsp", "tablespoons": "tbsp", "tbsp": "tbsp", "tbsp.": "tbsp", "tbs": "tbsp", "tbl": "tbsp", "T": "tbsp",
	"teaspoon": "tsp", "teaspoons": "tsp", "tsp": "tsp", "tsp.": "tsp", "t": "tsp",
	"ounce": "oz", "ounces": "oz", "oz": "oz", "oz.": "oz",
	"fl oz": "fl oz", "fluid ounce": "fl oz", "fluid ounces": "fl oz", "fl. oz.": "fl oz",
	"pound": "lb", "pounds": "lb", "lb": "lb", "lbs": "lb", "lb.": "lb", "lbs.": "lb",
	"pint": "pint", "pints": "pint", "pt": "pint",
	"quart": "quart", "quarts": "quart", "qt": "quart",
	"gallon": "gallon", "gallons": "gallon", "gal": "gallon",
	"gram": "g", "grams": "g", "g": "g", "gr": "g",
	"kilogram": "kg", "kilograms": "kg", "kg": "kg",
	"milliliter": "ml", "milliliters": "ml", "millilitre": "ml", "millilitres": "ml", "ml": "ml",
	"liter": "l", "liters": "l", "litre": "l", "litres": "l", "l": "l",
	"pinch": "pinch", "pinches": "pinch", "dash": "dash", "dashes": "dash",
	"drop": "drop", "drops": "drop",
	"clove": "clove", "cloves": "clove", "can": "can", "cans": "can", "tin": "can", "tins": "can", "stick": "stick", "sticks": "stick",
	"package": "package", "packages": "package", "pkg": "package", "slice": "slice", "slices": "slice",
	"bunch": "bunch", "bunches": "bunch", "sprig": "sprig", "sprigs": "sprig", "head": "head", "heads": "head",
	"jar": "jar", "jars": "jar", "bottle": "bottle", "bottles": "bottle", "handful": "handful",
}

var fractions = map[rune]float64{'½': .5, '⅓': 1.0 / 3, '⅔': 2.0 / 3, '¼': .25, '¾': .75, '⅕': .2, '⅛': .125, '⅜': .375, '⅝': .625, '⅞': .875}

// qtyRE finds a leading amount: "1", "1.5", "1 1/2", "1/2", "1½", "2-3", "2 to 3".
var qtyRE = regexp.MustCompile(`^\s*((?:\d+\s+)?\d+/\d+|\d*[.,]?\d+\s*[½⅓⅔¼¾⅕⅛⅜⅝⅞]?|[½⅓⅔¼¾⅕⅛⅜⅝⅞])(?:\s*(?:-|–|to)\s*((?:\d+\s+)?\d+/\d+|\d*[.,]?\d+\s*[½⅓⅔¼¾⅕⅛⅜⅝⅞]?|[½⅓⅔¼¾⅕⅛⅜⅝⅞]))?\s*`)

// ParseLine understands an ingredient line as far as a rule can:
// "1 ½ cups flour, sifted" → 1.5 cup "flour" (note "sifted").
func ParseLine(line string) Ingredient {
	in := Ingredient{Line: strings.TrimSpace(line)}
	rest := strings.TrimLeft(in.Line, "-•*▢□ \t")
	if m := qtyRE.FindStringSubmatch(rest); m != nil && strings.TrimSpace(m[0]) != "" {
		if q, ok := parseQty(m[1]); ok {
			in.Qty = &q
			if m[2] != "" {
				if q2, ok := parseQty(m[2]); ok && q2 > q {
					in.QtyMax = &q2
				}
			}
			rest = rest[len(m[0]):]
		}
	}
	// "(8 oz)" right after the amount stays in the note.
	var notes []string
	if strings.HasPrefix(rest, "(") {
		if end := strings.Index(rest, ")"); end > 0 {
			notes = append(notes, strings.TrimSpace(rest[1:end]))
			rest = strings.TrimSpace(rest[end+1:])
		}
	}
	// "a pinch of nutmeg", "an 8 oz package": one of the unit.
	if in.Qty == nil {
		for _, a := range []string{"a ", "an "} {
			if strings.HasPrefix(strings.ToLower(rest), a) {
				var one Ingredient
				if after := takeUnit(&one, rest[len(a):]); one.Unit != "" {
					q := 1.0
					in.Qty, in.Unit, rest = &q, one.Unit, after
				}
			}
		}
	}
	// "pinch of salt", "can of beans": one of the unit.
	if in.Qty == nil {
		var one Ingredient
		if after := takeUnit(&one, rest); one.Unit != "" && strings.HasPrefix(strings.ToLower(after), "of ") {
			q := 1.0
			in.Qty, in.Unit, rest = &q, one.Unit, after
		}
	}
	// "2 x 400g tins tomatoes": the size goes in the note, the tin is the unit.
	if m := timesRE.FindStringSubmatch(rest); in.Qty != nil && m != nil {
		notes = append(notes, strings.ReplaceAll(m[1], ",", ".")+" "+strings.ToLower(m[2]))
		rest = rest[len(m[0]):]
	}
	if in.Qty != nil && in.Unit == "" {
		rest = takeUnit(&in, rest)
	}
	// "1 tablespoon + 1 teaspoon sugar": the second amount joins the first.
	if m := plusRE.FindStringSubmatch(rest); in.Qty != nil && m != nil {
		if q2, ok := parseQty(m[1]); ok {
			var two Ingredient
			after := takeUnit(&two, rest[len(m[0]):])
			if f1, f2, ok := sameKind(in.Unit, two.Unit); ok {
				q := math.Round((*in.Qty+q2*f2/f1)*1000) / 1000
				in.Qty, rest = &q, after
			}
		}
	}
	food := rest
	if i := strings.Index(food, ","); i >= 0 {
		notes = append(notes, strings.TrimSpace(food[i+1:]))
		food = food[:i]
	}
	if i := strings.Index(food, "("); i >= 0 {
		if end := strings.Index(food[i:], ")"); end > 0 {
			notes = append(notes, strings.TrimSpace(food[i+1:i+end]))
			food = food[:i] + food[i+end+1:]
		}
	}
	in.Food = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(food), "of "))
	// "Salt and pepper to taste": "to taste" is a note, not part of the food.
	// A weight after the food ("chocolate chips 340g") is a note, not part of its name.
	if m := trailingWeight.FindStringSubmatchIndex(in.Food); m != nil && m[0] > 0 {
		notes = append(notes, strings.TrimSpace(in.Food[m[2]:m[3]]))
		in.Food = strings.TrimSpace(in.Food[:m[0]])
	}
	for _, t := range trailingNotes {
		if low := strings.ToLower(in.Food); strings.HasSuffix(low, t) && len(low) > len(t) {
			in.Food = strings.TrimSpace(in.Food[:len(in.Food)-len(t)])
			notes = append([]string{strings.TrimSpace(t)}, notes...)
			break
		}
	}
	in.Note = strings.Join(nonEmpty(notes), "; ")
	if in.Food == "" {
		in.Food = in.Line
	}
	return in
}

// takeUnit moves a unit word from the start of rest into in.Unit.
func takeUnit(in *Ingredient, rest string) string {
	words := strings.Fields(rest)
	for n := min(3, len(words)); n >= 1; n-- {
		key := strings.Join(words[:n], " ")
		u, ok := unitName(key)
		if !ok && strings.HasSuffix(key, ".") { // cards write "T." and "pt."
			key = strings.TrimSuffix(key, ".")
			u, ok = unitName(key)
		}
		if !ok {
			continue
		}
		// A one-letter unit only counts when more words follow ("1 c flour").
		if len(key) == 1 && len(words) == n {
			return rest
		}
		in.Unit = u
		return strings.Join(words[n:], " ")
	}
	return rest
}

func parseQty(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	total := 0.0
	for _, part := range strings.Fields(s) {
		// A glyph fraction may be glued on: "1½".
		var frac float64
		for r, v := range fractions {
			if strings.ContainsRune(part, r) {
				frac = v
				part = strings.ReplaceAll(part, string(r), "")
			}
		}
		total += frac
		if part == "" {
			continue
		}
		if a, b, ok := strings.Cut(part, "/"); ok {
			x, e1 := strconv.ParseFloat(a, 64)
			y, e2 := strconv.ParseFloat(b, 64)
			if e1 != nil || e2 != nil || y == 0 {
				return 0, false
			}
			total += x / y
			continue
		}
		f, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return 0, false
		}
		total += f
	}
	return total, total > 0
}

func nonEmpty(ss []string) []string {
	out := ss[:0]
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func unitName(key string) (string, bool) {
	u, ok := unitNames[key]
	if !ok && key != "T" { // "T" is a tablespoon, never lower-cased to "t"
		u, ok = unitNames[strings.ToLower(key)]
	}
	return u, ok
}

var (
	// trailingWeight: "340g" or "(340 g)" at the end of a food name.
	trailingWeight = regexp.MustCompile(`(?i)\s+\(?(\d+(?:[.,]\d+)?\s*(?:g|kg|grams?|ml|oz|lbs?))\)?\.?$`)
	timesRE        = regexp.MustCompile(`(?i)^[x×]\s*(\d+(?:[.,]\d+)?)\s*(g|kg|ml|l|oz|lb)\b\.?\s*`)
	plusRE         = regexp.MustCompile(`(?i)^(?:\+|plus)\s+(\d+(?:\s+\d+/\d+)?|\d+/\d+|\d*[.,]\d+)\s+`)
	// trailingNotes end a food name but say how it's used.
	trailingNotes = []string{" to taste", " as needed", " for serving", " to serve", " for garnish", " to garnish",
		" (optional)", " optional"}
	unitML = map[string]float64{"tsp": 4.929, "tbsp": 14.787, "cup": 236.588, "fl oz": 29.574, "ml": 1, "l": 1000,
		"pint": 473.176, "quart": 946.353, "gallon": 3785.41}
	unitG = map[string]float64{"g": 1, "kg": 1000, "oz": 28.3495, "lb": 453.592}
)

// sameKind gives both units' size in one measure (ml or g) when they can be added.
func sameKind(a, b string) (float64, float64, bool) {
	if unitML[a] > 0 && unitML[b] > 0 {
		return unitML[a], unitML[b], true
	}
	if unitG[a] > 0 && unitG[b] > 0 {
		return unitG[a], unitG[b], true
	}
	return 0, 0, false
}

// Convert changes an amount between units of the same kind (volume or
// weight); ok is false when they can't be compared ("cup" and "lb").
func Convert(q float64, from, to string) (float64, bool) {
	if from == to {
		return q, true
	}
	if a, b, ok := sameKind(from, to); ok {
		return q * a / b, true
	}
	return 0, false
}
