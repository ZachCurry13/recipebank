package api

import (
	"testing"
	"time"
)

func TestPlanAndTonight(t *testing.T) {
	c, _ := setup(t)
	var kid, guest struct{ ID int64 }
	c.do("POST", "/api/people", map[string]any{"name": "Kid", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "allergic"}}}, &kid)
	c.do("POST", "/api/people", map[string]any{"name": "Grandpa", "is_guest": true, "heat_max": -1,
		"rules": []map[string]string{{"kind": "diet", "key": "vegetarian"}}}, &guest)
	var mac, salad struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Mac and Cheese", "servings": 4, "ingredients": []map[string]string{
		{"line": "8 oz macaroni"}, {"line": "2 cups shredded cheddar cheese"}}}, &mac)
	c.do("POST", "/api/recipes", map[string]any{"title": "Green Salad", "ingredients": []map[string]string{
		{"line": "1 head lettuce"}, {"line": "1 cucumber"}, {"line": "2 tbsp olive oil"}}}, &salad)

	today := time.Now()
	date := today.Format(day)
	tomorrow := today.AddDate(0, 0, 1).Format(day)

	// Nothing planned: ideas are only recipes everyone home can eat.
	var tn struct {
		Meals []tonightMeal `json:"meals"`
		Ideas []planRecipe  `json:"ideas"`
	}
	c.do("GET", "/api/tonight?date="+date, nil, &tn)
	if len(tn.Meals) != 0 || len(tn.Ideas) != 1 || tn.Ideas[0].ID != salad.ID {
		t.Fatalf("ideas: %+v", tn.Ideas)
	}

	// Plan mac and cheese tonight with Grandpa visiting.
	if code := c.do("POST", "/api/plan", map[string]any{"date": date, "meal": "dinner", "recipe_id": mac.ID, "servings": 8}, nil); code != 200 {
		t.Fatalf("plan: %d", code)
	}
	c.do("POST", "/api/plan", map[string]any{"date": tomorrow, "meal": "lunch", "title": "Leftovers"}, nil)
	c.do("PUT", "/api/plan/day/"+date, map[string]any{"who": []int64{kid.ID, guest.ID}}, nil)
	c.do("GET", "/api/tonight?date="+date, nil, &tn)
	if len(tn.Meals) != 1 || len(tn.Meals[0].Full) != 2 || len(tn.Ideas) != 0 {
		t.Fatalf("tonight: %+v", tn)
	}
	for _, v := range tn.Meals[0].Full {
		if v.Status != "no" && v.Name == "Kid" {
			t.Errorf("mac and cheese for a milk allergy: %s", v.Status)
		}
	}

	var plan struct {
		Days []planDay `json:"days"`
	}
	c.do("GET", "/api/plan?from="+date+"&days=2", nil, &plan)
	if len(plan.Days) != 2 || !plan.Days[0].WhoSet || len(plan.Days[1].Meals) != 1 || plan.Days[1].Meals[0].Title != "Leftovers" {
		t.Fatalf("plan: %+v", plan.Days)
	}

	// The week to the shopping list, scaled to 8 servings (twice the recipe).
	c.do("POST", "/api/plan/shopping", map[string]any{"from": date, "days": 7}, nil)
	var list shopList
	c.do("GET", "/api/shopping", nil, &list)
	if mc := list.find("macaroni"); mc == nil || mc.Qty != 16 || mc.Unit != "oz" {
		t.Fatalf("macaroni: %+v", mc)
	}
	if code := c.do("POST", "/api/plan", map[string]any{"date": "someday", "meal": "dinner", "title": "x"}, nil); code != 400 {
		t.Fatalf("bad date accepted: %d", code)
	}
}
