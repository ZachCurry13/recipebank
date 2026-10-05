package shopping

import (
	"math"
	"testing"
)

func TestAdd(t *testing.T) {
	near := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }
	if q, ok := Add(2, "cup", 1, "cup"); !ok || q != 3 {
		t.Errorf("cups: %v %v", q, ok)
	}
	if q, ok := Add(1, "cup", 4, "tbsp"); !ok || !near(q, 1.25) {
		t.Errorf("cup + tbsp: %v %v", q, ok)
	}
	if q, ok := Add(1, "lb", 8, "oz"); !ok || !near(q, 1.5) {
		t.Errorf("lb + oz: %v %v", q, ok)
	}
	if _, ok := Add(2, "cup", 100, "g"); ok {
		t.Error("cups and grams were added")
	}
	if q, ok := Add(3, "", 2, ""); !ok || q != 5 {
		t.Errorf("eggs: %v %v", q, ok)
	}
	if q, ok := Add(2, "cup", 0, ""); !ok || q != 2 {
		t.Errorf("'to taste' changed the amount: %v %v", q, ok)
	}
}

func TestSection(t *testing.T) {
	cases := map[[2]string]string{
		{"red onion", "kitchen"}: "Produce", {"whole milk", "kitchen"}: "Dairy & eggs", {"chicken thighs", "kitchen"}: "Meat & fish",
		{"all-purpose flour", "kitchen"}: "Spices & baking", {"chicken broth", "kitchen"}: "Pantry",
		{"mint toothpaste", "home"}: "Personal care", {"laundry detergent", "home"}: "Household",
		{"white vinegar", "home"}: "Household", {"frozen peas", "kitchen"}: "Frozen", {"mystery item", "kitchen"}: "Other",
	}
	for in, want := range cases {
		if got := Section(in[0], in[1]); got != want {
			t.Errorf("Section(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}

func TestCleanName(t *testing.T) {
	for in, want := range map[string]string{
		"lemon wedges to serve": "lemon wedges",
		"sunflower or vegetable oil plus a little extra for frying": "sunflower or vegetable oil",
		"salt and pepper to taste":                                  "salt and pepper",
		"butter, divided":                                           "butter",
		"flour":                                                     "flour",
	} {
		if got := CleanName(in); got != want {
			t.Errorf("CleanName(%q) = %q, want %q", in, got, want)
		}
	}
}
