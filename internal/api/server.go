// Package api wires the HTTP routes for the JSON API and embedded web app.
package api

import (
	"io/fs"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/config"
	"github.com/zachcurry13/recipebank/internal/store"
)

type Server struct {
	Cfg       config.Config
	Store     *store.Store
	Auth      *auth.Manager
	Web       fs.FS        // embedded static assets
	PhotoDir  string       // recipe photos (under the data folder)
	Fetch     *http.Client // outside pages and images; tests swap it
	logins    *loginLimiter
	etags     sync.Map  // static file name → ETag
	buildOnce sync.Once // buildID, computed once
	build     string
	indexOnce sync.Once // index.html with versioned addresses
	index     []byte
}

func (s *Server) Router() http.Handler {
	s.logins = newLoginLimiter()
	if s.Fetch == nil {
		s.Fetch = newFetchClient()
	}
	r := chi.NewRouter()
	r.Use(realIP(s.Cfg.TrustProxy))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(securityHeaders)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })

	r.Route("/api", func(r chi.Router) {
		r.Use(noStore)
		r.Use(cors(s.Cfg.CORSOrigins))
		r.Use(csrfGuard)

		r.Get("/setup", s.handleSetupStatus)
		r.Post("/setup", s.handleSetup)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)

		r.Group(func(r chi.Router) {
			r.Use(s.Auth.RequireUser)
			r.Get("/me", s.handleMe)
			r.Put("/me/prefs", s.handlePrefs)
			r.Put("/me/password", s.handleChangePassword)
			r.Get("/info", s.handleInfo)
			r.Get("/people", s.handleListPeople)
			r.Get("/recipes", s.handleListRecipes)
			r.Get("/recipes/{id}", s.handleGetRecipe)
			r.Put("/recipes/{id}/rating", s.handleRating)
			r.Get("/photos/{name}", s.handlePhoto)

			// Parents: add and edit recipes and people.
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireManager)
				r.Post("/import/url", s.handleImportURL)
				r.Post("/import/photo", s.handleImportPhoto)
				r.Post("/import/text", s.handleImportText)
				r.Post("/check", s.handleCheckDraft)
				r.Post("/recipes", s.handleSaveRecipe)
				r.Put("/recipes/{id}", s.handleSaveRecipe)
				r.Delete("/recipes/{id}", s.handleDeleteRecipe)
				r.Post("/recipes/{id}/version", s.handleVersion)
				r.Put("/recipes/{id}/label", s.handleLabelChecked)
				r.Post("/photos", s.handleUploadPhoto)
				r.Post("/people", s.handleSavePerson)
				r.Put("/people/{id}", s.handleSavePerson)
				r.Delete("/people/{id}", s.handleDeletePerson)
				r.Put("/household", s.handleHousehold)
			})

			// Admins: the AI and accounts.
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireAdmin)
				r.Get("/admin/settings", s.handleGetSettings)
				r.Put("/admin/settings", s.handlePutSettings)
				r.Post("/admin/ai/test", s.handleTestAI)
				r.Get("/admin/users", s.handleListUsers)
				r.Post("/admin/users", s.handleCreateUser)
				r.Put("/admin/users/{id}", s.handleUpdateUser)
				r.Put("/admin/users/{id}/password", s.handleResetPassword)
				r.Delete("/admin/users/{id}", s.handleDeleteUser)
			})
		})
	})

	r.NotFound(s.serveStatic)
	return r
}
