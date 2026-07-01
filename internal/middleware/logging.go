package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

type LoggingOptions struct {
	// SkipPaths are exact paths that are not logged (e.g. health probes).
	SkipPaths map[string]struct{}
}

func Logging(logger *slog.Logger, opts LoggingOptions) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, skip := opts.SkipPaths[r.URL.Path]; skip {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}

			next.ServeHTTP(rec, r)

			duration := time.Since(start)
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}

			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration_ms", duration.Milliseconds(),
				"bytes", rec.bytes,
				"remote_addr", clientIP(r),
				"user_agent", r.UserAgent(),
			}

			if id := RequestIDFromContext(r.Context()); id != "" {
				attrs = append(attrs, "request_id", id)
			}

			if query := r.URL.RawQuery; query != "" {
				attrs = append(attrs, "query", query)
			}

			switch {
			case status >= http.StatusInternalServerError:
				logger.Error("request", attrs...)
			case status >= http.StatusBadRequest:
				logger.Warn("request", attrs...)
			default:
				logger.Info("request", attrs...)
			}
		})
	}
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if ip, _, ok := strings.Cut(forwarded, ","); ok {
			return strings.TrimSpace(ip)
		}
		return strings.TrimSpace(forwarded)
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	host, _, err := strings.Cut(r.RemoteAddr, ":")
	if err {
		return r.RemoteAddr
	}
	return host
}
