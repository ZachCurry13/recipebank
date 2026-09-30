package safety

// Allergen is one allergen with the words that mean it (including hidden
// names), look-alikes that don't ("coconut milk"), and foods that often
// contain it so the label must be checked.
type Allergen struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	EU    bool    `json:"eu"` // only on the EU's list of 14
	terms phrases // contains it
	not   phrases // look-alikes, masked before terms are matched
	maybe phrases // may contain it: check the label
}

var milkNot = []string{"coconut milk", "almond milk", "oat milk", "soy milk", "soya milk", "rice milk", "cashew milk",
	"hemp milk", "pea milk", "flax milk", "macadamia milk", "coconut cream", "cream of coconut", "cashew cream",
	"cream of tartar", "peanut butter", "almond butter", "cashew butter", "nut butter", "sunflower butter",
	"sunflower seed butter", "seed butter", "apple butter", "cocoa butter", "shea butter", "mango butter",
	"butternut", "butter bean", "butter lettuce", "vegan butter", "plant butter", "vegan cheese", "dairy free",
	"non dairy", "nondairy", "coconut yogurt", "vegan", "plant based", "dairy free cheese", "dairy free butter",
	"dairy free milk", "dairy free yogurt", "vegan cream cheese", "vegan sour cream", "vegan yogurt", "non dairy milk",
	"non dairy creamer", "plant based milk", "plant based butter"}

var wheatNot = []string{"buckwheat", "rice flour", "almond flour", "coconut flour", "oat flour", "corn flour",
	"cornflour", "chickpea flour", "gram flour", "cassava flour", "tapioca flour", "potato flour", "sorghum flour",
	"teff flour", "arrowroot flour", "millet flour", "quinoa flour", "banana flour", "gluten free", "rice noodle",
	"glass noodle", "zucchini noodle", "shirataki noodle", "kelp noodle", "rice paper", "rice cracker",
	"coconut aminos", "cauliflower crust", "lettuce wrap", "rice cake", "gluten free flour", "gluten free pasta",
	"gluten free bread", "gluten free breadcrumb", "gluten free noodle", "gluten free tortilla", "gluten free oat",
	"gluten free cracker", "gluten free soy sauce", "gluten free flour blend"}

var wheatTerms = []string{"wheat", "flour", "semolina", "durum", "spelt", "farro", "bulgur", "couscous", "seitan",
	"einkorn", "emmer", "kamut", "graham", "wheat germ", "bran", "bread", "breadcrumb", "bread crumb", "panko",
	"crouton", "pasta", "spaghetti", "macaroni", "penne", "fettuccine", "linguine", "lasagna", "lasagne", "orzo",
	"rigatoni", "fusilli", "ziti", "rotini", "farfalle", "tortellini", "ravioli", "noodle", "udon", "ramen",
	"flour tortilla", "pita", "bagel", "bun", "croissant", "pie crust", "pie dough", "puff pastry", "pastry",
	"phyllo", "filo", "wonton", "dumpling", "cracker", "biscuit", "pretzel", "soy sauce", "shoyu", "teriyaki",
	"roux", "matzo", "matzah", "farina", "cream of wheat", "triticale", "freekeh", "naan", "baguette", "brioche",
	"challah", "sourdough", "english muffin", "graham cracker", "cookie", "cake", "muffin", "waffle", "pancake",
	"stuffing", "breading", "gnocchi", "vital wheat gluten", "gluten"}

var glutenGrains = []string{"barley", "rye", "oat", "oatmeal", "rolled oat", "malt", "malted", "malt extract",
	"malt vinegar", "beer", "ale", "lager", "brewer's yeast", "pearl barley", "oat flour", "oat milk", "oat bran"}

