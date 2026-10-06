package api

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/foodfacts"
	"github.com/zachcurry13/recipebank/internal/push"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

var dateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func (s *Server) handleListStock(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListStock(r.URL.Query().Get("area"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// handleSaveStock adds (POST) or changes (PUT) an item in the pantry or closet.
func (s *Server) handleSaveStock(w http.ResponseWriter, r *http.Request) {
	var it store.StockItem
	if !readJSON(w, r, &it, 64<<10) {
		return
	}
	it.ID = 0
	if id, ok := pathID(r, "id"); ok {
		it.ID = id
	}
	if msg := checkStock(&it); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	id, err := s.Store.SaveStock(&it)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	saved, err := s.Store.StockItem(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func checkStock(it *store.StockItem) string {
	it.Name = strings.TrimSpace(it.Name)
	switch {
	case it.Name == "" || len(it.Name) > 120:
		return "a name (up to 120 letters) is needed"
	case it.UseBy != "" && !dateRE.MatchString(it.UseBy):
		return "use-by must be a date"
	case it.Barcode != "" && !foodfacts.ValidBarcode(it.Barcode):
		return "a barcode is 8, 12, 13 or 14 digits"
	case it.LabelSource != "" && it.LabelSource != "off" && it.LabelSource != "parent":
		return "unknown label source"
	}
	for _, list := range [][]string{it.Allergens, it.Traces} {
		for _, k := range list {
			if _, ok := safety.AllergenByKey(k); !ok {
				return "unknown allergen " + k
			}
		}
	}
	return ""
}

// handleAdjustStock changes an amount ({"delta": -1}); anyone may, so a kid
// can mark the toothpaste used up.
func (s *Server) handleAdjustStock(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var body struct {
		Delta float64 `json:"delta"`
	}
	if !ok || !readJSON(w, r, &body, 1<<10) {
		return
	}
	before, err := s.Store.StockItem(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	it, err := s.Store.AdjustStock(id, body.Delta)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	// Just ran low: it goes on the shopping list by itself.
	if it.Low() && !before.Low() {
		_, _ = s.addLow(it, auth.UserFrom(r).Username)
		page, what := "/#/pantry", "🥫 Running low"
		if it.Area == "home" {
			page, what = "/#/supplies", "🧽 Running low"
		}
		s.Push.Notify("low", 0, push.Message{Title: what, Body: it.Name + " is running low. It's on the shopping list.", URL: page, Tag: "low"})

	}
	writeJSON(w, http.StatusOK, it)
}

func (s *Server) handleDeleteStock(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Store.DeleteStock(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleLookupBarcode finds a barcode: an item already in the house, else
// the open product databases (?barcode=…&area=kitchen|home).
func (s *Server) handleLookupBarcode(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.URL.Query().Get("barcode"))
	area := r.URL.Query().Get("area")
	if area != "home" {
		area = "kitchen"
	}
	if !foodfacts.ValidBarcode(code) {
		writeErr(w, http.StatusBadRequest, "a barcode is 8, 12, 13 or 14 digits")
		return
	}
	if it, err := s.Store.StockByBarcode(area, code); err == nil {
		writeJSON(w, http.StatusOK, map[string]any{"existing": it})
		return
	}
	bases := s.Products.FoodBases
	if area == "home" {
		bases = s.Products.HomeBases
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	p, err := foodfacts.Client{HTTP: s.Fetch, Bases: bases}.Lookup(ctx, code)
	if errors.Is(err, foodfacts.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "That barcode isn't in the open product databases yet. Type the name instead.")
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": p})
}

// pantryHint is a pantry product whose label can settle a "not sure" line.
type pantryHint struct {
	Ingredient int      `json:"ingredient"`
	StockID    int64    `json:"stock_id"`
	Name       string   `json:"name"`
	Brand      string   `json:"brand"`
	Allergens  []string `json:"allergens"`
	Traces     []string `json:"traces"`
}

// pantryHints matches the lines someone isn't sure about to labelled products in the pantry.
func (s *Server) pantryHints(rc *recipe.Recipe, verdicts []safety.Verdict) []pantryHint {
	out := []pantryHint{}
	unsure := map[int]bool{}
	for _, v := range verdicts {
		for _, x := range v.Reasons {
			if x.Status == safety.Unsure && x.Allergen != "" {
				for _, i := range x.Ingredients {
					unsure[i] = true
				}
			}
		}
	}
	if len(unsure) == 0 {
		return out
	}
	items, err := s.Store.ListStock("kitchen")
	if err != nil {
		return out
	}
	for i, in := range rc.Ingredients {
		if !unsure[i] {
			continue
		}
		for _, it := range items {
			if it.LabelSource != "" && safety.SameFood(in.Food, it.Name) {
				out = append(out, pantryHint{Ingredient: i, StockID: it.ID, Name: it.Name, Brand: it.Brand,
					Allergens: it.Allergens, Traces: it.Traces})
				break
			}
		}
	}
	return out
}
