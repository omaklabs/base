package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/omakase-dev/go-boilerplate/internal/db"
)

func TestCurrentUserReturnsNilWhenNoUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	user := CurrentUser(req)
	if user != nil {
		t.Errorf("expected nil user from bare request, got %+v", user)
	}
}

func TestCurrentUserReturnsUserFromContext(t *testing.T) {
	want := &db.User{
		ID:        42,
		Email:     "alice@example.com",
		Name:      "Alice",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), userKey, want)
	req = req.WithContext(ctx)

	got := CurrentUser(req)
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != want.ID {
		t.Errorf("user ID = %d, want %d", got.ID, want.ID)
	}
	if got.Email != want.Email {
		t.Errorf("user Email = %q, want %q", got.Email, want.Email)
	}
	if got.Name != want.Name {
		t.Errorf("user Name = %q, want %q", got.Name, want.Name)
	}
}

func TestRequireAuthRedirectsWhenNoUser(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called when unauthenticated")
	})

	handler := RequireAuth(inner)
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/login" {
		t.Errorf("Location = %q, want %q", location, "/login")
	}
}

func TestRequireAuthPassesThroughWhenAuthenticated(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireAuth(inner)

	user := &db.User{
		ID:        1,
		Email:     "bob@example.com",
		Name:      "Bob",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	ctx := context.WithValue(req.Context(), userKey, user)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called for authenticated request")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
