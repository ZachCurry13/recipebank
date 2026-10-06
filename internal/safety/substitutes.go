package safety

// subTable: common kitchen substitutions for when something's missing
// ("no cumin? use chili powder"). Combinations are written with "and" so
// each part is checked against everyone's rules. The allergy swaps
// (swapTable) are offered after these.
var subTable = []struct {
	when  phrases
	ideas []swapIdea
}{
	{compile("cumin", "ground cumin"), []swapIdea{{"chili powder", "same amount; it contains cumin and adds a little heat"},
		{"ground coriander", "same amount; milder"}, {"smoked paprika", "half as much; smoky"}}},
	{compile("coriander", "ground coriander"), []swapIdea{{"cumin", "half as much"}, {"caraway", "half as much"}}},
	{compile("smoked paprika"), []swapIdea{{"paprika", "same amount, less smoky"}, {"chili powder", "half as much"}}},
	{compile("paprika"), []swapIdea{{"chili powder", "half as much; hotter"}, {"cayenne", "a pinch; much hotter"}}},
	{compile("chili powder"), []swapIdea{{"cumin and paprika and oregano", "mix equal parts; add cayenne for heat"}, {"paprika", "same amount, milder"}}},
	{compile("cayenne", "red pepper flake", "chili flake"), []swapIdea{{"hot sauce", "a few drops"}, {"black pepper", "for a little bite"}}},
	{compile("oregano"), []swapIdea{{"basil", "same amount"}, {"thyme", "same amount"}, {"italian seasoning", "same amount"}}},
	{compile("basil"), []swapIdea{{"oregano", "half as much"}, {"italian seasoning", "same amount"}}},
	{compile("thyme"), []swapIdea{{"oregano", "same amount"}, {"rosemary", "half as much"}}},
	{compile("rosemary"), []swapIdea{{"thyme", "same amount"}, {"sage", "half as much"}}},
	{compile("sage"), []swapIdea{{"thyme", "same amount"}, {"rosemary", "half as much"}}},
	{compile("cilantro"), []swapIdea{{"parsley", "same amount, with a squeeze of lime"}}},
	{compile("parsley"), []swapIdea{{"cilantro", "same amount"}, {"basil", "same amount"}}},
	{compile("nutmeg"), []swapIdea{{"cinnamon", "half as much"}, {"allspice", "same amount"}}},
	{compile("allspice"), []swapIdea{{"cinnamon and nutmeg", "mostly cinnamon, a pinch of nutmeg"}}},
	{compile("cinnamon"), []swapIdea{{"nutmeg", "a quarter as much"}, {"allspice", "half as much"}}},
	{compile("ginger", "ground ginger"), []swapIdea{{"cinnamon", "same amount"}, {"allspice", "same amount"}}},
	{compile("garlic", "garlic clove"), []swapIdea{{"garlic powder", "1/8 tsp per clove"}}},
	{compile("onion"), []swapIdea{{"onion powder", "1 tbsp per onion"}, {"shallot", "3 shallots per onion"}}},
	{compile("shallot"), []swapIdea{{"onion", "a little less, finely chopped"}}},
	{compile("green onion", "scallion"), []swapIdea{{"chive", "same amount"}, {"onion", "a little, finely chopped"}}},
	{compile("lemon juice"), []swapIdea{{"lime juice", "same amount"}, {"white vinegar", "half as much"}}},
	{compile("lime juice"), []swapIdea{{"lemon juice", "same amount"}}},
	{compile("buttermilk"), []swapIdea{{"milk and lemon juice", "1 tbsp lemon juice in 1 cup milk; wait 5 minutes"},
		{"plain yogurt and milk", "3/4 cup yogurt and 1/4 cup milk"}}},
	{compile("sour cream"), []swapIdea{{"plain greek yogurt", "same amount"}}},
	{compile("greek yogurt", "plain yogurt", "yogurt"), []swapIdea{{"sour cream", "same amount"}}},
	{compile("heavy cream", "whipping cream"), []swapIdea{{"milk and butter", "3/4 cup milk with 1/4 cup melted butter (won't whip)"}}},
	{compile("half and half"), []swapIdea{{"milk and heavy cream", "equal parts"}, {"milk", "richer with a little butter"}}},
	{compile("brown sugar"), []swapIdea{{"sugar and molasses", "1 cup sugar with 1 tbsp molasses"}, {"sugar", "same amount, less moist"}}},
	{compile("powdered sugar", "confectioners sugar"), []swapIdea{{"sugar and cornstarch", "blend 1 cup sugar with 1 tbsp cornstarch until fine"}}},
	{compile("maple syrup"), []swapIdea{{"honey", "same amount"}, {"brown sugar", "3/4 as much and a splash of water"}}},
	{compile("molasses"), []swapIdea{{"honey", "same amount"}, {"maple syrup", "same amount"}}},
	{compile("baking powder"), []swapIdea{{"baking soda and cream of tartar", "1/4 tsp soda and 1/2 tsp cream of tartar per tsp"}}},
	{compile("baking soda"), []swapIdea{{"baking powder", "3 times as much"}}},
	{compile("cornstarch"), []swapIdea{{"flour", "twice as much, for thickening"}, {"arrowroot", "same amount"}}},
	{compile("self rising flour"), []swapIdea{{"flour and baking powder and salt", "1 cup flour, 1 1/2 tsp baking powder, 1/4 tsp salt"}}},
	{compile("cake flour"), []swapIdea{{"flour and cornstarch", "1 cup minus 2 tbsp flour, plus 2 tbsp cornstarch"}}},
	{compile("wine", "white wine", "red wine"), []swapIdea{{"broth and white vinegar", "same amount of broth, a splash of vinegar"}}},
	{compile("rice vinegar"), []swapIdea{{"apple cider vinegar", "same amount, with a pinch of sugar"}}},
	{compile("apple cider vinegar"), []swapIdea{{"white vinegar", "same amount"}, {"lemon juice", "same amount"}}},
	{compile("white vinegar"), []swapIdea{{"apple cider vinegar", "same amount"}, {"lemon juice", "same amount"}}},
	{compile("tomato paste"), []swapIdea{{"ketchup", "same amount; a little sweeter"}, {"tomato sauce", "three times as much, cook it down"}}},
	{compile("tomato sauce"), []swapIdea{{"tomato paste and water", "1 part paste to 1 part water"}}},
	{compile("bread crumb", "breadcrumb", "panko"), []swapIdea{{"cracker", "crushed, same amount"}, {"rolled oats", "same amount"}}},
	{compile("mayonnaise", "mayo"), []swapIdea{{"plain greek yogurt", "same amount"}, {"sour cream", "same amount"}}},
	{compile("vegetable oil", "canola oil"), []swapIdea{{"olive oil", "same amount"}, {"melted butter", "same amount"}}},
	{compile("olive oil"), []swapIdea{{"vegetable oil", "same amount"}}},
	{compile("broth", "stock"), []swapIdea{{"bouillon and water", "1 cube or 1 tsp per cup of water"}, {"water, salt and herbs", "less flavor"}}},
}

// Substitutes lists replacements for a missing food: kitchen substitutions
// first, then the allergy swaps. Not yet checked against anyone's rules: use OKForAll.
func Substitutes(food string) []SwapIdea {
	t := newText(food)
	var out []SwapIdea
	seen := map[string]bool{}
	add := func(ideas []swapIdea) {
		for _, i := range ideas {
			if !seen[i.to] {
				seen[i.to] = true
				out = append(out, SwapIdea{To: i.to, Note: i.note})
			}
		}
	}
	for _, row := range subTable {
		if t.has(row.when) {
			add(row.ideas)
			break
		}
	}
	add(swapIdeas(t))
	return out
}

func swapIdeas(t text) []swapIdea {
	for _, row := range swapTable {
		if t.has(row.when) {
			return row.ideas
		}
	}
	return nil
}
