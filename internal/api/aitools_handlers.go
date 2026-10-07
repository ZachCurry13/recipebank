package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/zachcurry13/recipebank/internal/aitools"
	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/ollama"
	"github.com/zachcurry13/recipebank/internal/safe"
	"github.com/zachcurry13/recipebank/internal/store"
)

// AI machines (Admin → AI tools): newer versions of the Ollama models in use,
// and the speed test.

func (s *Server) handleAIToolsStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.AITools.Status())
}

// handleCheckModelUpdates checks for newer model versions in the background.
func (s *Server) handleCheckModelUpdates(w http.ResponseWriter, r *http.Request) {
	safe.Go("model update check", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		s.AITools.CheckUpdates(ctx)
	})
	w.WriteHeader(http.StatusAccepted)
}

// handleModelUpdated forgets an update once the model was downloaded again.
func (s *Server) handleModelUpdated(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Server string `json:"server"`
		Model  string `json:"model"`
	}
	if !readJSON(w, r, &body, 2<<10) {
		return
	}
	s.AITools.Updated(body.Server, body.Model)
	w.WriteHeader(http.StatusNoContent)
}

// handleStartBench times one AI's models ({"which": "main" | "photo"}).
func (s *Server) handleStartBench(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Which string `json:"which"`
	}
	if !readJSON(w, r, &body, 1<<10) {
		return
	}
	if err := s.AITools.StartBench(body.Which); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, aitools.ErrBusy) {
			status = http.StatusConflict
		}
		writeErr(w, status, err.Error())
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// aiHealth is whether one AI is answering, checked without spending tokens.
type aiHealth struct {
	Which   string   `json:"which"` // "main" or "photo"
	OK      bool     `json:"ok"`
	Checked bool     `json:"checked"` // false: an online service that isn't asked (Claude)
	Error   string   `json:"error,omitempty"`
	Missing []string `json:"missing,omitempty"` // models it doesn't have
}

// handleAIHealth says whether each AI that's set up is answering.
func (s *Server) handleAIHealth(w http.ResponseWriter, r *http.Request) {
	out := []aiHealth{}
	for _, m := range []struct {
		which string
		ai    store.AIConfig
	}{{"main", s.Store.AIConfig()}, {"photo", s.Store.PhotoAIConfig()}} {
		if !m.ai.Ready() {
			continue
		}
		h := aiHealth{Which: m.which}
		if m.ai.Provider == "anthropic" {
			h.OK = true // Claude's API isn't asked: it would need a request that costs tokens
			out = append(out, h)
			continue
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		missing, err := llm.Reachable(ctx, m.ai.BaseURL, m.ai.APIKey, append(append([]string{}, m.ai.Models...), m.ai.Vision...))
		cancel()
		h.Checked, h.Missing = true, missing
		if err != nil {
			h.Error = err.Error()
			if llm.HostDown(err) {
				h.Error = "can't be reached: is that computer on?"
			}
		}
		h.OK = err == nil && len(missing) == 0
		out = append(out, h)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ais": out, "cost": s.Store.CostThisMonth()})
}

// handleAIModels lists a service's models for Admin's model boxes, from the
// form as it is now (a blank key means the saved one), so names don't have
// to be guessed (Google retires old Gemini names, for example).
func (s *Server) handleAIModels(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Which    string `json:"which"` // "llm" (the main AI) or "photo"
		Provider string `json:"provider"`
		BaseURL  string `json:"base_url"`
		APIKey   string `json:"api_key"`
	}
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	if body.APIKey == "" {
		key := store.KeyLLMAPIKey
		if body.Which == "photo" {
			key = store.KeyPhotoAPIKey
		}
		body.APIKey = s.Store.Setting(key)
	}
	if body.Provider != "anthropic" && body.BaseURL == "" {
		writeErr(w, http.StatusBadRequest, "type the address first")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	models, err := llm.ListModels(ctx, body.Provider, body.BaseURL, body.APIKey)
	if err != nil {
		if llm.HostDown(err) {
			err = errors.New("it can't be reached: check the address, and that the computer is on")
		}
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	// On Ollama, say which models read photos, for the photo model boxes.
	var photo []string
	if base, err := ollama.Normalize(body.BaseURL); err == nil && body.Provider != "anthropic" && ollama.IsOllama(ctx, base) {
		photo = ollama.Photo(ctx, base, models)
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": models, "photo": photo})
}
