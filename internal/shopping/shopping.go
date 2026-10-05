// Package shopping combines amounts for the shopping list and sorts what to
// buy into store sections. No database or network code.
package shopping

import (
	"math"
	"strings"
)

// toTsp and toG convert volumes (in US teaspoons, so cups and spoons add up
// exactly) and weights to one base unit.
var (
	toTsp = map[string]float64{"tsp": 1, "tbsp": 3, "fl oz": 6, "cup": 48, "pint": 96, "quart": 192, "gallon": 768,
		"ml": 1 / 4.92892, "l": 1000 / 4.92892}
	toG = map[string]float64{"oz": 28.3495, "lb": 453.592, "g": 1, "kg": 1000}
)

// Add combines two amounts of the same food into the first one's unit. ok is
// false when they can't be added (cups and grams, cans and cloves).
func Add(q1 float64, u1 string, q2 float64, u2 string) (float64, bool) {
	switch {
	case q2 == 0:
		return q1, true // "salt to taste" adds nothing to count
	case q1 == 0 && u1 == "":
		return q2, false // the first had no amount: keep them apart
	case u1 == u2:
		return q1 + q2, true
	case toTsp[u1] > 0 && toTsp[u2] > 0:
		return round(q1 + q2*toTsp[u2]/toTsp[u1]), true
	case toG[u1] > 0 && toG[u2] > 0:
		return round(q1 + q2*toG[u2]/toG[u1]), true
	}
	return 0, false
}

// round keeps amounts tidy (4 decimal places).
func round(q float64) float64 { return math.Round(q*10000) / 10000 }

// Sections, in the order a shop is walked.
var Sections = []string{"Produce", "Bakery", "Meat & fish", "Dairy & eggs", "Frozen", "Pantry", "Spices & baking",
	"Drinks", "Household", "Personal care", "Other"}

var sectionWords = []struct {
	section string
	words   []string
}{
	{"Frozen", []string{"frozen", "ice cream"}},
	{"Personal care", []string{"toothpaste", "mouthwash", "floss", "shampoo", "conditioner", "deodorant", "lotion",
		"soap bar", "body wash", "razor", "toothbrush", "sunscreen", "tissue", "cotton"}},
	{"Household", []string{"bleach", "detergent", "dish soap", "dishwasher", "laundry", "sponge", "paper towel",
		"toilet paper", "trash bag", "cleaner", "castile soap", "borax", "washing soda", "hydrogen peroxide",
		"rubbing alcohol", "isopropyl", "spray bottle", "foil", "plastic wrap", "parchment", "zip"}},
	// Shelf goods before meat: "chicken broth" is on the soup aisle.
	{"Pantry", []string{"broth", "stock", "bouillon", "soup", "canned", "sauce", "tomato paste", "curry paste"}},
	{"Spices & baking", []string{"salt", "pepper", "cinnamon", "nutmeg", "paprika", "cumin", "oregano", "chili powder",
		"baking soda", "baking powder", "yeast", "vanilla", "cocoa", "sugar", "flour", "cornstarch", "spice", "seasoning",
		"extract", "chocolate chip", "sprinkle", "powdered sugar", "brown sugar", "essential oil", "xylitol"}},
	{"Dairy & eggs", []string{"milk", "butter", "cheese", "yogurt", "cream", "egg", "sour cream", "buttermilk"}},
	{"Meat & fish", []string{"chicken", "beef", "pork", "turkey", "lamb", "bacon", "sausage", "ham", "fish", "salmon",
		"tuna", "cod", "shrimp", "steak", "ground meat"}},
	{"Bakery", []string{"bread", "bun", "roll", "tortilla", "bagel", "pita", "croissant", "baguette", "naan"}},
	{"Drinks", []string{"juice", "soda", "coffee", "tea", "wine", "beer", "sparkling", "water bottle"}},
	{"Produce", []string{"onion", "garlic", "tomato", "lettuce", "spinach", "kale", "carrot", "celery", "potato",
		"pepper bell", "bell pepper", "cucumber", "zucchini", "squash", "broccoli", "cauliflower", "cabbage", "mushroom",
		"apple", "banana", "lemon", "lime", "orange", "berry", "strawberr", "blueberr", "grape", "avocado", "herb",
		"parsley", "cilantro", "basil", "mint", "dill", "ginger", "scallion", "green onion", "leek", "jalapeno",
		"jalapeño", "peach", "pear", "mango", "pineapple", "corn on the cob", "green bean", "asparagus", "fruit"}},
	{"Pantry", []string{"rice", "pasta", "noodle", "oil", "vinegar", "broth", "stock", "can", "canned", "bean",
		"lentil", "chickpea", "sauce", "ketchup", "mustard", "mayonnaise", "honey", "syrup", "jam", "peanut butter",
		"oat", "cereal", "cracker", "nut", "raisin", "coconut milk", "tomato paste", "salsa", "soup"}},
}

// Section picks the store section for something to buy. Home & Care things
// that aren't personal care go under Household.
func Section(name, area string) string {
	n := strings.ToLower(name)
	for _, s := range sectionWords {
		for _, w := range s.words {
			if strings.Contains(n, w) {
				if area == "home" && s.section != "Personal care" && s.section != "Household" {
					return "Household"
				}
				return s.section
			}
		}
	}
	if area == "home" {
		return "Household"
	}
	return "Other"
}

// trailing phrases say how a recipe uses something, not what to buy.
var trailing = []string{" plus ", " for ", " to serve", " to taste", " to garnish", " as needed", " if needed",
	" (optional)", " optional", ", divided", " divided"}

// CleanName is what to buy from an ingredient's food: "lemon wedges to serve"
// → "lemon wedges", "oil plus a little extra for frying" → "oil".
func CleanName(food string) string {
	name := strings.TrimSpace(food)
	low := strings.ToLower(name)
	cut := len(name)
	for _, t := range trailing {
		if i := strings.Index(low, t); i > 0 && i < cut {
			cut = i
		}
	}
	if out := strings.TrimSpace(strings.TrimRight(name[:cut], ",;")); out != "" {
		return out
	}
	return name
}
