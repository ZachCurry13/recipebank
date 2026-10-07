package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
)

// CanSee asks Ollama what each model can do and reports which can read
// photos. Models it couldn't ask about are left out of the answer.
func CanSee(ctx context.Context, base string, models []string) map[string]bool {
	out := map[string]bool{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, m := range models {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if see, err := canSee(ctx, base, m); err == nil {
				mu.Lock()
				out[m] = see
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return out
}

// Photo lists the models CanSee says read photos, in the order given.
func Photo(ctx context.Context, base string, models []string) []string {
	see := CanSee(ctx, base, models)
	out := []string{}
	for _, m := range models {
		if see[m] {
			out = append(out, m)
		}
	}
	return out
}

// canSee reads Ollama's "vision" capability; Ollama older than 0.6.5 doesn't
// list capabilities, so there a picture reader in the model gives it away.
func canSee(ctx context.Context, base, model string) (bool, error) {
	body, _ := json.Marshal(map[string]string{"model": model, "name": model}) // newer and older Ollama
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/show", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("ollama: %s", resp.Status)
	}
	var show struct {
		Capabilities []string `json:"capabilities"`
		Details      struct {
			Families []string `json:"families"`
		} `json:"details"`
		Projector map[string]any `json:"projector_info"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&show); err != nil {
		return false, err
	}
	if show.Capabilities != nil {
		return slices.Contains(show.Capabilities, "vision"), nil
	}
	return len(show.Projector) > 0 || slices.Contains(show.Details.Families, "clip") ||
		slices.Contains(show.Details.Families, "mllama"), nil
}

// IsOllama reports whether an address answers like Ollama.
func IsOllama(ctx context.Context, base string) bool {
	var ver struct {
		Version string `json:"version"`
	}
	return getJSON(ctx, base+"/api/version", &ver) == nil && ver.Version != ""
}
