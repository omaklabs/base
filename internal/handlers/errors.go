package handlers

import (
	"net/http"
	"runtime/debug"

	"github.com/omakase-dev/go-boilerplate/internal/middleware"
	"github.com/omakase-dev/go-boilerplate/templates/pages"
)

// RenderError renders a 500 error page. In dev mode, includes error details
// such as the error message, stack trace, and request information.
// In production mode, renders the generic 500 error page.
func RenderError(w http.ResponseWriter, r *http.Request, err error, isDev bool) {
	w.WriteHeader(http.StatusInternalServerError)

	if isDev {
		stack := string(debug.Stack())
		requestID := middleware.RequestIDFromContext(r.Context())
		pages.ErrorDev(r.Method, r.URL.Path, requestID, err.Error(), stack).Render(r.Context(), w)
		return
	}

	pages.Error500().Render(r.Context(), w)
}

// RenderNotFound renders a 404 page.
func RenderNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	pages.Error404().Render(r.Context(), w)
}
