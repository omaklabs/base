// Package server provides shared HTTP infrastructure for all domain packages.
// Deps holds injected dependencies. Module describes a domain's contributions.
// IsHTMX(), RenderError(), RenderNotFound() are shared helpers.
// Domains import this package; this package never imports domains.
package server

import (
	"github.com/omaklabs/base/internal/db"
	"github.com/omaklabs/base/internal/email"
	"github.com/omaklabs/base/internal/jobs"
	"github.com/omaklabs/base/internal/logger"
	"github.com/omaklabs/base/internal/storage"
)

// Deps holds shared dependencies injected into every domain package.
// Adding a new dependency = one field here, one line in cmd/app/serve.go.
type Deps struct {
	Queries *db.Queries
	Queue   *jobs.Queue
	Mailer  email.Mailer
	Storage storage.Storage
	Logger  *logger.Logger
	IsDev   bool
}
