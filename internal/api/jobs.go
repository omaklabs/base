package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/omakase-dev/go-boilerplate/internal/db"
	"github.com/omakase-dev/go-boilerplate/internal/jobs"
)

func handleListJobs(queue *jobs.Queue) http.HandlerFunc {
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

		jobList, err := queue.List(r.Context(), status, limit, offset)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list jobs"})
			return
		}

		// Return empty array instead of null when no jobs found
		if jobList == nil {
			jobList = []db.Job{}
		}

		writeJSON(w, http.StatusOK, jobList)
	}
}

func handleGetJob(queue *jobs.Queue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid job id"})
			return
		}

		job, err := queue.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get job"})
			return
		}

		writeJSON(w, http.StatusOK, job)
	}
}

func handleRetryJob(queue *jobs.Queue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid job id"})
			return
		}

		// Verify the job exists before retrying
		_, err = queue.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get job"})
			return
		}

		if err := queue.Retry(r.Context(), id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retry job"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "retried"})
	}
}

func handleCancelJob(queue *jobs.Queue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid job id"})
			return
		}

		// Verify the job exists before cancelling
		_, err = queue.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get job"})
			return
		}

		if err := queue.Cancel(r.Context(), id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to cancel job"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
	}
}

func handleJobStats(queue *jobs.Queue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := queue.Stats(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get job stats"})
			return
		}

		writeJSON(w, http.StatusOK, stats)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
