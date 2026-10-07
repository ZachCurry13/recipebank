package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/llm"
	"github.com/zachcurry13/recipebank/internal/store"
)

// editableKeys are the settings the admin page may change.
var editableKeys = map[string]bool{
	store.KeyLLMProvider: true, store.KeyLLMBaseURL: true, store.KeyLLMAPIKey: true, store.KeyLLMModel: true,
	store.KeyLLMFallbackModel: true, store.KeyLLMVisionModel: true, store.KeyLLMJSONMode: true,
	store.KeyLLMTimeoutSeconds: true, store.KeySessionDays: true, store.KeyAllergenList: true, store.KeyDefaultUnits: true,
	store.KeyMorningHour: true, store.KeyTonightHour: true, store.KeyCheckUpdates: true,
	store.KeyUSDAKey: true, store.KeySMTPHost: true, store.KeySMTPPort: true, store.KeySMTPUser: true,
	store.KeySMTPPassword: true, store.KeySMTPFrom: true, store.KeyCurrency: true, store.KeyBudgetWeekly: true, store.KeyEmailWho: true,
	store.KeyPhotoProvider: true, store.KeyPhotoBaseURL: true, store.KeyPhotoAPIKey: true, store.KeyPhotoModel: true,
	store.KeyPhotoJSONMode: true,
}

func init() {
	for _, f := range store.Features {
		editableKeys[store.FeatureKey(f)] = true
	}
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	all, err := s.Store.AllSettings()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	out := map[string]any{}
	for k, v := range all {
		if store.SecretKeys[k] {
			out[k+"_set"] = v != "" // never send the secret itself
			continue
		}
		out[k] = v
	}
	out["tokens_this_month"] = s.Store.TokensThisMonth()
	out["folders"] = map[string]any{"data": s.Cfg.DataDir, "photos": s.PhotoDir, "cache": s.CacheDir,
		"photos_own": !inside(s.Cfg.DataDir, s.PhotoDir), "cache_own": !inside(s.Cfg.DataDir, s.CacheDir)}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if !readJSON(w, r, &body, 16<<10) {
		return
	}
	for k, v := range body {
		if !editableKeys[k] {
			writeErr(w, http.StatusBadRequest, "unknown setting "+k)
			return
		}
		v = strings.TrimSpace(v)
		if store.SecretKeys[k] && v == "" {
			continue // blank keeps the saved secret
		}
		if k == store.KeyPhotoProvider && v != "" && v != "openai" && v != "anthropic" {
			writeErr(w, http.StatusBadRequest, "the photo AI is OpenAI-compatible, Anthropic, or none")
			return
		}
		if k == store.KeyLLMBaseURL || k == store.KeyPhotoBaseURL {
			v = llm.NormalizeBaseURL(v)
		}
		if err := s.Store.SetSetting(k, v); err != nil {
			writeStoreErr(w, err)
			return
		}
	}
	s.handleGetSettings(w, r)
}

// handleTestAI asks the AI (?which=photo: the photo AI) a tiny question to
// prove the settings work.
func (s *Server) handleTestAI(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	ai := s.Store.AIConfig()
	if r.URL.Query().Get("which") == "photo" {
		if ai = s.Store.PhotoAIConfig(); !ai.Ready() {
			writeErr(w, http.StatusBadRequest, "no photo AI is set up: choose its kind, address and model, then Save")
			return
		}
	}
	start := time.Now()
	model, err := llm.AskWith(ctx, s.Store, ai, "You are a test. Reply with JSON only.", `Reply exactly {"ok": true}`, func(out string) error {
		var v struct {
			OK bool `json:"ok"`
		}
		if json.Unmarshal([]byte(jsonObject(out)), &v) != nil || !v.OK {
			return errors.New("unexpected answer: " + truncate(out, 120))
		}
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "seconds": time.Since(start).Seconds(), "model": model})
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	us, err := s.Store.ListUsers()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, us)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
		PersonID *int64 `json:"person_id"`
	}
	if !readJSON(w, r, &body, 4<<10) {
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if !usernameRE.MatchString(body.Username) || !store.ValidRole(body.Role) {
		writeErr(w, http.StatusBadRequest, "username must be 2-40 letters or digits; role admin, editor or kid")
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := s.Store.CreateUser(body.Username, hash, body.Role)
	if err != nil {
		writeErr(w, http.StatusConflict, "that username is taken")
		return
	}
	if body.PersonID != nil {
		_ = s.Store.UpdateUser(u.ID, u.Role, body.PersonID)
	}
	writeJSON(w, http.StatusCreated, u)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var body struct {
		Role     string `json:"role"`
		PersonID *int64 `json:"person_id"`
	}
	if !ok || !readJSON(w, r, &body, 4<<10) {
		return
	}
	u, err := s.Store.UserByID(id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	if u.IsAdmin() && body.Role != store.RoleAdmin && s.lastAdmin() {
		writeErr(w, http.StatusBadRequest, "keep at least one admin")
		return
	}
	if body.PersonID != nil && *body.PersonID == 0 {
		body.PersonID = nil
	}
	if err := s.Store.UpdateUser(id, body.Role, body.PersonID); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var body struct {
		Password string `json:"password"`
	}
	if !ok || !readJSON(w, r, &body, 4<<10) {
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Store.SetPassword(id, hash); err != nil {
		writeStoreErr(w, err)
		return
	}
	_ = s.Store.DeleteUserSessions(id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	if id == auth.UserFrom(r).ID {
		writeErr(w, http.StatusBadRequest, "you can't delete your own account")
		return
	}
	if u, err := s.Store.UserByID(id); err == nil && u.IsAdmin() && s.lastAdmin() {
		writeErr(w, http.StatusBadRequest, "keep at least one admin")
		return
	}
	if err := s.Store.DeleteUser(id); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) lastAdmin() bool {
	n, err := s.Store.CountAdmins()
	return err == nil && n <= 1
}

func jsonObject(s string) string {
	i, j := strings.Index(s, "{"), strings.LastIndex(s, "}")
	if i < 0 || j < i {
		return s
	}
	return s[i : j+1]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// inside reports whether path is dir or a folder within it.
func inside(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
