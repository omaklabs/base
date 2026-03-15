package flash

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetAndGet(t *testing.T) {
	// Set a flash message.
	rec := httptest.NewRecorder()
	Set(rec, "Record created", "success")

	// Build a new request carrying the cookie that was just set.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}

	got := Get(req)
	if got == nil {
		t.Fatal("Get() returned nil, want flash")
	}
	if got.Message != "Record created" {
		t.Errorf("Message = %q, want %q", got.Message, "Record created")
	}
	if got.Variant != "success" {
		t.Errorf("Variant = %q, want %q", got.Variant, "success")
	}
}

func TestGetReturnsNilWithoutCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	got := Get(req)

	if got != nil {
		t.Errorf("Get() = %+v, want nil", got)
	}
}

func TestGetAndClear(t *testing.T) {
	// Set a flash.
	setRec := httptest.NewRecorder()
	Set(setRec, "Deleted", "error")

	// Build request with the cookie.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range setRec.Result().Cookies() {
		req.AddCookie(c)
	}

	// GetAndClear should return the flash and clear the cookie.
	clearRec := httptest.NewRecorder()
	got := GetAndClear(clearRec, req)

	if got == nil {
		t.Fatal("GetAndClear() returned nil, want flash")
	}
	if got.Message != "Deleted" {
		t.Errorf("Message = %q, want %q", got.Message, "Deleted")
	}

	// The clear cookie should have MaxAge -1.
	cookies := clearRec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "flash" {
			found = true
			if c.MaxAge != -1 {
				t.Errorf("clear cookie MaxAge = %d, want -1", c.MaxAge)
			}
		}
	}
	if !found {
		t.Error("expected a clear cookie to be set")
	}
}

func TestClear(t *testing.T) {
	rec := httptest.NewRecorder()
	Clear(rec)

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "flash" {
			found = true
			if c.MaxAge != -1 {
				t.Errorf("MaxAge = %d, want -1", c.MaxAge)
			}
			if c.Value != "" {
				t.Errorf("Value = %q, want empty string", c.Value)
			}
		}
	}
	if !found {
		t.Error("expected flash cookie to be set for clearing")
	}
}

func TestVariants(t *testing.T) {
	variants := []string{"success", "error", "warning", "info"}

	for _, variant := range variants {
		t.Run(variant, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Set(rec, "test message", variant)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for _, c := range rec.Result().Cookies() {
				req.AddCookie(c)
			}

			got := Get(req)
			if got == nil {
				t.Fatalf("Get() returned nil for variant %q", variant)
			}
			if got.Variant != variant {
				t.Errorf("Variant = %q, want %q", got.Variant, variant)
			}
			if got.Message != "test message" {
				t.Errorf("Message = %q, want %q", got.Message, "test message")
			}
		})
	}
}
