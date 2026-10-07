package api

import (
	"fmt"
	"testing"
)

// A person allergic to soy who can have soybean oil: recipes with the oil
// are fine for her, recipes with soy lecithin aren't, and "soy" itself can't
// be an exception.
func TestAllergyExceptionsSaved(t *testing.T) {
	c, _ := setup(t)
	var sis struct {
		ID    int64
		Rules []struct {
			Key   string
			Allow []string
		}
	}
	if code := c.do("POST", "/api/people", map[string]any{"name": "Sister", "heat_max": -1, "rules": []map[string]any{
		{"kind": "allergy", "key": "soy", "severity": "allergic", "allow": []string{" Soybean  Oil ", "soybean oil"}}}}, &sis); code >= 300 {
		t.Fatalf("save: %d", code)
	}
	var people []struct {
		ID    int64
		Rules []struct {
			Key   string
			Allow []string
		}
	}
	c.do("GET", "/api/people", nil, &people)
	if len(people) != 1 || len(people[0].Rules) != 1 || len(people[0].Rules[0].Allow) != 1 || people[0].Rules[0].Allow[0] != "soybean oil" {
		t.Fatalf("saved tidy: %+v", people)
	}
	verdict := func(lines ...string) string {
		var ings []map[string]string
		for _, l := range lines {
			ings = append(ings, map[string]string{"line": l})
		}
		var rc struct{ ID int64 }
		c.do("POST", "/api/recipes", map[string]any{"title": lines[0], "ingredients": ings}, &rc)
		var page struct{ Verdicts []struct{ Status string } }
		c.do("GET", fmt.Sprintf("/api/recipes/%d?who=%d", rc.ID, people[0].ID), nil, &page)
		return page.Verdicts[0].Status
	}
	if v := verdict("2 cups flour", "1/2 cup soybean oil"); v != "ok" {
		t.Fatalf("soybean oil: %s", v)
	}
	if v := verdict("1 cup chocolate chips with soy lecithin"); v != "no" {
		t.Fatalf("soy lecithin: %s", v)
	}
	for _, bad := range []string{"soy", "Soya", "x", "soy; drop table"} {
		code := c.do("POST", "/api/people", map[string]any{"name": "Bad", "heat_max": -1, "rules": []map[string]any{
			{"kind": "allergy", "key": "soy", "severity": "allergic", "allow": []string{bad}}}}, nil)
		if code != 400 {
			t.Fatalf("%q: %d", bad, code)
		}
	}
	if code := c.do("POST", "/api/people", map[string]any{"name": "Bad", "heat_max": -1, "rules": []map[string]any{
		{"kind": "diet", "key": "vegan", "allow": []string{"honey"}}}}, nil); code != 400 {
		t.Fatalf("diets have no exceptions: %d", code)
	}
}
