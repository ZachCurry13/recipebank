package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
)

// Embed turns each text into an embedding (numbers capturing its meaning)
// with an OpenAI-compatible service: Ollama, Gemini, OpenAI. Texts that are
// alike in meaning give vectors that point the same way (see Cosine).
func Embed(ctx context.Context, hc *http.Client, baseURL, apiKey, model string, texts []string) ([][]float32, error) {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(model) == "" {
		return nil, errors.New("an address and an embedding model are needed")
	}
	if hc == nil {
		hc = &http.Client{}
	}
	body, _ := json.Marshal(map[string]any{"model": model, "input": texts})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	var out struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("embeddings %s: %s", resp.Status, truncate(string(raw), 200))
	}
	if out.Error != nil {
		return nil, fmt.Errorf("embeddings error: %s", out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK || len(out.Data) != len(texts) {
		return nil, fmt.Errorf("embeddings %s: got %d answers for %d texts", resp.Status, len(out.Data), len(texts))
	}
	vecs := make([][]float32, len(texts))
	for i, d := range out.Data {
		at := d.Index
		if at < 0 || at >= len(vecs) || vecs[at] != nil {
			at = i
		}
		if len(d.Embedding) == 0 {
			return nil, errors.New("the model gave an empty embedding: is it an embedding model (for example nomic-embed-text)?")
		}
		vecs[at] = d.Embedding
	}
	return vecs, nil
}

// Cosine is how alike two embeddings are: 1 = same meaning, 0 = unrelated.
func Cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
