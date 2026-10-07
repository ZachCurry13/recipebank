package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// HostDown reports whether err means the AI server can't answer at all
// right now: it couldn't be reached (switched off, wrong address) or it
// didn't answer in time. Its other models are skipped then, so a dead or
// stuck server doesn't cost a wait per model before the backup AI.
func HostDown(err error) bool {
	var op *net.OpError
	if errors.As(err, &op) && op.Op == "dial" {
		return true
	}
	var dns *net.DNSError
	return errors.As(err, &dns) || errors.Is(err, context.DeadlineExceeded)
}

// TooLong reports whether the AI refused the request for not fitting its
// context window ("exceeds the available context size", "maximum context
// length", "input length exceeds the context length").
func TooLong(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context") && (strings.Contains(msg, "exceed") || strings.Contains(msg, "maximum") ||
		strings.Contains(msg, "too long") || strings.Contains(msg, "n_ctx"))
}

// Reasoning reports whether a model thinks before it answers (DeepSeek-R1,
// QwQ, Qwen3, gpt-oss, o1/o3…): it needs room for its thinking, which is
// stripped from the answer (stripThinking).
func Reasoning(model string) bool {
	m := strings.ToLower(model)
	for _, k := range []string{"deepseek-r1", "-r1", "r1:", "qwq", "qwen3", "gpt-oss", "magistral", "reason", "think", "o1-", "o3", "o4-mini"} {
		if strings.Contains(m, k) {
			return true
		}
	}
	return m == "o1" || strings.HasPrefix(m, "o1:")
}

// MaxTokens is how long an answer may be: a recipe as JSON is up to a few
// thousand tokens, plus a reasoning model's thinking.
func MaxTokens(model string) int {
	if Reasoning(model) {
		return 8000
	}
	return 4000
}

// Reachable asks an OpenAI-compatible server for its models (it costs no
// tokens) and says which of want it doesn't have. Ollama, LM Studio, OpenAI
// and Gemini all list their models this way.
func Reachable(ctx context.Context, baseURL, apiKey string, want []string) (missing []string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, NormalizeBaseURL(baseURL)+"/models", nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
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
	if json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&list) != nil || len(list.Data) == 0 {
		return nil, nil // answered, but doesn't list its models: nothing more to check
	}
	have := map[string]bool{}
	for _, m := range list.Data {
		id := strings.TrimPrefix(m.ID, "models/") // Gemini says "models/gemini-…"
		have[id], have[strings.TrimSuffix(id, ":latest")] = true, true
	}
	for _, m := range want {
		if !have[m] && !have[strings.TrimSuffix(m, ":latest")] {
			missing = append(missing, m)
		}
	}
	return missing, nil
}
