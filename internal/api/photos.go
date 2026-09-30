package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zachcurry13/recipebank/internal/recipe"
)

const maxPhoto = 8 << 20

var photoName = regexp.MustCompile(`^[a-f0-9]{32}\.(jpg|png|webp|gif)$`)

var photoExt = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif"}

// savePhoto stores an image under a random name and returns the name.
func (s *Server) savePhoto(data []byte) (string, error) {
	if len(data) == 0 || len(data) > maxPhoto {
		return "", errors.New("photos must be under 8 MB")
	}
	ext, ok := photoExt[http.DetectContentType(data)]
	if !ok {
		return "", errors.New("that file isn't a photo (JPEG, PNG, WebP or GIF)")
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	name := hex.EncodeToString(b) + ext
	if err := os.MkdirAll(s.PhotoDir, 0o750); err != nil {
		return "", err
	}
	return name, os.WriteFile(filepath.Join(s.PhotoDir, name), data, 0o640)
}

// decodeDataURL reads "data:image/jpeg;base64,…".
func decodeDataURL(d string) ([]byte, string, error) {
	head, b64, ok := strings.Cut(d, ",")
	if !ok || !strings.HasPrefix(head, "data:") || !strings.HasSuffix(head, ";base64") {
		return nil, "", errors.New("photos must be sent as data: URLs")
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, "", errors.New("that photo couldn't be read")
	}
	mt := http.DetectContentType(data)
	if _, ok := photoExt[mt]; !ok {
		return nil, "", errors.New("that file isn't a photo (JPEG, PNG, WebP or GIF)")
	}
	return data, mt, nil
}

// handleUploadPhoto stores a dish photo ({"image": "data:…"}) → {"name": …}.
func (s *Server) handleUploadPhoto(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Image string `json:"image"`
	}
	if !readJSON(w, r, &body, 12<<20) {
		return
	}
	data, _, err := decodeDataURL(body.Image)
	if err == nil {
		var name string
		if name, err = s.savePhoto(data); err == nil {
			writeJSON(w, http.StatusOK, map[string]string{"name": name})
			return
		}
	}
	writeErr(w, http.StatusBadRequest, err.Error())
}

// handlePhoto serves a stored photo. Names are random and never reused, so
// browsers may keep them.
func (s *Server) handlePhoto(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if !photoName.MatchString(name) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Del("Pragma")
	http.ServeFile(w, r, filepath.Join(s.PhotoDir, name))
}

// downloadPhoto saves a web recipe's photo.
func (s *Server) downloadPhoto(ctx context.Context, raw string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	data, _, err := s.fetch(ctx, raw, maxPhoto)
	if err != nil {
		return "", err
	}
	return s.savePhoto(data)
}

// ownPhotos checks that a recipe only points at photos stored here.
func (s *Server) ownPhotos(rc *recipe.Recipe) bool {
	names := append([]string{rc.Photo}, rc.SourcePhotos...)
	for _, n := range names {
		if n == "" {
			continue
		}
		if !photoName.MatchString(n) {
			return false
		}
		if _, err := os.Stat(filepath.Join(s.PhotoDir, n)); err != nil {
			return false
		}
	}
	return true
}

// cleanPhotos removes photos no recipe uses any more, once they're a day
// old (younger ones may belong to a draft still being edited).
func (s *Server) cleanPhotos() {
	used, err := s.Store.PhotosInUse()
	if err != nil {
		return
	}
	entries, err := os.ReadDir(s.PhotoDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || used[e.Name()] || !photoName.MatchString(e.Name()) || time.Since(info.ModTime()) < 24*time.Hour {
			continue
		}
		_ = os.Remove(filepath.Join(s.PhotoDir, e.Name()))
	}
}

// CleanPhotos is cleanPhotos for the daily background job.
func (s *Server) CleanPhotos() { s.cleanPhotos() }
