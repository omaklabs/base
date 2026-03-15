package server

import (
	"net/http"
	"runtime/debug"

	"github.com/omakase-dev/go-boilerplate/internal/middleware"
)

// IsHTMX checks if the request was made via HTMX.
func IsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// RenderError renders a 500 error page. In dev mode, includes error details
// such as the error message, stack trace, and request information.
// In production mode, renders the generic 500 error page.
func RenderError(w http.ResponseWriter, r *http.Request, err error, isDev bool) {
	w.WriteHeader(http.StatusInternalServerError)

	if isDev {
		stack := string(debug.Stack())
		requestID := middleware.RequestIDFromContext(r.Context())
		ErrorDev(r.Method, r.URL.Path, requestID, err.Error(), stack).Render(r.Context(), w)
		return
	}

	Error500().Render(r.Context(), w)
}

// RenderNotFound renders a 404 page.
func RenderNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	Error404().Render(r.Context(), w)
}

// HandleNotFound returns a handler that renders the 404 page.
func HandleNotFound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		Error404().Render(r.Context(), w)
	}
}
