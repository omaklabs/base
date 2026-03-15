package api

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/omaklabs/base/internal/config"
	"github.com/omaklabs/base/internal/db"
	"github.com/omaklabs/base/internal/email"
	"github.com/omaklabs/base/internal/jobs"
	"github.com/omaklabs/base/internal/logger"
	"github.com/omaklabs/base/migrations"
	"github.com/pressly/goose/v3"
)

// testEnv holds everything needed to run API tests.
type testEnv struct {
	DB         *sql.DB
	Queue      *jobs.Queue
	EmailStore *email.Store
	Logger     *logger.Logger
	Router     chi.Router
}

// setupTestEnv creates a fresh SQLite database, runs migrations, and mounts
// the API routes on a new chi router.
func setupTestEnv(t *testing.T) testEnv {
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

	queries := db.New(conn)
	queue := jobs.NewQueue(queries)
	emailStore := email.NewStore(queries)
	log := logger.New(nil)

	cfg := config.Config{
		Addr:           ":8080",
		Env:            "development",
		DatabasePath:   dbPath,
		CSRFKey:        "test-csrf-key",
		InternalAPIKey: "test-api-key",
		Mail: config.MailConfig{
			From:       "noreply@localhost",
			Host:       "localhost",
			Port:       587,
			Username:   "",
			Password:   "smtp-secret",
			Encryption: "tls",
		},
	}
	reloadFn := func() error { return nil }

	r := chi.NewRouter()
	Mount(r, conn, dbPath, queue, emailStore, log, cfg, reloadFn)

	return testEnv{
		DB:         conn,
		Queue:      queue,
		EmailStore: emailStore,
		Logger:     log,
		Router:     r,
	}
}

func TestHealthEndpoint(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify expected fields are present (includes new SQLite stats)
	for _, field := range []string{"status", "uptime", "go_version", "goroutines", "memory_mb", "db_ok", "db_size_mb", "db_tables"} {
		if _, ok := body[field]; !ok {
			t.Errorf("missing field %q in health response", field)
		}
	}

	if body["status"] != "ok" {
		t.Errorf("status = %v, want %q", body["status"], "ok")
	}
	if body["db_ok"] != true {
		t.Errorf("db_ok = %v, want true", body["db_ok"])
	}
}

func TestJobsListEndpoint(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body []any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(body) != 0 {
		t.Errorf("expected empty array, got %d items", len(body))
	}
}

func TestJobsStatsEndpoint(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Enqueue some jobs to have stats
	_ = env.Queue.Enqueue(ctx, "test_type", map[string]string{"a": "1"})
	_ = env.Queue.Enqueue(ctx, "test_type", map[string]string{"b": "2"})

	req := httptest.NewRequest(http.MethodGet, "/jobs/stats", nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	pending, ok := body["pending"]
	if !ok {
		t.Fatal("missing 'pending' key in stats response")
	}
	if pending.(float64) != 2 {
		t.Errorf("pending = %v, want 2", pending)
	}
}

func TestLogStreamEndpoint(t *testing.T) {
	env := setupTestEnv(t)

	// Use a context with cancel so we can stop the SSE stream
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/logs", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	// Run the handler in a goroutine since SSE blocks
	done := make(chan struct{})
	go func() {
		defer close(done)
		env.Router.ServeHTTP(rec, req)
	}()

	// Give the handler time to set up the subscriber
	time.Sleep(50 * time.Millisecond)

	// Log a message so the subscriber receives it
	env.Logger.Info("test log message", "key", "value")

	// Give it time to write
	time.Sleep(50 * time.Millisecond)

	// Cancel the context to stop the SSE stream
	cancel()

	// Wait for the handler to finish
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE handler to finish")
	}

	// Verify SSE headers
	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/event-stream")
	}
	cacheControl := rec.Header().Get("Cache-Control")
	if cacheControl != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", cacheControl, "no-cache")
	}
	connection := rec.Header().Get("Connection")
	if connection != "keep-alive" {
		t.Errorf("Connection = %q, want %q", connection, "keep-alive")
	}

	// Verify we received at least one SSE data line
	body := rec.Body.String()
	scanner := bufio.NewScanner(strings.NewReader(body))
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			found = true
			jsonData := strings.TrimPrefix(line, "data: ")
			var entry logger.Entry
			if err := json.Unmarshal([]byte(jsonData), &entry); err != nil {
				t.Fatalf("failed to unmarshal SSE data: %v", err)
			}
			if entry.Msg != "test log message" {
				t.Errorf("entry.Msg = %q, want %q", entry.Msg, "test log message")
			}
			break
		}
	}
	if !found {
		t.Error("expected at least one SSE data line in response")
	}
}

func TestGetJobNotFound(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/jobs/99999", nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["error"] != "job not found" {
		t.Errorf("error = %q, want %q", body["error"], "job not found")
	}
}

