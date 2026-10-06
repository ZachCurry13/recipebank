package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fits(recs []Rec) map[string]string {
	m := map[string]string{}
	for _, r := range recs {
		m[r.Name] = r.Fit
	}
	return m
}

func TestRecommend(t *testing.T) {
	gb := func(n float64) int64 { return int64(n * (1 << 30)) }
	// An 8 GB card: 7B for text; 3B for photos (7B VL needs 8.5).
	r := Recommend(GPU{Kind: "about", VRAMBytes: gb(8)})
	text, photos := fits(r.Text), fits(r.Photos)
	if text["qwen2.5:7b"] != "best" || text["llama3.1:8b"] != "powerful" || text["qwen2.5:14b"] != "too_big" ||
		photos["qwen2.5vl:3b"] != "best" || photos["gemma3:4b"] != "powerful" || photos["qwen2.5vl:7b"] != "too_big" {
		t.Fatalf("8 GB: %v %v", text, photos)
	}
	// 12 GB: 14B for text is best and biggest; 7B VL best for photos, Gemma 3 12B the biggest.
	r = Recommend(GPU{Kind: "manual", VRAMBytes: gb(12)})
	text, photos = fits(r.Text), fits(r.Photos)
	if text["qwen2.5:14b"] != "best_powerful" || photos["qwen2.5vl:7b"] != "best" || photos["gemma3:12b"] != "powerful" ||
		photos["gemma3:27b"] != "too_big" {
		t.Fatalf("12 GB: %v %v", text, photos)
	}
	// 24 GB: the 32B VL model fits for messy handwriting.
	if photos := fits(Recommend(GPU{Kind: "manual", VRAMBytes: gb(24)}).Photos); photos["qwen2.5vl:32b"] != "powerful" || photos["qwen2.5vl:7b"] != "best" {
		t.Fatalf("24 GB: %v", photos)
	}
	// CPU only: small text models are OK; photos are slow.
	r = Recommend(GPU{Kind: "none"})
	if fits(r.Text)["llama3.2"] != "cpu_ok" || fits(r.Text)["qwen2.5:7b"] != "cpu_slow" || fits(r.Photos)["qwen2.5vl:3b"] != "cpu_slow" {
		t.Fatalf("cpu: %v", r)
	}
	if r := Recommend(GPU{Kind: "unknown"}); fits(r.Text)["qwen2.5:7b"] != "" {
		t.Fatalf("unknown should not label: %v", r)
	}
}

// fakeGPUOllama answers tags/ps/generate; loading puts part of the model on the GPU.
func fakeGPUOllama(t *testing.T, vram int64) (*httptest.Server, *[]string) {
	var calls []string
	loaded := false
	size := int64(5_000_000_000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			fmt.Fprintf(w, `{"models":[{"name":"llama3.2:latest","size":2000000000},{"name":"qwen2.5:7b","size":%d}]}`, size)
		case "/api/ps":
			if loaded {
				fmt.Fprintf(w, `{"models":[{"name":"qwen2.5:7b","size":%d,"size_vram":%d}]}`, size, vram)
			} else {
				fmt.Fprint(w, `{"models":[]}`)
			}
		case "/api/generate":
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			calls = append(calls, fmt.Sprintf("%v:%v", req["model"], req["keep_alive"]))
			loaded = req["keep_alive"] != "0"
			fmt.Fprint(w, `{"done":true}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestMeasureGPU(t *testing.T) {
	srv, calls := fakeGPUOllama(t, 3_000_000_000) // 3 GB of 5 on the GPU
	if g, _ := MeasureGPU(context.Background(), srv.URL, false); g.Kind != "unknown" {
		t.Fatalf("without probe and nothing loaded: %+v", g)
	}
	g, err := MeasureGPU(context.Background(), srv.URL, true)
	if err != nil || g.Kind != "about" || g.VRAMBytes != 3_000_000_000 || g.Basis != "qwen2.5:7b" {
		t.Fatalf("probe: %+v %v", g, err)
	}
	if strings.Join(*calls, " ") != "qwen2.5:7b:1m qwen2.5:7b:0" {
		t.Fatalf("should load the largest model then unload it: %v", *calls)
	}
	cpu, _ := fakeGPUOllama(t, 0)
	if g, _ := MeasureGPU(context.Background(), cpu.URL, true); g.Kind != "none" {
		t.Fatalf("CPU only: %+v", g)
	}
}
