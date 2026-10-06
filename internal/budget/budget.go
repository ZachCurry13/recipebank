// Package budget estimates what the shopping and recipes cost from the
// prices the family gives pantry and supply items: a price for one item as
// counted ("$3.49") and, if known, its package size ("16 oz", "12 count").
package budget

import (
	"math"
	"strings"

	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

// size reads a package size: "16 oz" → 16 oz, "12 count" or "12" → 12 (a count).
func size(s string) (qty float64, unit string, ok bool) {
	in := recipe.ParseLine(s)
	if in.Qty == nil || *in.Qty <= 0 {
		return 0, "", false
	}
	return *in.Qty, in.Unit, true
}

// Buy is what buying enough for an amount costs, in whole packages: "2 cups
// flour" is one bag; "6 eggs" from a 12-count carton is one carton.
func Buy(it store.StockItem, qty float64, unit string) (float64, bool) {
	if it.Price <= 0 {
		return 0, false
	}
	pq, pu, sized := size(it.Size)
	switch {
	case qty <= 0:
		return it.Price, true
	case !sized && unit == "":
		return it.Price * math.Ceil(qty), true // priced each: "2 lemons"
	case sized && pu == "" && unit == "":
		return it.Price * math.Ceil(qty/pq), true
	case sized && pu != "":
		if n, ok := inUnit(it.Name, qty, unit, pu); ok {
			return it.Price * math.Ceil(n/pq-1e-9), true
		}
	}
	return it.Price, true // can't compare: one package
}

// Use is the share of the price a recipe uses: 2 cups of a 5 lb bag of
// flour. ok is false when the amounts can't be compared.
func Use(it store.StockItem, qty float64, unit string) (float64, bool) {
	if it.Price <= 0 || qty <= 0 {
		return 0, false
	}
	pq, pu, sized := size(it.Size)
	switch {
	case !sized && unit == "":
		return it.Price * qty, true
	case sized && pu == "" && unit == "":
		return it.Price * qty / pq, true
	case sized && pu != "":
		if n, ok := inUnit(it.Name, qty, unit, pu); ok {
			return it.Price * n / pq, true
		}
	}
	return 0, false
}

// Match finds the priced pantry or supply item that is food, if any.
func Match(stock []store.StockItem, food string) *store.StockItem {
	for i := range stock {
		if stock[i].Price > 0 && (safety.Covers(stock[i].Name, food) || safety.SameFood(food, stock[i].Name)) {
			return &stock[i]
		}
	}
	return nil
}

// Cost is an estimate: the total of the lines that could be priced.
type Cost struct {
	Total  float64 `json:"total"`
	Priced int     `json:"priced"`
	Lines  int     `json:"lines"` // lines that count (salt, pepper and water don't)
}

// Recipe estimates what making rc (times factor) costs.
func Recipe(rc *recipe.Recipe, stock []store.StockItem, factor float64) Cost {
	var c Cost
	for _, in := range rc.Ingredients {
		if in.Food == "" || safety.Staple(in.Food) {
			continue
		}
		c.Lines++
		it := Match(stock, in.Food)
		if it == nil || in.Qty == nil {
			continue
		}
		q := *in.Qty
		if in.QtyMax != nil {
			q = (q + *in.QtyMax) / 2
		}
		if v, ok := Use(*it, q*factor, in.Unit); ok {
			c.Total += v
			c.Priced++
		}
	}
	return c
}

// Shopping prices each line still to buy (by id) and adds them up.
func Shopping(items []store.ShopItem, stock []store.StockItem) (map[int64]float64, Cost) {
	costs := map[int64]float64{}
	var c Cost
	for _, sh := range items {
		if sh.Checked {
			continue
		}
		c.Lines++
		var it *store.StockItem
		for i := range stock {
			if sh.StockID != nil && stock[i].ID == *sh.StockID && stock[i].Price > 0 {
				it = &stock[i]
			}
		}
		if it == nil {
			it = Match(stock, sh.Name)
		}
		if it == nil {
			continue
		}
		if v, ok := Buy(*it, sh.Qty, sh.Unit); ok {
			costs[sh.ID] = v
			c.Total += v
			c.Priced++
		}
	}
	return costs, c
}

// cupGrams lets cups of foods sold by weight be priced: a cup of flour is
// 125 g. The first name an item's name contains wins (look-alikes first).
var cupGrams = []struct {
	name string
	g    float64
}{{"buttermilk", 0}, {"vinegar", 0}, {"milk", 0}, {"flour", 125}, {"powdered sugar", 120}, {"brown sugar", 213},
	{"sugar", 200}, {"butter", 227}, {"oats", 90}, {"rice", 185}, {"cocoa", 85}, {"chocolate chips", 170}, {"cornstarch", 128}}

// inUnit converts a recipe amount to the package's unit, weighing cups of
// foods sold by weight.
func inUnit(name string, qty float64, unit, want string) (float64, bool) {
	if n, ok := recipe.Convert(qty, unit, want); ok {
		return n, true
	}
	cups, ok := recipe.Convert(qty, unit, "cup")
	if !ok {
		return 0, false
	}
	low := strings.ToLower(name)
	for _, c := range cupGrams {
		if strings.Contains(low, c.name) {
			if c.g == 0 {
				return 0, false
			}
			return recipe.Convert(cups*c.g, "g", want)
		}
	}
	return 0, false
}
