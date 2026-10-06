package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// DishPrompt asks the vision model what a photographed meal most likely is.
const DishPrompt = `This photo shows a meal or a dish. Say what it most likely is. Reply with JSON only:
{"name": "the dish's usual name", "description": "one short sentence", "ingredients": ["main foods you can see, or that it usually has"], "cuisine": ""}
If the photo doesn't show food, reply {"name": ""}.`

// Dish is what the AI thinks a photographed meal is.
type Dish struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Ingredients []string `json:"ingredients"`
	Cuisine     string   `json:"cuisine"`
}

// ParseDish reads the answer, keeping it short.
func ParseDish(out string) (Dish, error) {
	var d Dish
	if err := json.Unmarshal([]byte(jsonObject(out)), &d); err != nil {
		return Dish{}, errors.New("the AI's answer wasn't valid JSON")
	}
	return d.Clean()
}

// Clean trims a dish to short, sensible lengths; it's an error without a name.
func (d Dish) Clean() (Dish, error) {
	d.Name, d.Description, d.Cuisine = clip(d.Name, 80), clip(d.Description, 240), clip(d.Cuisine, 40)
	if d.Name == "" {
		return Dish{}, errors.New("the AI didn't see a dish in the photo")
	}
	var ings []string
	for _, i := range d.Ingredients {
		if i = clip(i, 40); i != "" && len(ings) < 15 {
			ings = append(ings, i)
		}
	}
	d.Ingredients = ings
	return d, nil
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		s = strings.TrimSpace(s[:n])
	}
	return s
}

// DishDraftSystem and DishDraftPrompt ask for a recipe for a described dish,
// written as plain text that the usual reader turns into a recipe.
const DishDraftSystem = "You write clear, simple home recipes."

func DishDraftPrompt(d Dish) string {
	p := fmt.Sprintf("Write a simple home recipe for %q", d.Name)
	if d.Description != "" {
		p += " (" + d.Description + ")"
	}
	if len(d.Ingredients) > 0 {
		p += ". It likely has: " + strings.Join(d.Ingredients, ", ")
	}
	return p + ".\nUse common US measures. Reply in plain text exactly like this:\n" +
		"Title (4 serv.)\nIngredients:\none ingredient per line, with its amount\nInstructions:\none step per line"
}
