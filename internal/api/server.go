// Package api wires the HTTP routes for the JSON API and embedded web app.
package api

import (
	"html/template"
	"io/fs"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/zachcurry13/recipebank/internal/aitools"
	"github.com/zachcurry13/recipebank/internal/auth"
	"github.com/zachcurry13/recipebank/internal/books"
	"github.com/zachcurry13/recipebank/internal/config"
	"github.com/zachcurry13/recipebank/internal/foodfacts"
	"github.com/zachcurry13/recipebank/internal/mail"
	"github.com/zachcurry13/recipebank/internal/nutrition"
	"github.com/zachcurry13/recipebank/internal/ollama"
	"github.com/zachcurry13/recipebank/internal/push"
	"github.com/zachcurry13/recipebank/internal/store"
	"github.com/zachcurry13/recipebank/internal/tunnel"
	"github.com/zachcurry13/recipebank/internal/updates"
)

type Server struct {
	Cfg         config.Config
	Store       *store.Store
	Auth        *auth.Manager
	Web         fs.FS        // embedded static assets
	PhotoDir    string       // recipe photos and card scans
	OldPhotos   string       // <data>/photos when photos moved to their own folder; still read from
	CacheDir    string       // previews (re-creatable)
	Fetch       *http.Client // outside pages and images; tests swap it
	Products    productBases // Open Food Facts databases; tests point them at a fake
	Tunnel      *tunnel.Manager
	Push        *push.Service                         // phone notifications
	Updates     *updates.Checker                      // newer releases on GitHub; tests point it at a fake
	Nutrition   string                                // USDA FoodData Central's address; tests point it at a fake
	BookBase    string                                // Open Library's address; tests point it at a fake
	Pulls       *ollama.Puller                        // Ollama model downloads, one at a time
	AITools     *aitools.Service                      // model updates and the speed test
	SendMail    func(mail.Config, mail.Message) error // tests catch emails here; nil = mail.Send
	mails       mailLimiter
	shareOnce   sync.Once // the shared-recipe page's template, parsed once
	shareTmpl   *template.Template
	timers      push.Timers // cook-mode timers that buzz the phone
	logins      *loginLimiter
	guestWrites *loginLimiter // changes through events' guest links, per address
	semMu       sync.Mutex    // "Search by meaning": the last indexing error, and recent searches' vectors
	semErr      string
	semCache    map[string][]float32
	semOnce     sync.Once
	semCh       chan struct{}
	tv          tvHub         // kitchen screens waiting for a phone
	tvNew       *loginLimiter // new screens, per address
	tvTries     *loginLimiter // codes tried from phones, per person
	etags       sync.Map      // static file name → ETag
	laterOnce   sync.Once     // "Read it later": the signal to read now
	laterCh     chan struct{}
	buildOnce   sync.Once // buildID, computed once
	build       string
	indexOnce   sync.Once // index.html with versioned addresses
	index       []byte
}

func (s *Server) Router() http.Handler {
	s.logins = newLoginLimiter()
	s.guestWrites = newRateLimiter(40, 10*60)
	s.tvNew = newRateLimiter(20, 10*60)
	s.tvTries = newRateLimiter(15, 10*60)
	if s.Fetch == nil {
		s.Fetch = newFetchClient()
	}
	if s.Push == nil {
		s.Push = push.New(s.Store)
	}
	if s.Tunnel == nil {
		s.Tunnel = tunnel.New()
	}
	if s.Updates == nil {
		s.Updates = updates.New()
	}
	if s.Nutrition == "" {
		s.Nutrition = nutrition.DefaultBase
	}
	if s.BookBase == "" {
		s.BookBase = books.DefaultBase
	}
	if s.Pulls == nil {
		s.Pulls = &ollama.Puller{}
	}
	if s.AITools == nil {
		s.AITools = aitools.New(s.Store)
	}
	if s.Products.FoodBases == nil {
		s.Products = productBases{FoodBases: foodfacts.FoodBases, HomeBases: foodfacts.HomeBases}
	}
	r := chi.NewRouter()
	r.Use(realIP(s.Cfg.TrustProxy))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(securityHeaders)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	r.Get("/share", s.handleShare)         // the phone's share menu; the app asks to sign in if needed
	r.Get("/s/{token}", s.handleSharePage) // a shared recipe, no sign-in
	r.Get("/s/{token}/photo", s.handleSharePhoto)
	r.Get("/tv", s.handleTVPage)                // the kitchen screen, no sign-in
	r.Get("/e/{token}", s.handleGuestPage)      // an event's guest page, no sign-in
	r.Get("/cal/{token}", s.handleCalendarFeed) // the meal plan for a calendar app, no sign-in

	r.Route("/api", func(r chi.Router) {
		r.Use(noStore)
		r.Use(cors(s.Cfg.CORSOrigins))
		r.Use(csrfGuard)

		// An event's guest link (no sign-in): see guest_handlers.go.
		r.Post("/tv/new", s.handleTVNew) // the kitchen screen starts here (see tv_handlers.go)
		r.Get("/tv/screen", s.handleTVScreen)
		r.Get("/guest/{token}", s.handleGuestView)
		r.Put("/guest/{token}/me", s.handleGuestMe)
		r.Delete("/guest/{token}/me", s.handleGuestLeave)
		r.Post("/guest/{token}/dishes", s.handleGuestDish)
		r.Delete("/guest/{token}/dishes/{id}", s.handleGuestDropDish)

		r.Get("/setup", s.handleSetupStatus)
		r.Post("/setup", s.handleSetup)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)

		r.Group(func(r chi.Router) {
			r.Use(s.Auth.RequireUser)
			s.memberRoutes(r)

			// Parents: add and edit recipes and people.
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireManager)
				s.parentRoutes(r)
			})

			// Admins: the AI and accounts.
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireAdmin)
				s.adminRoutes(r)
			})
		})
	})

	r.NotFound(s.serveStatic)
	return r
}

// productBases are the product databases for the pantry and the closet.
type productBases struct{ FoodBases, HomeBases []string }
