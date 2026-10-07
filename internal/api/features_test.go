package api

import (
	"fmt"
	"testing"
)

// Features turned off by the admin are refused (and listed as off for the
// web app); each person can keep their own plan to some meals and hide pages.
func TestFeaturesAndSimplerView(t *testing.T) {
	c, _ := setup(t)
	var info struct{ Features map[string]bool }
	c.do("GET", "/api/info", nil, &info)
	if len(info.Features) < 10 || !info.Features["share"] || !info.Features["plan"] {
		t.Fatalf("everything starts on: %v", info.Features)
	}
	var rc struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Soup", "ingredients": []map[string]string{{"line": "1 onion"}}}, &rc)
	var link struct{ Share struct{ Token string } }
	if code := c.do("POST", fmt.Sprintf("/api/recipes/%d/share", rc.ID), nil, &link); code != 200 || link.Share.Token == "" {
		t.Fatalf("share: %d", code)
	}
	if code := c.do("PUT", "/api/admin/settings", map[string]string{"feature_share": "false", "feature_plan": "false"}, nil); code != 200 {
		t.Fatalf("turn off: %d", code)
	}
	c.do("GET", "/api/info", nil, &info)
	if info.Features["share"] || info.Features["plan"] || !info.Features["events"] {
		t.Fatalf("after: %v", info.Features)
	}
	if code := c.do("GET", "/s/"+link.Share.Token, nil, nil); code != 410 {
		t.Fatalf("an old link stops working while sharing is off: %d", code)
	}
	if code := c.do("POST", fmt.Sprintf("/api/recipes/%d/share", rc.ID), nil, nil); code != 403 {
		t.Fatalf("no new links: %d", code)
	}
	if code := c.do("DELETE", fmt.Sprintf("/api/recipes/%d/share", rc.ID), nil, nil); code >= 300 {
		t.Fatalf("stopping a link still works: %d", code)
	}
	if code := c.do("PUT", "/api/admin/settings", map[string]string{"feature_nope": "false"}, nil); code != 400 {
		t.Fatalf("unknown feature: %d", code)
	}

	if code := c.do("PUT", "/api/me/simpler", map[string]any{"plan_meals": []string{"dinner"}, "hidden_pages": []string{"events", "books"},
		"menu_order": []string{"add", "tonight", "kitchen"}}, nil); code != 204 {
		t.Fatalf("simpler: %d", code)
	}
	var me struct {
		PlanMeals   string `json:"plan_meals"`
		HiddenPages string `json:"hidden_pages"`
		MenuOrder   string `json:"menu_order"`
	}
	c.do("GET", "/api/me", nil, &me)
	if me.PlanMeals != "dinner" || me.HiddenPages != "events,books" || me.MenuOrder != "add,tonight,kitchen" {
		t.Fatalf("me: %+v", me)
	}
	for _, bad := range []map[string]any{{"plan_meals": []string{"elevenses"}}, {"hidden_pages": []string{"admin"}}, {"hidden_pages": []string{"tonight"}},
		{"menu_order": []string{"admin"}}, {"menu_order": []string{"plan", "plan"}}} {
		if code := c.do("PUT", "/api/me/simpler", bad, nil); code != 400 {
			t.Fatalf("%v: %d", bad, code)
		}
	}
	c.do("PUT", "/api/me/simpler", map[string]any{"plan_meals": []string{"breakfast", "lunch", "dinner", "snack"}}, nil)
	c.do("GET", "/api/me", nil, &me)
	if me.PlanMeals != "" || me.HiddenPages != "" {
		t.Fatalf("every meal is the default again: %+v", me)
	}
}
