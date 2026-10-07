package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// Ollama says which models read photos: newer ones list a "vision"
// capability, older ones show a picture reader in the model's details.
func TestCanSee(t *testing.T) {
	shows := map[string]string{
		"qwen2.5vl:3b":    `{"capabilities":["completion","vision"],"details":{"families":["qwen25vl"]}}`,
		"qwen2.5:3b":      `{"capabilities":["completion","tools"],"details":{"families":["qwen2"]}}`,
		"llava:7b":        `{"details":{"families":["llama","clip"]}}`,
		"llama3.2-vision": `{"details":{"families":["mllama"]}}`,
		"old-text":        `{"details":{"families":["llama"]}}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
			fmt.Fprint(w, `{"version":"0.9.0"}`)
		case "/api/show":
			var body struct{ Model string }
			json.NewDecoder(r.Body).Decode(&body)
			if s, ok := shows[body.Model]; ok {
				fmt.Fprint(w, s)
				return
			}
			http.Error(w, `{"error":"model not found"}`, http.StatusNotFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	ctx := context.Background()
	names := []string{"qwen2.5vl:3b", "qwen2.5:3b", "llava:7b", "llama3.2-vision", "old-text", "missing"}
	see := CanSee(ctx, srv.URL, names)
	want := map[string]bool{"qwen2.5vl:3b": true, "qwen2.5:3b": false, "llava:7b": true, "llama3.2-vision": true, "old-text": false}
	if len(see) != len(want) {
		t.Fatalf("got %v", see)
	}
	for m, w := range want {
		if see[m] != w {
			t.Errorf("%s: %v", m, see[m])
		}
	}
	if got := Photo(ctx, srv.URL, names); !slices.Equal(got, []string{"qwen2.5vl:3b", "llava:7b", "llama3.2-vision"}) {
		t.Errorf("photo models: %v", got)
	}
	if !IsOllama(ctx, srv.URL) || IsOllama(ctx, srv.URL+"/v1beta/openai") {
		t.Error("only Ollama answers like Ollama")
	}
}
