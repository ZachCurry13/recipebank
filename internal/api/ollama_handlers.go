package api

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/ollama"
	"github.com/zachcurry13/recipebank/internal/store"
)

// Ollama helpers (Admin → AI), ported from NovelCheck: find Ollama on the
// network, download models with progress, pick which RecipeBank uses, see
// what the graphics card can hold, and remove models that aren't used.

// ollamaBase is an AI address's Ollama server, when it's on one of Ollama's ports.
func ollamaBase(llmURL string) string {
	base, err := ollama.Normalize(llmURL)
	if err != nil {
		return ""
	}
	if u, perr := url.Parse(base); perr != nil || !llm.IsOllamaPort(u.Port()) {
		return ""
	}
	return base
}

// handleOllamaFind looks for Ollama servers (and at ?url=) and lists their models.
func (s *Server) handleOllamaFind(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	var extra []string
	if u := r.URL.Query().Get("url"); u != "" {
		n, err := ollama.Normalize(u)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		extra = append(extra, n)
	}
	for _, k := range []string{store.KeyLLMBaseURL, store.KeyPhotoBaseURL} {
		if b := ollamaBase(s.Store.Setting(k)); b != "" {
			extra = append(extra, b)
		}
	}
	servers := ollama.Discover(ctx, extra...)
	sctx, scancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer scancel()
	for i := range servers {
		servers[i].Photo = ollama.Photo(sctx, servers[i].URL, servers[i].Models)
	}
	// catalog: the download picks before the graphics card is checked
	writeJSON(w, http.StatusOK, map[string]any{"servers": servers, "catalog": ollama.Recommend(ollama.GPU{})})
}

type ollamaModelReq struct {
	URL    string   `json:"url"`
	Model  string   `json:"model"`
	Models []string `json:"models"` // "use": in order, the first one first
	Use    string   `json:"use"`    // "use": "text" (the main AI) or "photos"
}

func (s *Server) handleOllamaPull(w http.ResponseWriter, r *http.Request) {
	var body ollamaModelReq
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	base, err := ollama.Normalize(body.URL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Pulls.Start(base, strings.TrimSpace(body.Model)); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, s.Pulls.Status())
}

func (s *Server) handleOllamaPullStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Pulls.Status())
}

// handleOllamaUse points RecipeBank at downloaded models. For text, they
// become the main AI. For photos: on the main AI's own Ollama they become its
// photo models; on another Ollama (a bigger computer) they become the photo AI.
func (s *Server) handleOllamaUse(w http.ResponseWriter, r *http.Request) {
	var body ollamaModelReq
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	base, err := ollama.Normalize(body.URL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	srv, err := ollama.Probe(ctx, base)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "can't reach Ollama at "+base)
		return
	}
	models := body.Models
	if len(models) == 0 {
		models = []string{body.Model}
	}
	var chosen []string
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m == "" || slices.Contains(chosen, m) {
			continue
		}
		if !slices.Contains(srv.Models, m) && !slices.Contains(srv.Models, m+":latest") {
			writeErr(w, http.StatusBadRequest, m+" isn't downloaded on this Ollama yet")
			return
		}
		chosen = append(chosen, m)
	}
	if len(chosen) == 0 {
		writeErr(w, http.StatusBadRequest, "pick at least one model")
		return
	}
	if body.Use == "photos" {
		see := ollama.CanSee(ctx, base, chosen)
		for _, m := range chosen {
			if can, known := see[m]; known && !can {
				writeErr(w, http.StatusBadRequest, m+" can't read photos. Pick a model marked 📷, or download a photo model.")
				return
			}
		}
	}
	v1 := base + "/v1"
	mainBase, _ := ollama.Normalize(s.Store.Setting(store.KeyLLMBaseURL))
	var set map[string]string
	where := "the main AI"
	switch {
	case body.Use == "text":
		set = map[string]string{store.KeyLLMProvider: "openai", store.KeyLLMBaseURL: v1, store.KeyLLMAPIKey: "",
			store.KeyLLMModel: chosen[0], store.KeyLLMFallbackModel: strings.Join(chosen[1:], ","), store.KeyLLMJSONMode: "true"}
	case body.Use == "photos" && mainBase == base:
		set = map[string]string{store.KeyLLMVisionModel: strings.Join(chosen, ",")}
		where = "the main AI's photo model"
	case body.Use == "photos":
		set = map[string]string{store.KeyPhotoProvider: "openai", store.KeyPhotoBaseURL: v1, store.KeyPhotoAPIKey: "",
			store.KeyPhotoModel: strings.Join(chosen, ","), store.KeyPhotoJSONMode: "true"}
		where = "the AI for photos"
	default:
		writeErr(w, http.StatusBadRequest, "use it for text or photos")
		return
	}
	for k, v := range set {
		if err := s.Store.SetSetting(k, v); err != nil {
			writeStoreErr(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"base_url": v1, "models": strings.Join(chosen, ", "), "where": where})
}

