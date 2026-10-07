package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Gemini's thinking models are asked to think little and get room to answer,
// so a photo doesn't run past the time limit; other servers get neither.
func TestGeminiThinksLittle(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = map[string]any{}
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"ok\":true}"}}]}`))
	}))
	defer srv.Close()

	gemini := &Client{BaseURL: srv.URL + "/generativelanguage.googleapis.com/v1beta/openai"}
	if _, _, err := gemini.ReadImages(context.Background(), "gemini-3.8-flash", "read it", []Image{{Data: []byte("x"), MediaType: "image/jpeg"}}); err != nil {
		t.Fatal(err)
	}
	if got["reasoning_effort"] != "low" || got["max_tokens"].(float64) < geminiMinTokens {
		t.Fatalf("gemini: %v %v", got["reasoning_effort"], got["max_tokens"])
	}
	other := &Client{BaseURL: srv.URL + "/v1"}
	if _, _, err := other.Complete(context.Background(), "qwen2.5:3b", "sys", "hi"); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["reasoning_effort"]; ok || got["max_tokens"].(float64) >= geminiMinTokens {
		t.Fatalf("others: %v", got)
	}
}
