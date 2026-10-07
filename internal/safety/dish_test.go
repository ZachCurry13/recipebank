package safety

import "testing"

func TestDishStatus(t *testing.T) {
	milk := allergic("milk", Allergic)
	if DishStatus([]string{"milk", "wheat"}, milk) != No {
		t.Fatal("declared milk")
	}
	if DishStatus([]string{"wheat"}, milk) != Unsure || DishStatus(nil, milk) != Unsure {
		t.Fatal("anything else in it isn't known: not sure for a real allergy")
	}
	if DishStatus([]string{"milk"}, allergic("milk", Avoid)) != No {
		t.Fatal("avoid still avoids what's declared")
	}
	if DishStatus(nil, allergic("milk", Avoid)) != OK {
		t.Fatal("avoid isn't a real allergy")
	}
	if DishStatus(nil, Person{HeatMax: -1, Diets: []string{"vegan"}}) != Unsure {
		t.Fatal("a diet can't be checked without the recipe")
	}
	if DishStatus([]string{"egg"}, Person{HeatMax: -1}) != OK {
		t.Fatal("no rules")
	}
}
