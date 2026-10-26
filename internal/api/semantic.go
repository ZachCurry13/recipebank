package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/store"
)

// Search by meaning (Admin → AI): an embedding model turns each recipe and
// each search into numbers; recipes whose numbers point the same way as the
// search are alike in meaning, even without sharing its words ("something
// warm for a cold night"). It only ranks and finds recipes: the allergy checks
// never depend on it. Word matches always come first.

// embedTarget is where embeddings are asked for: the main AI's service, or
// the photo AI's when the main one can't (Claude has no embeddings).
func (s *Server) embedTarget() (baseURL, key, model string, ok bool) {
	model = s.Store.Setting(store.KeyEmbedModel)
	if model == "" {
		return "", "", "", false
	}
	for _, ai := range []store.AIConfig{s.Store.AIConfig(), s.Store.PhotoAIConfig()} {
		if ai.Provider != "anthropic" && ai.BaseURL != "" {
			return ai.BaseURL, ai.APIKey, model, true
		}
	}
	return "", "", model, false
}

// recipeDoc is the text a recipe is embedded from.
func recipeDoc(rc *recipe.Recipe) string {
	var b strings.Builder
	b.WriteString(rc.Title + ". " + rc.Summary + " ")
	for _, p := range []string{rc.Course, rc.Cuisine, rc.Protein} {
		if p != "" {
			b.WriteString(p + ". ")
		}
	}
	b.WriteString("Ingredients: ")
	for _, in := range rc.Ingredients {
		b.WriteString(in.Line + ", ")
	}
	if r := []rune(b.String()); len(r) > 2000 {
		return string(r[:2000])
	}
	return b.String()
}

func docHash(doc string) string {
	h := sha256.Sum256([]byte(doc))
	return hex.EncodeToString(h[:8])
}

func (s *Server) semanticWake() chan struct{} {
	s.semOnce.Do(func() { s.semCh = make(chan struct{}, 1) })
	return s.semCh
}

// IndexMeaning keeps every recipe's embedding up to date in the background.
func (s *Server) IndexMeaning(ctx context.Context) {
	t := time.NewTicker(2 * time.Minute)
	defer t.Stop()
	for {
		s.indexRecipes(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-s.semanticWake():
		}
	}
}

func (s *Server) setSemErr(msg string) {
	s.semMu.Lock()
	s.semErr = msg
	s.semMu.Unlock()
}

func (s *Server) indexRecipes(ctx context.Context) {
	base, key, model, ok := s.embedTarget()
	if !ok {
		s.setSemErr("")
		return
	}
	all, err := s.Store.ListRecipes("")
	if err != nil {
		return
	}
	have, err := s.Store.EmbeddedHashes(model)
	if err != nil {
		return
	}
	type job struct {
		id        int64
		doc, hash string
	}
	var todo []job
	for i := range all {
		doc := recipeDoc(&all[i])
		if h := docHash(doc); have[all[i].ID] != h {
			todo = append(todo, job{all[i].ID, doc, h})
		}
	}
	for len(todo) > 0 {
		n := min(len(todo), 8)
		docs := make([]string, n)
		for i := range docs {
			docs[i] = todo[i].doc
		}
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		vecs, err := llm.Embed(cctx, nil, base, key, model, docs)
		cancel()
		if err != nil {
			if ctx.Err() == nil {
				s.setSemErr(err.Error())
			}
			return
		}
		for i := range vecs {
			if err := s.Store.SaveEmbedding(todo[i].id, model, todo[i].hash, vecs[i]); err != nil {
				return
			}
		}
		todo = todo[n:]
	}
	s.setSemErr("")
}

// semanticScores finds the recipes closest in meaning to a search: the few
// best, and only those near the best one. It returns nil when the feature is
// off, nothing is indexed yet, or the AI doesn't answer quickly.
func (s *Server) semanticScores(ctx context.Context, q string) map[int64]float64 {
	q = strings.TrimSpace(q)
	if len([]rune(q)) < 3 {
		return nil
	}
	base, key, model, ok := s.embedTarget()
	if !ok {
		return nil
	}
	vecs, err := s.Store.Embeddings(model)
	if err != nil || len(vecs) == 0 {
		return nil
	}
	cacheKey := model + "|" + strings.ToLower(q)
	s.semMu.Lock()
	qv := s.semCache[cacheKey]
	s.semMu.Unlock()
	if qv == nil {
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		out, err := llm.Embed(cctx, nil, base, key, model, []string{q})
		if err != nil {
			return nil
		}
		qv = out[0]
		s.semMu.Lock()
		if s.semCache == nil || len(s.semCache) > 200 {
			s.semCache = map[string][]float32{}
		}
		s.semCache[cacheKey] = qv
		s.semMu.Unlock()
	}
	type hit struct {
		id    int64
		score float64
	}
	var hits []hit
	for id, v := range vecs {
		hits = append(hits, hit{id, llm.Cosine(qv, v)})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if len(hits) > 8 {
		hits = hits[:8]
	}
	cut := max(hits[0].score-0.1, 0.3)
	out := map[int64]float64{}
	for _, h := range hits {
		if h.score >= cut {
			out[h.id] = h.score
		}
	}
	return out
}

// handleEmbedStatus tells Admin how far indexing has got.
func (s *Server) handleEmbedStatus(w http.ResponseWriter, r *http.Request) {
	_, _, model, ok := s.embedTarget()
	indexed, total := 0, 0
	if model != "" {
		if have, err := s.Store.EmbeddedHashes(model); err == nil {
			indexed = len(have)
		}
	}
	total, _ = s.Store.CountRecipes()
	s.semMu.Lock()
	errMsg := s.semErr
	s.semMu.Unlock()
	ai := s.Store.AIConfig()
	out := map[string]any{"model": model, "ready": ok, "indexed": indexed, "total": total, "error": errMsg,
		"ollama": ollamaBase(ai.BaseURL) != "", "anthropic_only": ai.Provider == "anthropic" && !s.Store.PhotoAIConfig().Ready()}
	if b := ollamaBase(ai.BaseURL); b != "" {
		out["ollama_url"] = b
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleEmbedSet(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model string `json:"model"`
	}
	if !readJSON(w, r, &body, 1<<10) {
		return
	}
	model := strings.TrimSpace(body.Model)
	if len(model) > 120 || strings.ContainsAny(model, " \n\t\"'") {
		writeErr(w, http.StatusBadRequest, "that doesn't look like a model name, for example nomic-embed-text")
		return
	}
	if err := s.Store.SetSetting(store.KeyEmbedModel, model); err != nil {
		writeStoreErr(w, err)
		return
	}
	_ = s.Store.DropOtherEmbeddings(model)
	s.setSemErr("")
	select {
	case s.semanticWake() <- struct{}{}:
	default:
	}
	s.handleEmbedStatus(w, r)
}
