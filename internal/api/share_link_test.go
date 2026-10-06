package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestShareLink(t *testing.T) {
	c, srv := setup(t)
	c.do("POST", "/api/people", map[string]any{"name": "Kiddo", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "severe"}}}, nil)
	var photo, card struct{ Name string }
	img := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpegBytes(40, 30))
	c.do("POST", "/api/photos", map[string]string{"image": img}, &photo)
	c.do("POST", "/api/photos", map[string]string{"image": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpegBytes(30, 40))}, &card)
	var rc struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Aunt's <b>pie</b>", "photo": photo.Name, "source_photos": []string{card.Name},
		"notes": "Bake on the low rack.", "ingredients": []map[string]string{{"line": "2 cups milk"}}, "steps": []map[string]string{{"text": "Bake."}}}, &rc)
	path := fmt.Sprintf("/api/recipes/%d/share", rc.ID)
	var a, b struct {
		Share struct{ Token, ExpiresAt string }
	}
	c.do("POST", path, map[string]any{}, &a)
	c.do("POST", path, map[string]any{}, &b)
	if a.Share.Token == "" || len(a.Share.Token) < 20 || b.Share.Token != a.Share.Token {
		t.Fatalf("links: %+v %+v", a, b)
	}

	public := &http.Client{} // no sign-in
	get := func(p string) (int, string, http.Header) {
		resp, err := public.Get(c.base + p)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(body), resp.Header
	}
	code, page, h := get("/s/" + a.Share.Token)
	if code != 200 || !strings.Contains(page, "Aunt&#39;s &lt;b&gt;pie&lt;/b&gt;") || !strings.Contains(page, "2 cups milk") ||
		!strings.Contains(page, "Bake on the low rack.") || h.Get("X-Robots-Tag") == "" {
		t.Fatalf("page %d: %s", code, page)
	}
	if strings.Contains(page, card.Name) || strings.Contains(page, "Kiddo") || strings.Contains(page, "allergy") && strings.Contains(page, "milk:") {
		t.Fatal("the page shows card photos or people")
	}
	if code, _, _ := get("/s/" + a.Share.Token + "/photo"); code != 200 {
		t.Fatalf("photo: %d", code)
	}
	if code, _, _ := get("/s/nope"); code != 410 {
		t.Fatalf("unknown link: %d", code)
	}

	// Expired, then stopped.
	_, _ = srv.Store.DB.Exec(`UPDATE shares SET expires_at = '2000-01-01T00:00:00Z'`)
	if code, page, _ := get("/s/" + a.Share.Token); code != 410 || !strings.Contains(page, "run out") {
		t.Fatalf("expired: %d", code)
	}
	c.do("POST", path, map[string]any{}, &b)
	if b.Share.Token == a.Share.Token {
		t.Fatal("an expired link was handed out again")
	}
	c.do("DELETE", path, nil, nil)
	if code, _, _ := get("/s/" + b.Share.Token); code != 410 {
		t.Fatalf("after stop sharing: %d", code)
	}
}
