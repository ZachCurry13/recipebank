package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zachcurry13/recipebank/internal/store"
)

// ErrNoAI means no AI is set up yet (Admin → AI).
var ErrNoAI = errors.New("no AI is set up yet: an admin can add one under Admin → AI")

// Ask sends one question to each model in order until parse accepts an answer.
// Text goes to the main AI, or to the photo AI when it's the only one. When
// the main AI can't answer (its server is off, or a model won't load on its
// graphics card), the photo AI answers instead: it reads text too.
func Ask(ctx context.Context, st *store.Store, system, user string, parse func(string) error) error {
	main, photo := st.AIConfig(), st.PhotoAIConfig()
	if !main.Ready() {
		_, err := askTextWith(ctx, st, photo, true, system, user, parse)
		return err
	}
	_, err := AskWith(ctx, st, main, system, user, parse)
	if err == nil || ctx.Err() != nil || !photo.Ready() || isDown(photo.BaseURL) {
		return err
	}
	if _, perr := askTextWith(ctx, st, photo, true, system, user, parse); perr == nil {
		return nil
	} else if HostDown(perr) {
		markDown(photo.BaseURL)
	}
	return err // the main AI's problem is the one to fix
}

// AskWith asks one AI's models (Ask's work; also the admin's "Test").
func AskWith(ctx context.Context, st *store.Store, ai store.AIConfig, system, user string, parse func(string) error) (string, error) {
	return askTextWith(ctx, st, ai, false, system, user, parse)
}

// askTextWith asks one AI; quick gives up connecting after a few seconds
// (the photo AI may be on a computer that's switched off).
func askTextWith(ctx context.Context, st *store.Store, ai store.AIConfig, quick bool, system, user string, parse func(string) error) (string, error) {
	if !ai.Ready() {
		return "", ErrNoAI
	}
	client := New(ai.Provider, ai.BaseURL, ai.APIKey, ai.JSONMode)
	if c, ok := client.(*Client); ok && quick {
		c.HTTP = quickDial
	}
	return try(ctx, st, ai, ai.Models, parse, func(cctx context.Context, model string) (string, Usage, error) {
		return client.Complete(cctx, model, system, user)
	})
}

// AskImages shows photos to each vision model in order until parse accepts
// an answer, and says which model answered: the photo AI's first (when there
// is one and it's on), then the main AI's.
func AskImages(ctx context.Context, st *store.Store, prompt string, images []Image, parse func(string) error) (string, error) {
	main, photo := st.AIConfig(), st.PhotoAIConfig()
	var photoErr error
	if photo.Ready() && !isDown(photo.BaseURL) {
		model, err := askImagesWith(ctx, st, photo, true, prompt, images, parse)
		if err == nil || ctx.Err() != nil || TooLong(err) {
			return model, err // too long: askPhotos makes the photos smaller and asks again
		}
		photoErr = err
		if HostDown(err) {
			markDown(photo.BaseURL) // off or asleep: don't wait for it again for a while
		}
	}
	if !main.Ready() {
		if photoErr != nil {
			return "", photoErr
		}
		return "", ErrNoAI
	}
	model, err := askImagesWith(ctx, st, main, false, prompt, images, parse)
	if err != nil && photoErr != nil && !TooLong(err) {
		return "", fmt.Errorf("the photo AI couldn't (%v), and neither could the main AI (%w)", photoErr, err)
	}
	return model, err
}

func askImagesWith(ctx context.Context, st *store.Store, ai store.AIConfig, quick bool, prompt string, images []Image,
	parse func(string) error) (string, error) {
	client := New(ai.Provider, ai.BaseURL, ai.APIKey, ai.JSONMode)
	if c, ok := client.(*Client); ok && quick {
		c.HTTP = quickDial
	}
	reader, ok := client.(ImageReader)
	if !ok {
		return "", errors.New("this AI can't read photos")
	}
	return try(ctx, st, ai, ai.Vision, parse, func(cctx context.Context, model string) (string, Usage, error) {
		return reader.ReadImages(cctx, model, prompt, images)
	})
}

// quickDial gives up connecting after a few seconds, so a photo AI on a
// computer that's switched off doesn't hold up the main AI.
var quickDial = &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, TLSHandshakeTimeout: 10 * time.Second}}

// downFor is how long a photo AI that couldn't be reached is skipped.
const downFor = 2 * time.Minute

var down = struct {
	sync.Mutex
	until map[string]time.Time
}{until: map[string]time.Time{}}

func markDown(baseURL string) {
	down.Lock()
	defer down.Unlock()
	down.until[baseURL] = time.Now().Add(downFor)
}

func isDown(baseURL string) bool {
	down.Lock()
	defer down.Unlock()
	return time.Now().Before(down.until[baseURL])
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
