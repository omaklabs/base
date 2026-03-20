package main

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"

	"github.com/omaklabs/base/assets"
	"github.com/omaklabs/base/internal/api"
	"github.com/omaklabs/base/internal/auth"
	"github.com/omaklabs/base/internal/config"
	"github.com/omaklabs/base/internal/db"
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

	// In development, mark requests as plaintext HTTP so gorilla/csrf
	// skips HTTPS-only origin/referer checks on localhost.
	if cfg.IsDev() {
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), csrf.PlaintextHTTPContextKey, true)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
	}

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

	// Dev-only auto-login route for testing
	if cfg.IsDev() {
		router.Get("/dev/login", func(w http.ResponseWriter, r *http.Request) {
			q := db.New(dbConn)
			user, err := q.GetUserByEmail(r.Context(), "test@example.com")
			if err != nil {
				user, err = q.CreateUser(r.Context(), db.CreateUserParams{
					Email: "test@example.com",
					Name:  "Test User",
				})
				if err != nil {
					http.Error(w, "failed to create test user", http.StatusInternalServerError)
					return
				}
			}
			token, err := auth.CreateSession(r.Context(), q, user.ID)
			if err != nil {
				http.Error(w, "failed to create session", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "session_token",
				Value:    token,
				Path:     "/",
				MaxAge:   86400 * 30,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			http.Redirect(w, r, "/", http.StatusSeeOther)
		})
	}

	// App routes — mount all domain modules from app.go
	router.Get("/", server.HandleWelcome())
	for _, m := range modules {
		if m.Mount != nil {
			m := m // capture loop variable
			router.Route(m.Path, func(r chi.Router) { m.Mount(r, deps) })
		}
	}
	router.NotFound(server.HandleNotFound())

	// Assets: serve from filesystem in dev (instant CSS/JS changes without rebuild),
	// embedded FS in production (single binary, no external files).
	if cfg.IsDev() {
		router.Handle("/assets/*", http.StripPrefix("/assets/",
			http.FileServer(http.Dir("assets"))))
	} else {
		router.Handle("/assets/*", http.StripPrefix("/assets/",
			http.FileServer(http.FS(assets.Files))))
	}

	return router
}
