package safety

import "strings"

// packWords on the end of a product name only say how it's sold.
var packWords = map[string]bool{"dozen": true, "pack": true, "count": true, "ct": true, "bag": true, "box": true,
	"jar": true, "can": true, "bottle": true, "oz": true, "lb": true, "g": true, "kg": true, "ml": true, "l": true,
	"fl": true, "carton": true, "tub": true, "tube": true, "family": true, "size": true, "value": true}

// core is a food's identifying words: describing words and numbers left out
// (all of them kept when every word describes, like "salt").
func core(food string) []string {
	var all, plain []string
	for _, w := range words(food) {
		if isNumber(w) {
			continue
		}
		all = append(all, w)
		if !descriptors[w] {
			plain = append(plain, w)
		}
	}
	if len(plain) == 0 {
		return all
	}
	return plain
}

// SameFood reports whether a product in the house (its name) is the food a
// recipe line names: the food's words must end the product's name, and the
// word before them mustn't make it another food ("Brand Chicken Broth 32 oz"
// is chicken broth; "Peanut Butter" isn't butter, "Chicken Broth" isn't chicken).
func SameFood(food, product string) bool {
	f := core(food)
	if len(f) == 0 {
		return false
	}
	var n []string
	for _, w := range words(product) {
		if (w == "with" || w == "in") && len(n) > 0 {
			break // "Chicken Broth with Herbs", "Tuna in Water"
		}
		if !isNumber(w) {
			n = append(n, w)
		}
	}
	for len(n) > 0 && (packWords[n[len(n)-1]] || descriptors[n[len(n)-1]]) && !knownFoods[n[len(n)-1]] {
		n = n[:len(n)-1]
	}
	// Describing words inside the name don't count ("Chicken Broth, Low Sodium").
	var kept []string
	for _, w := range n {
		if !descriptors[w] || knownFoods[w] {
			kept = append(kept, w)
		}
	}
	n = kept
	if len(n) < len(f) || strings.Join(n[len(n)-len(f):], " ") != strings.Join(f, " ") {
		return false
	}
	if i := len(n) - len(f) - 1; i >= 0 {
		before := n[i]
		if knownFoods[before] || modifierFoods[before] {
			return false
		}
	}
	return true
}

// modifierFoods turn the next word into a different food ("peanut butter",
// "corn syrup", "rice vinegar").
var modifierFoods = map[string]bool{"peanut": true, "almond": true, "apple": true, "cocoa": true, "shea": true,
	"coconut": true, "soy": true, "oat": true, "rice": true, "cashew": true, "sunflower": true, "corn": true,
	"maple": true, "garlic": true, "onion": true, "chili": true, "sesame": true, "olive": true, "nut": true,
	"chicken": true, "beef": true, "vegetable": true, "fish": true, "tomato": true, "cream": true, "milk": true}

// FoodKey is a food's identifying words, so "2 large eggs" and "1 egg" end
// up on one shopping line.
func FoodKey(food string) string { return strings.Join(core(food), " ") }
