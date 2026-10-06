package nutrition

import "strings"

// FoodData Central's search ranks oddly for one-word foods ("flour" finds
// arrowroot flour first), so everyday foods are searched by the name USDA
// gives the plain kind, and the best of the first results is picked.
var plainNames = map[string]string{
	"flour": "wheat flour white all-purpose enriched bleached", "all purpose flour": "wheat flour white all-purpose enriched bleached",
	"plain flour": "wheat flour white all-purpose enriched bleached", "bread flour": "wheat flour white bread enriched",
	"whole wheat flour": "wheat flour whole-grain", "sugar": "sugars granulated", "granulated sugar": "sugars granulated",
	"white sugar": "sugars granulated", "brown sugar": "sugars brown", "powdered sugar": "sugars powdered",
	"confectioners sugar": "sugars powdered", "milk": "milk whole 3.25% milkfat", "whole milk": "milk whole 3.25% milkfat",
	"butter": "butter salted", "unsalted butter": "butter without salt", "egg": "egg whole raw fresh",
	"salt": "salt table", "kosher salt": "salt table", "sea salt": "salt table", "pepper": "spices pepper black",
	"black pepper": "spices pepper black", "olive oil": "oil olive salad or cooking", "vegetable oil": "oil soybean salad or cooking",
	"canola oil": "oil canola", "water": "water tap drinking", "baking powder": "leavening agents baking powder double-acting",
	"baking soda": "leavening agents baking soda", "heavy cream": "cream fluid heavy whipping", "sour cream": "cream sour cultured",
	"cheddar": "cheese cheddar", "cheddar cheese": "cheese cheddar", "rice": "rice white long-grain regular raw enriched",
	"white rice": "rice white long-grain regular raw enriched", "onion": "onions raw", "garlic": "garlic raw",
	"ground beef": "beef ground 80% lean meat 20% fat raw", "chicken breast": "chicken broilers or fryers breast meat only raw",
	"vanilla": "vanilla extract", "vanilla extract": "vanilla extract", "cinnamon": "spices cinnamon ground",
	"cumin": "spices cumin seed", "honey": "honey", "potato": "potatoes flesh and skin raw", "carrot": "carrots raw",
	"tomato": "tomatoes red ripe raw year round average", "lemon juice": "lemon juice raw", "parmesan": "cheese parmesan grated",
	"mozzarella": "cheese mozzarella whole milk", "spinach": "spinach raw", "cornstarch": "cornstarch", "yeast": "leavening agents yeast baker's active dry",
}

// searchTerms is what to ask FoodData Central for a recipe's food.
func searchTerms(food string) string {
	f := strings.ToLower(strings.TrimSpace(food))
	if q, ok := plainNames[f]; ok {
		return q
	}
	if q, ok := plainNames[strings.TrimSuffix(f, "s")]; ok { // "eggs", "onions"
		return q
	}
	return f
}

// score rates a result's description for a search: every searched word it
// has counts, the food's own name (before the first comma) more, and shorter,
// plainer names win ("Onions, raw" over "Onions, frozen, chopped, cooked").
func score(query, desc string) int {
	d := strings.ToLower(desc)
	head, _, _ := strings.Cut(d, ",")
	words := strings.Fields(query)
	s := 0
	for _, w := range words {
		w = strings.TrimSuffix(w, "s")
		if strings.Contains(d, w) {
			s += 10
		}
	}
	if len(words) > 0 && strings.Contains(head, strings.TrimSuffix(words[len(words)-1], "s")) {
		s += 4
	}
	if strings.Contains(d, "raw") {
		s += 2
	}
	return s - len(strings.Fields(strings.ReplaceAll(d, ",", " ")))
}
