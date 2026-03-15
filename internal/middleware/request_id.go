package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)

// contextKey is an unexported type used for context keys in this package,
// preventing collisions with keys from other packages.
type contextKey int

const (
	requestIDKey contextKey = iota
	userKey
)

// generateUUID produces a version 4 UUID using crypto/rand.
func generateUUID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

// RequestID is middleware that generates a unique request ID for every incoming
// request. The ID is added to the request context and set as the X-Request-ID
// response header. If the incoming request already carries an X-Request-ID
// header, that value is reused.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = generateUUID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext extracts the request ID from the given context.
// Returns an empty string if no request ID is present.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
