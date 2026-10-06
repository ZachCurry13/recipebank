package apps

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

// paprikaRecipe reads Paprika's recipe JSON.
func paprikaRecipe(data []byte) (Found, error) {
	var p struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Ingredients string   `json:"ingredients"`
		Directions  string   `json:"directions"`
		Notes       string   `json:"notes"`
		Servings    string   `json:"servings"`
		PrepTime    string   `json:"prep_time"`
		CookTime    string   `json:"cook_time"`
		TotalTime   string   `json:"total_time"`
		Source      string   `json:"source"`
		SourceURL   string   `json:"source_url"`
		Categories  []string `json:"categories"`
		Rating      int      `json:"rating"`
		PhotoData   string   `json:"photo_data"`
	}
	if err := json.Unmarshal(data, &p); err != nil || strings.TrimSpace(p.Name) == "" {
		return Found{}, errors.New("not a Paprika recipe")
	}
	r := recipe.Recipe{Title: p.Name, Summary: p.Description, Notes: p.Notes, Rating: p.Rating, Heat: -1,
		PrepMin: minutesOf(p.PrepTime), CookMin: minutesOf(p.CookTime), TotalMin: minutesOf(p.TotalTime),
		SourceURL: webAddress(p.SourceURL), SourceNote: "From Paprika" + by(p.Source), Course: courseOf(p.Categories)}
	r.Servings, r.YieldText = servings(p.Servings)
	r.Ingredients = ingredientLines(strings.Split(p.Ingredients, "\n"))
	r.Steps = stepLines(p.Directions)
	f := Found{Recipe: r}
	if p.PhotoData != "" {
		if img, err := base64.StdEncoding.DecodeString(p.PhotoData); err == nil && len(img) <= maxPhoto {
			f.Photo = img
		}
	}
	return f, nil
}

// genericRecipe reads a Mealie or schema.org recipe (camelCase or snake_case keys).
func genericRecipe(m map[string]any) (recipe.Recipe, bool) {
	get := func(keys ...string) any {
		for _, k := range keys {
			if v, ok := m[k]; ok && v != nil {
				return v
			}
		}
		return nil
	}
	title := text(get("name", "title"))
	ings := get("recipeIngredient", "recipe_ingredient", "ingredients")
	if title == "" || ings == nil {
		return recipe.Recipe{}, false
	}
	r := recipe.Recipe{Title: title, Summary: text(get("description")), Heat: -1, SourceNote: "Imported",
		PrepMin: minutesOf(get("prepTime", "prep_time")), CookMin: minutesOf(get("performTime", "perform_time", "cookTime", "cook_time")),
		TotalMin: minutesOf(get("totalTime", "total_time")), SourceURL: webAddress(text(get("orgURL", "org_url", "url", "source_url")))}
	r.Servings, r.YieldText = servings(get("recipeServings", "recipe_servings", "recipeYield", "recipe_yield", "servings"))
	if n, ok := get("rating").(float64); ok && n >= 0 && n <= 5 {
		r.Rating = int(n + 0.5)
	}
	r.Course = courseOf(names(get("recipeCategory", "recipe_category")))
	switch v := ings.(type) {
	case string:
		r.Ingredients = ingredientLines(strings.Split(v, "\n"))
	case []any:
		section := ""
		for _, it := range v {
			switch in := it.(type) {
			case string:
				r.Ingredients = append(r.Ingredients, recipe.Ingredient{Line: in, Section: section})
			case map[string]any:
				if t := text(in["title"]); t != "" {
					section = t
				}
				line := firstOf(text(in["originalText"]), text(in["original_text"]), text(in["display"]))
				if line == "" {
					qty := ""
					if q, ok := in["quantity"].(float64); ok && q > 0 {
						qty = strconv.FormatFloat(q, 'f', -1, 64)
					}
					line = joinWords(qty, nameOf(in["unit"]), nameOf(in["food"])) + comma(text(in["note"]))
				}
				if line != "" {
					r.Ingredients = append(r.Ingredients, recipe.Ingredient{Line: line, Section: section})
				}
			}
		}
	}
	r.Steps = instructions(get("recipeInstructions", "recipe_instructions", "instructions"), "")
	r.Notes = notesOf(get("notes"))
	return r, true
}

// instructions reads steps given as text, a list of texts, or HowToStep and
// HowToSection objects ({"text"}, {"title"/"name", "itemListElement"}).
func instructions(v any, section string) []recipe.Step {
	var out []recipe.Step
	switch t := v.(type) {
	case string:
		for _, s := range stepLines(t) {
			s.Section = section
			out = append(out, s)
		}
	case []any:
		for _, it := range t {
			if m, ok := it.(map[string]any); ok {
				if sub, ok := m["itemListElement"]; ok {
					out = append(out, instructions(sub, firstOf(text(m["name"]), text(m["title"])))...)
					continue
				}
				if title := text(m["title"]); title != "" && section == "" {
					out = append(out, instructions(m["text"], title)...)
					continue
				}
				out = append(out, instructions(m["text"], section)...)
				continue
			}
			out = append(out, instructions(it, section)...)
		}
	}
	return out
}
