package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zachcurry13/recipebank/internal/llm"
)

// busyAI answers like Gemini when it's overloaded.
func busyAI() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`[{"error":{"code":503,"message":"This model is currently experiencing high demand.","status":"UNAVAILABLE"}}]`))
	}))
}

// A card no AI can read right now goes on the "Read it later" list with its
// photos; it's read in the background once an AI works, then opened as a
// draft to check, and taken off the list.
func TestReadItLater(t *testing.T) {
	defer func(old time.Duration) { llm.BusyWait = old }(llm.BusyWait)
	llm.BusyWait = 0
	c, srv := setup(t)
	busy := busyAI()
	defer busy.Close()
	useAI(srv.Store, busy.URL)

	var failed struct {
		Error  string
		Later  bool
		Photos []string
	}
	if code := c.do("POST", "/api/import/photo", photoBody(600, 400), &failed); code != 502 || !failed.Later || len(failed.Photos) != 1 ||
		!strings.Contains(failed.Error, "busy") {
		t.Fatalf("busy AI: %d %+v", code, failed)
	}
	var added struct{ ID int64 }
	src := map[string]any{"pile": 3, "title": "Biscuits card"}
	if code := c.do("POST", "/api/later", map[string]any{"kind": "photo", "photos": failed.Photos, "source": src}, &added); code != 200 {
		t.Fatalf("add: %d", code)
	}
	if code := c.do("POST", "/api/later", map[string]any{"kind": "photo", "photos": []string{"0123456789abcdef0123456789abcdef.jpg"}}, nil); code != 400 {
		t.Fatalf("photos that aren't here: %d", code)
	}
	used, _ := srv.Store.PhotosInUse()
	if !used[failed.Photos[0]] {
		t.Fatal("the daily clean-up must keep a waiting card's photos")
	}

	// Still busy: it waits for another try, saying why.
	now := time.Now()
	if srv.readNextLater(context.Background(), now.Add(3*time.Minute)) {
		t.Fatal("stop for now when the AI is still busy")
	}
	var list struct {
		Items []struct {
			ID                   int64
			Status, Error, Title string
			Tries                int
			Source               string
		}
	}
	c.do("GET", "/api/later", nil, &list)
	if len(list.Items) != 1 || list.Items[0].Status != "waiting" || list.Items[0].Tries != 1 || !strings.Contains(list.Items[0].Error, "busy") {
		t.Fatalf("after a busy try: %+v", list.Items)
	}

	var sides []int
	good := fakeVisionAI(t, 4000, &sides)
	defer good.Close()
	useAI(srv.Store, good.URL)
	if srv.readNextLater(context.Background(), now.Add(4*time.Minute)) {
		t.Fatal("not its turn yet (5 minutes after the last try)")
	}
	c.do("POST", fmt.Sprintf("/api/later/%d/now", added.ID), nil, nil) // Try now
	if !srv.readNextLater(context.Background(), time.Now().Add(time.Second)) {
		t.Fatal("read with a working AI")
	}
	c.do("GET", "/api/later", nil, &list)
	if it := list.Items[0]; it.Status != "ready" || it.Title != "Grandma's Biscuits" || !strings.Contains(it.Source, "Biscuits card") {
		t.Fatalf("ready: %+v", it)
	}
	var draft struct {
		Recipe struct {
			Title        string
			SourceKind   string   `json:"source_kind"`
			SourcePhotos []string `json:"source_photos"`
		}
		Verdicts []any
	}
	if code := c.do("GET", fmt.Sprintf("/api/later/%d", added.ID), nil, &draft); code != 200 || draft.Recipe.Title != "Grandma's Biscuits" ||
		draft.Recipe.SourceKind != "photo" || len(draft.Recipe.SourcePhotos) != 1 || draft.Recipe.SourcePhotos[0] != failed.Photos[0] {
		t.Fatalf("the draft: %d %+v", code, draft)
	}
	if code := c.do("DELETE", fmt.Sprintf("/api/later/%d", added.ID), nil, nil); code != 200 {
		t.Fatalf("delete: %d", code)
	}
	if c.do("GET", "/api/later", nil, &list); len(list.Items) != 0 {
		t.Fatalf("after saving: %+v", list.Items)
	}
}

// Pasted text waits too while the AI's computer is off; after two days of
// trying it stops and says so. A page with no recipe isn't offered for later.
func TestReadItLaterGivesUp(t *testing.T) {
	c, srv := setup(t)
	off := httptest.NewServer(http.NotFoundHandler())
	off.Close() // a computer that's switched off
	useAI(srv.Store, off.URL)
	var failed struct {
		Later bool
		Error string
	}
	if code := c.do("POST", "/api/import/text", map[string]string{"text": "Grandma's soup, a little of this and that"}, &failed); code != 502 || !failed.Later {
		t.Fatalf("AI off: %d %+v", code, failed)
	}
	var added struct{ ID int64 }
	c.do("POST", "/api/later", map[string]any{"kind": "text", "text": "Grandma's soup, a little of this and that"}, &added)
	srv.readNextLater(context.Background(), time.Now().Add(3*time.Minute))
	srv.readNextLater(context.Background(), time.Now().Add(49*time.Hour))
	l, _ := srv.Store.LaterByID(added.ID)
	if l.Status != "failed" || !strings.Contains(l.Error, "two days") || l.Tries != 2 {
		t.Fatalf("gave up: %+v", l)
	}

	var calls atomic.Int32
	noRecipe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"no_recipe\":true}"}}]}`))
	}))
	defer noRecipe.Close()
	useAI(srv.Store, noRecipe.URL)
	var none struct{ Later bool }
	if code := c.do("POST", "/api/import/text", map[string]string{"text": "Nice weather today"}, &none); code == 200 || none.Later || calls.Load() == 0 {
		t.Fatalf("no recipe isn't for later: %d %+v", code, none)
	}
}
