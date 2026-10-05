// Package config loads process-level configuration from environment variables.
// Runtime-tunable options (the AI, the allergen list) live in the settings
// table instead so admins can change them from the web page.
package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/zachcurry13/recipebank/internal/files"
)

type Config struct {
	Addr          string   // HTTP listen address
	DataDir       string   // recipebank.db (SSD dataset)
	PhotosDir     string   // recipe photos and card scans: /photos when mounted, else <data>/photos
	CacheDir      string   // previews that can be made again: /cache when mounted, else <data>/cache
	AdminUser     string   // bootstrap admin username
	AdminPassword string   // optional: pre-create the first admin (else set up in the browser)
	TrustProxy    bool     // honor X-Forwarded-For / CF-Connecting-IP
	CORSOrigins   []string // allowed cross-origin callers (empty = same-origin only)
	SessionDays   int      // session lifetime
}

func Load() Config {
	c := Config{
		Addr:          env("RECIPEBANK_ADDR", ":8080"),
		DataDir:       env("RECIPEBANK_DATA_DIR", "/data"),
		AdminUser:     env("RECIPEBANK_ADMIN_USER", "admin"),
		AdminPassword: os.Getenv("RECIPEBANK_ADMIN_PASSWORD"),
		TrustProxy:    envBool("RECIPEBANK_TRUST_PROXY", true),
		CORSOrigins:   splitList(os.Getenv("RECIPEBANK_CORS_ORIGINS")),
		SessionDays:   envInt("RECIPEBANK_SESSION_DAYS", 30),
	}
	c.PhotosDir = files.Dir("RECIPEBANK_PHOTOS_DIR", "/photos", c.DataDir, "photos")
	c.CacheDir = files.Dir("RECIPEBANK_CACHE_DIR", "/cache", c.DataDir, "cache")
	return c
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	if b, err := strconv.ParseBool(os.Getenv(key)); err == nil {
		return b
	}
	return def
}

func envInt(key string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil && n > 0 {
		return n
	}
	return def
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
