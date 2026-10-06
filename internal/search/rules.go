// Package search reads plain-words requests ("quick dairy-free dinners
// everyone can eat") into rules RecipeBank can check without the AI.
package search

import (
	"regexp"
	"strconv"
	"strings"
)

// Rules are what a request asks for that can be checked for sure.
type Rules struct {
	Diets    []string `json:"diets"`    // diet keys every recipe must pass
	MaxMin   int      `json:"max_min"`  // 0 = any time
	Everyone bool     `json:"everyone"` // OK for everyone eating
	Quick    bool     `json:"quick"`    // "quick", "fast", "easy weeknight": 30 minutes unless a time is given
}

var dietWords = []struct {
	key   string
	words []string
}{
	{"dairyfree", []string{"dairy-free", "dairy free", "no dairy", "without dairy", "milk-free", "lactose-free"}},
	{"glutenfree", []string{"gluten-free", "gluten free", "no gluten", "celiac"}},
	{"nutfree", []string{"nut-free", "nut free", "no nuts", "without nuts"}},
	{"vegan", []string{"vegan"}},
	{"vegetarian", []string{"vegetarian", "veggie"}},
	{"pescatarian", []string{"pescatarian"}},
	{"meatless", []string{"meatless", "no meat", "without meat", "lent", "lenten", "friday"}},
	{"halal", []string{"halal"}},
	{"kosher", []string{"kosher"}},
}

var (
	minutesRE = regexp.MustCompile(`(?i)(?:under|less than|within|in|max(?:imum)?|no more than)\s+(\d{1,3})\s*(?:minutes|mins?|m\b)|(\d{1,3})\s*(?:minutes|mins?)\s*(?:or less|max)`)
	hoursRE   = regexp.MustCompile(`(?i)(?:under|less than|within|in)\s+(?:an?|one|1)\s+hour`)
	quickRE   = regexp.MustCompile(`(?i)\b(quick|fast|speedy|weeknight|busy night|in a hurry)\b`)
	everyRE   = regexp.MustCompile(`(?i)\b(everyone|everybody|whole family|all of us|the family) (can|could) (eat|have)|\bsafe for (everyone|all)\b|\bfor everyone\b`)
)

// Parse reads the rules out of a request.
func Parse(text string) Rules {
	low := strings.ToLower(text)
	var r Rules
	for _, d := range dietWords {
		for _, w := range d.words {
			if strings.Contains(low, w) {
				if d.key == "vegetarian" && strings.Contains(low, "vegan") {
					break
				}
				r.Diets = append(r.Diets, d.key)
				break
			}
		}
	}
	if m := minutesRE.FindStringSubmatch(text); m != nil {
		n, _ := strconv.Atoi(m[1] + m[2])
		r.MaxMin = n
	} else if hoursRE.MatchString(text) {
		r.MaxMin = 60
	}
	if quickRE.MatchString(text) {
		r.Quick = true
		if r.MaxMin == 0 {
			r.MaxMin = 30
		}
	}
	r.Everyone = everyRE.MatchString(text)
	if r.Diets == nil {
		r.Diets = []string{}
	}
	return r
}

// stop words carry no meaning for matching recipes.
var stop = map[string]bool{"a": true, "an": true, "the": true, "and": true, "or": true, "with": true, "for": true, "of": true,
	"to": true, "in": true, "on": true, "that": true, "we": true, "can": true, "our": true, "my": true, "some": true,
	"something": true, "recipes": true, "recipe": true, "food": true, "meals": true, "meal": true, "ideas": true, "like": true,
	"want": true, "make": true, "kids": true, "family": true, "everyone": true, "eat": true, "good": true, "nice": true,
	"under": true, "minutes": true, "minute": true, "min": true, "quick": true, "easy": true, "fast": true, "free": true,
	"no": true, "without": true, "all": true, "is": true, "are": true, "be": true, "it": true, "this": true, "these": true}

// Keywords are a request's meaningful words, for matching recipe text.
func Keywords(text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '\'' || r > 127)
	}) {
		if len(w) < 3 || stop[w] || seen[w] {
			continue
		}
		if _, err := strconv.Atoi(w); err == nil {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	return out
}
