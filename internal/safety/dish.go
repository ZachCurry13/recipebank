package safety

// DishStatus is how a dish without a recipe (a guest's casserole, a
// store-bought pie) suits a person, from the allergens its cook says it has.
// Its other contents aren't known, so for a real allergy or a diet it's
// never OK, only "not sure": unknown is never safe.
func DishStatus(contains []string, p Person) string {
	for _, k := range contains {
		if _, ok := p.Allergies[k]; ok {
			return No
		}
	}
	if realAllergy(p) || len(p.Diets) > 0 {
		return Unsure
	}
	return OK
}
