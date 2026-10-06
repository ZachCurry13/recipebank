package llm

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

const recipeJSON = `Reply with JSON only, in this shape:
{"title": "", "summary": "one short sentence", "area": "kitchen or home",
 "servings": 0, "yield": "", "prep_min": 0, "cook_min": 0, "total_min": 0,
 "course": "", "cuisine": "", "protein": "", "heat": -1,
 "ingredients": [{"line": "the ingredient line exactly as written", "section": "", "unsure": false}],
 "steps": [{"text": "", "section": "", "unsure": false}],
 "notes": "", "storage": ""}

Rules:
- Copy ingredient lines and steps as written, with their amounts. Never add, drop or guess ingredients.
- "area" is "home" for household and personal-care recipes (cleaners, laundry, soap, toothpaste,
  mouthwash, lotion, deodorant), otherwise "kitchen".
- "course": for kitchen recipes one of breakfast, main, side, soup, salad, appetizer, snack, dessert,
  baking, bread, drink, sauce; for home recipes one of cleaning, laundry, oral care, personal care, other.
- "protein": the main protein (chicken, beef, pork, fish, seafood, turkey, lamb, eggs, beans, tofu,
  cheese) or "" if none. "cuisine": e.g. Italian, Mexican, American, or "".
- "heat": 0 (no heat) to 5 (extreme), from the chilies used; -1 if you can't tell.
- "section" is the heading of a part only when the recipe itself has headings for its parts; otherwise "".
  Never make up headings.
- "storage": how to store it and how long it keeps, if the recipe says.
- Use 0 or "" for anything not given.`

// TextSystem is the system prompt for pulling a recipe out of pasted text or a web page.
const TextSystem = `You turn text into a structured recipe for a family recipe app.
If the text has no recipe, reply {"title": ""}.
` + recipeJSON

type aiRecipe struct {
	Title       string  `json:"title"`
	Summary     string  `json:"summary"`
	Area        string  `json:"area"`
	Servings    float64 `json:"servings"`
	Yield       string  `json:"yield"`
	PrepMin     int     `json:"prep_min"`
	CookMin     int     `json:"cook_min"`
	TotalMin    int     `json:"total_min"`
	Course      string  `json:"course"`
	Cuisine     string  `json:"cuisine"`
	Protein     string  `json:"protein"`
	Heat        *int    `json:"heat"`
	Ingredients []struct {
		Line    string `json:"line"`
		Section string `json:"section"`
		Unsure  bool   `json:"unsure"`
	} `json:"ingredients"`
	Steps []struct {
		Text    string `json:"text"`
		Section string `json:"section"`
		Unsure  bool   `json:"unsure"`
	} `json:"steps"`
	Notes   string `json:"notes"`
	Storage string `json:"storage"`
}

// ErrNoRecipe means the AI found no recipe in what it was given.
var ErrNoRecipe = errors.New("no recipe found there")

// ParseRecipe reads the AI's JSON into a recipe draft.
func ParseRecipe(out string) (recipe.Recipe, error) {
	var a aiRecipe
	if err := json.Unmarshal([]byte(jsonObject(out)), &a); err != nil {
		return recipe.Recipe{}, errors.New("the AI's answer wasn't valid JSON")
	}
	if strings.TrimSpace(a.Title) == "" {
		return recipe.Recipe{}, ErrNoRecipe
	}
	if len(a.Ingredients) == 0 && len(a.Steps) == 0 {
		return recipe.Recipe{}, errors.New("the AI found a title but no ingredients or steps")
	}
	r := recipe.Recipe{Title: a.Title, Summary: a.Summary, Area: a.Area, Servings: max(0, a.Servings),
		YieldText: a.Yield, PrepMin: max(0, a.PrepMin), CookMin: max(0, a.CookMin), TotalMin: max(0, a.TotalMin),
		Course: strings.ToLower(strings.TrimSpace(a.Course)), Cuisine: a.Cuisine,
		Protein: strings.ToLower(strings.TrimSpace(a.Protein)), Heat: -1, Notes: a.Notes, Storage: a.Storage}
	if a.Heat != nil && *a.Heat >= 0 && *a.Heat <= 5 {
		r.Heat = *a.Heat
	}
	for _, in := range a.Ingredients {
		p := recipe.ParseLine(in.Line)
		p.Section, p.Unsure = in.Section, in.Unsure
		r.NeedsReview = r.NeedsReview || in.Unsure
		r.Ingredients = append(r.Ingredients, p)
	}
	for _, st := range a.Steps {
		r.Steps = append(r.Steps, recipe.Step{Text: st.Text, Section: st.Section, Unsure: st.Unsure})
		r.NeedsReview = r.NeedsReview || st.Unsure
	}
	r.Clean()
	return r, nil
}

// jsonObject cuts the JSON object out of an answer that may be wrapped in
// ``` fences or a sentence.
func jsonObject(s string) string {
	i, j := strings.Index(s, "{"), strings.LastIndex(s, "}")
	if i < 0 || j < i {
		return s
	}
	return s[i : j+1]
}
