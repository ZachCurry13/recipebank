// Package aitools looks after the family's AI machines (from NovelCheck): it
// checks once a day whether the Ollama models in use have a newer version
// (Admin shows an Update button), and times each model on a made-up recipe
// card, as text and as a photo (the speed test).
package aitools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/ollama"
	"github.com/zachcurry13/recipebank/internal/safe"
	"github.com/zachcurry13/recipebank/internal/store"
)

// keyBenchmarks keeps the speed test results ("ai|job|model" → Bench).
const keyBenchmarks = "ai_benchmarks"

// Service holds the latest update check and speed test.
type Service struct {
	Store *store.Store

	mu        sync.Mutex
	updates   []ollama.Update
	checkedAt time.Time
	checkErr  string
	benching  bool
}

func New(st *store.Store) *Service { return &Service{Store: st} }

// machine is one AI and the models it's set to use.
type machine struct {
	name string // "main" or "photo"
	ai   store.AIConfig
}

func (s *Service) machines() []machine {
	var out []machine
	if ai := s.Store.AIConfig(); ai.Ready() {
		out = append(out, machine{"main", ai})
	}
	if ai := s.Store.PhotoAIConfig(); ai.Ready() {
		out = append(out, machine{"photo", ai})
	}
	return out
}

// Loop checks for model updates once a day (the first time 15 minutes after start).
func (s *Service) Loop(ctx context.Context) {
	wait := 15 * time.Minute
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = 24 * time.Hour
		_ = safe.Run("model update check", func() error { s.CheckUpdates(ctx); return nil })
	}
}

// CheckUpdates asks each Ollama on the home network and Ollama's model
// library whether the models in use have a newer version.
func (s *Service) CheckUpdates(ctx context.Context) []ollama.Update {
	var all []ollama.Update
	var errs []string
	for _, m := range s.machines() {
		if m.ai.Provider == "anthropic" || !llm.IsLocal(m.ai.BaseURL) {
			continue
		}
		base, err := ollama.Normalize(m.ai.BaseURL)
		if err != nil {
			continue
		}
		models := append(append([]string{}, m.ai.Models...), m.ai.Vision...)
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		ups, err := ollama.Outdated(cctx, base, unique(models))
		cancel()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s AI: %v", m.name, err))
			continue
		}
		all = append(all, ups...)
	}
	s.mu.Lock()
	s.updates, s.checkedAt, s.checkErr = all, time.Now(), strings.Join(errs, "; ")
	s.mu.Unlock()
	return all
}

func unique(list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range list {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// Status is what Admin shows.
type Status struct {
	Updates   []ollama.Update  `json:"updates"`
	CheckedAt string           `json:"checked_at"`
	CheckErr  string           `json:"check_error"`
	Benching  bool             `json:"benching"`
	Results   map[string]Bench `json:"results"`
}

// Status returns the latest check and speed test.
func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{Updates: s.updates, CheckErr: s.checkErr, Benching: s.benching, Results: s.benchmarks()}
	if st.Updates == nil {
		st.Updates = []ollama.Update{}
	}
	if !s.checkedAt.IsZero() {
		st.CheckedAt = s.checkedAt.UTC().Format(time.RFC3339)
	}
	return st
}

// Updated forgets an update once its model was downloaded again.
func (s *Service) Updated(server, model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.updates[:0]
	for _, u := range s.updates {
		if !(u.Model == model && u.Server == server) {
			kept = append(kept, u)
		}
	}
	s.updates = kept
}

func (s *Service) benchmarks() map[string]Bench {
	out := map[string]Bench{}
	_ = json.Unmarshal([]byte(s.Store.Setting(keyBenchmarks)), &out)
	return out
}
