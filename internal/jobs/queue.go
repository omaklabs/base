package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/omakase-dev/go-boilerplate/internal/db"
)

// JobHandler is the function signature that job processors must implement.
// The payload is the raw JSON bytes that were enqueued with the job.
type JobHandler func(ctx context.Context, payload []byte) error

// Queue is a SQLite-backed background job queue. Register handlers for
// specific job types, enqueue work, and call Process to poll for jobs.
type Queue struct {
	queries  *db.Queries
	handlers map[string]JobHandler
	mu       sync.RWMutex
}

// NewQueue creates a new Queue backed by the given SQLC queries instance.
func NewQueue(queries *db.Queries) *Queue {
	return &Queue{
		queries:  queries,
		handlers: make(map[string]JobHandler),
	}
}

// Register associates a handler function with a job type. When a job of
// the given type is claimed by Process, this handler will be invoked.
func (q *Queue) Register(jobType string, handler JobHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[jobType] = handler
}

// Enqueue creates a new pending job that is eligible to run immediately.
// The payload is JSON-marshalled before storage.
func (q *Queue) Enqueue(ctx context.Context, jobType string, payload any) error {
	return q.EnqueueAt(ctx, jobType, payload, time.Now().UTC())
}

// EnqueueAt creates a new pending job scheduled to run at the specified time.
// The payload is JSON-marshalled before storage.
func (q *Queue) EnqueueAt(ctx context.Context, jobType string, payload any, runAt time.Time) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling job payload: %w", err)
	}

	_, err = q.queries.CreateJob(ctx, db.CreateJobParams{
		Type:    jobType,
		Payload: string(data),
		RunAt:   runAt,
	})
	if err != nil {
		return fmt.Errorf("creating job: %w", err)
	}

	return nil
}

// Process is the main worker loop. It polls for pending jobs, dispatches
// them to registered handlers, and updates their status accordingly.
//
// The loop runs until ctx is cancelled. When no jobs are available it
// sleeps for 1 second before polling again.
func (q *Queue) Process(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		job, err := q.queries.ClaimNextJob(ctx)
		if err != nil {
			if err == sql.ErrNoRows {
				// No jobs available — back off and try again.
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(1 * time.Second):
					continue
				}
			}
			return fmt.Errorf("claiming next job: %w", err)
		}

		// Check if the job has exceeded its max attempts.
		if job.Attempts > job.MaxAttempts {
			failErr := q.queries.FailJob(ctx, db.FailJobParams{
				Error: "max attempts exceeded",
				ID:    job.ID,
			})
			if failErr != nil {
				log.Printf("jobs: failed to mark job %d as failed (max attempts): %v", job.ID, failErr)
			}
			continue
		}

		// Look up the handler for this job type.
		q.mu.RLock()
		handler, ok := q.handlers[job.Type]
		q.mu.RUnlock()

		if !ok {
			failErr := q.queries.FailJob(ctx, db.FailJobParams{
				Error: fmt.Sprintf("no handler registered for job type %q", job.Type),
				ID:    job.ID,
			})
			if failErr != nil {
				log.Printf("jobs: failed to mark job %d as failed (no handler): %v", job.ID, failErr)
			}
			continue
		}

		// Execute the handler.
		if handlerErr := handler(ctx, []byte(job.Payload)); handlerErr != nil {
			failErr := q.queries.FailJob(ctx, db.FailJobParams{
				Error: handlerErr.Error(),
				ID:    job.ID,
			})
			if failErr != nil {
				log.Printf("jobs: failed to mark job %d as failed: %v", job.ID, failErr)
			}
			continue
		}

		// Handler succeeded — mark as done.
		if completeErr := q.queries.CompleteJob(ctx, job.ID); completeErr != nil {
			log.Printf("jobs: failed to mark job %d as complete: %v", job.ID, completeErr)
		}
	}
}

// List returns jobs, optionally filtered by status. Pass an empty string
// for status to return all jobs.
func (q *Queue) List(ctx context.Context, status string, limit, offset int) ([]db.Job, error) {
	return q.queries.ListJobs(ctx, db.ListJobsParams{
		FilterStatus: status,
		QueryLimit:   int64(limit),
		QueryOffset:  int64(offset),
	})
}

// Get returns a single job by its ID.
func (q *Queue) Get(ctx context.Context, id int64) (db.Job, error) {
	return q.queries.GetJob(ctx, id)
}

// Retry resets a failed job back to pending status so it can be retried.
func (q *Queue) Retry(ctx context.Context, id int64) error {
	return q.queries.RetryJob(ctx, id)
}

// Cancel marks a pending job as cancelled, preventing it from running.
func (q *Queue) Cancel(ctx context.Context, id int64) error {
	return q.queries.CancelJob(ctx, id)
}

// Stats returns a map of job status to count.
func (q *Queue) Stats(ctx context.Context) (map[string]int64, error) {
	rows, err := q.queries.CountJobsByStatus(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[row.Status] = row.Count
	}
	return result, nil
}
