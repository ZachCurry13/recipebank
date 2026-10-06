package recipe

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	ingHead  = regexp.MustCompile(`(?i)^\s*(ingredients?|you will need|what you need)\s*:?\s*$`)
	stepHead = regexp.MustCompile(`(?i)^\s*(directions?|instructions?|method|steps|preparation|how to make( it)?)\s*:?\s*$`)
	noteHead = regexp.MustCompile(`(?i)^\s*(notes?|tips?|storage)\s*:?\s*$`)
	stepNum  = regexp.MustCompile(`^\s*(step\s*)?\d+[.):]\s*`)
	servRE   = regexp.MustCompile(`(?i)\s*\(\s*(?:serves\s*)?(\d+)\s*(?:servings?|serv\.?|portions?)?\s*\)\s*$`)
	unsureRE = regexp.MustCompile(`\s*\[\?\]`)
	bullet   = regexp.MustCompile(`^\s*[-•*▢□]\s*`)
)

// FromText reads a pasted recipe without the AI: the first line is the
// title, lines under "Ingredients" are ingredients, lines under
// "Directions"/"Method"/"Steps" are steps. ok is false without both headings.
func FromText(text string) (r Recipe, ok bool) {
	r = Recipe{SourceKind: "text", Heat: -1}
	part := "intro"
	var notes []string
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r", ""), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case line == "":
			continue
		case ingHead.MatchString(line):
			part = "ing"
			continue
		case stepHead.MatchString(line):
			part = "steps"
			continue
		case noteHead.MatchString(line):
			part = "notes"
			continue
		}
		// "[?]" marks words the reader couldn't make out.
		unsure := unsureRE.MatchString(line)
		line = strings.TrimSpace(unsureRE.ReplaceAllString(line, ""))
		r.NeedsReview = r.NeedsReview || unsure
		switch part {
		case "intro":
			if r.Title == "" {
				r.Title = line
				// "Unstuffed Peppers (4 serv.)": the servings go in their own field.
				if m := servRE.FindStringSubmatch(line); m != nil && strings.ContainsAny(line, "(") {
					if n, err := strconv.Atoi(m[1]); err == nil && n > 0 && n < 100 {
						r.Servings = float64(n)
						r.Title = strings.TrimSpace(servRE.ReplaceAllString(line, ""))
					}
				}
			} else if r.Summary == "" {
				r.Summary = line
			}
		case "ing":
			line = bullet.ReplaceAllString(line, "")
			if strings.HasSuffix(line, ":") && len(line) < 40 {
				continue // a sub-heading such as "For the sauce:"
			}
			in := ParseLine(line)
			in.Unsure = unsure
			r.Ingredients = append(r.Ingredients, in)
		case "steps":
			if line = stepNum.ReplaceAllString(bullet.ReplaceAllString(line, ""), ""); line != "" {
				r.Steps = append(r.Steps, Step{Text: line, Unsure: unsure})
			}
		case "notes":
			notes = append(notes, line)
		}
	}
	r.Notes = strings.Join(notes, "\n")
	return r, r.Title != "" && len(r.Ingredients) > 0 && len(r.Steps) > 0
}
