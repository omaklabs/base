package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyLimitAllowsSmallBody(t *testing.T) {
	const limit = 1024 // 1 KB

	var bodyContent string
	handler := BodyLimit(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		bodyContent = string(b)
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.NewReader("small body")
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if bodyContent != "small body" {
		t.Errorf("body = %q, want %q", bodyContent, "small body")
	}
}

func TestBodyLimitRejectsLargeBody(t *testing.T) {
	const limit = 16 // 16 bytes

	var readErr error
	handler := BodyLimit(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Create a body that exceeds the limit.
	largeBody := strings.NewReader(strings.Repeat("x", 1024))
	req := httptest.NewRequest(http.MethodPost, "/upload", largeBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if readErr == nil {
		t.Fatal("expected read error due to body size limit, got nil")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}
