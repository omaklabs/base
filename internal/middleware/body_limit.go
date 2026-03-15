package middleware

import "net/http"

// BodyLimit returns middleware that limits the size of incoming request bodies.
// It wraps r.Body with http.MaxBytesReader so that any read beyond maxBytes
// returns an error. A typical default is 10 MB (10 * 1024 * 1024).
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
