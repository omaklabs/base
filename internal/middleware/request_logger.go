package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/omaklabs/base/internal/logger"
)

// responseWriter wraps http.ResponseWriter to capture the status code and
// number of bytes written for logging purposes.
type responseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int
	wroteHeader  bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Flush implements http.Flusher. It delegates to the underlying ResponseWriter
// if it supports flushing (needed for SSE / streaming responses).
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// RequestLogger returns middleware that logs every HTTP request using the
// provided structured logger. Each log entry includes the method, path,
// status code, duration in milliseconds, request ID, user agent, and bytes
// written. The log level is chosen based on the response status code:
// Info for 2xx/3xx, Warn for 4xx, and Error for 5xx.
func RequestLogger(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			durationMs := float64(duration.Nanoseconds()) / 1e6

			msg := fmt.Sprintf("%s %s %d %.1fms", r.Method, r.URL.Path, rw.status, durationMs)

			fields := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", durationMs,
				"request_id", RequestIDFromContext(r.Context()),
				"user_agent", r.UserAgent(),
				"bytes_written", rw.bytesWritten,
			}

			switch {
			case rw.status >= 500:
				log.Error(msg, fields...)
			case rw.status >= 400:
				log.Warn(msg, fields...)
			default:
				log.Info(msg, fields...)
			}
		})
	}
}
