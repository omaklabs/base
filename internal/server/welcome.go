package server

import (
	"net/http"
)

// HandleWelcome returns a handler that renders the welcome page.
func HandleWelcome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Welcome().Render(r.Context(), w)
	}
}
