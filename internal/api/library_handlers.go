package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
)

// recipeCard is a recipe in the Library: the facts on its card and each
// diner's verdict.
type recipeCard struct {
	ID          int64    `json:"id"`
	Area        string   `json:"area"`
	Title       string   `json:"title"`
	Photo       string   `json:"photo"`
	TotalMin    int      `json:"total_min"`
	Heat        int      `json:"heat"`
	Course      string   `json:"course"`
	Cuisine     string   `json:"cuisine"`
	Protein     string   `json:"protein"`
	Rating      int      `json:"rating"`
	NeedsReview bool     `json:"needs_review"`
	Verdicts    []status `json:"verdicts"`
}

type status struct {
	PersonID int64  `json:"person_id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
}

// handleListRecipes lists an area's recipes, filtered, with verdicts for
// the chosen diners. ?ok=1 keeps only recipes OK for all of them
// (?ok=unsure also keeps "not sure").
func (s *Server) handleListRecipes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	all, err := s.Store.ListRecipes(q.Get("area"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	diners, err := s.diners(r)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	search := strings.ToLower(strings.TrimSpace(q.Get("q")))
	maxMin, _ := strconv.Atoi(q.Get("max_min"))
	maxHeat, heatErr := strconv.Atoi(q.Get("heat"))
	diet := q.Get("diet")
	cards := []recipeCard{}
	facets := map[string]map[string]int{"course": {}, "cuisine": {}, "protein": {}}
	for i := range all {
		rc := &all[i]
		heat := heatOf(rc)
		if search != "" && !strings.Contains(rc.Text(), search) ||
			maxMin > 0 && (rc.TotalMin == 0 || rc.TotalMin > maxMin) ||
			heatErr == nil && heat > maxHeat ||
			!matches(q.Get("course"), rc.Course) || !matches(q.Get("cuisine"), rc.Cuisine) || !matches(q.Get("protein"), rc.Protein) {
			continue
		}
		if diet != "" && safety.Check(rc, safety.Person{HeatMax: -1, Diets: []string{diet}}).Status != safety.OK {
			continue
		}
		card, worst := cardFor(rc, diners)
		if ok := q.Get("ok"); ok == "1" && worst != safety.OK || ok == "unsure" && worst == safety.No {
			continue
		}
		cards = append(cards, card)
		for k, v := range map[string]string{"course": rc.Course, "cuisine": rc.Cuisine, "protein": rc.Protein} {
			if v != "" {
				facets[k][strings.ToLower(v)]++
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"recipes": cards, "facets": facets})
}

// cardFor is a recipe's Library card, and the worst verdict among diners.
func cardFor(rc *recipe.Recipe, diners []safety.Person) (recipeCard, string) {
	card := recipeCard{ID: rc.ID, Area: rc.Area, Title: rc.Title, Photo: rc.Photo, TotalMin: rc.TotalMin, Heat: heatOf(rc),
		Course: rc.Course, Cuisine: rc.Cuisine, Protein: rc.Protein, Rating: rc.Rating, NeedsReview: rc.NeedsReview,
		Verdicts: []status{}}
	worst := safety.OK
	for _, p := range diners {
		v := safety.Check(rc, p)
		card.Verdicts = append(card.Verdicts, status{p.ID, p.Name, v.Status})
		worst = worse(worst, v.Status)
	}
	return card, worst
}

func matches(want, have string) bool { return want == "" || strings.EqualFold(want, have) }

func worse(a, b string) string {
	rank := map[string]int{safety.OK: 0, safety.Unsure: 1, safety.No: 2}
	if rank[b] > rank[a] {
		return b
	}
	return a
}
