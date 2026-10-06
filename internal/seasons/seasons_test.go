package seasons

import (
	"testing"
	"time"

	"github.com/zachcurry13/recipebank/internal/recipe"
)

func TestCalendar(t *testing.T) {
	if e := Easter(2027); e.Month() != time.March || e.Day() != 28 {
		t.Errorf("Easter 2027: %v", e)
	}
	if a := AdventSunday(2026); a.Month() != time.November || a.Day() != 29 {
		t.Errorf("Advent 2026: %v", a)
	}
	if th := Thanksgiving(2026); th.Month() != time.November || th.Day() != 26 {
		t.Errorf("Thanksgiving 2026: %v", th)
	}
	lent, _ := Find("lent")
	if !lent.In(time.Date(2027, 3, 12, 15, 0, 0, 0, time.UTC)) || lent.In(time.Date(2027, 3, 30, 0, 0, 0, 0, time.UTC)) {
		t.Error("Lent 2027 (Feb 10 - Mar 25) is wrong")
	}
	xmas, _ := Find("christmas")
	if !xmas.In(time.Date(2027, 1, 3, 0, 0, 0, 0, time.UTC)) {
		t.Error("Christmas should run into January")
	}
}

func TestFits(t *testing.T) {
	cookies := &recipe.Recipe{Area: "kitchen", Title: "Grandma's Gingerbread Cookies", Ingredients: []recipe.Ingredient{recipe.ParseLine("3 cups flour")}}
	advent, _ := Find("advent")
	if !advent.Fits(cookies) {
		t.Error("gingerbread cookies aren't Advent baking")
	}
	lent, _ := Find("lent")
	fish := &recipe.Recipe{Area: "kitchen", Title: "Baked cod", Ingredients: []recipe.Ingredient{recipe.ParseLine("1 lb cod"), recipe.ParseLine("1 lemon")}}
	beef := &recipe.Recipe{Area: "kitchen", Title: "Beef stew", Ingredients: []recipe.Ingredient{recipe.ParseLine("2 lb beef chuck")}}
	if !lent.Fits(fish) || lent.Fits(beef) {
		t.Error("Lent Fridays: cod in, beef out")
	}
	grill, _ := Find("grilling")
	if grill.Fits(&recipe.Recipe{Area: "kitchen", Title: "Grilled peaches"}) == false {
		t.Error("grilled peaches belong to summer grilling")
	}
}
