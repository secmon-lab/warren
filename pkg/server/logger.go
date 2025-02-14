package server

import (
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

		sw := &statusResponseWriter{ResponseWriter: w}

		next.ServeHTTP(sw, r)

		logger.Info("Request", "method", r.Method, "path", r.URL.Path, "status", sw.status)
	})
}
