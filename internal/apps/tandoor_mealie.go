package apps

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

// tandoorZip reads one recipe from Tandoor's export: recipe.json and image.*.
func tandoorZip(zr *zip.Reader) (Found, error) {
	var data, photo []byte
	for _, f := range zr.File {
		switch base := strings.ToLower(path.Base(f.Name)); {
		case base == "recipe.json":
			data, _ = readAll(f, maxJSON)
		case strings.HasPrefix(base, "image."):
			photo, _ = readAll(f, maxPhoto)
		}
	}
	if data == nil {
		return Found{}, errors.New("no recipe.json")
	}
	var t struct {
		Name         string  `json:"name"`
		Description  string  `json:"description"`
		Servings     float64 `json:"servings"`
		ServingsText string  `json:"servings_text"`
		WorkingTime  int     `json:"working_time"`
		WaitingTime  int     `json:"waiting_time"`
		SourceURL    string  `json:"source_url"`
		Keywords     []struct {
			Name string `json:"name"`
		} `json:"keywords"`
		Steps []struct {
			Name        string `json:"name"`
			Instruction string `json:"instruction"`
			Ingredients []struct {
				Food *struct {
					Name string `json:"name"`
				} `json:"food"`
				Unit *struct {
					Name string `json:"name"`
				} `json:"unit"`
				Amount       float64 `json:"amount"`
				Note         string  `json:"note"`
				OriginalText string  `json:"original_text"`
				IsHeader     bool    `json:"is_header"`
				NoAmount     bool    `json:"no_amount"`
			} `json:"ingredients"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(data, &t); err != nil || strings.TrimSpace(t.Name) == "" {
		return Found{}, errors.New("not a Tandoor recipe")
	}
	r := recipe.Recipe{Title: t.Name, Summary: t.Description, Servings: t.Servings, Heat: -1, PrepMin: t.WorkingTime,
		TotalMin: t.WorkingTime + t.WaitingTime, SourceURL: webAddress(t.SourceURL), SourceNote: "From Tandoor"}
	if t.ServingsText != "" && t.ServingsText != "servings" {
		r.YieldText = strings.TrimSpace(fmt.Sprintf("%g %s", t.Servings, t.ServingsText))
	}
	var kw []string
	for _, k := range t.Keywords {
		kw = append(kw, k.Name)
	}
	r.Course = courseOf(kw)
	section := ""
	for _, st := range t.Steps {
		for _, in := range st.Ingredients {
			food, unit := "", ""
			if in.Food != nil {
				food = in.Food.Name
			}
			if in.Unit != nil {
				unit = in.Unit.Name
			}
			if in.IsHeader {
				section = strings.TrimSpace(firstOf(in.Note, food, in.OriginalText))
				continue
			}
			line := strings.TrimSpace(in.OriginalText)
			if line == "" {
				amount := ""
				if !in.NoAmount && in.Amount > 0 {
					amount = strconv.FormatFloat(in.Amount, 'f', -1, 64)
				}
				line = joinWords(amount, unit, food) + comma(in.Note)
			}
			if line != "" {
				r.Ingredients = append(r.Ingredients, recipe.Ingredient{Line: line, Section: section})
			}
		}
		for _, s := range stepLines(st.Instruction) {
			s.Section = strings.TrimSpace(st.Name)
			r.Steps = append(r.Steps, s)
		}
	}
	return Found{Recipe: r, Photo: photo}, nil
}

// mealieBackup reads Mealie's backup: database.json's tables, photos under data/recipes/<id>/images.
func mealieBackup(data []byte, files map[string]*zip.File) ([]Found, error) {
	var db map[string][]map[string]any
	if err := json.Unmarshal(data, &db); err != nil || db["recipes"] == nil {
		return nil, errors.New("not a Mealie backup")
	}
	byID := func(table string) map[string]string {
		out := map[string]string{}
		for _, row := range db[table] {
			out[text(row["id"])] = text(row["name"])
		}
		return out
	}
	foods, units := byID("ingredient_foods"), byID("ingredient_units")
	ings, steps, notes := map[string][]map[string]any{}, map[string][]map[string]any{}, map[string][]map[string]any{}
	for _, row := range db["recipes_ingredients"] {
		ings[text(row["recipe_id"])] = append(ings[text(row["recipe_id"])], row)
	}
	for _, row := range db["recipe_instructions"] {
		steps[text(row["recipe_id"])] = append(steps[text(row["recipe_id"])], row)
	}
	for _, row := range db["notes"] {
		notes[text(row["recipe_id"])] = append(notes[text(row["recipe_id"])], row)
	}
	var out []Found
	for _, row := range db["recipes"] {
		id := text(row["id"])
		var list []any
		for _, in := range sortByPosition(ings[id]) {
			in["food"], in["unit"] = foods[text(in["food_id"])], units[text(in["unit_id"])]
			list = append(list, in)
		}
		row["recipe_ingredient"] = list
		var st []any
		for _, s := range sortByPosition(steps[id]) {
			st = append(st, s)
		}
		row["recipe_instructions"] = st
		var ns []any
		for _, n := range notes[id] {
			ns = append(ns, n)
		}
		row["notes"] = ns
		r, ok := genericRecipe(row)
		if !ok {
			continue
		}
		r.SourceNote = "From Mealie"
		f := Found{Recipe: r}
		for _, name := range []string{"original.webp", "original.jpg", "original.png", "min-original.webp"} {
			if zf := files["data/recipes/"+id+"/images/"+name]; zf != nil {
				f.Photo, _ = readAll(zf, maxPhoto)
				break
			}
		}
		out = append(out, f)
	}
	return out, nil
}

func sortByPosition(rows []map[string]any) []map[string]any {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && pos(rows[j]) < pos(rows[j-1]); j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
	return rows
}

func pos(m map[string]any) float64 {
	n, _ := m["position"].(float64)
	return n
}
