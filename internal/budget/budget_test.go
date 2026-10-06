package budget

import (
	"math"
	"testing"

	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/store"
)

func near(a, b float64) bool { return math.Abs(a-b) < 0.01 }

var (
	flour  = store.StockItem{ID: 1, Name: "All-purpose flour", Price: 4.00, Size: "5 lb"}
	eggs   = store.StockItem{ID: 2, Name: "Eggs", Price: 3.60, Size: "12 count"}
	lemons = store.StockItem{ID: 3, Name: "Lemons", Price: 0.50}
	milk   = store.StockItem{ID: 4, Name: "Whole milk", Price: 4.00, Size: "1 gallon"}
	cheap  = store.StockItem{ID: 5, Name: "Salt", Price: 1.00}
	free   = store.StockItem{ID: 6, Name: "Cumin"}
)

func TestBuyAndUse(t *testing.T) {
	for _, c := range []struct {
		name     string
		it       store.StockItem
		qty      float64
		unit     string
		buy, use float64
		useKnown bool
	}{
		{"2 cups flour: a bag; 250 g of 5 lb", flour, 2, "cup", 4.00, 0.44, true},
		{"2 lb flour", flour, 2, "lb", 4.00, 1.60, true},
		{"11 lb flour: three bags", flour, 11, "lb", 12.00, 8.80, true},
		{"6 eggs: a carton; half", eggs, 6, "", 3.60, 1.80, true},
		{"18 eggs: two cartons", eggs, 18, "", 7.20, 5.40, true},
		{"2 lemons, priced each", lemons, 2, "", 1.00, 1.00, true},
		{"2 cups milk", milk, 2, "cup", 4.00, 0.50, true},
		{"milk, no amount: one", milk, 0, "", 4.00, 0, false},
	} {
		if b, ok := Buy(c.it, c.qty, c.unit); !ok || !near(b, c.buy) {
			t.Errorf("%s: buy %.2f (%v), want %.2f", c.name, b, ok, c.buy)
		}
		if u, ok := Use(c.it, c.qty, c.unit); ok != c.useKnown || !near(u, c.use) {
			t.Errorf("%s: use %.2f (%v), want %.2f (%v)", c.name, u, ok, c.use, c.useKnown)
		}
	}
	if _, ok := Buy(free, 1, "tsp"); ok {
		t.Error("an item without a price was priced")
	}
}

func TestRecipeAndShopping(t *testing.T) {
	stock := []store.StockItem{flour, eggs, lemons, milk, cheap, free}
	rc := &recipe.Recipe{}
	for _, l := range []string{"1 lb flour", "3 eggs", "2 cups milk", "1 tsp cumin", "salt to taste", "1 cup sugar"} {
		rc.Ingredients = append(rc.Ingredients, recipe.ParseLine(l))
	}
	c := Recipe(rc, stock, 2) // doubled
	// flour 2 lb = 1.60, eggs 6 = 1.80, milk 4 cups = 1.00; cumin has no price, sugar isn't in the pantry, salt doesn't count.
	if !near(c.Total, 4.40) || c.Priced != 3 || c.Lines != 5 {
		t.Fatalf("recipe: %+v", c)
	}
	id := int64(2)
	costs, sc := Shopping([]store.ShopItem{{ID: 10, Name: "flour", Qty: 3, Unit: "cup"}, {ID: 11, Name: "eggs", Qty: 6, StockID: &id},
		{ID: 12, Name: "bread"}, {ID: 13, Name: "lemons", Qty: 3, Checked: true}}, stock)
	if !near(sc.Total, 7.60) || sc.Priced != 2 || sc.Lines != 3 || !near(costs[11], 3.60) || costs[12] != 0 {
		t.Fatalf("shopping: %+v %v", sc, costs)
	}
}
