package websocket

import (
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/usecase"
)

// NewTestHandler creates a WebSocket handler configured for testing
// It automatically sets the frontend URL to match the test server
func NewTestHandler(hub *Hub, repository interfaces.Repository, useCases *usecase.UseCases) (*Handler, *httptest.Server) {
	// Create a test server to get a URL
	r := chi.NewRouter()
	server := httptest.NewServer(r)

	// Create handler with test server URL as frontend URL
	handler := NewHandler(hub, repository, useCases).
		WithFrontendURL(server.URL)

	return handler, server
}
