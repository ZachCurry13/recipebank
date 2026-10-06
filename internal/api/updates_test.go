package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zachcurry13/recipebank/internal/updates"
	"github.com/zachcurry13/recipebank/internal/version"
)

// A newer release on GitHub shows for parents with the bundled changelog;
// "seen" is kept per person; admins' phones hear about each release once.
func TestUpdatesSeenAndGuide(t *testing.T) {
	old := version.Version
	version.Version = "0.2.0"
	defer func() { version.Version = old }()
	c, srv := setup(t)
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"tag_name":"v0.3.0","name":"0.3.0","body":"## New\n- Substitute","html_url":"https://example.invalid/0.3.0"}]`))
	}))
	defer gh.Close()
	srv.Updates = &updates.Checker{URL: gh.URL, HTTP: gh.Client()}

	var up struct {
		Status struct {
			Current         string `json:"current"`
			Latest          string `json:"latest"`
			UpdateAvailable bool   `json:"update_available"`
		} `json:"status"`
		Changelog string `json:"changelog"`
	}
	c.do("GET", "/api/updates", nil, &up)
	if !up.Status.UpdateAvailable || up.Status.Latest != "v0.3.0" || !strings.Contains(up.Changelog, "## [0.2.0]") {
		t.Fatalf("updates: %+v (changelog %d bytes)", up.Status, len(up.Changelog))
	}

	var me struct {
		SeenVersion string `json:"seen_version"`
	}
	c.do("GET", "/api/me", nil, &me)
	if me.SeenVersion != "" {
		t.Fatalf("a new admin hasn't seen the welcome: %q", me.SeenVersion)
	}
	c.do("PUT", "/api/me/seen", map[string]any{}, nil)
	c.do("GET", "/api/me", nil, &me)
	if me.SeenVersion != "0.2.0" {
		t.Fatalf("seen: %q", me.SeenVersion)
	}

	type guide struct {
		Steps []struct {
			Key  string
			Done bool
		}
		Hidden bool
	}
	var g guide
	c.do("GET", "/api/admin/guide", nil, &g)
	done := func() string {
		var ks []string
		for _, s := range g.Steps {
			if s.Done {
				ks = append(ks, s.Key)
			}
		}
		return strings.Join(ks, ",")
	}
	if len(g.Steps) != 6 || done() != "admin" || g.Hidden {
		t.Fatalf("a new install's guide: %+v", g)
	}
	c.do("POST", "/api/recipes", map[string]any{"title": "Toast", "ingredients": []map[string]string{{"line": "1 slice bread"}}}, nil)
	c.do("PUT", "/api/admin/guide", map[string]any{"step": "phones", "done": true}, &g)
	if done() != "admin,recipe,phones" {
		t.Fatalf("after a recipe and ticking phones: %s", done())
	}
	c.do("PUT", "/api/admin/guide", map[string]any{"step": "hidden", "done": true}, &g)
	if !g.Hidden || done() != "admin,recipe,phones" {
		t.Fatalf("hidden: %+v", g)
	}
	if code := c.do("PUT", "/api/admin/guide", map[string]any{"step": "nope", "done": true}, nil); code != 400 {
		t.Fatalf("unknown step: %d", code)
	}

	srv.notifyUpdate()
	if got := srv.Store.Setting("update_notified"); got != "v0.3.0" {
		t.Fatalf("update_notified = %q", got)
	}
}
