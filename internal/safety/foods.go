package safety

import "strings"

// knownFoods are plain, single-ingredient foods free of every listed
// allergen. Strict mode (a severe allergy) accepts an ingredient only when
// it's one of these or a food whose allergens the word lists already know.
var knownFoods = map[string]bool{}

func init() {
	for _, f := range []string{
		// Single foods whose allergens the word lists already name (flour is
		// wheat, butter is milk): known, so they don't trip strict mode for
		// other allergies.
		"flour", "plain flour", "all purpose flour", "bread flour", "cake flour", "pastry flour", "whole wheat flour",
		"self rising flour", "self raising flour", "egg", "egg white", "egg yolk", "milk", "whole milk", "skim milk",
		"buttermilk", "butter", "unsalted butter", "salted butter", "cream", "heavy cream", "double cream",
		"single cream", "whipping cream", "sour cream", "yogurt", "greek yogurt", "cream cheese", "cheddar",
		"parmesan", "mozzarella", "feta", "ricotta", "peanut", "almond", "walnut", "pecan", "cashew", "hazelnut",
		"pistachio", "tofu", "sesame seed", "tahini", "shrimp", "salmon", "tuna", "cod", "oat", "rolled oat",
		"sunflower", "sunflower seed",
		// Home & Care: plain chemicals and single-plant oils (nut oils are
		// allergen words already).
		"baking soda", "washing soda", "borax", "hydrogen peroxide", "rubbing alcohol", "isopropyl alcohol",
		"citric acid", "xylitol", "glycerin", "vegetable glycerin", "distilled water", "witch hazel", "epsom salt",
		"bentonite clay", "calcium carbonate", "activated charcoal", "stevia", "bleach", "ammonia", "oxygen bleach",
		"sodium percarbonate", "cornstarch", "arrowroot powder", "aloe vera", "aloe vera gel", "beeswax", "shea butter",
		"cocoa butter", "jojoba oil", "essential oil", "peppermint oil", "spearmint oil", "tea tree oil", "lavender oil",
		"lemon oil", "orange oil", "eucalyptus oil", "clove oil", "cinnamon oil", "rosemary oil", "grapefruit oil",
		"lime oil", "wintergreen oil", "pine oil", "lemongrass oil", "thyme oil", "oregano oil", "vodka",
		// Basics
		"salt", "pepper", "black pepper", "white pepper", "peppercorn", "sugar", "brown sugar", "powdered sugar",
		"confectioners sugar", "icing sugar", "cane sugar", "granulated sugar", "honey", "maple syrup", "agave",
		"agave nectar", "corn syrup", "molasses", "water", "ice", "ice water", "boiling water", "hot water",
		// Oils and vinegars
		"olive oil", "extra virgin olive oil", "vegetable oil", "canola oil", "avocado oil", "coconut oil",
		"sunflower oil", "corn oil", "grapeseed oil", "safflower oil", "white vinegar", "distilled vinegar",
		"distilled white vinegar", "apple cider vinegar", "rice vinegar", "cider vinegar",
		// Vegetables
		"garlic", "garlic clove", "garlic powder", "onion", "onion powder", "shallot", "scallion", "green onion",
		"spring onion", "leek", "chive", "tomato", "tomato paste", "potato", "sweet potato", "yam", "carrot",
		"bell pepper", "jalapeno", "serrano", "poblano", "chili", "chile", "chili pepper", "habanero", "celery",
		"spinach", "kale", "lettuce", "romaine", "arugula", "cabbage", "broccoli", "cauliflower", "zucchini",
		"squash", "butternut squash", "pumpkin", "cucumber", "mushroom", "eggplant", "asparagus", "green bean",
		"pea", "snap pea", "snow pea", "bean sprout", "beet", "radish", "turnip", "parsnip", "okra", "artichoke",
		"corn", "sweet corn", "avocado", "olive", "caper", "ginger", "fresh ginger", "ginger root", "lemongrass",
		"cilantro", "parsley", "basil", "mint", "dill", "thyme", "rosemary", "sage", "oregano", "bay leaf", "chard",
		"collard green", "bok choy", "fennel", "jicama", "watercress",
		// Fruit
		"lemon", "lime", "orange", "lemon juice", "lime juice", "orange juice", "lemon zest", "lime zest",
		"orange zest", "apple", "banana", "pear", "peach", "plum", "cherry", "grape", "strawberry", "blueberry",
		"raspberry", "blackberry", "cranberry", "mango", "pineapple", "kiwi", "watermelon", "melon", "cantaloupe",
		"honeydew", "date", "fig", "prune", "apricot", "nectarine", "pomegranate", "grapefruit", "coconut",
		"shredded coconut", "coconut milk", "coconut cream", "coconut water", "applesauce",
		// Grains and starches (gluten-free ones)
		"rice", "white rice", "brown rice", "basmati rice", "jasmine rice", "wild rice", "arborio rice", "quinoa",
		"cornmeal", "cornstarch", "corn starch", "polenta", "grit", "masa harina", "potato starch", "tapioca",
		"tapioca starch", "arrowroot", "millet", "sorghum", "amaranth", "teff", "rice flour", "corn tortilla",
		// Legumes
		"black bean", "kidney bean", "pinto bean", "cannellini bean", "navy bean", "great northern bean",
		"chickpea", "garbanzo bean", "lentil", "red lentil", "split pea", "lima bean", "bean",
		// Meat (plain)
		"chicken", "chicken breast", "chicken thigh", "chicken wing", "chicken drumstick", "whole chicken", "beef",
		"ground beef", "steak", "chuck roast", "pork", "pork chop", "pork loin", "pork shoulder", "pork tenderloin",
		"ground pork", "lamb", "turkey", "ground turkey", "turkey breast", "veal", "duck", "venison", "bison",
		// Spices and baking
		"cinnamon", "nutmeg", "ground ginger", "turmeric", "cumin", "coriander", "paprika", "smoked paprika",
		"cayenne", "cayenne pepper", "chili flake", "red pepper flake", "crushed red pepper", "allspice", "cardamom",
		"fennel seed", "star anise", "saffron", "clove", "caraway", "anise", "sumac", "baking soda", "baking powder",
		"yeast", "active dry yeast", "instant yeast", "vanilla", "vanilla extract", "vanilla bean", "cocoa",
		"cocoa powder", "unsweetened cocoa", "gelatin", "agar", "cream of tartar", "nutritional yeast",
		"flaxseed", "ground flaxseed", "flax seed", "chia seed", "sunflower seed", "pumpkin seed", "pepita",
		"herb", "fresh herb", "dried herb", "italian seasoning", "herbes de provence", "flax egg", "chia egg",
		"zucchini noodle", "sweet paprika",
	} {
		knownFoods[strings.Join(words(f), " ")] = true
	}
}

