package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/config"
	"github.com/zachcurry13/recipebank/internal/db"
	"github.com/zachcurry13/recipebank/internal/store"
)

type client struct {
	t    *testing.T
	base string
	http *http.Client
}

// do sends a JSON request and decodes the answer into out (if not nil).
func (c *client) do(method, path string, body, out any) int {
	c.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, c.base+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-RecipeBank", "1")
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		_ = json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

// setup starts a server with a fresh database and a signed-in admin.
func setup(t *testing.T) (*client, *Server) {
	t.Helper()
	d, err := db.OpenDSN("file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	st := store.New(d)
	srv := &Server{Cfg: config.Config{}, Store: st, Auth: &auth.Manager{Store: st, SessionDays: 30},
		Web: fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}, PhotoDir: t.TempDir()}
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	c := &client{t: t, base: ts.URL, http: &http.Client{Jar: jar}}
	if code := c.do("POST", "/api/setup", map[string]string{"username": "admin", "password": "password123"}, nil); code != 201 {
		t.Fatalf("setup: %d", code)
	}
	return c, srv
}

const cookiePage = `<html><script type="application/ld+json">{"@type":"Recipe","name":"Butter Cookies",
"recipeIngredient":["2 cups flour","1 cup butter","1/2 cup sugar"],"recipeInstructions":"Mix.\nBake 12 minutes."}</script></html>`

func TestImportSaveAndCheck(t *testing.T) {
	c, srv := setup(t)
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(cookiePage)) }))
	defer site.Close()
	srv.Fetch = site.Client()

	var person store.Person
	c.do("POST", "/api/people", map[string]any{"name": "Kid", "heat_max": -1,
		"rules": []map[string]string{{"kind": "allergy", "key": "milk", "severity": "severe"}}}, &person)
	if person.ID == 0 || len(person.Rules) != 1 {
		t.Fatalf("person: %+v", person)
	}

	var draft struct {
		Recipe   map[string]any   `json:"recipe"`
		Verdicts []map[string]any `json:"verdicts"`
	}
	if code := c.do("POST", "/api/import/url", map[string]string{"url": site.URL}, &draft); code != 200 {
		t.Fatalf("import: %d", code)
	}
	if draft.Recipe["title"] != "Butter Cookies" || len(draft.Verdicts) != 1 || draft.Verdicts[0]["status"] != "no" {
		t.Fatalf("draft: %+v", draft)
	}

	var saved struct{ ID int64 }
	if code := c.do("POST", "/api/recipes", draft.Recipe, &saved); code != 200 || saved.ID == 0 {
		t.Fatalf("save: %d %+v", code, saved)
	}
	var list struct {
		Recipes []recipeCard `json:"recipes"`
	}
	c.do("GET", "/api/recipes?area=kitchen", nil, &list)
	if len(list.Recipes) != 1 || list.Recipes[0].Verdicts[0].Status != "no" {
		t.Fatalf("list: %+v", list)
	}
	c.do("GET", "/api/recipes?area=kitchen&ok=1", nil, &list)
	if len(list.Recipes) != 0 {
		t.Fatalf("ok=1 kept a recipe with milk: %+v", list)
	}
}

func TestKidsCantEdit(t *testing.T) {
	c, _ := setup(t)
	c.do("POST", "/api/admin/users", map[string]string{"username": "kid", "password": "password123", "role": "kid"}, nil)
	jar, _ := cookiejar.New(nil)
	kid := &client{t: t, base: c.base, http: &http.Client{Jar: jar}}
	if code := kid.do("POST", "/api/auth/login", map[string]string{"username": "kid", "password": "password123"}, nil); code != 200 {
		t.Fatalf("kid login: %d", code)
	}
	if code := kid.do("POST", "/api/recipes", map[string]string{"title": "x"}, nil); code != 403 {
		t.Fatalf("kid saved a recipe: %d", code)
	}
	if code := kid.do("GET", "/api/recipes", nil, nil); code != 200 {
		t.Fatalf("kid can't read recipes: %d", code)
	}
}

func TestCSRFHeaderRequired(t *testing.T) {
	c, _ := setup(t)
	req, _ := http.NewRequest("POST", c.base+"/api/people", bytes.NewBufferString(`{"name":"x"}`))
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("no X-RecipeBank header: %d", resp.StatusCode)
	}
}
