package aitools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/safe"
	"github.com/zachcurry13/recipebank/internal/store"
)

// Bench is one model's time on one job of the speed test.
type Bench struct {
	AI           string  `json:"ai"`  // "main" or "photo"
	Job          string  `json:"job"` // "text": organize the card's words; "photo": read the card's photo
	Model        string  `json:"model"`
	Seconds      float64 `json:"seconds"`
	TokensPerSec float64 `json:"tok_s,omitempty"`
	ReadRight    bool    `json:"read_right"` // the answer had what's on the card
	At           string  `json:"at"`
	Error        string  `json:"error,omitempty"`
}

// ErrBusy means a speed test is already running.
var ErrBusy = errors.New("a speed test is already running")

// StartBench times every model of one AI ("main" or "photo") in the background.
func (s *Service) StartBench(which string) error {
	var m *machine
	for _, x := range s.machines() {
		if x.name == which {
			m = &x
		}
	}
	if m == nil {
		return errors.New("that AI isn't set up")
	}
	s.mu.Lock()
	if s.benching {
		s.mu.Unlock()
		return ErrBusy
	}
	s.benching = true
	s.mu.Unlock()
	safe.Go("speed test", func() {
		defer func() {
			s.mu.Lock()
			s.benching = false
			s.mu.Unlock()
		}()
		s.bench(*m)
	})
	return nil
}

func (s *Service) bench(m machine) {
	client := llm.New(m.ai.Provider, m.ai.BaseURL, m.ai.APIKey, m.ai.JSONMode)
	timed := func(model string, call func(context.Context) (string, llm.Usage, error)) (string, float64, float64, error) {
		ctx, cancel := context.WithTimeout(context.Background(), llm.Timeout(m.ai.BaseURL, s.Store.SettingInt(store.KeyLLMTimeoutSeconds)))
		defer cancel()
		start := time.Now()
		out, u, err := call(ctx)
		took := time.Since(start)
		if u.Total() > 0 {
			_ = s.Store.RecordUsage(model, u.PromptTokens, u.CompletionTokens, took)
		}
		tps := 0.0
		if took > 0 && u.CompletionTokens > 0 {
			tps = float64(u.CompletionTokens) / took.Seconds()
		}
		return out, took.Seconds(), tps, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if m.name == "main" {
		for _, model := range m.ai.Models {
			out, secs, tps, err := timed(model, func(ctx context.Context) (string, llm.Usage, error) {
				return client.Complete(ctx, model, llm.TextSystem, llm.CardText(SampleText()))
			})
			b := Bench{AI: m.name, Job: "text", Model: model, Seconds: secs, TokensPerSec: tps, At: now}
			if err == nil {
				rc, perr := llm.ParseRecipe(out)
				b.ReadRight = perr == nil && len(rc.Ingredients) >= 6
				if perr != nil {
					err = errors.New("it answered, but not with a recipe: " + perr.Error())
				}
			}
			s.save(b, err)
		}
	}
	reader, ok := client.(llm.ImageReader)
	if !ok {
		return
	}
	card := []llm.Image{{Data: SampleCard(), MediaType: "image/jpeg"}}
	for _, model := range unique(m.ai.Vision) {
		out, secs, tps, err := timed(model, func(ctx context.Context) (string, llm.Usage, error) {
			return reader.ReadImages(ctx, model, llm.TranscribePrompt(nil), card)
		})
		low := strings.ToLower(out)
		s.save(Bench{AI: m.name, Job: "photo", Model: model, Seconds: secs, TokensPerSec: tps, At: now,
			ReadRight: err == nil && strings.Contains(low, sampleWord) && strings.Contains(low, "350")}, err)
	}
}

// save keeps one result, replacing that model's last one for the job.
func (s *Service) save(b Bench, err error) {
	if err != nil {
		b.Error = err.Error()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.benchmarks()
	all[b.AI+"|"+b.Job+"|"+b.Model] = b
	if data, err := json.Marshal(all); err == nil {
		_ = s.Store.SetSetting(keyBenchmarks, string(data))
	}
}
