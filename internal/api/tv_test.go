package api

import (
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
)

// A TV opens /tv and gets a code; a signed-in phone enters it and sends steps.
// Only the screen with the secret can read them; the phone can't send before
// pairing, and wrong codes are refused.
func TestKitchenScreen(t *testing.T) {
	c, _ := setup(t)
	var tv struct{ Code, Secret string }
	if code := c.do("POST", "/api/tv/new", nil, &tv); code != 200 || len(tv.Code) != 6 || tv.Secret == "" {
		t.Fatalf("new screen: %d %+v", code, tv)
	}
	var st struct {
		Paired  bool
		View    *tvView
		Version int
	}
	read := func(secret string) int {
		st = struct {
			Paired  bool
			View    *tvView
			Version int
		}{}
		return c.do("GET", "/api/tv/screen?code="+strings.ToLower(tv.Code)+"&secret="+secret, nil, &st)
	}
	if read("wrong") != 404 {
		t.Fatal("a screen's secret is needed to read it")
	}
	if read(tv.Secret) != 200 || st.Paired || st.View != nil {
		t.Fatalf("waiting: %+v", st)
	}
	view := tvView{Title: "Soup", At: 1, Total: 3, Text: "Chop the onion.", Items: []tvItem{{Amount: "1", Name: "onion"}},
		Timers: []tvTimer{{Label: "Simmer", Left: 600}}}
	if code := c.do("POST", "/api/tv/show", map[string]any{"code": tv.Code, "view": view}, nil); code != 409 {
		t.Fatalf("show before pairing: %d", code)
	}
	if code := c.do("POST", "/api/tv/pair", map[string]string{"code": "NOPE12"}, nil); code != 404 {
		t.Fatalf("wrong code: %d", code)
	}
	if code := c.do("POST", "/api/tv/pair", map[string]string{"code": tv.Code}, nil); code != 200 {
		t.Fatalf("pair: %d", code)
	}
	view.Text = strings.Repeat("x", 5000)
	c.do("POST", "/api/tv/show", map[string]any{"code": tv.Code, "view": view}, nil)
	if read(tv.Secret); !st.Paired || st.View == nil || st.View.Title != "Soup" || len([]rune(st.View.Text)) != 2000 ||
		st.View.Timers[0].Left != 600 || st.Version < 2 {
		t.Fatalf("shown: %+v", st)
	}
	c.do("DELETE", "/api/tv/"+tv.Code, nil, nil)
	if read(tv.Secret); st.View != nil || !st.Paired {
		t.Fatalf("cleared: %+v", st)
	}
	// Signed out, a phone can't pair or send.
	jar, _ := cookiejar.New(nil)
	anon := &client{t: t, base: c.base, http: &http.Client{Jar: jar}}
	if code := anon.do("POST", "/api/tv/pair", map[string]string{"code": tv.Code}, nil); code != 401 {
		t.Fatalf("pairing needs a sign-in: %d", code)
	}
}
