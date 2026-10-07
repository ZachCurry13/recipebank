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
	Cfg       config.Config
	Store     *store.Store
	Auth      *auth.Manager
	Web       fs.FS        // embedded static assets
	PhotoDir  string       // recipe photos and card scans
	OldPhotos string       // <data>/photos when photos moved to their own folder; still read from
	CacheDir  string       // previews (re-creatable)
	Fetch     *http.Client // outside pages and images; tests swap it
	Products  productBases // Open Food Facts databases; tests point them at a fake
	Tunnel    *tunnel.Manager
	Push      *push.Service                         // phone notifications
	Updates   *updates.Checker                      // newer releases on GitHub; tests point it at a fake
	Nutrition string                                // USDA FoodData Central's address; tests point it at a fake
	BookBase  string                                // Open Library's address; tests point it at a fake
	Pulls     *ollama.Puller                        // Ollama model downloads, one at a time
	AITools   *aitools.Service                      // model updates and the speed test
	SendMail  func(mail.Config, mail.Message) error // tests catch emails here; nil = mail.Send
	mails     mailLimiter
	shareOnce sync.Once // the shared-recipe page's template, parsed once
	shareTmpl *template.Template
	timers    push.Timers // cook-mode timers that buzz the phone
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
			r.Put("/me/seen", s.handleSeen)
			r.Get("/updates", s.handleUpdates)
			r.Put("/me/prefs", s.handlePrefs)
			r.Put("/me/simpler", s.handleSimpler)
			r.Put("/me/password", s.handleChangePassword)
			r.Get("/info", s.handleInfo)
			r.Get("/people", s.handleListPeople)
			r.Get("/recipes", s.handleListRecipes)
			r.Get("/recipes/{id}", s.handleGetRecipe)
			r.Put("/recipes/{id}/rating", s.handleRating)
			r.Post("/recipes/{id}/substitute", s.handleSubstitute)
			r.Get("/recipes/{id}/cooks", s.handleListCooks)
			r.With(s.feature("nutrition")).Get("/recipes/{id}/nutrition", s.handleNutrition)
			r.Get("/cookbook", s.handleCookbook)
			r.Get("/books", s.handleListBooks)
			r.Get("/books/search", s.handleSearchBooks)
			r.Get("/books/{id}", s.handleGetBook)
			r.Get("/pile", s.handleListPile)
			r.Get("/events", s.handleListEvents)
			r.Get("/events/{id}", s.handleGetEvent)
			r.Post("/events/{id}/shopping", s.handleEventShopping)
			r.With(s.feature("email")).Post("/recipes/{id}/email", s.handleEmailRecipe)
			r.With(s.feature("email")).Post("/shopping/email", s.handleEmailShopping)
			r.Post("/recipes/{id}/cooks", s.handleAddCook)
			r.Post("/make", s.handleMake)
			r.Post("/make/photo", s.handleMakePhoto)
			r.Get("/photos/{name}", s.handlePhoto)
			r.Get("/stock", s.handleListStock)
			r.Post("/stock/{id}/adjust", s.handleAdjustStock)
			r.Get("/shopping", s.handleShopping)
			r.Post("/shopping", s.handleAddShopping)
			r.Post("/shopping/recipe", s.handleShopRecipe)
			r.Post("/shopping/clear", s.handleClearShopping)
			r.Put("/shopping/{id}", s.handleUpdateShopping)
			r.Delete("/shopping/{id}", s.handleDeleteShopping)
			r.Get("/plan", s.handlePlan)
			r.Get("/tonight", s.handleTonight)
			r.Post("/plan/shopping", s.handlePlanShopping)
			r.Get("/collections", s.handleCollections)
			r.Get("/collections/{id}", s.handleCollection)
			r.Get("/seasons/{key}", s.handleSeason)
			r.Post("/search", s.handleSearch)
			r.Get("/push", s.handlePushStatus)
			r.Post("/push/subscribe", s.handlePushSubscribe)
			r.Post("/push/unsubscribe", s.handlePushUnsubscribe)
			r.Post("/push/test", s.handlePushTest)
			r.Post("/push/timer", s.handleTimerStart)
			r.Delete("/push/timer/{id}", s.handleTimerStop)

			// Parents: add and edit recipes and people.
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireManager)
				r.Post("/import/url", s.handleImportURL)
				r.Post("/import/photo", s.handleImportPhoto)
				r.Post("/import/photo/again", s.handleReadAgain)
				r.Post("/import/dish", s.handleDish)
				r.Post("/import/dish/draft", s.handleDishDraft)
				r.Post("/import/text", s.handleImportText)
				r.Post("/import/app", s.handleImportApp)
				r.Post("/check", s.handleCheckDraft)
				r.Post("/recipes", s.handleSaveRecipe)
				r.Put("/recipes/{id}", s.handleSaveRecipe)
				r.Delete("/recipes/{id}", s.handleDeleteRecipe)
				r.Post("/recipes/{id}/version", s.handleVersion)
				r.Put("/recipes/{id}/label", s.handleLabelChecked)
				r.Post("/recipes/{id}/card-checked", s.handleCardChecked)
				r.Get("/recipes/{id}/share", s.handleGetShare)
				r.With(s.feature("share")).Post("/recipes/{id}/share", s.handleShareRecipe)
				r.Delete("/recipes/{id}/share", s.handleStopSharing)
				r.Post("/plan/suggest", s.handlePlanSuggest)
				r.Put("/recipes/{id}/book", s.handleRecipeBook)
				r.Get("/books/lookup", s.handleLookupBook)
				r.Post("/books", s.handleSaveBook)
				r.Put("/books/{id}", s.handleSaveBook)
				r.Delete("/books/{id}", s.handleDeleteBook)
				r.Post("/books/{id}/entries", s.handleAddBookEntries)
				r.Post("/books/index", s.handleBookIndex)
				r.Delete("/books/entries/{id}", s.handleDeleteBookEntry)
				r.Post("/pile", s.handleAddPile)
				r.Put("/pile/{id}", s.handleUpdatePile)
				r.Delete("/pile/{id}", s.handleDeletePile)
				r.Post("/events", s.handleSaveEvent)
				r.Put("/events/{id}", s.handleSaveEvent)
				r.Delete("/events/{id}", s.handleDeleteEvent)
				r.Post("/events/{id}/dishes", s.handleAddDish)
				r.Put("/events/{id}/dishes/{dish}", s.handleDishBrings)
				r.Delete("/events/{id}/dishes/{dish}", s.handleDeleteDish)
				r.Delete("/cooks/{id}", s.handleDeleteCook)
				r.Post("/photos", s.handleUploadPhoto)
				r.Post("/people", s.handleSavePerson)
				r.Put("/people/{id}", s.handleSavePerson)
				r.Delete("/people/{id}", s.handleDeletePerson)
				r.Put("/household", s.handleHousehold)
				r.Get("/stock/lookup", s.handleLookupBarcode)
				r.Post("/stock/receipt", s.handleReceipt)
				r.Post("/stock/receipt/apply", s.handleReceiptApply)
				r.Post("/stock", s.handleSaveStock)
				r.Put("/stock/{id}", s.handleSaveStock)
				r.Delete("/stock/{id}", s.handleDeleteStock)
				r.Post("/plan", s.handleSavePlan)
				r.Put("/plan/day/{date}", s.handlePlanDay)
				r.Put("/plan/{id}", s.handleSavePlan)
				r.Delete("/plan/{id}", s.handleDeletePlan)
				r.Post("/collections", s.handleSaveCollection)
				r.Put("/collections/{id}", s.handleSaveCollection)
				r.Delete("/collections/{id}", s.handleDeleteCollection)
				r.Post("/collections/{id}/recipes", s.handleCollectionAdd)
				r.Delete("/collections/{id}/recipes/{rid}", s.handleCollectionRemove)
				r.Post("/collections/{id}/suggest", s.handleCollectionSuggest)
			})

			// Admins: the AI and accounts.
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireAdmin)
				r.Get("/admin/settings", s.handleGetSettings)
				r.Get("/admin/guide", s.handleGuide)
				r.Get("/admin/backup", s.handleBackup)
				r.Post("/admin/backup", s.handleRestore)
				r.Put("/admin/guide", s.handleGuideUpdate)
				r.Put("/admin/settings", s.handlePutSettings)
				r.Post("/admin/ai/test", s.handleTestAI)
				r.Post("/admin/email/test", s.handleEmailTest)
				r.Get("/admin/hints", s.handleListHints)
				r.Post("/admin/hints", s.handleAddHint)
				r.Delete("/admin/hints/{id}", s.handleDeleteHint)
				r.Get("/admin/users", s.handleListUsers)
				r.Post("/admin/users", s.handleCreateUser)
				r.Put("/admin/users/{id}", s.handleUpdateUser)
				r.Put("/admin/users/{id}/password", s.handleResetPassword)
				r.Delete("/admin/users/{id}", s.handleDeleteUser)
				r.Get("/admin/aitools", s.handleAIToolsStatus)
				r.Get("/admin/ai/health", s.handleAIHealth)
				r.Post("/admin/aitools/check-updates", s.handleCheckModelUpdates)
				r.Post("/admin/aitools/updated", s.handleModelUpdated)
				r.Post("/admin/aitools/bench", s.handleStartBench)
				r.Get("/admin/ollama/find", s.handleOllamaFind)
				r.Post("/admin/ollama/pull", s.handleOllamaPull)
				r.Get("/admin/ollama/pull", s.handleOllamaPullStatus)
				r.Post("/admin/ollama/use", s.handleOllamaUse)
				r.Get("/admin/ollama/gpu", s.handleOllamaGPU)
				r.Get("/admin/ollama/models", s.handleOllamaModels)
				r.Post("/admin/ollama/delete", s.handleOllamaDelete)
				r.Get("/admin/tunnel", s.handleTunnelStatus)
				r.Put("/admin/tunnel", s.handleTunnelSave)
			})
		})
	})

	r.NotFound(s.serveStatic)
	return r
}

// productBases are the product databases for the pantry and the closet.
type productBases struct{ FoodBases, HomeBases []string }
