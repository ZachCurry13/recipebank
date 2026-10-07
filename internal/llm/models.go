package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// modelIDs asks a server which models it has (it costs no tokens): an
// OpenAI-compatible one at /models (Ollama, LM Studio, OpenAI, Gemini), Claude
// at its own /v1/models. nil without an error: it answered but lists none.
func modelIDs(ctx context.Context, provider, baseURL, apiKey string) ([]string, error) {
	url := NormalizeBaseURL(baseURL) + "/models"
	if provider == "anthropic" {
		base := strings.TrimRight(baseURL, "/")
		if base == "" {
			base = "https://api.anthropic.com"
		}
		url = base + "/v1/models?limit=100"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	switch {
	case provider == "anthropic":
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case apiKey != "":
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := quickDial.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, errors.New("the API key was refused")
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("it answered %s", resp.Status)
	}
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&list) != nil {
		return nil, nil
	}
	var ids []string
	for _, m := range list.Data {
		ids = append(ids, strings.TrimPrefix(m.ID, "models/")) // Gemini says "models/gemini-…"
	}
	return ids, nil
}

// Reachable asks an OpenAI-compatible server for its models and says which
// of want it doesn't have.
func Reachable(ctx context.Context, baseURL, apiKey string, want []string) (missing []string, err error) {
	ids, err := modelIDs(ctx, "openai", baseURL, apiKey)
	if err != nil || len(ids) == 0 {
		return nil, err // answered, but doesn't list its models: nothing more to check
	}
	have := map[string]bool{}
	for _, id := range ids {
		have[id], have[strings.TrimSuffix(id, ":latest")] = true, true
	}
	for _, m := range want {
		if !have[m] && !have[strings.TrimSuffix(m, ":latest")] {
			missing = append(missing, m)
		}
	}
	return missing, nil
}

// notChat are models that can't answer a recipe question: embeddings, speech,
// and ones that make pictures or video.
var notChat = regexp.MustCompile(`(?i)embed|tts|audio|speech|whisper|transcri|imagen|image-generation|dall-e|veo|aqa|moderation|realtime|computer-use`)

// ListModels is a service's models that can answer questions, for Admin's
// "List models", so its current names don't have to be guessed.
func ListModels(ctx context.Context, provider, baseURL, apiKey string) ([]string, error) {
	ids, err := modelIDs(ctx, provider, baseURL, apiKey)
	if err != nil {
		return nil, err
	}
	out := []string{}
	seen := map[string]bool{}
	for _, id := range ids {
		if id != "" && !seen[id] && !notChat.MatchString(id) {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, nil
}
