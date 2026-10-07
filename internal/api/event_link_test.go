package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// A guest opens the event's link (no account), says they're coming with a
// peanut allergy, sees the menu marked for them and adds what they bring.
// The host sees them and their allergies; the guest page shows nobody else's.
func TestEventGuestLink(t *testing.T) {
	c, _ := setup(t)
	var kid struct{ ID int64 }
	c.do("POST", "/api/people", map[string]any{"name": "Kiddo", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "allergic"}}}, &kid)
	var mac struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Mac and cheese", "ingredients": []map[string]string{{"line": "2 cups cheddar cheese"}, {"line": "8 oz macaroni"}}}, &mac)
	date := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	var ev struct{ ID int64 }
	c.do("POST", "/api/events", map[string]any{"name": "Potluck", "date": date, "who": []int64{kid.ID}}, &ev)
	c.do("POST", fmt.Sprintf("/api/events/%d/dishes", ev.ID), map[string]any{"recipe_id": mac.ID}, nil)
	c.do("POST", fmt.Sprintf("/api/events/%d/dishes", ev.ID), map[string]any{"title": "Dinner rolls"}, nil)

	var link struct{ Link struct{ Token string } }
	if code := c.do("POST", fmt.Sprintf("/api/events/%d/link", ev.ID), nil, &link); code != 200 || link.Link.Token == "" {
		t.Fatalf("link: %d", code)
	}
	guest := func(method, path, key string, body any) (int, []byte) {
		var rd io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rd = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, c.base+"/api/guest/"+link.Link.Token+path, rd)
		req.Header.Set("X-RecipeBank", "1")
		req.Header.Set("Content-Type", "application/json")
		if key != "" {
			req.Header.Set("X-Guest-Key", key)
		}
		resp, err := http.DefaultClient.Do(req) // no cookies: a guest has no account
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, raw
	}
	code, raw := guest("GET", "", "", nil)
	if code != 200 || strings.Contains(string(raw), "Kiddo") || strings.Contains(strings.ToLower(string(raw)), "allergic") {
		t.Fatalf("the guest page shows no family names or allergies: %d %s", code, raw)
	}
	code, raw = guest("PUT", "/me", "", map[string]any{"name": "Sam", "allergies": map[string]string{"peanut": "allergic"}})
	var joined struct{ Key string }
	json.Unmarshal(raw, &joined)
	if code != 200 || joined.Key == "" {
		t.Fatalf("join: %d %s", code, raw)
	}
	if code, _ := guest("POST", "/dishes", joined.Key, map[string]any{"title": "Peanut butter cookies", "contains": []string{"peanut", "wheat"}}); code != 200 {
		t.Fatalf("bring: %d", code)
	}
	if code, _ := guest("POST", "/dishes", "", map[string]any{"title": "Mystery"}); code != 403 {
		t.Fatalf("say you're coming first: %d", code)
	}
	_, raw = guest("GET", "", joined.Key, nil)
	var view struct {
		Coming int
		Dishes []struct {
			Title, Status string
			Mine          bool
			Contains      []string
		}
	}
	json.Unmarshal(raw, &view)
	status := map[string]string{}
	for _, d := range view.Dishes {
		status[d.Title] = d.Status
	}
	if view.Coming != 2 || status["Mac and cheese"] != "ok" || status["Dinner rolls"] != "unsure" || status["Peanut butter cookies"] != "no" {
		t.Fatalf("view: %+v", view)
	}
	if last := view.Dishes[2]; !last.Mine || len(last.Contains) != 2 {
		t.Fatalf("my dish: %+v", last)
	}

	var host struct {
		Coming []struct {
			ID     int64
			Name   string
			ByLink bool `json:"by_link"`
			Rules  string
		}
		Dishes []struct {
			Title    string
			Verdicts []struct {
				ID     int64
				Status string
			}
		}
	}
	c.do("GET", fmt.Sprintf("/api/events/%d", ev.ID), nil, &host)
	if len(host.Coming) != 2 || !host.Coming[1].ByLink || host.Coming[1].Rules != "Peanuts (allergic)" {
		t.Fatalf("host sees the guest: %+v", host.Coming)
	}
	for _, d := range host.Dishes {
		if d.Title == "Peanut butter cookies" && (d.Verdicts[0].Status != "unsure" || d.Verdicts[1].Status != "no") {
			t.Fatalf("checked for everyone: %+v", d)
		}
	}

	// Another guest can't take back Sam's dish.
	_, raw = guest("PUT", "/me", "", map[string]any{"name": "Alex"})
	var other struct{ Key string }
	json.Unmarshal(raw, &other)
	_, raw = guest("GET", "", joined.Key, nil)
	json.Unmarshal(raw, &view)
	var cookies int64
	var full struct {
		Dishes []struct {
			ID    int64
			Title string
		}
	}
	json.Unmarshal(raw, &full)
	for _, d := range full.Dishes {
		if d.Title == "Peanut butter cookies" {
			cookies = d.ID
		}
	}
	if code, _ := guest("DELETE", fmt.Sprintf("/dishes/%d", cookies), other.Key, nil); code != 404 {
		t.Fatalf("not yours: %d", code)
	}
	if code, _ := guest("DELETE", fmt.Sprintf("/dishes/%d", cookies), joined.Key, nil); code != 204 {
		t.Fatalf("yours: %d", code)
	}
	c.do("DELETE", fmt.Sprintf("/api/events/%d/link", ev.ID), nil, nil)
	if code, _ := guest("GET", "", "", nil); code != 410 {
		t.Fatalf("stopped: %d", code)
	}
	var past struct{ ID int64 }
	c.do("POST", "/api/events", map[string]any{"name": "Last year", "date": "2020-01-01"}, &past)
	if code := c.do("POST", fmt.Sprintf("/api/events/%d/link", past.ID), nil, nil); code != 400 {
		t.Fatalf("a past event gets no link: %d", code)
	}
}
