package view

import (
	"context"
)

// contextKey is an unexported type used for context keys in this package,
// preventing collisions with keys from other packages.
type contextKey int

const (
	csrfTokenKey contextKey = iota
	flashKey
)

// Flash holds a single flash message to display on the next page load.
type Flash struct {
	Message string
	Variant string // success, error, warning, info
}

// WithCSRFToken returns a copy of ctx carrying the CSRF token string.
func WithCSRFToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfTokenKey, token)
}

// CSRFToken extracts the CSRF token from ctx. Returns an empty string when no
// token is present.
func CSRFToken(ctx context.Context) string {
	tok, _ := ctx.Value(csrfTokenKey).(string)
	return tok
}

// WithFlash returns a copy of ctx carrying the given flash message.
func WithFlash(ctx context.Context, f *Flash) context.Context {
	return context.WithValue(ctx, flashKey, f)
}

// GetFlash extracts the flash message from ctx. Returns nil when no flash is
// present.
func GetFlash(ctx context.Context) *Flash {
	f, _ := ctx.Value(flashKey).(*Flash)
	return f
}
