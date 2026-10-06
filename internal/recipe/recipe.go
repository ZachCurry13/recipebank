// Package recipe holds RecipeBank's recipe type and turns outside material
// (web pages, lines of text, the AI's JSON) into it. It has no database or
// network code of its own.
package recipe

import "strings"

// Areas: food recipes, and Home & Care (cleaners, toothpaste, mouthwash…).
const (
	AreaKitchen = "kitchen"
	AreaHome    = "home"
)

// Recipe is one recipe, saved or a draft from an import.
type Recipe struct {
	ID           int64        `db:"id" json:"id"`
	Area         string       `db:"area" json:"area"`
	Title        string       `db:"title" json:"title"`
	Summary      string       `db:"summary" json:"summary"`
	Servings     float64      `db:"servings" json:"servings"`
	YieldText    string       `db:"yield_text" json:"yield_text"`
	PrepMin      int          `db:"prep_min" json:"prep_min"`
	CookMin      int          `db:"cook_min" json:"cook_min"`
	TotalMin     int          `db:"total_min" json:"total_min"`
	Heat         int          `db:"heat" json:"heat"` // 0-5; -1 = not set
	Course       string       `db:"course" json:"course"`
	Cuisine      string       `db:"cuisine" json:"cuisine"`
	Protein      string       `db:"protein" json:"protein"`
	Ingredients  []Ingredient `db:"-" json:"ingredients"`
	Steps        []Step       `db:"-" json:"steps"`
	Notes        string       `db:"notes" json:"notes"`
	Storage      string       `db:"storage" json:"storage"`
	SourceKind   string       `db:"source_kind" json:"source_kind"`
	SourceURL    string       `db:"source_url" json:"source_url"`
	SourceNote   string       `db:"source_note" json:"source_note"`
	Photo        string       `db:"photo" json:"photo"`
	SourcePhotos []string     `db:"-" json:"source_photos"`
	NeedsReview  bool         `db:"needs_review" json:"needs_review"`
	ReadBy       string       `db:"read_by" json:"read_by"`       // the AI model that read the photos
	AIReading    string       `db:"ai_reading" json:"ai_reading"` // what it copied from them, as plain text
	Difficulty   string       `db:"difficulty" json:"difficulty"` // easy, medium, hard; "" = worked out (Level)
	Rating       int          `db:"rating" json:"rating"`
	VersionOf    *int64       `db:"version_of" json:"version_of"`
	CreatedBy    string       `db:"created_by" json:"created_by"`
	CreatedAt    string       `db:"created_at" json:"created_at"`
	UpdatedAt    string       `db:"updated_at" json:"updated_at"`

	ImageURL string `db:"-" json:"image_url,omitempty"` // a web import's photo, fetched when saved
}

// Ingredient is one ingredient line, as written plus what was understood.
type Ingredient struct {
	Line    string   `json:"line"`
	Qty     *float64 `json:"qty,omitempty"`
	QtyMax  *float64 `json:"qty_max,omitempty"` // "2-3 cups"
	Unit    string   `json:"unit,omitempty"`    // canonical: cup, tbsp, tsp, g, ml, oz…
	Food    string   `json:"food,omitempty"`
	Note    string   `json:"note,omitempty"` // "chopped", "to taste"
	Section string   `json:"section,omitempty"`
	Unsure  bool     `json:"unsure,omitempty"`  // a photo line the AI couldn't read clearly
	Checked []string `json:"checked,omitempty"` // allergens a parent ruled out on the label ("*" = all)
}

// Step is one instruction.
type Step struct {
	Text    string `json:"text"`
	Section string `json:"section,omitempty"`
	Unsure  bool   `json:"unsure,omitempty"`
}

// Clean trims and fills the fields every recipe must have, and parses any
// ingredient line that hasn't been understood yet.
func (r *Recipe) Clean() {
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" {
		r.Title = "Untitled recipe"
	}
	if r.Area != AreaHome {
		r.Area = AreaKitchen
	}
	if r.Heat < -1 || r.Heat > 5 {
		r.Heat = -1
	}
	if r.Rating < 0 || r.Rating > 5 {
		r.Rating = 0
	}
	if r.TotalMin == 0 && (r.PrepMin > 0 || r.CookMin > 0) {
		r.TotalMin = r.PrepMin + r.CookMin
	}
	switch r.Difficulty {
	case Easy, Medium, Hard:
	default:
		r.Difficulty = ""
	}
	switch r.SourceKind {
	case "web", "photo", "text", "manual", "ai", "app": // "ai": drafted by the AI from a photo of a dish; "app": another app's export
	default:
		r.SourceKind = "manual"
	}
	ings := r.Ingredients[:0]
	for _, in := range r.Ingredients {
		in.Line = strings.TrimSpace(in.Line)
		if in.Line == "" && in.Food == "" {
			continue
		}
		if in.Line == "" {
			in.Line = in.Food
		}
		if in.Food == "" {
			p := ParseLine(in.Line)
			p.Section, p.Unsure, p.Checked = in.Section, in.Unsure, in.Checked
			in = p
		}
		ings = append(ings, in)
	}
	r.Ingredients = ings
	steps := r.Steps[:0]
	for _, st := range r.Steps {
		if st.Text = strings.TrimSpace(st.Text); st.Text != "" {
			steps = append(steps, st)
		}
	}
	r.Steps = steps
	if r.SourcePhotos == nil {
		r.SourcePhotos = []string{}
	}
	if r.Ingredients == nil {
		r.Ingredients = []Ingredient{}
	}
	if r.Steps == nil {
		r.Steps = []Step{}
	}
}

// Text is everything searchable about a recipe, lower-cased.
func (r *Recipe) Text() string {
	var b strings.Builder
	for _, s := range []string{r.Title, r.Summary, r.Course, r.Cuisine, r.Protein, r.Notes, r.SourceNote} {
		b.WriteString(s)
		b.WriteByte(' ')
	}
	for _, in := range r.Ingredients {
		b.WriteString(in.Line)
		b.WriteByte(' ')
	}
	return strings.ToLower(b.String())
}
