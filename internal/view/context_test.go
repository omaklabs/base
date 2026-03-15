package view

import (
	"context"
	"testing"
)

func TestWithCSRFTokenAndCSRFToken(t *testing.T) {
	ctx := context.Background()
	token := "abc123"

	ctx = WithCSRFToken(ctx, token)
	got := CSRFToken(ctx)

	if got != token {
		t.Errorf("CSRFToken() = %q, want %q", got, token)
	}
}

func TestCSRFTokenReturnsEmptyWhenMissing(t *testing.T) {
	ctx := context.Background()
	got := CSRFToken(ctx)

	if got != "" {
		t.Errorf("CSRFToken() = %q, want empty string", got)
	}
}

func TestWithFlashAndGetFlash(t *testing.T) {
	ctx := context.Background()
	f := &Flash{Message: "Saved!", Variant: "success"}

	ctx = WithFlash(ctx, f)
	got := GetFlash(ctx)

	if got == nil {
		t.Fatal("GetFlash() returned nil, want flash")
	}
	if got.Message != f.Message {
		t.Errorf("Message = %q, want %q", got.Message, f.Message)
	}
	if got.Variant != f.Variant {
		t.Errorf("Variant = %q, want %q", got.Variant, f.Variant)
	}
}

func TestGetFlashReturnsNilWhenMissing(t *testing.T) {
	ctx := context.Background()
	got := GetFlash(ctx)

	if got != nil {
		t.Errorf("GetFlash() = %+v, want nil", got)
	}
}
