package api

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// The meal plan as a calendar link: planned meals at the person's times with
// a link to the recipe, events all day; only the meals they plan; a new link
// stops the old one.
func TestCalendarFeed(t *testing.T) {
	c, _ := setup(t)
	tomorrow := time.Now().AddDate(0, 0, 1)
	date := tomorrow.Format(day)
	var soup struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Soup, the family's", "ingredients": []map[string]string{{"line": "1 onion"}}}, &soup)
	c.do("POST", "/api/plan", map[string]any{"date": date, "meal": "dinner", "recipe_id": soup.ID, "note": "Double it; Grandma comes"}, nil)
	c.do("POST", "/api/plan", map[string]any{"date": date, "meal": "breakfast", "title": "Pancakes"}, nil)
	c.do("POST", "/api/events", map[string]any{"name": "Potluck", "date": tomorrow.AddDate(0, 0, 3).Format(day)}, nil)

	var cal struct {
		On     bool
		Path   string
		Times  map[string]string
		Events bool
	}
	c.do("GET", "/api/me/calendar", nil, &cal)
	if cal.On || cal.Times["dinner"] != "18:00" {
		t.Fatalf("before: %+v", cal)
	}
	c.do("POST", "/api/me/calendar", nil, &cal)
	if !cal.On || !strings.HasPrefix(cal.Path, "/cal/") {
		t.Fatalf("made: %+v", cal)
	}
	feed := func(path string) (int, string) {
		resp, err := http.Get(c.base + path) // a calendar app: no sign-in
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	code, ics := feed(cal.Path)
	ymd := tomorrow.Format("20060102")
	for _, want := range []string{"BEGIN:VCALENDAR", "SUMMARY:🍽 Dinner: Soup\\, the family's", "DTSTART:" + ymd + "T180000",
		"Double it\\; Grandma comes", "/#/recipe/", "SUMMARY:🍳 Breakfast: Pancakes", "DTSTART:" + ymd + "T080000",
		"SUMMARY:🎉 Potluck", "DTSTART;VALUE=DATE:" + tomorrow.AddDate(0, 0, 3).Format("20060102")} {
		if !strings.Contains(strings.ReplaceAll(ics, "\r\n ", ""), want) {
			t.Errorf("feed is missing %q", want)
		}
	}
	if code != 200 || !strings.Contains(ics, "\r\nEND:VCALENDAR\r\n") {
		t.Fatalf("feed: %d", code)
	}

	// Dinner at 19:30, no events; and only dinners on their plan.
	c.do("PUT", "/api/me/calendar", map[string]any{"times": map[string]string{"dinner": "19:30", "lunch": "25:00"}, "events": false}, &cal)
	if cal.Times["dinner"] != "19:30" || cal.Times["lunch"] != "12:00" || cal.Events {
		t.Fatalf("times: %+v", cal)
	}
	c.do("PUT", "/api/me/simpler", map[string]any{"plan_meals": []string{"dinner"}}, nil)
	_, ics = feed(cal.Path)
	if !strings.Contains(ics, "DTSTART:"+ymd+"T193000") || strings.Contains(ics, "Pancakes") || strings.Contains(ics, "Potluck") {
		t.Fatalf("dinner only, at 19:30, no events:\n%s", ics)
	}

	old := cal.Path
	c.do("POST", "/api/me/calendar", nil, &cal)
	if code, _ := feed(old); code != 404 || cal.Path == old || cal.Times["dinner"] != "19:30" {
		t.Fatalf("a new link stops the old one: %d %+v", code, cal)
	}
	c.do("DELETE", "/api/me/calendar", nil, nil)
	if code, _ := feed(cal.Path); code != 404 {
		t.Fatalf("turned off: %d", code)
	}
}

// Long lines fold at 75 bytes without splitting a character; text is escaped.
func TestICSText(t *testing.T) {
	long := strings.Repeat("🍲 Soup ", 20)
	out := buildICS("X", []calItem{{UID: "a", Date: "2026-10-08", Time: "18:00", Summary: long}}, time.Now())
	for _, l := range strings.Split(out, "\r\n") {
		if len(l) > 75 {
			t.Errorf("line too long (%d): %q", len(l), l)
		}
		if strings.ToValidUTF8(l, "?") != l {
			t.Errorf("a character was split: %q", l)
		}
	}
	if icsText("a,b;c\\d\ne") != `a\,b\;c\\d\ne` {
		t.Error(icsText("a,b;c\\d\ne"))
	}
}
