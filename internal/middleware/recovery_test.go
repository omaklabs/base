package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoveryReturns500OnPanic(t *testing.T) {
	mw := Recovery(nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Internal Server Error") {
		t.Errorf("body = %q, want it to contain 'Internal Server Error'", body)
	}
}

func TestRecoveryLogsStackTrace(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	mw := Recovery(logger)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic message")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	logged := buf.String()
	if !strings.Contains(logged, "test panic message") {
		t.Errorf("log output should contain panic value, got: %s", logged)
	}
	if !strings.Contains(logged, "goroutine") {
		t.Errorf("log output should contain stack trace, got: %s", logged)
	}
}

func TestRecoveryPassesThroughNormally(t *testing.T) {
	mw := Recovery(nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "ok")
	}
}

func TestRecoveryWithCustomLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "CUSTOM: ", 0)

	mw := Recovery(logger)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("custom logger panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	logged := buf.String()
	if !strings.Contains(logged, "CUSTOM: ") {
		t.Errorf("log output should use custom logger prefix, got: %s", logged)
	}
}
