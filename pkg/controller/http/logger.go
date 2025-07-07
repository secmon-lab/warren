package http

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logging.From(r.Context()).With("request_id", uuid.New().String())

		attrs := []any{
			slog.Any("method", r.Method),
			slog.Any("path", r.URL.Path),
			slog.Any("query", r.URL.Query()),
			slog.Any("headers", r.Header),
		}

		if logger.Enabled(r.Context(), slog.LevelDebug) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Warn("failed to read request body", "error", err)
			} else {
				attrs = append(attrs, slog.Any("body", string(body)))
			}
			r.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		sw := &statusResponseWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r.WithContext(logging.With(r.Context(), logger)))
		attrs = append(attrs, slog.Int("status", sw.status))

		logger.Info("Access Log", attrs...)
	})
}
