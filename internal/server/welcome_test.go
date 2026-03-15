package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleWelcome(t *testing.T) {
	handler := HandleWelcome()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}
