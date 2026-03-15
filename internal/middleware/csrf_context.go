package middleware

import (
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/omakase-dev/go-boilerplate/internal/view"
)

// CSRFContext is middleware that extracts the CSRF token provided by the
// gorilla/csrf package and stores it in the request context via the view
// package. This allows Templ components to retrieve the token with
// view.CSRFToken(ctx) without needing direct access to the *http.Request.
func CSRFContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := csrf.Token(r)
		ctx := view.WithCSRFToken(r.Context(), token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
