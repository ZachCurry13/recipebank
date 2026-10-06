package llm

import (
	"encoding/json"
	"errors"
	"strings"
)

// FridgePrompt asks the vision model what food a photo shows, for "What can
// I make?". The person ticks off what's really there before it's used.
const FridgePrompt = `These photos show food at home: a fridge, a pantry shelf or a counter. List the foods you can
clearly see, as plain names a recipe would use ("eggs", "cheddar cheese", "carrots"), without brands or
amounts. Leave out anything you can't identify. Reply with JSON only: {"foods": ["...", "..."]}`

// ParseFoods reads the list (at most 40 short, different names).
func ParseFoods(out string) ([]string, error) {
	var a struct {
		Foods []string `json:"foods"`
	}
	if err := json.Unmarshal([]byte(jsonObject(out)), &a); err != nil {
		return nil, errors.New("the AI's answer wasn't valid JSON")
	}
	foods, seen := []string{}, map[string]bool{}
	for _, f := range a.Foods {
		f = strings.TrimSpace(f)
		if k := strings.ToLower(f); f != "" && len(f) <= 40 && !seen[k] {
			seen[k] = true
			if foods = append(foods, f); len(foods) == 40 {
				break
			}
		}
	}
	if len(foods) == 0 {
		return nil, errors.New("the AI didn't see any food in the photo")
	}
	return foods, nil
}
