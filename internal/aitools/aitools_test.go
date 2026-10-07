package aitools

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zachcurry13/recipebank/internal/db"
	"github.com/zachcurry13/recipebank/internal/ollama"
	"github.com/zachcurry13/recipebank/internal/store"
)

func testStore(t *testing.T) *store.Store {
	d, err := db.OpenDSN("file:" + filepath.Join(t.TempDir(), "t.db") + "?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return &store.Store{DB: d}
}

func TestSampleCard(t *testing.T) {
	img, err := jpeg.Decode(bytes.NewReader(SampleCard()))
	if err != nil || img.Bounds().Dx() < 600 {
		t.Fatalf("card: %v %v", err, img)
	}
	if !strings.Contains(SampleText(), "3 cups rolled oats") {
		t.Fatal("the text matches the card")
	}
}

// The speed test times each text model on the card's words and each photo
// model on the card's photo, and says whether they read it right.
func TestBench(t *testing.T) {
	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		answer := `{"title":"Oat Cookies","ingredients":[{"line":"1 cup butter"},{"line":"1 cup brown sugar"},{"line":"2 eggs"},{"line":"1 tsp vanilla"},{"line":"1 1/2 cups flour"},{"line":"3 cups rolled oats"}],"steps":[{"text":"Bake."}]}`
		switch {
		case bytes.Contains(raw, []byte("image_url")) && bytes.Contains(raw, []byte(`"blurry-vision"`)):
			answer = `{"text":"Cookies with flour and sugar"}`
		case bytes.Contains(raw, []byte("image_url")):
			answer = `{"text":"Oat Cookies\n3 cups rolled oats\nBake at 350F for 10 minutes."}`
		case bytes.Contains(raw, []byte(`"broken-text"`)):
			answer = "I like cookies."
		}
		resp, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": answer}}},
			"usage": map[string]int{"prompt_tokens": 100, "completion_tokens": 50}})
		w.Write(resp)
	}))
	defer ai.Close()
	st := testStore(t)
	for k, v := range map[string]string{store.KeyLLMProvider: "openai", store.KeyLLMBaseURL: ai.URL, store.KeyLLMModel: "small-text,broken-text",
		store.KeyLLMVisionModel: "small-vision,blurry-vision", store.KeyLLMJSONMode: "false"} {
		st.SetSetting(k, v)
	}
	s := New(st)
	if err := s.StartBench("photo"); err == nil {
		t.Fatal("no photo AI is set up")
	}
	if err := s.StartBench("main"); err != nil {
		t.Fatal(err)
	}
	var res map[string]Bench
	for i := 0; i < 100; i++ {
		if st := s.Status(); !st.Benching && len(st.Results) == 4 {
			res = st.Results
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if b := res["main|text|small-text"]; !b.ReadRight || b.Error != "" || b.TokensPerSec <= 0 {
		t.Fatalf("text: %+v", res)
	}
	if b := res["main|text|broken-text"]; b.ReadRight || b.Error == "" {
		t.Fatalf("a model that doesn't answer with a recipe: %+v", b)
	}
	if !res["main|photo|small-vision"].ReadRight || res["main|photo|blurry-vision"].ReadRight {
		t.Fatalf("photos: %+v", res)
	}
}

// Models with a newer version in Ollama's library are listed.
func TestCheckUpdates(t *testing.T) {
	manifest := []byte(`{"schemaVersion":2}`)
	sum := sha256.Sum256(manifest)
	latest := hex.EncodeToString(sum[:])
	reg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(manifest) }))
	defer reg.Close()
	old := ollama.Registry
	ollama.Registry = reg.URL
	defer func() { ollama.Registry = old }()
	ol := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"models":[{"name":"qwen2.5:3b","digest":"sha256:%s"},{"name":"qwen2.5vl:3b","digest":"sha256:older"}]}`, latest)
	}))
	defer ol.Close()
	st := testStore(t)
	for k, v := range map[string]string{store.KeyLLMProvider: "openai", store.KeyLLMBaseURL: ol.URL + "/v1", store.KeyLLMModel: "qwen2.5:3b",
		store.KeyLLMVisionModel: "qwen2.5vl:3b"} {
		st.SetSetting(k, v)
	}
	s := New(st)
	ups := s.CheckUpdates(t.Context())
	if len(ups) != 1 || ups[0].Model != "qwen2.5vl:3b" {
		t.Fatalf("updates: %+v", ups)
	}
	s.Updated(ups[0].Server, ups[0].Model)
	if st := s.Status(); len(st.Updates) != 0 || st.CheckedAt == "" {
		t.Fatalf("after updating: %+v", st)
	}
}
