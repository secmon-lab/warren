package http

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
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

// Hijack implements the http.Hijacker interface for WebSocket support
func (w *statusResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logging.From(r.Context()).With("request_id", uuid.New().String())

		// Prepare debug attributes before handling request
		var debugAttrs []any
		var body []byte
		if logger.Enabled(r.Context(), slog.LevelDebug) {
			// Read and store request body for debug logging
			var err error
			body, err = io.ReadAll(r.Body)
			if err != nil {
				logger.Warn("failed to read request body", "error", err)
			}
			r.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		// Handle the request
		sw := &statusResponseWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r.WithContext(logging.With(r.Context(), logger)))

		// Basic attributes for INFO level
		infoAttrs := []any{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.Int("status", sw.status),
		}

		// Log at INFO level
		logger.Info("Access Log", infoAttrs...)

		// Log at DEBUG level with additional details
		if logger.Enabled(r.Context(), slog.LevelDebug) {
			debugAttrs = []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote", r.RemoteAddr),
				slog.Int("status", sw.status),
				slog.Any("query", r.URL.Query()),
				slog.Any("headers", r.Header),
			}
			if body != nil {
				debugAttrs = append(debugAttrs, slog.String("body", string(body)))
			}
			logger.Debug("Detailed Access Log", debugAttrs...)
		}
	})
}
