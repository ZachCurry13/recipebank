package recipe

import (
	"bytes"
	"encoding/json"
	"html"
	"regexp"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"
)

// FromHTML reads the schema.org Recipe most recipe sites embed as JSON-LD
// (<script type="application/ld+json">). ok is false when the page has none.
func FromHTML(page []byte) (r Recipe, ok bool) {
	for _, block := range ldBlocks(page) {
		var v any
		if json.Unmarshal([]byte(block), &v) != nil {
			continue
		}
		if node := findRecipe(v); node != nil {
			return fromNode(node), true
		}
	}
	return Recipe{}, false
}

// ldBlocks returns the text of every JSON-LD script on the page.
func ldBlocks(page []byte) []string {
	doc, err := xhtml.Parse(bytes.NewReader(page))
	if err != nil {
		return nil
	}
	var out []string
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && n.Data == "script" {
			for _, a := range n.Attr {
				if a.Key == "type" && strings.Contains(strings.ToLower(a.Val), "ld+json") && n.FirstChild != nil {
					out = append(out, n.FirstChild.Data)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

// findRecipe finds the Recipe object in a JSON-LD value: the object itself,
// an array of objects, or an @graph.
func findRecipe(v any) map[string]any {
	switch t := v.(type) {
	case []any:
		for _, e := range t {
			if r := findRecipe(e); r != nil {
				return r
			}
		}
	case map[string]any:
		if isType(t["@type"], "Recipe") {
			return t
		}
		if g, ok := t["@graph"]; ok {
			return findRecipe(g)
		}
		if e, ok := t["mainEntity"]; ok {
			return findRecipe(e)
		}
	}
	return nil
}

func isType(v any, want string) bool {
	switch t := v.(type) {
	case string:
		return strings.EqualFold(t, want)
	case []any:
		for _, e := range t {
			if isType(e, want) {
				return true
			}
		}
	}
	return false
}

func fromNode(n map[string]any) Recipe {
	r := Recipe{
		Area: AreaKitchen, SourceKind: "web", Heat: -1,
		Title:    clean(str(n["name"])),
		Summary:  clean(str(n["description"])),
		PrepMin:  Minutes(str(n["prepTime"])),
		CookMin:  Minutes(str(n["cookTime"])),
		TotalMin: Minutes(str(n["totalTime"])),
		Cuisine:  clean(first(n["recipeCuisine"])),
		Course:   strings.ToLower(clean(first(n["recipeCategory"]))),
		ImageURL: image(n["image"]),
	}
	r.Servings, r.YieldText = servings(n["recipeYield"])
	for _, line := range list(n["recipeIngredient"]) {
		if line = clean(line); line != "" {
			r.Ingredients = append(r.Ingredients, ParseLine(line))
		}
	}
	r.Steps = steps(n["recipeInstructions"], "")
	return r
}

// steps flattens recipeInstructions: a string, strings, HowToSteps, or
// HowToSections holding steps.
func steps(v any, section string) []Step {
	var out []Step
	switch t := v.(type) {
	case string:
		for _, line := range strings.Split(clean(t), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				out = append(out, Step{Text: line, Section: section})
			}
		}
	case []any:
		for _, e := range t {
			out = append(out, steps(e, section)...)
		}
	case map[string]any:
		if isType(t["@type"], "HowToSection") {
			return steps(t["itemListElement"], clean(str(t["name"])))
		}
		text := clean(str(t["text"]))
		if text == "" {
			text = clean(str(t["name"]))
		}
		if text != "" {
			out = append(out, Step{Text: text, Section: section})
		}
	}
	return out
}

var numRE = regexp.MustCompile(`\d+(\.\d+)?`)

func servings(v any) (float64, string) {
	var text string
	switch t := v.(type) {
	case float64:
		return t, ""
	case []any:
		for _, e := range t {
			if s := strings.TrimSpace(str(e)); s != "" && text == "" {
				text = s
			}
			if s := str(e); !strings.ContainsAny(s, " ") && numRE.MatchString(s) {
				n, _ := strconv.ParseFloat(numRE.FindString(s), 64)
				return n, text
			}
		}
	default:
		text = str(v)
	}
	text = clean(text)
	n, _ := strconv.ParseFloat(numRE.FindString(text), 64)
	if strings.TrimSpace(numRE.ReplaceAllString(text, "")) == "" || strings.Contains(strings.ToLower(text), "serv") {
		return n, ""
	}
	return n, text
}

var durRE = regexp.MustCompile(`(?i)^P(?:(\d+)D)?(?:T(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:\d+(?:\.\d+)?S)?)?$`)

// Minutes reads an ISO 8601 duration ("PT1H30M") as whole minutes.
func Minutes(iso string) int {
	m := durRE.FindStringSubmatch(strings.TrimSpace(iso))
	if m == nil {
		return 0
	}
	d, _ := strconv.Atoi(m[1])
	h, _ := strconv.ParseFloat(m[2], 64)
	mins, _ := strconv.ParseFloat(m[3], 64)
	return d*1440 + int(h*60+mins+0.5)
}

func image(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		if len(t) > 0 {
			return image(t[0])
		}
	case map[string]any:
		return str(t["url"])
	}
	return ""
}

func list(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			out = append(out, str(e))
		}
		return out
	}
	return nil
}

func first(v any) string {
	if l := list(v); len(l) > 0 {
		return l[0]
	}
	return ""
}

func str(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	}
	return ""
}

var tagRE = regexp.MustCompile(`<[^>]*>`)

// clean drops HTML tags and entities and tidies spaces.
func clean(s string) string {
	s = html.UnescapeString(tagRE.ReplaceAllString(s, " "))
	s = strings.ReplaceAll(s, " ", " ")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.Join(strings.Fields(l), " ")
	}
	return strings.TrimSpace(strings.Join(nonEmpty(lines), "\n"))
}
