package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/omaklabs/base/internal/flash"
	"github.com/omaklabs/base/internal/view"
)

func TestFlashContextMiddleware(t *testing.T) {
	// Set a flash cookie.
	setRec := httptest.NewRecorder()
	flash.Set(setRec, "Welcome back!", "success")

	// Build a request carrying the flash cookie.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range setRec.Result().Cookies() {
		req.AddCookie(c)
	}

	var gotFlash *view.Flash
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFlash = view.GetFlash(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := FlashContext(inner)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotFlash == nil {
		t.Fatal("expected flash in context, got nil")
	}
	if gotFlash.Message != "Welcome back!" {
		t.Errorf("Message = %q, want %q", gotFlash.Message, "Welcome back!")
	}
	if gotFlash.Variant != "success" {
		t.Errorf("Variant = %q, want %q", gotFlash.Variant, "success")
	}

	// Verify the cookie was cleared.
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "flash" {
			found = true
			if c.MaxAge != -1 {
				t.Errorf("flash cookie MaxAge = %d, want -1", c.MaxAge)
			}
		}
	}
	if !found {
		t.Error("expected flash cookie to be cleared")
	}
}

func TestFlashContextMiddlewareWithoutCookie(t *testing.T) {
	var gotFlash *view.Flash
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFlash = view.GetFlash(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := FlashContext(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotFlash != nil {
		t.Errorf("expected nil flash without cookie, got %+v", gotFlash)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
