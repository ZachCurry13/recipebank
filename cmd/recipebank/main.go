// Command recipebank runs the RecipeBank web app: the JSON API and the
// embedded web pages, on one port.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/zachcurry13/recipebank/internal/api"
	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/config"
	"github.com/zachcurry13/recipebank/internal/db"
	"github.com/zachcurry13/recipebank/internal/files"
	"github.com/zachcurry13/recipebank/internal/store"
	"github.com/zachcurry13/recipebank/internal/tunnel"
	"github.com/zachcurry13/recipebank/internal/version"
	"github.com/zachcurry13/recipebank/web"
)

func main() {
	cfg := config.Load()
	log.Printf("RecipeBank %s starting; data in %s", version.Version, cfg.DataDir)

	database, err := db.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()
	st := store.New(database)
	if st.Setting(store.KeySessionDays) == store.Defaults[store.KeySessionDays] && cfg.SessionDays > 0 {
		store.Defaults[store.KeySessionDays] = strconv.Itoa(cfg.SessionDays)
	}
	if err := auth.Bootstrap(st, cfg.AdminUser, cfg.AdminPassword); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	srv := &api.Server{
		Cfg:      cfg,
		Store:    st,
		Auth:     &auth.Manager{Store: st, SessionDays: cfg.SessionDays},
		Web:      web.FS(),
		PhotoDir: cfg.PhotosDir,
		CacheDir: cfg.CacheDir,
	}
	setUpFolders(cfg, srv)

	// Built-in Cloudflare Tunnel, for using RecipeBank away from home.
	srv.Tunnel = tunnel.New()
	if st.SettingBool(store.KeyTunnelEnabled) {
		if err := srv.Tunnel.Apply(true, st.Setting(store.KeyTunnelToken)); err != nil {
			log.Printf("remote access not started: %v", err)
		}
	}
	defer srv.Tunnel.Stop()
	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go housekeeping(ctx, st, srv)
	go srv.Reminders(ctx)
	go srv.AITools.Loop(ctx) // newer versions of the Ollama models in use, once a day

	go func() {
		log.Printf("listening on %s", cfg.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()
	<-ctx.Done()
	log.Printf("shutting down")
	shut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shut)
}

// setUpFolders checks the photo and cache folders can be written, and moves
// photos into a newly mounted /photos dataset from the data folder.
func setUpFolders(cfg config.Config, srv *api.Server) {
	log.Printf("photos in %s; previews in %s", cfg.PhotosDir, cfg.CacheDir)
	for _, d := range []string{cfg.PhotosDir, cfg.CacheDir} {
		if err := files.Writable(d); err != nil {
			log.Printf("WARNING: can't write to %s (%v). Give the apps user (568) write access to that dataset.", d, err)
		}
	}
	old := filepath.Join(cfg.DataDir, "photos")
	if filepath.Clean(old) == filepath.Clean(cfg.PhotosDir) {
		return
	}
	srv.OldPhotos = old
	if n := files.MoveAll(old, cfg.PhotosDir); n > 0 {
		log.Printf("moved %d photos from %s to %s", n, old, cfg.PhotosDir)
	}
}

// housekeeping drops expired sessions and unused photos now and then.
func housekeeping(ctx context.Context, st *store.Store, srv *api.Server) {
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	for {
		_ = st.PurgeExpiredSessions()
		srv.CleanPhotos()
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
