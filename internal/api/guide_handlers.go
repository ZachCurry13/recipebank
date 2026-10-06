package api

import (
	"net/http"
	"slices"
	"strings"

	"github.com/zachcurry13/recipebank/internal/store"
)

// The admin's "Getting started" checklist. Most steps tick themselves off;
// the AI, sign-ins and phones can also be ticked (or skipped) by hand.
var guideSteps = []string{"admin", "family", "recipe", "ai", "users", "phones"}

type guideStep struct {
	Key  string `json:"key"`
	Done bool   `json:"done"`
}

// guideState is what's stored: steps ticked by hand, and "hidden".
func (s *Server) guideState() []string {
	return store.SplitList(s.Store.Setting(store.KeyGuide))
}

func (s *Server) handleGuide(w http.ResponseWriter, r *http.Request) {
	manual := s.guideState()
	people, err := s.Store.ListPeople()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	users, err := s.Store.CountUsers()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	recipes, err := s.Store.CountRecipes()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	auto := map[string]bool{"admin": true, "family": len(people) > 0, "recipe": recipes > 0,
		"ai": s.Store.AnyAI(), "users": users > 1}
	steps := []guideStep{}
	for _, k := range guideSteps {
		steps = append(steps, guideStep{Key: k, Done: auto[k] || slices.Contains(manual, k)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"steps": steps, "hidden": slices.Contains(manual, "hidden")})
}

// handleGuideUpdate ticks a step by hand ({"step": "phones", "done": true})
// or hides and shows the checklist ({"step": "hidden", "done": true}).
func (s *Server) handleGuideUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Step string `json:"step"`
		Done bool   `json:"done"`
	}
	if !readJSON(w, r, &body, 1<<10) {
		return
	}
	if body.Step != "hidden" && !slices.Contains(guideSteps, body.Step) {
		writeErr(w, http.StatusBadRequest, "unknown step")
		return
	}
	manual := slices.DeleteFunc(s.guideState(), func(k string) bool { return k == body.Step })
	if body.Done {
		manual = append(manual, body.Step)
	}
	if err := s.Store.SetSetting(store.KeyGuide, strings.Join(manual, ",")); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.handleGuide(w, r)
}
