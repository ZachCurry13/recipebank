package api

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zachcurry13/recipebank/internal/store"
)

// Each AI is checked by asking for its models (no tokens spent): a missing
// model and a computer that's off are both said plainly. Costs come from
// each AI's prices.
func TestAIHealthAndCost(t *testing.T) {
	c, srv := setup(t)
	main := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"data":[{"id":"qwen2.5:3b"},{"id":"qwen2.5vl:3b"}]}`))
	}))
	defer main.Close()
	off := httptest.NewServer(http.NotFoundHandler())
	offURL := off.URL
	off.Close() // the PC is switched off
	for k, v := range map[string]string{store.KeyLLMProvider: "openai", store.KeyLLMBaseURL: main.URL + "/v1", store.KeyLLMModel: "qwen2.5:3b",
		store.KeyLLMVisionModel: "qwen2.5vl:7b", store.KeyPhotoProvider: "openai", store.KeyPhotoBaseURL: offURL + "/v1",
		store.KeyPhotoModel: "big-vision", store.KeyLLMPriceIn: "1", store.KeyLLMPriceOut: "2", store.KeyPhotoPriceIn: "10"} {
		srv.Store.SetSetting(k, v)
	}
	srv.Store.RecordUsage("qwen2.5:3b", 1_000_000, 500_000, time.Second)
	srv.Store.RecordUsage("big-vision", 100_000, 0, time.Second)

	var res struct {
		AIs []struct {
			Which, Error string
			OK, Checked  bool
			Missing      []string
		}
		Cost store.AICost
	}
	if code := c.do("GET", "/api/admin/ai/health", nil, &res); code != 200 || len(res.AIs) != 2 {
		t.Fatalf("health: %d %+v", code, res)
	}
	m, p := res.AIs[0], res.AIs[1]
	if m.OK || len(m.Missing) != 1 || m.Missing[0] != "qwen2.5vl:7b" {
		t.Fatalf("main AI is missing its photo model: %+v", m)
	}
	if p.OK || p.Error != "can't be reached: is that computer on?" {
		t.Fatalf("photo AI is off: %+v", p)
	}
	if math.Abs(res.Cost.Main-2) > 0.001 || math.Abs(res.Cost.Photo-1) > 0.001 || math.Abs(res.Cost.Total-3) > 0.001 {
		t.Fatalf("cost: %+v", res.Cost)
	}
}

// "List models" asks the service (with the saved key when none is typed) and
// leaves out models that can't answer questions.
func TestListModels(t *testing.T) {
	c, srv := setup(t)
	var auth string
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.Write([]byte(`{"data":[{"id":"models/gemini-3.8-flash"},{"id":"models/text-embedding-004"},{"id":"models/imagen-4"},{"id":"models/gemini-3.8-flash-lite"}]}`))
	}))
	defer svc.Close()
	srv.Store.SetSetting(store.KeyPhotoAPIKey, "saved-key")
	var res struct{ Models []string }
	if code := c.do("POST", "/api/admin/ai/models", map[string]string{"which": "photo", "provider": "openai", "base_url": svc.URL + "/v1"}, &res); code != 200 {
		t.Fatalf("list: %d", code)
	}
	if auth != "Bearer saved-key" || len(res.Models) != 2 || res.Models[0] != "gemini-3.8-flash" {
		t.Fatalf("models: %q %v", auth, res.Models)
	}
	if code := c.do("POST", "/api/admin/ai/models", map[string]string{"which": "llm", "provider": "openai"}, nil); code != 400 {
		t.Fatalf("no address: %d", code)
	}
}
