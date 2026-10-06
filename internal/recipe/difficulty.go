package recipe

import "strings"

// Difficulty levels. A recipe's Difficulty is one of them, or "" to work it out.
const (
	Easy   = "easy"
	Medium = "medium"
	Hard   = "hard"
)

// Techniques that take practice ("knead", "temper"…).
var techniques = []string{"knead", "let rise", "proof", "temper", "deep fry", "deep-fry", "candy thermometer", "caramel",
	"roux", "stiff peaks", "fold in", "laminat", "piping bag", "flamb", "sous vide", "blind bake", "puff pastry", "phyllo",
	"filo", "souffl", "custard", "emulsif", "debone", "spatchcock", "water bath", "julienne"}

// Level is the recipe's difficulty: the one chosen, or worked out from its
// active time (prep and cook, so a long rise doesn't count), how many steps
// and ingredients it has, and the techniques its steps use.
func (r *Recipe) Level() string {
	switch r.Difficulty {
	case Easy, Medium, Hard:
		return r.Difficulty
	}
	score := 0
	active := r.PrepMin + r.CookMin
	if active == 0 {
		active = r.TotalMin
	}
	switch {
	case active > 120:
		score += 2
	case active > 50:
		score++
	}
	switch n := len(r.Steps); {
	case n > 12:
		score += 2
	case n > 6:
		score++
	}
	if len(r.Ingredients) > 14 {
		score++
	}
	var text strings.Builder
	for _, s := range r.Steps {
		text.WriteString(strings.ToLower(s.Text) + " ")
	}
	found := 0
	for _, t := range techniques {
		if strings.Contains(text.String(), t) {
			found++
		}
	}
	score += min(found, 2)
	switch {
	case score >= 4:
		return Hard
	case score >= 2:
		return Medium
	}
	return Easy
}
