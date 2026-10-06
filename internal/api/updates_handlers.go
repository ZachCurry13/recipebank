package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/zachcurry13/recipebank"
	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/push"
	"github.com/zachcurry13/recipebank/internal/store"
	"github.com/zachcurry13/recipebank/internal/version"
)

// handleUpdates reports the running version, whether a newer release is out
// (for parents; kids aren't nagged), GitHub's release notes and the bundled
// changelog.
func (s *Server) handleUpdates(w http.ResponseWriter, r *http.Request) {
	enabled := s.Store.Setting(store.KeyCheckUpdates) == "true"
	st := s.Updates.Status(r.Context(), enabled)
	if !auth.UserFrom(r).CanManage() {
		st.UpdateAvailable = false
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": st, "checks_enabled": enabled, "changelog": recipebank.Changelog})
}

// handleSeen records that someone saw the welcome or "What's new" for the
// running version.
func (s *Server) handleSeen(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.SetSeen(auth.UserFrom(r).ID, version.Version); err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"seen_version": version.Version})
}

// notifyUpdate tells admins' phones once about each new release.
func (s *Server) notifyUpdate() {
	if s.Updates == nil || s.Store.Setting(store.KeyCheckUpdates) != "true" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	st := s.Updates.Status(ctx, true)
	if !st.UpdateAvailable || s.Store.Setting("update_notified") == st.Latest {
		return
	}
	_ = s.Store.SetSetting("update_notified", st.Latest)
	users, err := s.Store.ListUsers()
	if err != nil {
		return
	}
	m := push.Message{Title: "🆕 RecipeBank " + strings.TrimPrefix(st.Latest, "v") + " is out",
		Body: "See what's new, then update the app on your server.", URL: "/#/whatsnew", Tag: "updates"}
	for _, u := range users {
		if u.IsAdmin() {
			s.Push.Notify("updates", u.ID, m)
		}
	}
}
