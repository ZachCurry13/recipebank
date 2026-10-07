package llm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zachcurry13/recipebank/internal/db"
	"github.com/zachcurry13/recipebank/internal/store"
)

const (
	geminiBusy = `LLM 503 Service Unavailable: [{ "error": { "code": 503, "message": "This model is currently experiencing high demand.", "status": "UNAVAILABLE" } }]`
	clipFailed = `LLM error: llama-server process has terminated: error: Failed to load CLIP model from /root/.ollama/models/blobs/sha256-e9758e589d443f653821b7be9bb9092c1bf7434522b70ec6e83591b1320fdb4d`
)

// Each kind of failure is said plainly, with what to do.
func TestExplain(t *testing.T) {
	for _, c := range []struct{ err, want string }{
		{geminiBusy, "busy right now"},
		{"Claude API error 529", "busy right now"},
		{`LLM 429 Too Many Requests: {"error":{"status":"RESOURCE_EXHAUSTED"}}`, "used up its allowance"},
		{clipFailed, "picture reader"},
		{"llama runner process has terminated: vk::PhysicalDevice::createDevice: ErrorInitializationFailed", "Restart the Ollama app"},
	} {
		msg, ok := Explain(errors.New(c.err))
		if !ok || !strings.Contains(msg, c.want) || !strings.Contains(msg, "It said:") {
			t.Errorf("%q: %v %q", c.err, ok, msg)
		}
	}
	if _, ok := Explain(errors.New("LLM returned no choices")); ok {
		t.Error("nothing plainer to say about an unknown problem")
	}
	both, ok := Explain(&BothFailed{Photo: errors.New(geminiBusy), Main: errors.New(clipFailed)})
	if !ok || !strings.Contains(both, "The AI for photos is busy") || !strings.Contains(both, "The main AI couldn't load the photo model's picture reader") {
		t.Errorf("both AIs: %q", both)
	}
	if Busy(errors.New(clipFailed)) || Limited(errors.New(clipFailed)) {
		t.Error("numbers inside a file name aren't a busy or limited answer")
	}
}

func testStore(t *testing.T) *store.Store {
	t.Helper()
	d, err := db.OpenDSN("file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return store.New(d)
}

// An online AI that's busy for a moment is asked once more; other problems
// aren't repeated.
func TestBusyAskedAgain(t *testing.T) {
	old := busyWait
	busyWait = 0
	t.Cleanup(func() { busyWait = old })
	var calls atomic.Int32
	status := http.StatusServiceUnavailable
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(status)
			w.Write([]byte(`[{"error":{"code":503,"message":"This model is currently experiencing high demand."}}]`))
			return
		}
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"fine"}}]}`))
	}))
	defer srv.Close()
	st := testStore(t)
	ai := store.AIConfig{Provider: "openai", BaseURL: srv.URL, Models: []string{"gemini-3.8-flash"}}
	accept := func(string) error { return nil }
	if _, err := AskWith(context.Background(), st, ai, "sys", "hi", accept); err != nil || calls.Load() != 2 {
		t.Fatalf("busy: %v after %d calls", err, calls.Load())
	}

	calls.Store(0)
	status = http.StatusInternalServerError
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(status)
		w.Write([]byte(`{"error":{"message":"` + strings.ReplaceAll(clipFailed, "LLM error: ", "") + `"}}`))
	})
	_, err := AskWith(context.Background(), st, ai, "sys", "hi", accept)
	if err == nil || calls.Load() != 1 || !NoPictures(err) {
		t.Fatalf("a model that won't load: %v after %d calls", err, calls.Load())
	}
}
