package main

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"

	"github.com/omaklabs/base/assets"
	"github.com/omaklabs/base/internal/api"
	"github.com/omaklabs/base/internal/config"
	"github.com/omaklabs/base/internal/email"
	"github.com/omaklabs/base/internal/middleware"
	"github.com/omaklabs/base/internal/server"
)

func buildRouter(cfg config.Config, dbConn *sql.DB, dbPath string, deps *server.Deps, emailStore *email.Store, reloadFn ...func() error) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RequestLogger(deps.Logger))
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
		api.Mount(r, dbConn, dbPath, deps.Queue, emailStore, deps.Logger, cfg, rFn)
	})

	// App routes — mount all domain modules from app.go
	router.Get("/", server.HandleWelcome())
	for _, m := range modules {
		if m.Mount != nil {
			m := m // capture loop variable
			router.Route(m.Path, func(r chi.Router) { m.Mount(r, deps) })
		}
	}
	router.NotFound(server.HandleNotFound())

	// Embedded assets
	router.Handle("/assets/*", http.StripPrefix("/assets/",
		http.FileServer(http.FS(assets.Files))))

	return router
}
