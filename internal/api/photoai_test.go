package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/zachcurry13/recipebank/internal/store"
)

// receiptAI answers every question with one receipt line named after it,
// and counts the photos it was shown.
func receiptAI(name string, photos *int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content any `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		answer := `{"ok": true}`
		if len(req.Messages) > 0 {
			if _, isParts := req.Messages[len(req.Messages)-1].Content.([]any); isParts {
				atomic.AddInt32(photos, 1)
				answer = `{"items":[{"name":"` + name + `","qty":1,"price":1,"area":"kitchen"}]}`
			}
		}
		resp, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": answer}}}})
		w.Write(resp)
	}))
}

func usePhotoAI(st *store.Store, url string) {
	_ = st.SetSetting(store.KeyPhotoProvider, "openai")
	_ = st.SetSetting(store.KeyPhotoBaseURL, url)
	_ = st.SetSetting(store.KeyPhotoModel, "big-vision")
	_ = st.SetSetting(store.KeyPhotoJSONMode, "false")
}

// Photos go to the photo AI first; when it's off, the main AI reads them, and
// the photo AI is skipped for a while instead of being waited for each time.
func TestPhotoAI(t *testing.T) {
	c, srv := setup(t)
	var mainPhotos, bigPhotos int32
	main := receiptAI("from main", &mainPhotos)
	defer main.Close()
	big := receiptAI("from big", &bigPhotos)
	useAI(srv.Store, main.URL)
	usePhotoAI(srv.Store, big.URL)

	read := func() string {
		var res struct {
			Lines []struct{ Name string } `json:"lines"`
		}
		if code := c.do("POST", "/api/stock/receipt", photoBody(600, 900), &res); code != 200 || len(res.Lines) != 1 {
			t.Fatalf("receipt: %d %+v", code, res)
		}
		return res.Lines[0].Name
	}
	if got := read(); got != "from big" || bigPhotos != 1 || mainPhotos != 0 {
		t.Fatalf("the photo AI should read it: %q %d %d", got, bigPhotos, mainPhotos)
	}
	var test struct{ Model string }
	if code := c.do("POST", "/api/admin/ai/test?which=photo", nil, &test); code != 200 || test.Model != "big-vision" {
		t.Fatalf("test the photo AI: %d %+v", code, test)
	}

	big.Close() // the computer is switched off
	if got := read(); got != "from main" || mainPhotos != 1 {
		t.Fatalf("the main AI should step in: %q %d", got, mainPhotos)
	}
	if got := read(); got != "from main" || mainPhotos != 2 || bigPhotos != 1 {
		t.Fatalf("still the main AI: %q %d %d", got, mainPhotos, bigPhotos)
	}

	// With only a photo AI, it does the text work too.
	only := receiptAI("only", &bigPhotos)
	defer only.Close()
	_ = srv.Store.SetSetting(store.KeyLLMModel, "")
	usePhotoAI(srv.Store, only.URL)
	var info struct {
		AIReady bool `json:"ai_ready"`
	}
	c.do("GET", "/api/info", nil, &info)
	if !info.AIReady {
		t.Fatal("a photo AI alone counts as an AI")
	}
	if code := c.do("POST", "/api/admin/ai/test", nil, nil); code == 200 {
		t.Fatal("the main AI isn't set up, so its test can't pass")
	}
	if got := read(); got != "only" {
		t.Fatalf("photo AI alone: %q", got)
	}
}