// handleOllamaGPU reports what an Ollama server's graphics card can hold, and
// labels the text and photo models. probe=1 may load the largest downloaded
// model for a moment to measure; vram_gb=N uses the size the admin chose
// instead (0 = no graphics card).
func (s *Server) handleOllamaGPU(w http.ResponseWriter, r *http.Request) {
	base, err := ollama.Normalize(r.URL.Query().Get("url"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	var g ollama.GPU
	var msg string
	if v := r.URL.Query().Get("vram_gb"); v != "" {
		gb, err := strconv.ParseFloat(v, 64)
		if err != nil || gb < 0 || gb > 512 {
			writeErr(w, http.StatusBadRequest, "choose a graphics memory size")
			return
		}
		g = ollama.GPU{Kind: "manual", VRAMBytes: int64(gb * (1 << 30))}
		if gb == 0 {
			g = ollama.GPU{Kind: "none"}
		}
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Minute)
		defer cancel()
		g, err = ollama.MeasureGPU(ctx, base, r.URL.Query().Get("probe") == "1")
		if err != nil {
			msg = err.Error()
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"gpu": g, "models": ollama.Recommend(g), "message": msg})
}

// modelsInUse are the models RecipeBank is set to use, so they aren't deleted by accident.
func (s *Server) modelsInUse() map[string]bool {
	used := map[string]bool{}
	for _, ai := range []store.AIConfig{s.Store.AIConfig(), s.Store.PhotoAIConfig()} {
		for _, m := range append(append([]string{}, ai.Models...), ai.Vision...) {
			used[m] = true
		}
	}
	return used
}

// handleOllamaModels lists an Ollama's installed models with their disk space.
func (s *Server) handleOllamaModels(w http.ResponseWriter, r *http.Request) {
	base, err := ollama.Normalize(r.URL.Query().Get("url"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	models, err := ollama.Installed(ctx, base)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "couldn't reach Ollama at "+base+": "+err.Error())
		return
	}
	used := s.modelsInUse()
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.Name
	}
	see := ollama.CanSee(ctx, base, names)
	type row struct {
		ollama.Model
		InUse  bool `json:"in_use"`
		Photos bool `json:"photos"` // it reads photos
	}
	out := make([]row, 0, len(models))
	var total int64
	for _, m := range models {
		out = append(out, row{m, used[m.Name] || used[strings.TrimSuffix(m.Name, ":latest")], see[m.Name]})
		total += m.Size
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": out, "total": total})
}

// handleOllamaDelete removes a model, unless RecipeBank is set to use it.
func (s *Server) handleOllamaDelete(w http.ResponseWriter, r *http.Request) {
	var body ollamaModelReq
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	base, err := ollama.Normalize(body.URL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	used := s.modelsInUse()
	if used[body.Model] || used[strings.TrimSuffix(body.Model, ":latest")] {
		writeErr(w, http.StatusConflict, "RecipeBank is set to use "+body.Model+". Choose another model first.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if err := ollama.Delete(ctx, base, body.Model); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
