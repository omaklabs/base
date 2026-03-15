package middleware

import (
	"encoding/json"
	"net/http"
)

// InternalAPIKey returns middleware that protects routes with a shared secret.
// Requests must include an X-Internal-Key header whose value matches the
// expected key. Requests with a missing or incorrect key receive a 401 JSON
// response.
func InternalAPIKey(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Internal-Key") != key {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
