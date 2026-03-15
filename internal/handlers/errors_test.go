package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderErrorDevMode(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/boom", nil)
	testErr := errors.New("something broke")

	RenderError(w, r, testErr, true)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "something broke") {
		t.Error("dev mode response should contain the error message")
	}
	if !strings.Contains(body, "goroutine") {
		t.Error("dev mode response should contain a stack trace")
	}
	if !strings.Contains(body, "GET") {
		t.Error("dev mode response should contain the request method")
	}
	if !strings.Contains(body, "/boom") {
		t.Error("dev mode response should contain the request path")
	}
}

func TestRenderErrorProdMode(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/boom", nil)
	testErr := errors.New("secret database error")

	RenderError(w, r, testErr, false)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	body := w.Body.String()
	if strings.Contains(body, "secret database error") {
		t.Error("prod mode response should NOT contain the error message")
	}
	if strings.Contains(body, "goroutine") {
		t.Error("prod mode response should NOT contain a stack trace")
	}
}

func TestRenderNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)

	RenderNotFound(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