// Allergens: the 9 major US allergens, then the EU's extras.
var Allergens = []Allergen{
	{Key: "milk", Label: "Milk", terms: compile("milk", "butter", "buttermilk", "cream", "sour cream", "half and half",
		"cheese", "cheddar", "mozzarella", "parmesan", "parmigiano", "pecorino", "ricotta", "feta", "gouda", "brie",
		"camembert", "mascarpone", "cottage cheese", "cream cheese", "paneer", "provolone", "gruyere", "monterey jack",
		"colby", "asiago", "gorgonzola", "halloumi", "queso", "yogurt", "yoghurt", "kefir", "ghee", "whey", "casein",
		"caseinate", "lactose", "lactalbumin", "curd", "custard", "ice cream", "gelato", "condensed milk",
		"evaporated milk", "milk powder", "dulce de leche", "creme fraiche", "quark", "skyr", "labneh", "nougat",
		"milk chocolate", "white chocolate", "alfredo", "bechamel", "ranch", "pesto", "butterscotch", "caramel", "tzatziki"),
		not: compile(milkNot...),
		maybe: compile("vegan butter", "plant butter", "vegan cheese", "dairy free", "non dairy", "nondairy", "margarine",
			"chocolate", "coconut yogurt", "vegan", "plant based", "sausage", "hot dog", "deli meat")},
	{Key: "egg", Label: "Eggs", terms: compile("egg", "egg white", "egg yolk", "yolk", "albumen", "albumin",
		"ovalbumin", "mayonnaise", "mayo", "meringue", "aioli", "eggnog", "lysozyme", "hollandaise", "custard",
		"frittata", "quiche", "brioche", "challah", "egg noodle", "egg wash"),
		not: compile("egg free", "eggless", "vegan mayo", "vegan mayonnaise", "egg replacer", "flax egg", "chia egg", "eggplant"),
		maybe: compile("pasta", "noodle", "marshmallow", "vegan mayo", "vegan mayonnaise", "egg free", "egg replacer",
			"cake mix", "pancake mix", "breadcrumb", "bread crumb", "panko", "cookie", "cake", "muffin", "waffle")},
	{Key: "fish", Label: "Fish", terms: compile("fish", "salmon", "tuna", "cod", "anchovy", "sardine", "tilapia",
		"halibut", "trout", "mackerel", "haddock", "pollock", "bass", "catfish", "snapper", "swordfish", "mahi mahi",
		"flounder", "herring", "perch", "pike", "carp", "grouper", "branzino", "fish sauce", "worcestershire",
		"caesar dressing", "bonito", "katsuobushi", "dashi", "surimi", "imitation crab", "caviar", "roe", "fish stock",
		"nam pla", "lox", "smoked salmon", "sole fillet"),
		not:   compile("shellfish", "fish free"),
		maybe: compile("caesar", "furikake", "kimchi", "pad thai sauce")},
	{Key: "shellfish", Label: "Shellfish (crustaceans)", terms: compile("shellfish", "shrimp", "prawn", "crab",
		"lobster", "crawfish", "crayfish", "langoustine", "krill", "scampi", "shrimp paste"),
		not:   compile("imitation crab", "crab apple", "crabapple"),
		maybe: compile("imitation crab", "surimi", "seafood", "bouillabaisse", "fish sauce", "kimchi")},
	{Key: "treenut", Label: "Tree nuts", terms: compile("almond", "walnut", "pecan", "cashew", "pistachio", "hazelnut",
		"filbert", "macadamia", "brazil nut", "pine nut", "pignoli", "chestnut", "praline", "marzipan", "frangipane",
		"nutella", "gianduja", "amaretto", "pesto", "nut", "mixed nut", "baklava", "hickory nut", "beechnut", "nocino",
		"frangelico", "almond extract", "nut butter", "nut milk", "nut oil"),
		not:   compile("water chestnut", "butternut", "nut free", "tree nut free", "nutmeg", "coconut", "doughnut", "donut", "tiger nut"),
		maybe: compile("nut free", "granola", "trail mix", "nougat", "chocolate", "praline", "cereal", "baked good", "muesli")},
	{Key: "peanut", Label: "Peanuts", terms: compile("peanut", "peanut butter", "peanut oil", "groundnut", "arachis",
		"goober", "monkey nut"),
		not: compile("peanut free"),
		maybe: compile("mixed nut", "nut", "trail mix", "satay", "nut butter", "granola", "chocolate", "peanut free",
			"cereal", "chili sauce", "mole", "egg roll", "african", "thai")},
	{Key: "wheat", Label: "Wheat", terms: compile(wheatTerms...), not: compile(wheatNot...),
		maybe: compile("gluten free", "tortilla", "tamari", "hoisin", "gravy", "beer", "soup", "seasoning", "bouillon",
			"stock cube", "oat", "oatmeal", "rolled oat", "licorice", "imitation crab", "surimi")},
	{Key: "soy", Label: "Soy", terms: compile("soy", "soya", "soybean", "soy sauce", "tamari", "shoyu", "tofu",
		"tempeh", "edamame", "miso", "natto", "soy milk", "soy lecithin", "textured vegetable protein", "tvp",
		"soy protein", "teriyaki", "hoisin", "ponzu", "yuba"),
		not: compile("soy free"),
		maybe: compile("lecithin", "bouillon", "stock cube", "margarine", "vegan butter", "chocolate chip", "vegan",
			"plant based", "veggie burger", "vegetable broth", "bread", "soy free", "worcestershire")},
	{Key: "sesame", Label: "Sesame", terms: compile("sesame", "sesame oil", "sesame seed", "tahini", "benne",
		"gomasio", "gomashio", "halvah", "halva", "hummus", "za'atar", "zaatar", "furikake"),
		maybe: compile("dukkah", "bagel", "bun", "everything seasoning", "everything bagel", "sushi", "cracker")},

	{Key: "gluten", Label: "Gluten (wheat, barley, rye, oats)", EU: true,
		terms: compile(append(append([]string{}, wheatTerms...), glutenGrains...)...),
		not:   compile(without(wheatNot, "oat flour")...),
		maybe: compile("gluten free", "gluten free oat", "tortilla", "tamari", "hoisin", "gravy", "soup", "seasoning",
			"bouillon", "stock cube", "licorice", "imitation crab", "surimi")},
	{Key: "celery", Label: "Celery", EU: true, terms: compile("celery", "celeriac", "celery salt", "celery seed", "celery root"),
		maybe: compile("stock", "broth", "bouillon", "stock cube", "soup", "seasoning", "old bay", "spice blend", "mirepoix")},
	{Key: "mustard", Label: "Mustard", EU: true, terms: compile("mustard", "dijon", "mustard seed", "mustard powder",
		"dry mustard", "mustard green"),
		maybe: compile("mayonnaise", "mayo", "salad dressing", "dressing", "vinaigrette", "curry powder", "pickle",
			"relish", "ketchup", "seasoning", "spice blend")},
	{Key: "lupin", Label: "Lupin", EU: true, terms: compile("lupin", "lupine", "lupini", "lupin flour"),
		maybe: compile("gluten free flour", "gluten free bread", "gluten free pasta")},
	{Key: "mollusc", Label: "Molluscs", EU: true, terms: compile("clam", "mussel", "oyster", "scallop", "squid",
		"calamari", "octopus", "snail", "escargot", "oyster sauce", "abalone", "cuttlefish", "conch", "whelk", "cockle"),
		not:   compile("oyster mushroom"),
		maybe: compile("seafood", "fish sauce", "bouillabaisse", "paella")},
	{Key: "sulphite", Label: "Sulphites", EU: true, terms: compile("wine", "red wine", "white wine", "sherry",
		"vermouth", "champagne", "sulfite", "sulphite", "sulfur dioxide", "wine vinegar", "balsamic", "marsala", "port wine"),
		maybe: compile("dried fruit", "dried apricot", "golden raisin", "raisin", "vinegar", "bottled lemon juice",
			"molasses", "pickle", "dried cranberry", "maraschino", "cider")},
}

