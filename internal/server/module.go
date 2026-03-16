package server

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/omaklabs/base/internal/db"
	"github.com/omaklabs/base/internal/jobs"
)

// Module describes a domain package's contributions to the app.
// Each domain exports a Module var. The boilerplate iterates these
// for route mounting, job registration, seed execution, etc.
type Module struct {
	// Name is a human-readable identifier (e.g., "posts", "billing").
	Name string

	// Path is the URL prefix (e.g., "/posts", "/billing").
	Path string

	// Mount registers the domain's routes on the given router.
	// The domain applies its own middleware (e.g., RequireAuth) inside.
	Mount func(r chi.Router, deps *Deps)

	// Jobs lists background job handlers this domain provides.
	Jobs []Job

	// Schedules lists recurring tasks this domain needs.
	Schedules []jobs.Schedule

	// Seeds lists seed functions for development data.
	Seeds []Seed
}

// Job pairs a job type name with its handler function.
type Job struct {
	Type    string
	Handler jobs.JobHandler
}

// Seed pairs a seed name with its function.
type Seed struct {
	Name string
	Fn   func(ctx context.Context, q *db.Queries) error
}
