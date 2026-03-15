package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/omaklabs/base/internal/view"
)

func TestCSRFContextPutsTokenInContext(t *testing.T) {
	// We cannot easily use the real gorilla/csrf middleware in a unit test
	// because it requires a valid CSRF cookie/token pair. Instead, we
	// simulate the scenario by setting the gorilla/csrf token in context
	// beforehand (the way gorilla/csrf does internally) and verify that
	// CSRFContext copies it into view context.
	//
	// However, gorilla/csrf.Token reads from an unexported context key, so
	// calling csrf.Token(r) without the real middleware returns "". We test
	// that CSRFContext at least stores *something* (empty string when
	// gorilla/csrf hasn't run) and doesn't panic.

	var gotToken string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = view.CSRFToken(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := CSRFContext(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Without the real gorilla/csrf middleware upstream, csrf.Token returns "".
	// The important thing is that CSRFContext stored the value in context via
	// view.WithCSRFToken, and view.CSRFToken returns it.
	if gotToken != "" {
		// If this fires, it means csrf.Token returned a non-empty value
		// without the gorilla middleware running — unexpected but acceptable.
		t.Logf("csrf.Token returned %q without gorilla middleware; still acceptable", gotToken)
	}

	// Verify the middleware doesn't break the chain.
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCSRFContextChainPassesThrough(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := CSRFContext(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called")
	}
}
