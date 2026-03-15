package handlers

import (
	"net/http"

	"github.com/omakase-dev/go-boilerplate/templates/pages"
)

func handleWelcome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pages.Welcome().Render(r.Context(), w)
	}
}
