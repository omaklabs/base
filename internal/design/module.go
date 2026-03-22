package design

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omaklabs/base/internal/server"
)

// Module describes the design system showcase.
// Only mounted in development mode.
var Module = server.Module{
	Name:  "design",
	Path:  "/design",
	Mount: Mount,
}

// Mount registers the design system routes.
func Mount(r chi.Router, deps *server.Deps) {
	r.Get("/", handleDesign())
}

func handleDesign() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		DesignPage().Render(r.Context(), w)
	}
}
