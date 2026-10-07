package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/zachcurry13/recipebank/internal/aitools"
	"github.com/zachcurry13/recipebank/internal/safe"
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
