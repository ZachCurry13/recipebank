package safety

// Diet is a way of eating, checked word by word.
type Diet struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	no    phrases // never
	maybe phrases // not sure (e.g. "broth": chicken or vegetable?)
	not   phrases // look-alikes to ignore
	// allergens whose "contains" words also break the diet
	allergens []string
}

var (
	meat = []string{"chicken", "beef", "pork", "lamb", "mutton", "turkey", "veal", "venison", "duck", "goose",
		"bacon", "ham", "prosciutto", "pancetta", "salami", "pepperoni", "chorizo", "sausage", "hot dog",
		"bratwurst", "kielbasa", "steak", "brisket", "rib", "ground meat", "meatball", "meat", "lard", "suet",
		"tallow", "bone broth", "drippings", "liver", "rabbit", "bison", "goat", "quail", "pheasant", "spam",
		"jerky", "corned beef", "pastrami", "bologna", "chicken broth", "chicken stock", "beef broth", "beef stock",
		"chicken bouillon", "beef bouillon", "oxtail", "short rib", "pulled pork", "carnitas", "gelatin", "schmaltz"}
	meatNot = []string{"vegetable broth", "vegetable stock", "veggie broth", "mushroom broth", "plant based meat",
		"meatless", "vegan sausage", "veggie burger", "impossible", "beyond meat", "nutmeg", "coconut meat",
		"crab meat", "lobster meat", "imitation", "veggie", "vegetarian", "vegan"}
	pork    = []string{"pork", "bacon", "ham", "prosciutto", "pancetta", "lard", "chorizo", "pulled pork", "carnitas", "spam", "guanciale"}
	alcohol = []string{"wine", "beer", "rum", "vodka", "bourbon", "whiskey", "whisky", "brandy", "sherry", "liqueur",
		"sake", "mirin", "cooking wine", "marsala", "vermouth", "cognac", "tequila", "gin", "champagne", "amaretto",
		"kahlua", "grand marnier", "ale", "lager", "stout", "cider", "prosecco", "port wine", "triple sec"}
	unclearBroth = []string{"broth", "stock", "bouillon", "stock cube", "gravy", "drippings"}
)

var Diets = []Diet{
	{Key: "vegetarian", Label: "Vegetarian", no: compile(meat...), not: compile(meatNot...),
		maybe: compile(unclearBroth...), allergens: []string{"fish", "shellfish", "mollusc"}},
	{Key: "vegan", Label: "Vegan", no: compile(append(append([]string{}, meat...), "honey", "beeswax", "lanolin", "carmine", "shellac")...),
		not: compile(meatNot...), maybe: compile(append(append([]string{}, unclearBroth...), "sugar", "chocolate", "wine")...),
		allergens: []string{"fish", "shellfish", "mollusc", "milk", "egg"}},
	{Key: "pescatarian", Label: "Pescatarian (fish, no meat)", no: compile(meat...), not: compile(meatNot...),
		maybe: compile(unclearBroth...)},
	{Key: "meatless", Label: "Meatless (Fridays, Lent)", no: compile(meat...), not: compile(meatNot...),
		maybe: compile(unclearBroth...)},
	{Key: "glutenfree", Label: "Gluten-free", allergens: []string{"gluten"}},
	{Key: "dairyfree", Label: "Dairy-free", allergens: []string{"milk"}},
	{Key: "nutfree", Label: "Nut-free", allergens: []string{"treenut", "peanut"}},
	{Key: "halal", Label: "Halal", no: compile(append(append([]string{}, pork...), alcohol...)...),
		not:   compile("turkey bacon", "turkey ham", "beef bacon", "halal", "non alcoholic", "alcohol free"),
		maybe: compile(append(append([]string{}, meat...), "vanilla extract")...)},
	{Key: "kosher", Label: "Kosher-style", no: compile(pork...), not: compile("turkey bacon", "beef bacon"),
		maybe: compile("gelatin"), allergens: []string{"shellfish", "mollusc"}},
}

// DietByKey finds a diet; ok is false for unknown keys.
func DietByKey(key string) (Diet, bool) {
	for _, d := range Diets {
		if d.Key == key {
			return d, true
		}
	}
	return Diet{}, false
}

// Heat words, strongest first: 5 extreme … 1 mild. Sweet and bell peppers don't count.
var heatWords = []struct {
	level int
	words phrases
}{
	{5, compile("ghost pepper", "bhut jolokia", "carolina reaper", "reaper", "scorpion pepper", "trinidad scorpion")},
	{4, compile("habanero", "scotch bonnet", "thai chili", "thai chile", "bird's eye", "birds eye", "bird eye", "piri piri")},
	{3, compile("jalapeno", "serrano", "cayenne", "chile de arbol", "arbol", "fresno", "hot pepper")},
	{2, compile("chili", "chile", "chili powder", "chili flake", "red pepper flake", "crushed red pepper", "sriracha",
		"hot sauce", "chipotle", "gochujang", "gochugaru", "harissa", "sambal", "wasabi", "horseradish", "chili oil",
		"curry paste", "jerk", "cajun", "buffalo sauce", "sichuan", "szechuan")},
	{1, compile("black pepper", "pepper", "paprika", "white pepper", "poblano", "ginger", "curry powder", "mustard", "dijon")},
}

var heatNot = compile("bell pepper", "sweet pepper", "roasted red pepper", "sweet paprika", "pepper jack", "mild",
	"chili bean", "sweet chili", "peppermint", "pepperoncini")

// EstimateHeat guesses 0-5 peppers from the ingredients, with the word that decided it.
func EstimateHeat(lines []string) (int, int) {
	best, at := 0, -1
	for i, line := range lines {
		t := newText(line)
		t.mask(heatNot)
		for _, h := range heatWords {
			if h.level > best && t.has(h.words) {
				best, at = h.level, i
			}
		}
	}
	return best, at
}
