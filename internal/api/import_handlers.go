package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
)

// userAgent says honestly who is asking. Some publishers block apps on
// purpose (403, or 402 "pay per crawl"); RecipeBank respects that and
// suggests pasting the text instead.
const userAgent = "RecipeBank/0.1 (+https://github.com/ZachCurry13/recipebank)"

func newFetchClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("only web addresses can be read")
			}
			return nil
		},
	}
}

// fetch downloads an outside page or image (at most limit bytes).
func (s *Server) fetch(ctx context.Context, raw string, limit int64) ([]byte, string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, "", errors.New("that doesn't look like a web address (it should start with https://)")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,image/*;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := s.Fetch.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("couldn't open that page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusForbidden, http.StatusPaymentRequired, http.StatusUnauthorized, http.StatusTooManyRequests:
			return nil, "", errors.New("that site blocks apps from reading it. Open the recipe in your browser, select and copy it, then use Paste text")
		}
		return nil, "", fmt.Errorf("that page answered %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	return body, resp.Header.Get("Content-Type"), err
}

// handleImportURL reads a recipe from a web page: the page's own recipe
// data when it has some, else the AI reads the page. Nothing is saved yet.
func (s *Server) handleImportURL(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL  string `json:"url"`
		Area string `json:"area"`
	}
	if !readJSON(w, r, &body, 8<<10) {
		return
	}
	page, _, err := s.fetch(r.Context(), body.URL, 5<<20)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	rc, ok := recipe.FromHTML(page)
	if !ok {
		title, text := recipe.PageText(page, 15000)
		rc, err = s.askText(r.Context(), "Page title: "+title+"\n\n"+text)
		if err != nil {
			writeErr(w, http.StatusBadGateway, importErr(err))
			return
		}
	}
	rc.SourceKind, rc.SourceURL = "web", strings.TrimSpace(body.URL)
	s.finishDraft(w, r, &rc, body.Area)
}

// handleImportText reads a recipe from pasted text.
func (s *Server) handleImportText(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text string `json:"text"`
		Area string `json:"area"`
	}
	if !readJSON(w, r, &body, 256<<10) {
		return
	}
	if strings.TrimSpace(body.Text) == "" {
		writeErr(w, http.StatusBadRequest, "paste the recipe's text first")
		return
	}
	// With the AI set up it reads the text; without it, a simple reader
	// handles recipes with "Ingredients" and "Directions" headings.
	rc, ok := recipe.FromText(body.Text)
	if s.Store.AnyAI() {
		ai, err := s.askText(r.Context(), body.Text)
		if err == nil {
			rc, ok = ai, true
		} else if !ok {
			writeErr(w, http.StatusBadGateway, importErr(err))
			return
		}
	}
	if !ok {
		writeErr(w, http.StatusBadRequest, "Couldn't find the ingredients and steps. Put the title on the first line, then a line saying Ingredients, then a line saying Directions (or set up the AI under Admin → AI).")
		return
	}
	rc.SourceKind = "text"
	s.finishDraft(w, r, &rc, body.Area)
}

// handleImportPhoto reads a recipe card, page or clipping (up to 4 photos:
// front and back, or pages in order). The photos are kept with the recipe.
func (s *Server) handleImportPhoto(w http.ResponseWriter, r *http.Request) {
	s.importPhotos(w, r, true)
}

// handleReadAgain reads a saved recipe's card photos once more (turned the
// right way up, or after changing the AI) without keeping anything: the
// recipe page shows the new reading beside the old one.
func (s *Server) handleReadAgain(w http.ResponseWriter, r *http.Request) { s.importPhotos(w, r, false) }

func (s *Server) importPhotos(w http.ResponseWriter, r *http.Request, keep bool) {
	var body struct {
		Images []string `json:"images"` // data: URLs
		Area   string   `json:"area"`
	}
	if !readJSON(w, r, &body, 40<<20) {
		return
	}
	originals, ok := readImages(w, body.Images, 4)
	if !ok {
		return
	}
	var names []string
	for _, im := range originals {
		if !keep {
			break
		}
		name, err := s.savePhoto(im.Data) // the full photo stays with the recipe
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		names = append(names, name)
	}
	rc, err := s.readPhotos(r.Context(), originals)
	if err != nil {
		writeErr(w, http.StatusBadGateway, importErr(err))
		return
	}
	rc.SourceKind, rc.SourcePhotos = "photo", names
	if len(safety.CrossCheck(&rc)) > 0 {
		rc.NeedsReview = true
	}
	s.finishDraft(w, r, &rc, body.Area)
}

// readPhotos reads a recipe from photos in two steps: the vision model copies
// the card as plain text, then organize turns the copy into a recipe.
func (s *Server) readPhotos(ctx context.Context, originals []llm.Image) (recipe.Recipe, error) {
	var copied string
	model, err := s.askPhotos(ctx, llm.TranscribePrompt(s.readingHints()), originals, func(out string) (perr error) {
		copied, perr = llm.ParseTranscript(out)
		return perr
	})
	if err != nil {
		return recipe.Recipe{}, err
	}
	rc, err := s.organize(ctx, copied)
	rc.ReadBy, rc.AIReading = model, copied
	return rc, err
}

// organize turns a card's plain-text copy into a recipe: the AI fills in the
// title, servings, steps and tags, while the ingredient lines come from the
// copy itself when it has them all, so no amount is lost.
func (s *Server) organize(ctx context.Context, copied string) (recipe.Recipe, error) {
	plain, ok := recipe.FromText(copied)
	ai, err := s.askText(ctx, llm.CardText(copied))
	switch {
	case err != nil && ok:
		return plain, nil
	case err != nil:
		return recipe.Recipe{}, err
	case ok:
		if len(plain.Ingredients) >= len(ai.Ingredients) {
			ai.Ingredients = plain.Ingredients
		}
		if len(ai.Steps) == 0 {
			ai.Steps = plain.Steps
		}
		if ai.Servings == 0 {
			ai.Servings = plain.Servings
		}
		ai.NeedsReview = ai.NeedsReview || plain.NeedsReview
	}
	return ai, nil
}

var errPhotoTooBig = errors.New("photo too big for the AI")

func (s *Server) askText(ctx context.Context, text string) (recipe.Recipe, error) {
	var rc recipe.Recipe
	err := llm.Ask(ctx, s.Store, llm.TextSystem, text, func(out string) (err error) {
		rc, err = llm.ParseRecipe(out)
		return err
	})
	if err == nil {
		// The lines as the person wrote them, even if the AI shortened them.
		rc.Ingredients = recipe.RestoreLines(text, rc.Ingredients, rc.Steps)
	}
	return rc, err
}

// finishDraft tidies an imported recipe and sends it back checked, unsaved.
func (s *Server) finishDraft(w http.ResponseWriter, r *http.Request, rc *recipe.Recipe, area string) {
	if area == recipe.AreaHome {
		rc.Area = recipe.AreaHome
	}
	rc.Clean()
	s.writeChecked(w, r, rc)
}

// importErr puts the AI's failures in plain words.
func importErr(err error) string {
	switch {
	case errors.Is(err, llm.ErrNoAI):
		return "This page has no recipe in the standard format, so the AI is needed to read it, and no AI is set up yet (Admin → AI)."
	case errors.Is(err, errPhotoTooBig):
		return "The photo doesn't fit in this AI model's working memory, even made smaller. Try one photo at a time, " +
			"or give the model more room on the AI server (for Ollama, a context length of 8192 or more), or use a cloud AI."
	case llm.TooLong(err):
		return "The text is too long for this AI model's working memory. Paste just the recipe, or give the model more room on the AI server."
	case errors.Is(err, llm.ErrNoRecipe) || strings.Contains(err.Error(), llm.ErrNoRecipe.Error()):
		return "No recipe was found there. Try a clearer photo, or paste the recipe's text."
	}
	return err.Error()
}
