package api

import (
	"net/http"
	"sort"
	"strings"

	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/safety"
)

// substituteIdea is one stand-in for an ingredient someone doesn't have.
type substituteIdea struct {
	To   string `json:"to"`
	Note string `json:"note"`
	Have bool   `json:"have"`  // everything it needs is in the pantry (or a staple)
	ByAI bool   `json:"by_ai"` // an AI idea, still checked by the word lists
}

// handleSubstitute lists what can stand in for one ingredient line
// ({"line": 3, "ai": true}, ?who= the people eating): the substitution table,
// the allergy swaps and, when asked, the AI. Ideas that don't suit everyone
// eating are left out and counted; ones already in the pantry come first.
func (s *Server) handleSubstitute(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var body struct {
		Line int      `json:"line"`
		AI   bool     `json:"ai"`
		Have []string `json:"have"` // foods ticked or typed on "What can I make?"
	}
	if !ok || !readJSON(w, r, &body, 32<<10) {
		return
	}
	rc, err := s.Store.Recipe(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	if body.Line < 0 || body.Line >= len(rc.Ingredients) {
		writeErr(w, http.StatusBadRequest, "no such ingredient")
		return
	}
	in := rc.Ingredients[body.Line]
	food := in.Food
	if food == "" {
		food = in.Line
	}
	diners, err := s.diners(r)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	stock, err := s.Store.ListStock(rc.Area)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	onHand := cleanHave(body.Have)
	for _, it := range stock {
		if it.Qty > 0 {
			onHand = append(onHand, it.Name)
		}
	}
	var ideas []substituteIdea
	seen := map[string]bool{}
	add := func(to, note string, byAI bool) {
		if k := strings.ToLower(to); !seen[k] && !safety.SameFood(food, to) {
			seen[k] = true
			ideas = append(ideas, substituteIdea{To: to, Note: note, ByAI: byAI})
		}
	}
	for _, i := range safety.Substitutes(food) {
		add(i.To, i.Note, false)
	}
	aiErr := ""
	if body.AI {
		err := llm.Ask(r.Context(), s.Store, llm.SubstituteSystem, llm.SubstitutePrompt(rc.Title, in.Line, onHand), func(out string) error {
			more, perr := llm.ParseSubstitutes(out)
			for _, m := range more {
				add(m.To, m.Note, true)
			}
			return perr
		})
		if err != nil {
			aiErr = importErr(err)
		}
	}
	out, skipped := []substituteIdea{}, 0
	for _, i := range ideas {
		if !safety.OKForAll(rc.Area, i.To, diners) {
			skipped++
			continue
		}
		i.Have = haveAll(onHand, i.To)
		out = append(out, i)
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Have && !out[b].Have })
	writeJSON(w, http.StatusOK, map[string]any{"food": food, "ideas": out, "skipped": skipped,
		"ai_error": aiErr, "ai_ready": s.Store.AnyAI()})
}

// haveAll: every food an idea needs ("milk and lemon juice") is on hand or a staple.
func haveAll(onHand []string, to string) bool {
	name := strings.SplitN(to, " (", 2)[0]
	for _, part := range strings.FieldsFunc(strings.ReplaceAll(name, " and ", ","), func(c rune) bool { return c == ',' }) {
		if part = strings.TrimSpace(part); part != "" && !safety.Staple(part) && !covered(onHand, part) {
			return false
		}
	}
	return true
}
