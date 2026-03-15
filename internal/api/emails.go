package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/omaklabs/base/internal/db"
	"github.com/omaklabs/base/internal/email"
)

func handleListEmails(emailStore *email.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")

		limit := 20
		if v := r.URL.Query().Get("limit"); v != "" {
			parsed, err := strconv.Atoi(v)
			if err != nil || parsed < 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit parameter"})
				return
			}
			limit = parsed
		}

		offset := 0
		if v := r.URL.Query().Get("offset"); v != "" {
			parsed, err := strconv.Atoi(v)
			if err != nil || parsed < 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid offset parameter"})
				return
			}
			offset = parsed
		}

		emails, err := emailStore.List(r.Context(), status, limit, offset)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list emails"})
			return
		}

		// Return empty array instead of null when no emails found
		if emails == nil {
			emails = []db.Email{}
		}

		writeJSON(w, http.StatusOK, emails)
	}
}

func handleGetEmail(emailStore *email.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email id"})
			return
		}

		em, err := emailStore.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "email not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get email"})
			return
		}

		writeJSON(w, http.StatusOK, em)
	}
}

func handleEmailStats(emailStore *email.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := emailStore.Stats(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get email stats"})
			return
		}

		writeJSON(w, http.StatusOK, stats)
	}
}
