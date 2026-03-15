package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/omaklabs/base/internal/logger"
)

// logEntry mirrors logger.Entry for test assertions.
type logEntry struct {
	Level  string         `json:"level"`
	Msg    string         `json:"msg"`
	Fields map[string]any `json:"fields,omitempty"`
}

func parseLogEntry(t *testing.T, buf *bytes.Buffer) logEntry {
	t.Helper()
	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v\nraw: %s", err, buf.String())
	}
	return entry
}

func TestRequestLoggerLogs200(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf)

	handler := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	entry := parseLogEntry(t, &buf)

	if entry.Level != "info" {
		t.Errorf("level = %q, want %q", entry.Level, "info")
	}
	if entry.Fields["method"] != "GET" {
		t.Errorf("fields[method] = %v, want %q", entry.Fields["method"], "GET")
	}
	if entry.Fields["path"] != "/posts" {
		t.Errorf("fields[path] = %v, want %q", entry.Fields["path"], "/posts")
	}
	if entry.Fields["status"] != float64(200) {
		t.Errorf("fields[status] = %v, want %v", entry.Fields["status"], 200)
	}
	if entry.Fields["user_agent"] != "test-agent" {
		t.Errorf("fields[user_agent] = %v, want %q", entry.Fields["user_agent"], "test-agent")
	}
	if entry.Fields["bytes_written"] != float64(5) {
		t.Errorf("fields[bytes_written] = %v, want %v", entry.Fields["bytes_written"], 5)
	}
}

func TestRequestLoggerLogs404(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf)

	handler := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	entry := parseLogEntry(t, &buf)

	if entry.Level != "warn" {
		t.Errorf("level = %q, want %q", entry.Level, "warn")
	}
	if entry.Fields["status"] != float64(404) {
		t.Errorf("fields[status] = %v, want %v", entry.Fields["status"], 404)
	}
}

func TestRequestLoggerLogs500(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf)

	handler := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodPost, "/error", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	entry := parseLogEntry(t, &buf)

	if entry.Level != "error" {
		t.Errorf("level = %q, want %q", entry.Level, "error")
	}
	if entry.Fields["status"] != float64(500) {
		t.Errorf("fields[status] = %v, want %v", entry.Fields["status"], 500)
	}
}

func TestRequestLoggerDuration(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf)

	handler := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	entry := parseLogEntry(t, &buf)

	dur, ok := entry.Fields["duration_ms"].(float64)
	if !ok {
		t.Fatal("duration_ms field is missing or not a number")
	}
	if dur < 0 {
		t.Errorf("duration_ms = %v, want >= 0", dur)
	}
}

func TestResponseWriterCapturesStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	rw.WriteHeader(http.StatusCreated)

	if rw.status != http.StatusCreated {
		t.Errorf("status = %d, want %d", rw.status, http.StatusCreated)
	}
}

func TestResponseWriterDefaultsTo200(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	// Write without calling WriteHeader — status should default to 200.
	rw.Write([]byte("data"))

	if rw.status != http.StatusOK {
		t.Errorf("status = %d, want %d", rw.status, http.StatusOK)
	}
}
