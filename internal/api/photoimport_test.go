package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zachcurry13/recipebank/internal/store"
)

// fakeVisionAI answers like a home AI server with a small context: photos
// longer than maxSide are refused with llama.cpp's "exceeds the available
// context size" error; smaller ones get the card copied as text. A request
// without photos (organizing the copy) gets a recipe that lost a line.
func fakeVisionAI(t *testing.T, maxSide int, sides *[]int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var parts []struct {
			ImageURL *struct {
				URL string `json:"url"`
			} `json:"image_url"`
		}
		_ = json.Unmarshal(req.Messages[len(req.Messages)-1].Content, &parts)
		long := 0
		for _, part := range parts {
			if part.ImageURL == nil {
				continue
			}
			_, b64, _ := strings.Cut(part.ImageURL.URL, ",")
			data, _ := base64.StdEncoding.DecodeString(b64)
			cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				t.Errorf("AI got an unreadable photo: %v", err)
			}
			long = max(long, cfg.Width, cfg.Height)
		}
		w.Header().Set("Content-Type", "application/json")
		answer := `{"title":"Grandma's Biscuits","area":"kitchen","ingredients":[{"line":"2 cups flour"},{"line":"1 cup buttermilk","unsure":true}],"steps":[{"text":"Mix and bake 12 minutes."}]}`
		if long > 0 {
			*sides = append(*sides, long)
			answer = "Grandma's Biscuits (6 serv.)\nIngredients:\n2 c. flour\n1 c. buttermilk [?]\n1 T. baking powder\nInstructions:\nMix and bake 12 minutes."
		}
		if long > maxSide {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"code":400,"message":"request (4318 tokens) exceeds the available context size (4096 tokens), try increasing it","type":"exceed_context_size_error"}}`))
			return
		}
		resp, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": answer}}},
			"usage": map[string]int{"prompt_tokens": 900, "completion_tokens": 100}})
		w.Write(resp)
	}))
}

func photoBody(w, h int) map[string]any {
	return map[string]any{"images": []string{"data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpegBytes(w, h))}}
}

func useAI(st *store.Store, url string) {
	_ = st.SetSetting(store.KeyLLMProvider, "openai")
	_ = st.SetSetting(store.KeyLLMBaseURL, url)
	_ = st.SetSetting(store.KeyLLMModel, "test-vision")
	_ = st.SetSetting(store.KeyLLMJSONMode, "false")
}

func TestPhotoShrinksToFitSmallAI(t *testing.T) {
	c, srv := setup(t)
	var sides []int
	ai := fakeVisionAI(t, 896, &sides)
	defer ai.Close()
	useAI(srv.Store, ai.URL)

	var draft struct {
		Recipe map[string]any `json:"recipe"`
		Error  string         `json:"error"`
	}
	if code := c.do("POST", "/api/import/photo", photoBody(2000, 1500), &draft); code != 200 {
		t.Fatalf("photo import: %d %s (AI saw sides %v)", code, draft.Error, sides)
	}
	if draft.Recipe["title"] != "Grandma's Biscuits" || draft.Recipe["needs_review"] != true || draft.Recipe["servings"] != 6.0 {
		t.Fatalf("draft: %+v", draft.Recipe)
	}
	// The AI dropped the baking powder while organizing; the card's copy keeps it.
	if ings, _ := draft.Recipe["ingredients"].([]any); len(ings) != 3 {
		t.Fatalf("ingredients: %v", draft.Recipe["ingredients"])
	}
	if draft.Recipe["read_by"] != "test-vision" || !strings.Contains(draft.Recipe["ai_reading"].(string), "[?]") {
		t.Fatalf("reading not recorded: %q %q", draft.Recipe["read_by"], draft.Recipe["ai_reading"])
	}
	if len(sides) != 2 || sides[0] != 1280 || sides[1] != 896 {
		t.Fatalf("photo sizes sent to the AI: %v, want [1280 896]", sides)
	}
	if photos, _ := draft.Recipe["source_photos"].([]any); len(photos) != 1 {
		t.Fatalf("the card photo wasn't kept: %v", draft.Recipe["source_photos"])
	}
}

func TestPhotoTooBigForAnySizeSaysWhy(t *testing.T) {
	c, srv := setup(t)
	var sides []int
	ai := fakeVisionAI(t, 100, &sides)
	defer ai.Close()
	useAI(srv.Store, ai.URL)
	var out struct {
		Error string `json:"error"`
	}
	if code := c.do("POST", "/api/import/photo", photoBody(2000, 1500), &out); code != 502 || !strings.Contains(out.Error, "working memory") {
		t.Fatalf("got %d %q", code, out.Error)
	}
	if len(sides) != len(aiPhotoSides) {
		t.Fatalf("tried sizes %v", sides)
	}
}
