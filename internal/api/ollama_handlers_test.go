package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zachcurry13/recipebank/internal/store"
)

// fakeOllama answers like an Ollama server with a few models; it records deletes.
func fakeOllama(t *testing.T, models ...string) (*httptest.Server, *[]string) {
	var mu sync.Mutex
	var deleted []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
			fmt.Fprint(w, `{"version":"0.9.0"}`)
		case "/api/tags":
			var rows []string
			for i, m := range models {
				rows = append(rows, fmt.Sprintf(`{"name":%q,"size":%d,"details":{"parameter_size":"3B","quantization_level":"Q4_K_M"}}`, m, (i+1)*1_000_000_000))
			}
			fmt.Fprintf(w, `{"models":[%s]}`, strings.Join(rows, ","))
		case "/api/delete":
			mu.Lock()
			deleted = append(deleted, r.Method)
			mu.Unlock()
		case "/api/pull":
			fmt.Fprint(w, "{\"status\":\"pulling manifest\"}\n{\"status\":\"downloading\",\"total\":100,\"completed\":50}\n{\"status\":\"success\"}\n")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &deleted
}

// Models are picked for text or photos: photos on the main AI's own Ollama
// become its photo model; on another machine they become the AI for photos.
// Models in use can't be deleted.
func TestOllamaTools(t *testing.T) {
	c, srv := setup(t)
	nas, deleted := fakeOllama(t, "qwen2.5:3b", "qwen2.5vl:3b", "llama3.2:latest")
	pc, _ := fakeOllama(t, "qwen2.5vl:32b")

	var found struct{ Servers []struct{ URL string } }
	if code := c.do("GET", "/api/admin/ollama/find?url="+nas.URL, nil, &found); code != 200 || len(found.Servers) == 0 {
		t.Fatalf("find: %d %+v", code, found)
	}
	var used map[string]string
	if code := c.do("POST", "/api/admin/ollama/use", map[string]any{"url": nas.URL, "models": []string{"qwen2.5:3b", "llama3.2"}, "use": "text"}, &used); code != 200 {
		t.Fatalf("use for text: %d", code)
	}
	ai := srv.Store.AIConfig()
	if ai.BaseURL != nas.URL+"/v1" || strings.Join(ai.Models, ",") != "qwen2.5:3b,llama3.2" {
		t.Fatalf("main AI: %+v", ai)
	}
	if code := c.do("POST", "/api/admin/ollama/use", map[string]any{"url": nas.URL, "model": "qwen2.5vl:3b", "use": "photos"}, &used); code != 200 ||
		used["where"] != "the main AI's photo model" || srv.Store.Setting(store.KeyLLMVisionModel) != "qwen2.5vl:3b" || srv.Store.PhotoAIConfig().Ready() {
		t.Fatalf("photos on the main AI's own Ollama: %d %v", code, used)
	}
	if code := c.do("POST", "/api/admin/ollama/use", map[string]any{"url": nas.URL, "model": "nope", "use": "text"}, nil); code != 400 {
		t.Fatalf("a model that isn't downloaded: %d", code)
	}
	if code := c.do("POST", "/api/admin/ollama/use", map[string]any{"url": pc.URL, "model": "qwen2.5vl:32b", "use": "photos"}, &used); code != 200 || used["where"] != "the AI for photos" {
		t.Fatalf("photos on the PC: %d %v", code, used)
	}
	if p := srv.Store.PhotoAIConfig(); !p.Ready() || p.BaseURL != pc.URL+"/v1" || p.Models[0] != "qwen2.5vl:32b" {
		t.Fatalf("photo AI: %+v", p)
	}

	var list struct {
		Models []struct {
			Name  string
			InUse bool `json:"in_use"`
		}
	}
	c.do("GET", "/api/admin/ollama/models?url="+nas.URL, nil, &list)
	inUse := map[string]bool{}
	for _, m := range list.Models {
		inUse[m.Name] = m.InUse
	}
	if !inUse["qwen2.5:3b"] || !inUse["llama3.2:latest"] || !inUse["qwen2.5vl:3b"] {
		t.Fatalf("in use: %v", inUse)
	}
	srv.Store.SetSetting(store.KeyLLMVisionModel, "") // photos go to the PC now
	if code := c.do("POST", "/api/admin/ollama/delete", map[string]string{"url": nas.URL, "model": "qwen2.5:3b"}, nil); code != 409 {
		t.Fatalf("deleting a model in use: %d", code)
	}
	if code := c.do("POST", "/api/admin/ollama/delete", map[string]string{"url": nas.URL, "model": "qwen2.5vl:3b"}, nil); code != 200 || len(*deleted) != 1 {
		t.Fatalf("delete: %d %v", code, *deleted)
	}

	var gpu struct {
		Models struct {
			Text, Photos []struct{ Name, Fit string }
		}
	}
	c.do("GET", "/api/admin/ollama/gpu?url="+nas.URL+"&vram_gb=12", nil, &gpu)
	fit := map[string]string{}
	for _, m := range append(gpu.Models.Text, gpu.Models.Photos...) {
		fit[m.Name] = m.Fit
	}
	if fit["qwen2.5:14b"] != "best_powerful" || fit["qwen2.5vl:7b"] != "best" {
		t.Fatalf("12 GB: %v", fit)
	}

	if code := c.do("POST", "/api/admin/ollama/pull", map[string]string{"url": nas.URL, "model": "qwen2.5vl:7b"}, nil); code != 202 {
		t.Fatalf("pull: %d", code)
	}
	var st struct {
		Active, Done bool
		Percent      float64
	}
	for i := 0; i < 50; i++ {
		c.do("GET", "/api/admin/ollama/pull", nil, &st)
		if st.Done {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !st.Done || st.Percent != 100 {
		t.Fatalf("pull status: %+v", st)
	}
	if code := c.do("POST", "/api/admin/ollama/pull", map[string]string{"url": nas.URL, "model": "bad name!"}, nil); code != 400 {
		t.Fatalf("bad model name: %d", code)
	}
}
