package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
)

// Recovery returns middleware that recovers from panics in downstream handlers.
// When a panic occurs the stack trace is logged and a 500 Internal Server Error
// response is sent to the client. If logger is nil the standard log package is
// used.
func Recovery(logger *log.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()
					logger.Printf("panic recovered: %v\n%s", err, stack)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
