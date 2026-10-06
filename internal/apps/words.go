package apps

// Small readers shared by the formats: times, servings, courses, lines.

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

var durPart = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(d|days?|h|hrs?|hours?|m|mins?|minutes?)\b`)

// minutesOf reads a time: minutes as a number, ISO 8601 ("PT1H30M"), or words ("1 hr 30 mins").
func minutesOf(v any) int {
	switch t := v.(type) {
	case float64:
		if t > 0 && t < 100000 {
			return int(t + 0.5)
		}
	case string:
		s := strings.TrimSpace(t)
		if strings.HasPrefix(strings.ToUpper(s), "P") {
			return recipe.Minutes(s)
		}
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n < 100000 {
			return n
		}
		total := 0.0
		for _, m := range durPart.FindAllStringSubmatch(s, -1) {
			n, _ := strconv.ParseFloat(m[1], 64)
			switch strings.ToLower(m[2])[0] {
			case 'd':
				total += n * 1440
			case 'h':
				total += n * 60
			default:
				total += n
			}
		}
		return int(total + 0.5)
	}
	return 0
}

var leadNum = regexp.MustCompile(`\d+(?:\.\d+)?`)

// servings reads "4", "Serves 4-6" or "2 loaves": the number, and the words when there are any.
func servings(v any) (float64, string) {
	switch t := v.(type) {
	case float64:
		if t > 0 && t < 1000 {
			return t, ""
		}
	case []any:
		if len(t) > 0 {
			return servings(t[0])
		}
	case string:
		s := strings.TrimSpace(t)
		n, _ := strconv.ParseFloat(leadNum.FindString(s), 64)
		if n <= 0 || n >= 1000 {
			n = 0
		}
		if _, err := strconv.ParseFloat(s, 64); err == nil {
			return n, ""
		}
		return n, s
	}
	return 0, ""
}

var courses = map[string]string{"breakfast": "breakfast", "brunch": "breakfast", "main": "main", "main course": "main",
	"main dish": "main", "dinner": "main", "entree": "main", "entrée": "main", "side": "side", "side dish": "side", "sides": "side",
	"soup": "soup", "soups": "soup", "salad": "salad", "salads": "salad", "appetizer": "appetizer", "appetizers": "appetizer",
	"starter": "appetizer", "snack": "snack", "snacks": "snack", "dessert": "dessert", "desserts": "dessert", "baking": "baking",
	"bread": "bread", "breads": "bread", "drink": "drink", "drinks": "drink", "beverages": "drink", "sauce": "sauce", "sauces": "sauce"}

// courseOf picks a course from an app's categories, when one of them is one.
func courseOf(cats []string) string {
	for _, c := range cats {
		if k, ok := courses[strings.ToLower(strings.TrimSpace(c))]; ok {
			return k
		}
	}
	return ""
}

func ingredientLines(lines []string) []recipe.Ingredient {
	var out []recipe.Ingredient
	section := ""
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		// Paprika and others mark a group with a line ending in ":" ("For the sauce:").
		if strings.HasSuffix(l, ":") && len(l) < 60 && !strings.ContainsAny(l, "0123456789") {
			section = strings.TrimSuffix(l, ":")
			continue
		}
		out = append(out, recipe.Ingredient{Line: l, Section: section})
	}
	return out
}

var stepNum = regexp.MustCompile(`^(?:step\s*)?\d+[.)]\s+`)

// stepLines splits directions into steps: one per line, numbers dropped.
func stepLines(s string) []recipe.Step {
	var out []recipe.Step
	for _, l := range strings.Split(strings.ReplaceAll(s, "\r", ""), "\n") {
		if l = strings.TrimSpace(stepNum.ReplaceAllString(strings.TrimSpace(l), "")); l != "" {
			out = append(out, recipe.Step{Text: l})
		}
	}
	return out
}

func notesOf(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case []any:
		var parts []string
		for _, it := range t {
			if m, ok := it.(map[string]any); ok {
				parts = append(parts, strings.TrimSpace(joinWords(text(m["title"])+sep(text(m["title"])), text(m["text"]))))
			} else if s := text(it); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "\n\n")
	}
	return ""
}

func names(v any) []string {
	var out []string
	if list, ok := v.([]any); ok {
		for _, it := range list {
			if n := nameOf(it); n != "" {
				out = append(out, n)
			}
		}
	}
	return out
}

func nameOf(v any) string {
	if m, ok := v.(map[string]any); ok {
		return text(m["name"])
	}
	return text(v)
}

func text(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	}
	return ""
}

func firstOf(s ...string) string {
	for _, x := range s {
		if strings.TrimSpace(x) != "" {
			return strings.TrimSpace(x)
		}
	}
	return ""
}

func joinWords(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}

func comma(note string) string {
	if note = strings.TrimSpace(note); note != "" {
		return ", " + note
	}
	return ""
}

func sep(title string) string {
	if title != "" {
		return ":"
	}
	return ""
}

func by(source string) string {
	if source = strings.TrimSpace(source); source != "" && !strings.HasPrefix(source, "http") {
		return " (" + source + ")"
	}
	return ""
}

// webAddress keeps a source only when it's a web address.
func webAddress(u string) string {
	u = strings.TrimSpace(u)
	if strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
		return u
	}
	return ""
}
