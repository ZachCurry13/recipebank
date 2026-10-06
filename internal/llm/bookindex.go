package llm

import (
	"encoding/json"
	"errors"
	"strings"
)

// IndexPrompt asks the vision model for the recipes in a cookbook's index or
// table of contents, by page.
const IndexPrompt = `This photo shows a page of a cookbook's index or table of contents. List each recipe on it
with its page number. Reply with JSON only:
{"recipes": [{"title": "recipe name as printed", "page": "112"}]}
- Only recipes: leave out chapter names, ingredient headings that group recipes, and anything you can't read.
- page: the page number as printed ("112", or "112-113"); "" if none is shown.
If the photo isn't an index or contents page, reply {"recipes": []}.`

// IndexEntry is one recipe on an index page.
type IndexEntry struct {
	Title string `json:"title"`
	Page  string `json:"page"`
}

// ParseIndex reads the answer: at most 300 recipes, each with a name.
func ParseIndex(out string) ([]IndexEntry, error) {
	var a struct {
		Recipes []IndexEntry `json:"recipes"`
	}
	if err := json.Unmarshal([]byte(jsonObject(out)), &a); err != nil {
		return nil, errors.New("the AI's answer wasn't valid JSON")
	}
	var list []IndexEntry
	for _, e := range a.Recipes {
		e.Title, e.Page = strings.TrimSpace(clip(e.Title, 200)), strings.TrimSpace(clip(e.Page, 12))
		if e.Title == "" {
			continue
		}
		if list = append(list, e); len(list) == 300 {
			break
		}
	}
	if len(list) == 0 {
		return nil, errors.New("no recipes could be read from the page: try a flatter, closer photo")
	}
	return list, nil
}
