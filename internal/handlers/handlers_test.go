package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsHTMX(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{"with HX-Request header", "true", true},
		{"without HX-Request header", "", false},
		{"with wrong value", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				r.Header.Set("HX-Request", tt.header)
			}
			if got := isHTMX(r); got != tt.want {
				t.Errorf("isHTMX() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotFoundHandler(t *testing.T) {
	handler := handleNotFound()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/nonexistent", nil)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestWelcomeHandler(t *testing.T) {
	handler := handleWelcome()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}
