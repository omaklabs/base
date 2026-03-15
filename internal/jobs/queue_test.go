package jobs

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/omakase-dev/go-boilerplate/internal/db"
	"github.com/omakase-dev/go-boilerplate/migrations"
	"github.com/pressly/goose/v3"
)

// testEnv holds the database objects needed by each test.
type testEnv struct {
	DB      *sql.DB
	Queries *db.Queries
}

// setupTestDB creates a fresh SQLite database with migrations applied.
// The database lives in t.TempDir() so it is automatically cleaned up.
func setupTestDB(t *testing.T) testEnv {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	conn, err := db.Connect(dbPath)
	if err != nil {
		t.Fatalf("db.Connect: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("goose.SetDialect: %v", err)
	}
	if err := goose.Up(conn, "."); err != nil {
		t.Fatalf("goose.Up: %v", err)
	}

	q := db.New(conn)

	return testEnv{
		DB:      conn,
		Queries: q,
	}
}

func TestEnqueue(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()
	q := NewQueue(env.Queries)

	payload := map[string]string{"email": "user@example.com"}
	err := q.Enqueue(ctx, "send_email", payload)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Verify the job is in the database with status=pending.
	jobs, err := q.List(ctx, "pending", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	job := jobs[0]
	if job.Type != "send_email" {
		t.Errorf("job.Type = %q, want %q", job.Type, "send_email")
	}
	if job.Status != "pending" {
		t.Errorf("job.Status = %q, want %q", job.Status, "pending")
	}
	if job.Payload != `{"email":"user@example.com"}` {
		t.Errorf("job.Payload = %q, want JSON object", job.Payload)
	}
}

func TestEnqueueAt(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()
	q := NewQueue(env.Queries)

	futureTime := time.Now().UTC().Add(24 * time.Hour)
	payload := map[string]string{"task": "future_task"}
	err := q.EnqueueAt(ctx, "scheduled_job", payload, futureTime)
	if err != nil {
		t.Fatalf("EnqueueAt: %v", err)
	}

	// The job should exist but not be claimable (run_at is in the future).
	jobs, err := q.List(ctx, "pending", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	job := jobs[0]
	if job.Status != "pending" {
		t.Errorf("job.Status = %q, want %q", job.Status, "pending")
	}

	// ClaimNextJob should return sql.ErrNoRows since run_at is in the future.
	_, err = env.Queries.ClaimNextJob(ctx)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows for future job, got: %v", err)
	}
}

func TestRegisterAndProcess(t *testing.T) {
	env := setupTestDB(t)
	q := NewQueue(env.Queries)

	var (
		mu             sync.Mutex
		receivedPayload []byte
	)
	done := make(chan struct{})

	q.Register("test_job", func(ctx context.Context, payload []byte) error {
		mu.Lock()
		receivedPayload = payload
		mu.Unlock()
		close(done)
		return nil
	})

	ctx := context.Background()
	err := q.Enqueue(ctx, "test_job", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Run Process in a goroutine with a timeout context.
	processCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	go func() {
		_ = q.Process(processCtx)
	}()

	// Wait for the handler to be called or timeout.
	select {
	case <-done:
		// Handler was invoked.
	case <-processCtx.Done():
		t.Fatal("timed out waiting for handler to be called")
	}

	mu.Lock()
	defer mu.Unlock()
	if string(receivedPayload) != `{"key":"value"}` {
		t.Errorf("payload = %q, want %q", string(receivedPayload), `{"key":"value"}`)
	}

	// Give a moment for CompleteJob to run.
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Verify the job was marked as done.
	jobs, err := q.List(context.Background(), "done", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 done job, got %d", len(jobs))
	}
}

func TestProcessFailedJob(t *testing.T) {
	env := setupTestDB(t)
	q := NewQueue(env.Queries)

	done := make(chan struct{})

	q.Register("failing_job", func(ctx context.Context, payload []byte) error {
		defer close(done)
		return errors.New("something went wrong")
	})

	ctx := context.Background()
	err := q.Enqueue(ctx, "failing_job", nil)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	processCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	go func() {
		_ = q.Process(processCtx)
	}()

	select {
	case <-done:
	case <-processCtx.Done():
		t.Fatal("timed out waiting for handler to be called")
	}

	// Give a moment for FailJob to run.
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Verify the job was marked as failed.
	jobs, err := q.List(context.Background(), "failed", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 failed job, got %d", len(jobs))
	}
	if jobs[0].Error != "something went wrong" {
		t.Errorf("job.Error = %q, want %q", jobs[0].Error, "something went wrong")
	}
}

func TestProcessNoHandler(t *testing.T) {
	env := setupTestDB(t)
	q := NewQueue(env.Queries)

	// Do NOT register any handler for this type.
	ctx := context.Background()
	err := q.Enqueue(ctx, "unknown_job_type", map[string]string{"x": "1"})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	processCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	go func() {
		_ = q.Process(processCtx)
	}()

	// Poll until the job is marked as failed.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for job to be marked failed")
		default:
		}

		jobs, err := q.List(context.Background(), "failed", 10, 0)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(jobs) == 1 {
			if jobs[0].Error == "" {
				t.Error("expected non-empty error for unhandled job type")
			}
			cancel()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestList(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()
	q := NewQueue(env.Queries)

	// Enqueue several jobs.
	for i := 0; i < 5; i++ {
		if err := q.Enqueue(ctx, "list_test", i); err != nil {
			t.Fatalf("Enqueue %d: %v", i, err)
		}
	}

	// List all jobs (no status filter).
	all, err := q.List(ctx, "", 20, 0)
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("List all: got %d, want 5", len(all))
	}

	// List only pending jobs.
	pending, err := q.List(ctx, "pending", 20, 0)
	if err != nil {
		t.Fatalf("List pending: %v", err)
	}
	if len(pending) != 5 {
		t.Errorf("List pending: got %d, want 5", len(pending))
	}

	// List with a non-matching status.
	done, err := q.List(ctx, "done", 20, 0)
	if err != nil {
		t.Fatalf("List done: %v", err)
	}
	if len(done) != 0 {
		t.Errorf("List done: got %d, want 0", len(done))
	}

	// Test pagination with limit/offset.
	page, err := q.List(ctx, "", 2, 0)
	if err != nil {
		t.Fatalf("List page: %v", err)
	}
	if len(page) != 2 {
		t.Errorf("List page: got %d, want 2", len(page))
	}
}

func TestStats(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()
	q := NewQueue(env.Queries)

	// Enqueue 3 jobs.
	for i := 0; i < 3; i++ {
		if err := q.Enqueue(ctx, "stats_test", i); err != nil {
			t.Fatalf("Enqueue %d: %v", i, err)
		}
	}

	// Manually update some job statuses.
	jobs, err := q.List(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Mark one job as done and one as failed.
	if err := env.Queries.CompleteJob(ctx, jobs[0].ID); err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}
	if err := env.Queries.FailJob(ctx, db.FailJobParams{
		Error: "test error",
		ID:    jobs[1].ID,
	}); err != nil {
		t.Fatalf("FailJob: %v", err)
	}

	stats, err := q.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	if stats["done"] != 1 {
		t.Errorf("stats[done] = %d, want 1", stats["done"])
	}
	if stats["failed"] != 1 {
		t.Errorf("stats[failed] = %d, want 1", stats["failed"])
	}
	if stats["pending"] != 1 {
		t.Errorf("stats[pending] = %d, want 1", stats["pending"])
	}
}

func TestRetry(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()
	q := NewQueue(env.Queries)

	if err := q.Enqueue(ctx, "retry_test", "data"); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	jobs, err := q.List(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	jobID := jobs[0].ID

	// Mark the job as failed.
	if err := env.Queries.FailJob(ctx, db.FailJobParams{
		Error: "temporary error",
		ID:    jobID,
	}); err != nil {
		t.Fatalf("FailJob: %v", err)
	}

	// Verify it is failed.
	job, err := q.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if job.Status != "failed" {
		t.Fatalf("job.Status = %q, want %q", job.Status, "failed")
	}

	// Retry the job.
	if err := q.Retry(ctx, jobID); err != nil {
		t.Fatalf("Retry: %v", err)
	}

	// Verify it is back to pending with error cleared.
	job, err = q.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get after retry: %v", err)
	}
	if job.Status != "pending" {
		t.Errorf("job.Status = %q, want %q", job.Status, "pending")
	}
	if job.Error != "" {
		t.Errorf("job.Error = %q, want empty string", job.Error)
	}
}

func TestCancel(t *testing.T) {
	env := setupTestDB(t)
	ctx := context.Background()
	q := NewQueue(env.Queries)

	if err := q.Enqueue(ctx, "cancel_test", "data"); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	jobs, err := q.List(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	jobID := jobs[0].ID

	// Cancel the job.
	if err := q.Cancel(ctx, jobID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	// Verify it is cancelled.
	job, err := q.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if job.Status != "cancelled" {
		t.Errorf("job.Status = %q, want %q", job.Status, "cancelled")
	}
}
