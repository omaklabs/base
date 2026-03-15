package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omakase-dev/go-boilerplate/internal/db"
	"github.com/omakase-dev/go-boilerplate/internal/jobs"
	"github.com/omakase-dev/go-boilerplate/templates/pages"
)

func Mount(r chi.Router, queries *db.Queries, queue *jobs.Queue) {
	// Welcome page — replaced by the agent when generating the app
	r.Get("/", handleWelcome())

	// Error pages — used by renderError helpers
	r.NotFound(handleNotFound())
}

func handleNotFound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		pages.Error404().Render(r.Context(), w)
	}
}

// isHTMX checks if the request was made via HTMX.
func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