func TestRetryJob(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Enqueue a job and mark it as failed
	if err := env.Queue.Enqueue(ctx, "retry_test", "data"); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	jobList, err := env.Queue.List(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobList) == 0 {
		t.Fatal("expected at least one job")
	}
	jobID := jobList[0].ID

	// Fail the job first (Retry only works on failed jobs)
	queries := db.New(env.DB)
	if err := queries.FailJob(ctx, db.FailJobParams{Error: "test error", ID: jobID}); err != nil {
		t.Fatalf("FailJob: %v", err)
	}

	// Retry via the API
	req := httptest.NewRequest(http.MethodPost, "/jobs/"+itoa(jobID)+"/retry", nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "retried" {
		t.Errorf("status = %q, want %q", body["status"], "retried")
	}

	// Verify the job is back to pending
	job, err := env.Queue.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if job.Status != "pending" {
		t.Errorf("job.Status = %q, want %q", job.Status, "pending")
	}
}

func TestCancelJob(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Enqueue a pending job
	if err := env.Queue.Enqueue(ctx, "cancel_test", "data"); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	jobList, err := env.Queue.List(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(jobList) == 0 {
		t.Fatal("expected at least one job")
	}
	jobID := jobList[0].ID

	// Cancel via the API
	req := httptest.NewRequest(http.MethodDelete, "/jobs/"+itoa(jobID), nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "cancelled" {
		t.Errorf("status = %q, want %q", body["status"], "cancelled")
	}

	// Verify the job is cancelled
	job, err := env.Queue.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if job.Status != "cancelled" {
		t.Errorf("job.Status = %q, want %q", job.Status, "cancelled")
	}
}

func TestEmailsListEndpoint(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create some emails via the store
	_, _ = env.EmailStore.Create(ctx, email.Message{To: []string{"a@example.com"}, Subject: "A"})
	_, _ = env.EmailStore.Create(ctx, email.Message{To: []string{"b@example.com"}, Subject: "B"})

	req := httptest.NewRequest(http.MethodGet, "/emails", nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body []any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(body) != 2 {
		t.Errorf("expected 2 emails, got %d", len(body))
	}
}

func TestEmailsStatsEndpoint(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	id1, _ := env.EmailStore.Create(ctx, email.Message{To: []string{"a@example.com"}, Subject: "A"})
	_, _ = env.EmailStore.Create(ctx, email.Message{To: []string{"b@example.com"}, Subject: "B"})
	_ = env.EmailStore.MarkSent(ctx, id1)

	req := httptest.NewRequest(http.MethodGet, "/emails/stats", nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	sent, ok := body["sent"]
	if !ok {
		t.Fatal("missing 'sent' key in stats response")
	}
	if sent.(float64) != 1 {
		t.Errorf("sent = %v, want 1", sent)
	}

	pending, ok := body["pending"]
	if !ok {
		t.Fatal("missing 'pending' key in stats response")
	}
	if pending.(float64) != 1 {
		t.Errorf("pending = %v, want 1", pending)
	}
}

func TestGetEmailNotFound(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/emails/99999", nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["error"] != "email not found" {
		t.Errorf("error = %q, want %q", body["error"], "email not found")
	}
}

func TestConfigViewEndpoint(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Sensitive fields should be masked
	for _, key := range []string{"CSRF_KEY", "INTERNAL_API_KEY"} {
		val, ok := body[key]
		if !ok {
			t.Errorf("missing key %q in config view", key)
			continue
		}
		if val != "****" {
			t.Errorf("%s = %q, want %q", key, val, "****")
		}
	}

	// MAIL_PASSWORD has a value set, so it should be masked
	if body["MAIL_PASSWORD"] != "****" {
		t.Errorf("MAIL_PASSWORD = %q, want %q", body["MAIL_PASSWORD"], "****")
	}

	// Non-sensitive fields should be visible
	if body["ADDR"] != ":8080" {
		t.Errorf("ADDR = %q, want %q", body["ADDR"], ":8080")
	}
	if body["ENV"] != "development" {
		t.Errorf("ENV = %q, want %q", body["ENV"], "development")
	}
}

func TestConfigSchemaEndpoint(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/config/schema", nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	groups, ok := body["groups"]
	if !ok {
		t.Fatal("missing 'groups' key in schema response")
	}

	groupSlice, ok := groups.([]any)
	if !ok {
		t.Fatal("'groups' is not an array")
	}

	if len(groupSlice) < 4 {
		t.Errorf("expected at least 4 groups, got %d", len(groupSlice))
	}
}

func TestReloadEndpoint(t *testing.T) {
	env := setupTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/reload", nil)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "reloaded" {
		t.Errorf("status = %q, want %q", body["status"], "reloaded")
	}
}

// itoa converts int64 to string for URL building.
func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