// packaged foods can hold almost anything: with a real allergy their
// label must be read, whatever the allergen.
var packaged = compile("broth", "stock", "bouillon", "stock cube", "soup", "soup mix", "seasoning", "seasoning mix",
	"spice blend", "spice mix", "taco seasoning", "sauce", "dressing", "marinade", "ketchup", "sausage", "hot dog",
	"deli meat", "chocolate", "chocolate chip", "sprinkle", "candy", "cereal", "granola", "mix", "frosting", "icing",
	"cookie", "cracker", "chip", "store bought", "prepared", "canned soup", "condensed soup", "vegan", "gluten free",
	"dairy free", "nut free", "egg free", "plant based", "imitation", "jam", "jelly", "salsa", "bouillon cube",
	"pesto", "curry paste", "gravy", "relish", "margarine", "shortening", "cooking spray", "baking chip")

func without(list []string, drop string) []string {
	out := []string{}
	for _, s := range list {
		if s != drop {
			out = append(out, s)
		}
	}
	return out
}

// AllergenByKey finds an allergen; ok is false for unknown keys.
func AllergenByKey(key string) (Allergen, bool) {
	for _, a := range Allergens {
		if a.Key == key {
			return a, true
		}
	}
	return Allergen{}, false
}

// AllergenList returns the allergens offered for the house's list: "us"
// (the 9 major ones) or "eu" (also gluten, celery, mustard, lupin,
// molluscs, sulphites).
func AllergenList(list string) []Allergen {
	out := []Allergen{}
	for _, a := range Allergens {
		if !a.EU || list == "eu" {
			out = append(out, a)
		}
	}
	return out
}
