package main

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"

	"github.com/omakase-dev/go-boilerplate/assets"
	"github.com/omakase-dev/go-boilerplate/internal/api"
	"github.com/omakase-dev/go-boilerplate/internal/config"
	"github.com/omakase-dev/go-boilerplate/internal/db"
	"github.com/omakase-dev/go-boilerplate/internal/email"
	"github.com/omakase-dev/go-boilerplate/internal/handlers"
	"github.com/omakase-dev/go-boilerplate/internal/jobs"
	"github.com/omakase-dev/go-boilerplate/internal/logger"
	"github.com/omakase-dev/go-boilerplate/internal/middleware"
)

func buildRouter(cfg config.Config, dbConn *sql.DB, dbPath string, queries *db.Queries, queue *jobs.Queue, emailStore *email.Store, appLogger *logger.Logger, reloadFn ...func() error) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RequestLogger(appLogger))
	router.Use(middleware.Recovery(nil))
	router.Use(middleware.BodyLimit(10 << 20)) // 10 MB
	router.Use(csrf.Protect(
		[]byte(cfg.CSRFKey),
		csrf.Secure(!cfg.IsDev()),
		csrf.Path("/"),
	))
	router.Use(middleware.Session(dbConn))
	router.Use(middleware.CSRFContext)
	router.Use(middleware.FlashContext)

	// Health check
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Internal management API
	router.Route("/internal", func(r chi.Router) {
		r.Use(middleware.InternalAPIKey(cfg.InternalAPIKey))
		var rFn func() error
		if len(reloadFn) > 0 && reloadFn[0] != nil {
			rFn = reloadFn[0]
		} else {
			rFn = func() error { return nil }
		}
		api.Mount(r, dbConn, dbPath, queue, emailStore, appLogger, cfg, rFn)
	})

	// App handlers
	handlers.Mount(router, queries, queue)

	// Embedded assets
	router.Handle("/assets/*", http.StripPrefix("/assets/",
		http.FileServer(http.FS(assets.Files))))

	return router
}
