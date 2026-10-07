package safety

import (
	"fmt"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

// Severities of an allergy.
const (
	Avoid    = "avoid"    // a preference: only what the recipe clearly contains counts
	Allergic = "allergic" // also "may contain" and packaged foods: check the label
	Severe   = "severe"   // strict: every ingredient must be a food the lists know
)

// Verdict statuses.
const (
	OK     = "ok"
	No     = "no"
	Unsure = "unsure"
)

// Person is someone's rules, as the checker needs them.
type Person struct {
	ID            int64               `json:"id"`
	Name          string              `json:"name"`
	IsKid         bool                `json:"is_kid"`
	HeatMax       int                 `json:"heat_max"`          // -1 = no limit
	Allergies     map[string]string   `json:"allergies"`         // allergen key → severity
	Allowed       map[string][]string `json:"allowed,omitempty"` // allergen key → what they can have anyway (exceptions.go)
	Diets         []string            `json:"diets"`
	Dislikes      []string            `json:"dislikes"`
	Sensitivities []string            `json:"sensitivities"`
}

// Reason is one thing standing between a person and a recipe.
type Reason struct {
	Status      string `json:"status"`             // no or unsure
	Rule        string `json:"rule"`               // "Milk allergy", "Vegan", "Dislikes", "Heat"
	Text        string `json:"text"`               // "contains milk"
	Allergen    string `json:"allergen,omitempty"` // for "I checked the label"
	Ingredients []int  `json:"ingredients"`        // which lines
}

// Verdict is one person's answer for one recipe.
type Verdict struct {
	PersonID int64    `json:"person_id"`
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Reasons  []Reason `json:"reasons"`
}

// reasons collects reasons, one per rule and status, listing every line.
type reasons struct{ list []Reason }

func (rs *reasons) add(status, rule, text, allergen string, line int) {
	for i := range rs.list {
		r := &rs.list[i]
		if r.Status == status && r.Rule == rule && r.Text == text {
			if line >= 0 && (len(r.Ingredients) == 0 || r.Ingredients[len(r.Ingredients)-1] != line) {
				r.Ingredients = append(r.Ingredients, line)
			}
			return
		}
	}
	r := Reason{Status: status, Rule: rule, Text: text, Allergen: allergen, Ingredients: []int{}}
	if line >= 0 {
		r.Ingredients = append(r.Ingredients, line)
	}
	rs.list = append(rs.list, r)
}

// Check answers "can p eat (or use) r?". It never says OK for a real
// allergy unless every line is accounted for.
func Check(r *recipe.Recipe, p Person) Verdict {
	var rs reasons
	var hasMeat, hasMilk []int
	dislikes := make([]phrases, len(p.Dislikes))
	for i, w := range p.Dislikes {
		dislikes[i] = compile(w)
	}
	sensitive := make([]phrases, len(p.Sensitivities))
	for i, s := range p.Sensitivities {
		sensitive[i] = sensitivityWords(s)
	}
	for i, in := range r.Ingredients {
		t := newText(in.Line + " \n " + in.Food)
		food := in.Food
		if food == "" {
			food = in.Line
		}
		for key, sev := range p.Allergies {
			checkAllergen(&rs, t, food, key, sev, p.Allowed[key], in.Checked, i)
		}
		for _, d := range p.Diets {
			checkDiet(&rs, t, d, i)
		}
		for j, w := range p.Dislikes {
			if t.has(dislikes[j]) {
				rs.add(No, "Dislikes", w, "", i)
			}
		}
		for j, s := range p.Sensitivities {
			if t.has(sensitive[j]) {
				rs.add(No, "Sensitive to", s, "", i)
			}
		}
		if slicesHas(p.Diets, "kosher") {
			m := t.clone()
			m.mask(meatNotWords)
			if m.has(meatWords) {
				hasMeat = append(hasMeat, i)
			}
			if isAllergen(t, "milk") {
				hasMilk = append(hasMilk, i)
			}
		}
	}
	if len(hasMeat) > 0 && len(hasMilk) > 0 {
		for _, i := range append(hasMeat, hasMilk...) {
			rs.add(No, "Kosher-style", "mixes meat and dairy", "", i)
		}
	}
	if p.HeatMax >= 0 {
		heat, at := r.Heat, -1
		if heat < 0 {
			heat, at = EstimateHeat(lines(r))
		}
		if heat > p.HeatMax {
			rs.add(No, "Heat", fmt.Sprintf("too spicy (%d of 5, likes up to %d)", heat, p.HeatMax), "", at)
		}
	}
	// "Stir in the butter" with no butter line: a line may be missing.
	for key, sev := range p.Allergies {
		if sev == Avoid {
			continue
		}
		if w := stepsMention(r, key, p.Allowed[key]); w != "" {
			a, _ := AllergenByKey(key)
			rs.add(Unsure, a.Label+" allergy", "the steps mention "+w+", but no ingredient line has it", "", -1)
		}
	}
	// A line read wrong (or missed) could hide anything: until someone checks
	// the recipe against the card, a real allergy can't be OK.
	if r.NeedsReview && realAllergy(p) {
		why := "the AI wasn't sure about some lines"
		if r.SourceKind == "photo" {
			why = "read from a photo; check it against the card first"
		}
		rs.add(Unsure, "Not checked yet", why, "", -1)
	}
	v := Verdict{PersonID: p.ID, Name: p.Name, Status: OK, Reasons: rs.list}
	for _, x := range rs.list {
		if x.Status == No {
			v.Status = No
		} else if v.Status == OK {
			v.Status = Unsure
		}
	}
	if v.Reasons == nil {
		v.Reasons = []Reason{}
	}
	return v
}

func checkAllergen(rs *reasons, t text, food, key, sev string, allowed, checked []string, line int) {
	a, ok := AllergenByKey(key)
	if !ok {
		return
	}
	rule := a.Label + " allergy"
	m := t.clone()
	m.mask(a.not)
	maskAllowed(m, t, a, allowed)
	if hit := m.first(a.terms); hit != "" {
		rs.add(No, rule, "contains "+strings.ToLower(a.Label), key, line)
		return
	}
	if sev == Avoid || slicesHas(checked, key) || slicesHas(checked, "*") {
		return
	}
	// "May contain" is looked for without what the person may have ("soy
	// lecithin" isn't a maybe for someone who can have it; "lecithin" still is).
	mt := t.clone()
	maskAllowed(mt, t, a, allowed)
	switch {
	case mt.has(a.maybe):
		rs.add(Unsure, rule, "may contain "+strings.ToLower(a.Label)+": check the label", key, line)
	case t.has(packaged):
		rs.add(Unsure, rule, "packaged food: check the label", key, line)
	case sev == Severe && !known(food) && !onlyAllowed(food, t, a, allowed):
		rs.add(Unsure, rule, "strict mode: not sure what's in it, check it", key, line)
	}
}

func checkDiet(rs *reasons, t text, key string, line int) {
	d, ok := DietByKey(key)
	if !ok {
		return
	}
	m := t.clone()
	m.mask(d.not)
	if hit := m.first(d.no); hit != "" {
		rs.add(No, d.Label, "has "+hit, "", line)
		return
	}
	for _, a := range d.allergens {
		if isAllergen(t, a) {
			al, _ := AllergenByKey(a)
			rs.add(No, d.Label, "has "+strings.ToLower(al.Label), "", line)
			return
		}
	}
	if hit := m.first(d.maybe); hit != "" {
		rs.add(Unsure, d.Label, "not sure about "+hit, "", line)
		return
	}
	for _, a := range d.allergens {
		if al, _ := AllergenByKey(a); t.has(al.maybe) {
			rs.add(Unsure, d.Label, "may have "+strings.ToLower(al.Label)+": check the label", "", line)
			return
		}
	}
}

// isAllergen reports whether t clearly contains allergen key.
func isAllergen(t text, key string) bool {
	a, ok := AllergenByKey(key)
	if !ok {
		return false
	}
	m := t.clone()
	m.mask(a.not)
	return m.has(a.terms)
}

// sensitivityWords widens a few common sensitivities to what they cover.
func sensitivityWords(s string) phrases {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "essential oil", "essential oils":
		return compile(essentialOils...)
	case "fragrance", "fragrances", "scent", "perfume":
		return compile(append([]string{"fragrance", "perfume", "scented"}, essentialOils...)...)
	case "alcohol":
		return compile("alcohol", "rubbing alcohol", "isopropyl", "ethanol", "vodka", "witch hazel")
	}
	return compile(s)
}

var meatWords, meatNotWords = compile(meat...), compile(meatNot...)

func lines(r *recipe.Recipe) []string {
	out := make([]string, len(r.Ingredients))
	for i, in := range r.Ingredients {
		out[i] = in.Line
	}
	return out
}

func slicesHas(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// realAllergy: p has an allergy that isn't just a preference.
func realAllergy(p Person) bool {
	for _, sev := range p.Allergies {
		if sev == Allergic || sev == Severe {
			return true
		}
	}
	return false
}
