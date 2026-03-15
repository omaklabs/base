package middleware

import (
	"net/http"

	"github.com/omakase-dev/go-boilerplate/internal/flash"
	"github.com/omakase-dev/go-boilerplate/internal/view"
)

// FlashContext is middleware that reads a flash cookie from the request (if
// present), clears it immediately so it is only shown once, and stores the
// flash message in the request context via the view package. Templ components
// can then retrieve the flash with view.GetFlash(ctx).
func FlashContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f := flash.GetAndClear(w, r); f != nil {
			ctx := view.WithFlash(r.Context(), &view.Flash{
				Message: f.Message,
				Variant: f.Variant,
			})
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
