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
	"github.com/zachcurry13/recipebank/internal/store"
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
		PhotoDir: filepath.Join(cfg.DataDir, "photos"),
	}
	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go housekeeping(ctx, st, srv)

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
