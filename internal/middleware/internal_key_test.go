package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalAPIKeyValidKeyPasses(t *testing.T) {
	const secret = "test-secret-key"

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := InternalAPIKey(secret)(inner)

	req := httptest.NewRequest(http.MethodGet, "/internal/health", nil)
	req.Header.Set("X-Internal-Key", secret)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called with valid key")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestInternalAPIKeyMissingKeyReturns401(t *testing.T) {
	const secret = "test-secret-key"

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called without key")
	})

	handler := InternalAPIKey(secret)(inner)

	req := httptest.NewRequest(http.MethodGet, "/internal/health", nil)
	// No X-Internal-Key header
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["error"] != "unauthorized" {
		t.Errorf("error = %q, want %q", body["error"], "unauthorized")
	}
}

func TestInternalAPIKeyWrongKeyReturns401(t *testing.T) {
	const secret = "test-secret-key"

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called with wrong key")
	})

	handler := InternalAPIKey(secret)(inner)

	req := httptest.NewRequest(http.MethodGet, "/internal/health", nil)
	req.Header.Set("X-Internal-Key", "wrong-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}
}

func TestInternalAPIKeyEmptyKeyReturns401(t *testing.T) {
	const secret = "test-secret-key"

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called with empty key")
	})

	handler := InternalAPIKey(secret)(inner)

	req := httptest.NewRequest(http.MethodGet, "/internal/health", nil)
	req.Header.Set("X-Internal-Key", "")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
