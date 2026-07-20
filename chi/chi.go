// Package chi provides middleware for Chi-based HTTP services that integrates
// with the go-service-kit logging and telemetry packages.
//
// Typical setup in server.go:
//
//	r.Use(chimiddleware.RequestID)     // chi generates the request ID
//	r.Use(chikit.RequestIDMiddleware)  // bridges it into the logging context
//	r.Use(chikit.RequestLoggerMiddleware("my-service"))
package chi

import (
	"net/http"
	"strings"
	"time"

	"github.com/agentruntime-io/go-service-kit/logging"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestIDMiddleware reads the request ID that chi's RequestID middleware
// placed in the context under chi's own typed key and re-stores it under the
// logging kit's RequestIDKey.  This means CorrelationFields picks it up
// natively — no registered extractor is needed.
//
// Must be placed after chimiddleware.RequestID in the middleware chain.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := chimiddleware.GetReqID(r.Context()); id != "" {
			r = r.WithContext(logging.WithRequestID(r.Context(), id))
		}
		next.ServeHTTP(w, r)
	})
}

// RequestLoggerMiddleware returns a Chi middleware that emits one structured
// access-log entry per non-probe request via logging.LogRequestComplete.
// serviceName identifies the service in the log entry (e.g. "work").
//
// It replaces hand-rolled httplog packages — every Chi service should use
// this instead of writing its own request logger.
func RequestLoggerMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(ww, r)

			if isProbe(r) {
				return
			}

			route := r.URL.Path
			if rc := chi.RouteContext(r.Context()); rc != nil {
				if p := rc.RoutePattern(); p != "" {
					route = p
				}
			}

			logging.LogRequestComplete(logging.L(), r.Context(), logging.RequestComplete{
				Service:  serviceName,
				Method:   r.Method,
				Route:    route,
				Status:   ww.status,
				Duration: time.Since(start),
			})
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code and
// byte count written by the handler.
type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
	wrote  bool
}

func (w *responseWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.wrote = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// isProbe returns true for Kubernetes/ELB health-check requests that should
// not generate access-log entries.
func isProbe(r *http.Request) bool {
	if r == nil {
		return false
	}
	path := r.URL.Path
	if path == "/healthz" || path == "/readyz" {
		return true
	}
	return strings.Contains(r.Header.Get("User-Agent"), "ELB-HealthChecker")
}
