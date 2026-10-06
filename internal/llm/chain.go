package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/store"
)

// ErrNoAI means no AI is set up yet (Admin → AI).
var ErrNoAI = errors.New("no AI is set up yet: an admin can add one under Admin → AI")

// Ask sends one question to each model in order until parse accepts an answer.
func Ask(ctx context.Context, st *store.Store, system, user string, parse func(string) error) error {
	ai := st.AIConfig()
	if !ai.Ready() {
		return ErrNoAI
	}
	client := New(ai.Provider, ai.BaseURL, ai.APIKey, ai.JSONMode)
	_, err := try(ctx, st, ai, ai.Models, parse, func(cctx context.Context, model string) (string, Usage, error) {
		return client.Complete(cctx, model, system, user)
	})
	return err
}

// AskImages shows photos to each vision model in order until parse accepts
// an answer, and says which model answered.
func AskImages(ctx context.Context, st *store.Store, prompt string, images []Image, parse func(string) error) (string, error) {
	ai := st.AIConfig()
	if !ai.Ready() {
		return "", ErrNoAI
	}
	reader, ok := New(ai.Provider, ai.BaseURL, ai.APIKey, ai.JSONMode).(ImageReader)
	if !ok {
		return "", errors.New("this AI can't read photos")
	}
	return try(ctx, st, ai, ai.Vision, parse, func(cctx context.Context, model string) (string, Usage, error) {
		return reader.ReadImages(cctx, model, prompt, images)
	})
}

func try(ctx context.Context, st *store.Store, ai store.AIConfig, models []string, parse func(string) error,
	call func(context.Context, string) (string, Usage, error)) (string, error) {
	var errs []string
	for _, model := range models {
		cctx, cancel := context.WithTimeout(ctx, Timeout(ai.BaseURL, st.SettingInt(store.KeyLLMTimeoutSeconds)))
		start := time.Now()
		out, usage, err := call(cctx, model)
		cancel()
		if usage.Total() > 0 {
			_ = st.RecordUsage(model, usage.PromptTokens, usage.CompletionTokens, time.Since(start))
		}
		if err == nil {
			if err = parse(out); err == nil {
				return model, nil
			}
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		errs = append(errs, fmt.Sprintf("%s: %v", model, err))
		if HostDown(err) {
			break // the server is off or stuck: its other models won't answer either
		}
	}
	return "", errors.New("the AI couldn't do it (" + strings.Join(errs, "; ") + ")")
}
