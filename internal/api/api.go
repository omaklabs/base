// Package api provides the internal management API mounted at /internal/*.
// Protected by middleware.InternalAPIKey. Endpoints for health, config,
// jobs, emails, and log streaming. Never expose to end users.
package api

import (
	"database/sql"

	"github.com/go-chi/chi/v5"
	"github.com/omaklabs/base/internal/config"
	"github.com/omaklabs/base/internal/email"
	"github.com/omaklabs/base/internal/jobs"
	"github.com/omaklabs/base/internal/logger"
)

// Mount registers all internal management API routes on the given router.
// Route order matters: /jobs/stats MUST come before /jobs/{id} so Chi
// does not match "stats" as an {id} parameter.
// dbPath is the filesystem path to the SQLite database file, used for
// reporting file-size stats in the health endpoint.
func Mount(r chi.Router, dbConn *sql.DB, dbPath string, queue *jobs.Queue, emailStore *email.Store, log *logger.Logger, cfg config.Config, reloadFn func() error) {
	r.Get("/health", handleHealth(dbConn, dbPath))
	r.Get("/config", handleConfigView(cfg))
	r.Get("/config/schema", handleConfigSchema())
	r.Post("/reload", handleReload(reloadFn))
	r.Get("/jobs", handleListJobs(queue))
	r.Get("/jobs/stats", handleJobStats(queue))
	r.Get("/jobs/{id}", handleGetJob(queue))
	r.Post("/jobs/{id}/retry", handleRetryJob(queue))
	r.Delete("/jobs/{id}", handleCancelJob(queue))
	r.Get("/emails", handleListEmails(emailStore))
	r.Get("/emails/stats", handleEmailStats(emailStore))
	r.Get("/emails/{id}", handleGetEmail(emailStore))
	r.Get("/logs", handleLogStream(log))
}
