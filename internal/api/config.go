package api

import (
	"net/http"

	"github.com/omakase-dev/go-boilerplate/internal/config"
)

// handleConfigView returns the current config with sensitive values redacted.
func handleConfigView(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, cfg.Redacted())
	}
}

// handleConfigSchema returns the config schema with groups and fields.
func handleConfigSchema() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"groups": config.Schema(),
		})
	}
}

// handleReload triggers a config reload.
func handleReload(reloadFn func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := reloadFn(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
	}
}
