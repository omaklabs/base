package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDSetInResponseHeader(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("expected X-Request-ID header to be set, got empty string")
	}
}

func TestRequestIDAvailableInContext(t *testing.T) {
	var ctxID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if ctxID == "" {
		t.Fatal("expected request ID in context, got empty string")
	}

	headerID := rec.Header().Get("X-Request-ID")
	if ctxID != headerID {
		t.Errorf("context ID %q does not match header ID %q", ctxID, headerID)
	}
}

func TestRequestIDReusesIncomingHeader(t *testing.T) {
	incoming := "existing-request-id-12345"

	var ctxID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", incoming)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != incoming {
		t.Errorf("response header = %q, want %q", rec.Header().Get("X-Request-ID"), incoming)
	}
	if ctxID != incoming {
		t.Errorf("context ID = %q, want %q", ctxID, incoming)
	}
}

func TestRequestIDUniqueness(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		id := rec.Header().Get("X-Request-ID")
		if ids[id] {
			t.Fatalf("duplicate request ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestRequestIDFromContextReturnsEmptyWhenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	id := RequestIDFromContext(req.Context())
	if id != "" {
		t.Errorf("expected empty string from bare context, got %q", id)
	}
}
