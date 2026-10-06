package recipe

import (
	"strings"
	"testing"
)

func TestStepNeeds(t *testing.T) {
	cases := map[string]string{
		"Wash the strawberries and pat them dry.":             "",
		"Stir the flour, sugar and salt in a bowl.":           "",
		"Crack the eggs into a cup.":                          "",
		"Scatter the sprinkles over the cookies.":             "",
		"Cut out shapes with a cookie cutter.":                "knife", // careful: "cut" is marked
		"Press out shapes with a cookie cutter.":              "",
		"Finely chop the onion.":                              "knife",
		"Dice the potatoes into 1-inch cubes.":                "knife",
		"Add the diced tomatoes and a quarter cup of water.":  "",
		"Grate the cheese.":                                   "sharp",
		"Sprinkle the grated cheese on top.":                  "",
		"Blend until smooth.":                                 "sharp",
		"Stir until well blended.":                            "",
		"Heat the oil in a skillet over medium heat.":         "stove,hot",
		"Simmer for 20 minutes.":                              "stove",
		"Bring a large pot of water to a boil.":               "stove,hot",
		"Cook the onion until soft.":                          "stove",
		"Preheat the oven to 350°F.":                          "oven",
		"Bake 25 minutes, until golden.":                      "oven",
		"Whisk in the baking soda and baking powder.":         "",
		"Spread the batter in the baking dish.":               "",
		"Drain the pasta.":                                    "hot",
		"Microwave the butter until melted.":                  "hot",
		"Stir in the melted butter.":                          "",
		"Add a dash of hot sauce.":                            "",
		"Let the cake cool, then spread the frosting on top":  "",
		"Slice the bread and toast the slices in a pan.":      "knife,stove",
		"Carefully take the tray out of the oven.":            "oven,hot",
		"Roll the dough into balls and set on the tray.":      "",
		"Layer between baking parchment, then freeze.":        "",
		"Spray the tin with cooking spray.":                   "",
		"Pour into a hot pan and fry until crisp on the edge": "stove,hot",
	}
	for step, want := range cases {
		if got := strings.Join(StepNeeds(step), ","); got != want {
			t.Errorf("%q: got %q, want %q", step, got, want)
		}
	}
}

func TestKidsCanHelp(t *testing.T) {
	steps := func(texts ...string) []Step {
		var out []Step
		for _, s := range texts {
			out = append(out, Step{Text: s})
		}
		return out
	}
	cookies := &Recipe{Area: AreaKitchen, Steps: steps("Preheat the oven to 350°F.", "Stir the butter and sugar together.",
		"Mix in the flour and chocolate chips.", "Roll into balls on the tray.", "Bake 12 minutes.")}
	if !cookies.KidsCanHelp() {
		t.Fatalf("cookies: %v", cookies.Needs())
	}
	steak := &Recipe{Area: AreaKitchen, Steps: steps("Heat a skillet over high heat.", "Sear the steak 3 minutes a side.",
		"Rest, then slice.", "Season with salt.")}
	if steak.KidsCanHelp() {
		t.Fatalf("steak: %v", steak.Needs())
	}
	snack := &Recipe{Area: AreaKitchen, Steps: steps("Spread peanut butter on the celery.")}
	if !snack.KidsCanHelp() {
		t.Fatal("a one-step snack a kid can make")
	}
	cleaner := &Recipe{Area: AreaHome, Steps: steps("Pour the vinegar into the bottle.", "Add the water.")}
	if cleaner.KidsCanHelp() {
		t.Fatal("Home & Care recipes aren't for kids to make")
	}
}