// descriptors say how a food is cut, sized or kept, not what it is.
var descriptors = map[string]bool{}

func init() {
	for _, d := range strings.Fields(`fresh freshly chopped diced minced sliced grated shredded ground large small
		medium whole boneless skinless bone in finely roughly coarsely thinly thin thick peeled seeded cored dried
		frozen thawed cooked uncooked raw organic ripe packed softened melted cold warm hot room temperature extra
		lean plus more for serving garnish to taste optional about and or of a an the few some pinch dash red green
		yellow white black brown purple orange sweet baby roma plum cherry russet yukon gold golden idaho vidalia
		flat leaf curly english persian hass granny smith fuji gala honeycrisp jumbo light dark kosher sea coarse
		fine flaky iodized smoked mild crushed pure natural unsweetened wild long short grain rinsed drained
		halved quartered cubed julienned trimmed torn divided juiced zested stemmed pitted squeezed heaping level
		leaves leaf sprig bunch head stalk clove can canned jar low sodium salt reduced unsalted salted new
		split cut into piece inch strip wedge round mashed toasted roasted steamed boiled serve served little
		frying greasing brushing dusting sprinkling topping caster superfine icing confectioners granulated
		self rising raising all purpose plain double single heavy whipping skim essential therapeutic grade food
		distilled filtered`) {
		descriptors[singular(d)] = true
	}
}

// known reports whether an ingredient is a plain food strict mode can trust.
func known(food string) bool {
	var parts [][]string
	for _, seg := range strings.FieldsFunc(food, func(r rune) bool { return r == ',' || r == ';' }) {
		parts = append(parts, splitParts(seg)...)
	}
	if len(parts) == 0 {
		return false
	}
	for _, p := range parts {
		if !knownPart(p) {
			return false
		}
	}
	return true
}

// splitParts splits "salt and pepper" into its foods.
func splitParts(food string) [][]string {
	var parts [][]string
	var cur []string
	for _, w := range words(food) {
		if w == "and" || w == "or" || w == "&" || w == "plus" {
			if len(cur) > 0 {
				parts = append(parts, cur)
			}
			cur = nil
			continue
		}
		cur = append(cur, w)
	}
	if len(cur) > 0 {
		parts = append(parts, cur)
	}
	return parts
}

// knownPart: the whole phrase is a known food, or it is once the describing
// words are left out ("finely chopped fresh parsley" → parsley,
// "peppermint essential oil" → peppermint oil).
func knownPart(ws []string) bool {
	var core, plain []string
	for _, w := range ws {
		if isNumber(w) {
			continue
		}
		core = append(core, w)
		if !descriptors[w] {
			plain = append(plain, w)
		}
	}
	if len(core) == 0 {
		return false
	}
	if knownFoods[strings.Join(core, " ")] {
		return true
	}
	if len(plain) == 0 {
		return true // only describing words ("a little extra for frying")
	}
	return knownFoods[strings.Join(plain, " ")]
}

func isNumber(w string) bool {
	for _, r := range w {
		if (r < '0' || r > '9') && r != '.' && r != '/' {
			return false
		}
	}
	return true
}
