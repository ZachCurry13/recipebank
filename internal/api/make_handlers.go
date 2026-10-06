package api

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/shopping"
	"github.com/zachcurry13/recipebank/internal/store"
)

// makeResult is one recipe "What can I make?" found, with what's missing.
type makeResult struct {
	Card     recipeCard     `json:"card"`
	Have     int            `json:"have"`
	Total    int            `json:"total"`
	Missing  []string       `json:"missing"`
	UsesSoon []string       `json:"uses_soon"` // pantry food near its use-by date this recipe uses up
	Swaps    []missingSwaps `json:"swaps"`
}

type missingSwaps struct {
	Missing string            `json:"missing"`
	Use     []safety.SwapIdea `json:"use"`
}

// handleMake finds kitchen recipes for what's on hand ({"have": ["chicken",
// "rice"], "pantry": true}, ?who= the people eating). Recipes needing least
// come first, then those using up food near its date. Swaps are offered only
// when they're on hand and suit everyone eating.
func (s *Server) handleMake(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Have   []string `json:"have"`
		Pantry bool     `json:"pantry"`
	}
	if !readJSON(w, r, &body, 32<<10) {
		return
	}
	onHand := []string{}
	for _, h := range body.Have {
		if h = strings.TrimSpace(h); h != "" && len(h) <= 80 {
			onHand = append(onHand, h)
		}
	}
	stock, err := s.Store.ListStock("kitchen")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	var soon []store.StockItem
	limit := time.Now().AddDate(0, 0, 3).Format(day)
	for _, it := range stock {
		if it.Qty <= 0 {
			continue
		}
		if body.Pantry {
			onHand = append(onHand, it.Name)
		}
		if it.UseBy != "" && it.UseBy <= limit {
			soon = append(soon, it)
		}
	}
	if len(onHand) == 0 {
		writeErr(w, http.StatusBadRequest, "tick or type what you have first")
		return
	}
	diners, err := s.diners(r)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	all, err := s.Store.ListRecipes("kitchen")
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	results := []makeResult{}
	for i := range all {
		if res, ok := s.makeOne(&all[i], onHand, soon, diners); ok {
			results = append(results, res)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if len(a.Missing) != len(b.Missing) {
			return len(a.Missing) < len(b.Missing)
		}
		if len(a.UsesSoon) != len(b.UsesSoon) {
			return len(a.UsesSoon) > len(b.UsesSoon)
		}
		return a.Have*b.Total > b.Have*a.Total
	})
	if len(results) > 40 {
		results = results[:40]
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results, "on_hand": len(onHand)})
}

// makeOne measures one recipe against what's on hand; ok is false when none
// of its (non-staple) ingredients are.
func (s *Server) makeOne(rc *recipe.Recipe, onHand []string, soon []store.StockItem, diners []safety.Person) (makeResult, bool) {
	res := makeResult{Missing: []string{}, UsesSoon: []string{}, Swaps: []missingSwaps{}}
	matched := 0
	for _, in := range rc.Ingredients {
		food := in.Food
		if food == "" || strings.Contains(strings.ToLower(in.Note+" "+in.Line), "optional") {
			continue
		}
		res.Total++
		if safety.Staple(food) {
			res.Have++
			continue
		}
		if covered(onHand, food) {
			res.Have++
			matched++
			for _, it := range soon {
				if safety.Covers(it.Name, food) && !contains(res.UsesSoon, it.Name) {
					res.UsesSoon = append(res.UsesSoon, it.Name)
				}
			}
			continue
		}
		name := shopping.CleanName(food)
		res.Missing = append(res.Missing, name)
		var use []safety.SwapIdea
		for _, idea := range safety.Substitutes(food) {
			if covered(onHand, strings.SplitN(idea.To, " (", 2)[0]) && safety.OKForAll(rc.Area, idea.To, diners) {
				use = append(use, idea)
			}
		}
		if len(use) > 0 {
			res.Swaps = append(res.Swaps, missingSwaps{Missing: name, Use: use})
		}
	}
	if matched == 0 {
		return res, false
	}
	res.Card, _ = cardFor(rc, diners)
	return res, true
}

func covered(onHand []string, food string) bool {
	for _, h := range onHand {
		if safety.Covers(h, food) {
			return true
		}
	}
	return false
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
