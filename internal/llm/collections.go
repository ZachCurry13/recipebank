package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

// PickSystem asks the AI to choose recipes for a theme from a list.
const PickSystem = `You help a family choose recipes from their own recipe list.
Reply with JSON only: {"ids": [3, 7], "why": {"3": "a few words"}}.
Pick only recipes from the list that clearly fit the request, best first, at most 20.
If none fit, reply {"ids": []}. Never invent recipes or ids.`

// RecipeLine is one recipe as the AI sees it: short, so a home AI's small
// working memory holds the whole list.
func RecipeLine(r *recipe.Recipe) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d: %s", r.ID, r.Title)
	var facts []string
	for _, f := range []string{r.Course, r.Cuisine, r.Protein} {
		if f != "" {
			facts = append(facts, f)
		}
	}
	if r.TotalMin > 0 {
		facts = append(facts, fmt.Sprintf("%d min", r.TotalMin))
	}
	if len(facts) > 0 {
		b.WriteString(" (" + strings.Join(facts, ", ") + ")")
	}
	var foods []string
	for i, in := range r.Ingredients {
		if i == 8 {
			break
		}
		foods = append(foods, in.Food)
	}
	if len(foods) > 0 {
		b.WriteString(": " + strings.Join(foods, ", "))
	}
	s := b.String()
	if len(s) > 220 {
		s = s[:220]
	}
	return s
}

// PickPrompt is the user message: the request and the list.
func PickPrompt(request string, lines []string) string {
	return "Request: " + request + "\n\nRecipes:\n" + strings.Join(lines, "\n")
}

// ParsePicks reads the AI's choice, keeping only ids from the list.
func ParsePicks(out string, allowed map[int64]bool) ([]int64, map[int64]string, error) {
	var a struct {
		IDs []json.Number     `json:"ids"`
		Why map[string]string `json:"why"`
	}
	if err := json.Unmarshal([]byte(jsonObject(out)), &a); err != nil {
		return nil, nil, errors.New("the AI's answer wasn't valid JSON")
	}
	ids, why := []int64{}, map[int64]string{}
	seen := map[int64]bool{}
	for _, n := range a.IDs {
		id, err := strconv.ParseInt(n.String(), 10, 64)
		if err != nil || !allowed[id] || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
		if w := strings.TrimSpace(a.Why[n.String()]); w != "" {
			why[id] = w
		}
	}
	return ids, why, nil
}
