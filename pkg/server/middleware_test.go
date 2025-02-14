package server_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/server"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func TestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	r := chi.NewRouter()
	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := logging.New(&buf, slog.LevelDebug, logging.FormatJSON)
			h.ServeHTTP(w, r.WithContext(logging.With(r.Context(), logger)))
		})
	})
	r.Use(server.LoggingMiddleware)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer test_token")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	gt.S(t, buf.String()).Contains(`"method":"GET"`)
	gt.S(t, buf.String()).Contains(`"path":"/"`)
	gt.S(t, buf.String()).Contains(`"status":200`)
	gt.S(t, buf.String()).NotContains(`test_token`)
}
