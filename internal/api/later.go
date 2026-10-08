package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/push"
	"github.com/zachcurry13/recipebank/internal/recipe"
	"github.com/zachcurry13/recipebank/internal/safety"
	"github.com/zachcurry13/recipebank/internal/store"
)

// "Read it later" (the Add page): card photos and pasted recipes no AI could
// read when they were added wait on a list. ReadLater reads them in the
// background and keeps each as a draft until someone checks and saves it.

// laterKeep is how long RecipeBank keeps trying.
const laterKeep = 48 * time.Hour

// laterWaits are the pauses between tries: short at first, then every two hours.
var laterWaits = []time.Duration{2 * time.Minute, 5 * time.Minute, 10 * time.Minute, 20 * time.Minute,
	30 * time.Minute, time.Hour, 2 * time.Hour}

func laterWait(tries int) time.Duration {
	return laterWaits[min(tries, len(laterWaits)-1)]
}

var errPhotosGone = errors.New("its photos aren't here any more: add the card again")

// importFailed says why the AI couldn't read it. When an AI may manage
// later, the answer says so, with the photos already kept, and the page
// offers "Read it later".
func importFailed(w http.ResponseWriter, err error, photos []string) {
	out := map[string]any{"error": importErr(err)}
	if llm.Temporary(err) {
		out["later"] = true
		if photos != nil {
			out["photos"] = photos
		}
	}
	writeJSON(w, http.StatusBadGateway, out)
}

func (s *Server) handleAddLater(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind   string          `json:"kind"` // photo | text
		Photos []string        `json:"photos"`
		Text   string          `json:"text"`
		Area   string          `json:"area"`
		Source json.RawMessage `json:"source"` // the page's note of where it came from, kept for it
	}
	if !readJSON(w, r, &body, 300<<10) {
		return
	}
	l := store.Later{Kind: body.Kind, Area: recipe.AreaKitchen}
	if body.Area == recipe.AreaHome {
		l.Area = recipe.AreaHome
	}
	switch body.Kind {
	case "photo":
		if len(body.Photos) == 0 || len(body.Photos) > 4 {
			writeErr(w, http.StatusBadRequest, "add 1 to 4 photos")
			return
		}
		for _, p := range body.Photos {
			if _, ok := s.photoPath(p); !ok {
				writeErr(w, http.StatusBadRequest, errPhotosGone.Error())
				return
			}
		}
		l.Photos = body.Photos
	case "text":
		if strings.TrimSpace(body.Text) == "" {
			writeErr(w, http.StatusBadRequest, "paste the recipe's text first")
			return
		}
		l.Text = body.Text
	default:
		writeErr(w, http.StatusBadRequest, "a photo or pasted text")
		return
	}
	if src := string(body.Source); src != "" && src != "null" && len(src) < 2000 && json.Valid(body.Source) {
		l.Source = src
	}
	u := auth.UserFrom(r)
	l.UserID, l.AddedBy = u.ID, u.Username
	id, err := s.Store.AddLater(l, time.Now().Add(laterWait(0)), laterKeep)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (s *Server) handleLaters(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.Laters()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	if items == nil {
		items = []store.Later{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// handleLaterDraft opens what was read, checked for everyone like any import.
func (s *Server) handleLaterDraft(w http.ResponseWriter, r *http.Request) {
	l, ok := s.laterItem(w, r)
	if !ok {
		return
	}
	var rc recipe.Recipe
	if l.Status != "ready" || json.Unmarshal([]byte(l.Draft), &rc) != nil {
		writeErr(w, http.StatusConflict, "it hasn't been read yet")
		return
	}
	s.writeChecked(w, r, &rc)
}

// handleLaterNow puts an item back in line to be read at once.
func (s *Server) handleLaterNow(w http.ResponseWriter, r *http.Request) {
	l, ok := s.laterItem(w, r)
	if !ok {
		return
	}
	if err := s.Store.LaterNow(l.ID, time.Now(), laterKeep); err != nil {
		writeStoreErr(w, err)
		return
	}
	s.wakeLater()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteLater(w http.ResponseWriter, r *http.Request) {
	l, ok := s.laterItem(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteLater(l.ID); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) laterItem(w http.ResponseWriter, r *http.Request) (store.Later, bool) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return store.Later{}, false
	}
	l, err := s.Store.LaterByID(id)
	if err != nil {
		writeStoreErr(w, err)
		return l, false
	}
	return l, true
}

// ReadLater reads the waiting cards and pasted recipes in the background,
// one at a time, as their turns come.
func (s *Server) ReadLater(ctx context.Context) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		for s.readNextLater(ctx, time.Now()) {
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-s.laterSignal():
		}
	}
}

// wakeLater starts reading now ("Try now").
func (s *Server) wakeLater() {
	select {
	case s.laterSignal() <- struct{}{}:
	default:
	}
}

func (s *Server) laterSignal() chan struct{} {
	s.laterOnce.Do(func() { s.laterCh = make(chan struct{}, 1) })
	return s.laterCh
}

// readNextLater reads the item whose turn it is. It reports whether to go on
// to the next one now: not after an AI that's still busy or off.
func (s *Server) readNextLater(ctx context.Context, now time.Time) bool {
	l, due, err := s.Store.LaterDue(now)
	if err != nil || !due {
		return false
	}
	rctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	rc, err := s.readLaterItem(rctx, l)
	waiting := llm.Temporary(err) || errors.Is(err, llm.ErrNoAI)
	switch {
	case err == nil:
		if l.Area == recipe.AreaHome {
			rc.Area = recipe.AreaHome
		}
		rc.Clean()
		draft, _ := json.Marshal(rc)
		_ = s.Store.LaterRead(l.ID, rc.Title, string(draft))
		s.Push.Notify("ready", l.UserID, push.Message{Title: "📥 Ready to check",
			Body: rc.Title + " was read. Check it and save it.", URL: "/#/add", Tag: "ready"})
		return true
	case ctx.Err() != nil:
		return false // shutting down: it stays in line
	case waiting && now.Before(parseTime(l.GiveUp)):
		_ = s.Store.LaterRetry(l.ID, importErr(err), now.Add(laterWait(l.Tries+1)))
		return false
	case waiting:
		_ = s.Store.LaterFailed(l.ID, "No AI could read it for two days. "+importErr(err))
	default:
		_ = s.Store.LaterFailed(l.ID, importErr(err))
	}
	return true
}

func (s *Server) readLaterItem(ctx context.Context, l store.Later) (recipe.Recipe, error) {
	if l.Kind == "text" {
		rc, err := s.askText(ctx, l.Text)
		rc.SourceKind = "text"
		return rc, err
	}
	var images []llm.Image
	for _, name := range l.Photos {
		p, ok := s.photoPath(name)
		if !ok {
			return recipe.Recipe{}, errPhotosGone
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return recipe.Recipe{}, errPhotosGone
		}
		images = append(images, llm.Image{Data: data, MediaType: http.DetectContentType(data)})
	}
	rc, err := s.readPhotos(ctx, images)
	if err != nil {
		return rc, err
	}
	rc.SourceKind, rc.SourcePhotos = "photo", l.Photos
	if len(safety.CrossCheck(&rc)) > 0 {
		rc.NeedsReview = true
	}
	return rc, nil
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
