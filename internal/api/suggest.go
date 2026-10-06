package api

import (
	"context"
	"net/http"
	"sort"

	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/search"
)

type pick struct {
	recipeCard
	Why string `json:"why"`
}

type leftOut struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

type suggestion struct {
	Picks   []pick       `json:"picks"`
	LeftOut []leftOut    `json:"left_out"`
	Rules   search.Rules `json:"rules"`
	UsedAI  bool         `json:"used_ai"`
	AIError string       `json:"ai_error,omitempty"`
}

// maxForAI keeps the list small enough for a home AI's working memory.
const maxForAI = 60

// suggest finds recipes in area that fit a plain-words request. The AI
// (when set up) picks by meaning; without it, recipes sharing the request's
// words are ranked. Either way the sure rules (diets, time, everyone eating)
// have the last word, and what they drop is listed with the reason.
func (s *Server) suggest(ctx context.Context, r *http.Request, area, request string, skip map[int64]bool) (suggestion, error) {
	out := suggestion{Picks: []pick{}, LeftOut: []leftOut{}, Rules: search.Parse(request)}
	all, err := s.Store.ListRecipes(area)
	if err != nil {
		return out, err
	}
	words := safety.NewWords(search.Keywords(request), nil)
	type scored struct {
		rc    *recipe.Recipe
		score int
	}
	var cands []scored
	for i := range all {
		if !skip[all[i].ID] {
			cands = append(cands, scored{&all[i], words.Count(all[i].Text())})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })

	byID := map[int64]*recipe.Recipe{}
	var ids []int64
	why := map[int64]string{}
	if s.Store.AIConfig().Ready() {
		allowed := map[int64]bool{}
		var lines []string
		for i, c := range cands {
			if i == maxForAI {
				break
			}
			allowed[c.rc.ID] = true
			byID[c.rc.ID] = c.rc
			lines = append(lines, llm.RecipeLine(c.rc))
		}
		err := llm.Ask(ctx, s.Store, llm.PickSystem, llm.PickPrompt(request, lines), func(ans string) (perr error) {
			ids, why, perr = llm.ParsePicks(ans, allowed)
			return perr
		})
		if err == nil {
			out.UsedAI = true
		} else {
			out.AIError = err.Error()
			ids = nil
		}
	}
	if !out.UsedAI {
		// Recipes sharing the request's words; when none do but the request
		// has rules ("quick dairy-free"), the rules alone choose.
		rules := len(out.Rules.Diets) > 0 || out.Rules.MaxMin > 0 || out.Rules.Everyone
		for _, c := range cands {
			byID[c.rc.ID] = c.rc
			if c.score > 0 {
				ids = append(ids, c.rc.ID)
			}
		}
		if len(ids) == 0 && rules {
			for _, c := range cands {
				ids = append(ids, c.rc.ID)
			}
		}
	}

	var everyone []safety.Person
	if out.Rules.Everyone {
		if everyone, err = s.diners(r); err != nil {
			return out, err
		}
	}
	diners, err := s.diners(r)
	if err != nil {
		return out, err
	}
	for _, id := range ids {
		if len(out.Picks) == 20 {
			break
		}
		rc := byID[id]
		if ok, reason := keepIf(rc, out.Rules.Diets, out.Rules.MaxMin, everyone); !ok {
			out.LeftOut = append(out.LeftOut, leftOut{rc.ID, rc.Title, reason})
			continue
		}
		card, _ := cardFor(rc, diners)
		out.Picks = append(out.Picks, pick{card, why[id]})
	}
	return out, nil
}

// handleCollectionSuggest suggests recipes for a collection from its
// description; a parent picks which to add.
func (s *Server) handleCollectionSuggest(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	c, err := s.Store.Collection(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	have, err := s.Store.CollectionRecipeIDs(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	skip := map[int64]bool{}
	for _, h := range have {
		skip[h] = true
	}
	request := c.Name
	if c.Description != "" {
		request = c.Name + ": " + c.Description
	}
	sug, err := s.suggest(r.Context(), r, c.Area, request, skip)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sug)
}

// handleSearch answers a plain-words question about the recipes
// ({"q": "something quick with chicken everyone can eat", "area": "kitchen"},
// ?who= the people eating).
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Q    string `json:"q"`
		Area string `json:"area"`
	}
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	if len(body.Q) < 3 || len(body.Q) > 300 {
		writeErr(w, http.StatusBadRequest, "ask in a few words")
		return
	}
	if body.Area != "home" {
		body.Area = "kitchen"
	}
	sug, err := s.suggest(r.Context(), r, body.Area, body.Q, nil)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sug)
}
