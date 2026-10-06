package api

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/zachcurry13/recipebank/internal/mail"
	"github.com/zachcurry13/recipebank/internal/store"
)

func TestEmailRecipeAndList(t *testing.T) {
	c, srv := setup(t)
	var sent []mail.Message
	srv.SendMail = func(_ mail.Config, m mail.Message) error { sent = append(sent, m); return nil }
	var rc struct{ ID int64 }
	c.do("POST", "/api/recipes", map[string]any{"title": "Grandma's <Crêpes>", "servings": 4, "notes": "Thin!",
		"ingredients": []map[string]string{{"line": "1 cup flour"}}, "steps": []map[string]string{{"text": "Whisk & fry."}}}, &rc)
	path := fmt.Sprintf("/api/recipes/%d/email", rc.ID)

	var info struct {
		EmailReady bool `json:"email_ready"`
	}
	c.do("GET", "/api/info", nil, &info)
	if info.EmailReady {
		t.Fatal("email isn't set up yet")
	}
	_ = srv.Store.SetSetting(store.KeySMTPHost, "mail.example.com")
	_ = srv.Store.SetSetting(store.KeySMTPFrom, "kitchen@example.com")
	c.do("GET", "/api/info", nil, &info)
	if !info.EmailReady {
		t.Fatal("email is set up now")
	}

	if code := c.do("POST", path, map[string]string{"to": "a@example.com, b@example.com"}, nil); code != 400 || len(sent) != 0 {
		t.Fatalf("two addresses: %d", code)
	}
	if code := c.do("POST", path, map[string]string{"to": "mom@example.com", "note": "Try these!"}, nil); code != 200 || len(sent) != 1 {
		t.Fatalf("send: %d", code)
	}
	m := sent[0]
	if m.To != "mom@example.com" || !strings.HasPrefix(m.Text, "Try these!") || !strings.Contains(m.Text, "- 1 cup flour") ||
		!strings.Contains(m.Text, "1. Whisk & fry.") || !strings.Contains(m.HTML, "Grandma&#39;s &lt;Crêpes&gt;") || strings.Contains(m.HTML, "<Crêpes>") {
		t.Fatalf("message: %+v", m)
	}

	if code := c.do("POST", "/api/shopping/email", map[string]string{"to": "dad@example.com"}, nil); code != 400 {
		t.Fatalf("an empty list: %d", code)
	}
	c.do("POST", "/api/shopping", map[string]string{"text": "2 lemons"}, nil)
	if code := c.do("POST", "/api/shopping/email", map[string]string{"to": "dad@example.com"}, nil); code != 200 ||
		!strings.Contains(sent[len(sent)-1].Text, "[ ] 2 lemons") {
		t.Fatalf("list: %d %+v", code, sent[len(sent)-1])
	}

	for i := len(sent); i < mailsPerHour; i++ {
		c.do("POST", path, map[string]string{"to": "mom@example.com"}, nil)
	}
	if code := c.do("POST", path, map[string]string{"to": "mom@example.com"}, nil); code != 429 {
		t.Fatalf("over the hourly limit: %d", code)
	}
	// Kids can send only when Admin → Email allows everyone.
	c.do("POST", "/api/admin/users", map[string]string{"username": "kid", "password": "password123", "role": "kid"}, nil)
	jar, _ := cookiejar.New(nil)
	kid := &client{t: t, base: c.base, http: &http.Client{Jar: jar}}
	kid.do("POST", "/api/auth/login", map[string]string{"username": "kid", "password": "password123"}, nil)
	if code := kid.do("POST", "/api/shopping/email", map[string]string{"to": "dad@example.com"}, nil); code != 403 {
		t.Fatalf("a kid emailed with parents only: %d", code)
	}
	_ = srv.Store.SetSetting(store.KeyEmailWho, "everyone")
	if code := kid.do("POST", "/api/shopping/email", map[string]string{"to": "dad@example.com"}, nil); code == 403 {
		t.Fatal("a kid can't email when everyone may")
	}

	var settings map[string]any
	_ = srv.Store.SetSetting(store.KeySMTPPassword, "hunter2")
	c.do("GET", "/api/admin/settings", nil, &settings)
	if _, sentOut := settings[store.KeySMTPPassword]; sentOut || settings[store.KeySMTPPassword+"_set"] != true {
		t.Fatalf("the mail password must stay on the server: %v", settings)
	}
}
