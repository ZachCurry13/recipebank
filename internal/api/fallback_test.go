package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zachcurry13/recipebank/internal/store"
)

// ollamaBroken answers like Ollama when a model won't load on the graphics card.
func ollamaBroken() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"llama-server process has terminated: error loading model: vk::PhysicalDevice::createDevice: ErrorInitializationFailed"}}`))
	}))
}

// When the main AI's model won't load, the photo AI organizes the text
// instead; when there's no other AI, the person is told what to do.
func TestMainAIFallsBackToPhotoAI(t *testing.T) {
	c, srv := setup(t)
	broken := ollamaBroken()
	defer broken.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer := `{"title":"Toast","ingredients":[{"line":"2 slices bread"}],"steps":[{"text":"Toast it."}]}`
		resp, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": answer}}}})
		w.Write(resp)
	}))
	defer good.Close()
	useAI(srv.Store, broken.URL)
	paste := map[string]string{"text": "Toast\n2 slices bread\nToast it."}

	var bad struct{ Error string }
	if code := c.do("POST", "/api/import/text", paste, &bad); code != 502 || !strings.Contains(bad.Error, "Restart the Ollama app") {
		t.Fatalf("without another AI: %d %q", code, bad.Error)
	}
	if code := c.do("POST", "/api/admin/ai/test", nil, &bad); code != 502 || !strings.Contains(bad.Error, "graphics card") {
		t.Fatalf("Test the AI says it plainly too: %d %q", code, bad.Error)
	}

	usePhotoAI(srv.Store, good.URL)
	_ = srv.Store.SetSetting(store.KeyPhotoModel, "big")
	var draft struct{ Recipe struct{ Title string } }
	if code := c.do("POST", "/api/import/text", paste, &draft); code != 200 || draft.Recipe.Title != "Toast" {
		t.Fatalf("the photo AI steps in: %d %+v", code, draft)
	}
}
