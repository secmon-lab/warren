package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	server "github.com/secmon-lab/warren/pkg/controller/http"
	websocket_controller "github.com/secmon-lab/warren/pkg/controller/websocket"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestHTTPServer_WebSocketEndpoint(t *testing.T) {
	ctx := context.Background()

	// Create repository with test data
	repo := repository.NewMemory()
	testTicket := ticket.Ticket{
		ID:     types.TicketID("test-ticket"),
		Status: types.TicketStatusOpen,
		Metadata: ticket.Metadata{
			Title:       "Test Ticket",
			Description: "Test Description",
		},
	}
	err := repo.PutTicket(ctx, testTicket)
	gt.NoError(t, err)

	// Create WebSocket hub and handler
	hub := websocket_controller.NewHub(ctx)
	go hub.Run()
	defer hub.Close()

	wsHandler := websocket_controller.NewHandler(hub, repo, nil)

	// Create mock use case
	mockUC := &UseCaseMock{}

	// Create HTTP server with WebSocket handler
	httpServer := server.New(mockUC,
		server.WithWebSocketHandler(wsHandler),
		server.WithNoAuthorization(true), // Disable authorization for test
	)

	testServer := httptest.NewServer(httpServer)
	defer testServer.Close()

	// Test WebSocket endpoint without authentication (should require auth)
	resp, err := http.Get(testServer.URL + "/ws/chat/ticket/test-ticket")
	gt.NoError(t, err)
	defer resp.Body.Close()

	// Should return 401 Unauthorized (no user context)
	gt.Value(t, resp.StatusCode).Equal(http.StatusUnauthorized)
}

func TestHTTPServer_WebSocketEndpoint_WithAuth(t *testing.T) {
	// This test is simplified - full WebSocket integration testing
	// would require more complex setup with proper middleware simulation
	// We verify the endpoint exists and basic routing works

	ctx := context.Background()
	repo := repository.NewMemory()
	hub := websocket_controller.NewHub(ctx)
	go hub.Run()
	defer hub.Close()

	wsHandler := websocket_controller.NewHandler(hub, repo, nil)
	mockUC := &UseCaseMock{}

	httpServer := server.New(mockUC,
		server.WithWebSocketHandler(wsHandler),
		server.WithNoAuthorization(true),
	)

	testServer := httptest.NewServer(httpServer)
	defer testServer.Close()

	// Test endpoint without proper authentication - should fail appropriately
	resp, err := http.Get(testServer.URL + "/ws/chat/ticket/test-ticket")
	gt.NoError(t, err)
	defer resp.Body.Close()

	// Should return 401 (missing user ID) - this confirms endpoint routing works
	gt.Value(t, resp.StatusCode).Equal(http.StatusUnauthorized)
}

func TestHTTPServer_WithoutWebSocketHandler(t *testing.T) {
	// Create mock use case
	mockUC := &UseCaseMock{}

	// Create HTTP server without WebSocket handler
	httpServer := server.New(mockUC,
		server.WithNoAuthorization(true),
	)

	testServer := httptest.NewServer(httpServer)
	defer testServer.Close()

	// Test WebSocket endpoint should not exist
	resp, err := http.Get(testServer.URL + "/ws/chat/ticket/test-ticket")
	gt.NoError(t, err)
	defer resp.Body.Close()

	// When WebSocket handler is not configured, the /ws route doesn't exist
	// In this case, the request falls through to the SPA handler which returns index.html
	// So we get 200 OK instead of 404. This is correct behavior.
	gt.Value(t, resp.StatusCode).Equal(http.StatusOK)
}

func TestHTTPServer_WebSocketEndpoint_NonExistentTicket(t *testing.T) {
	// This test verifies basic endpoint routing without complex auth setup

	ctx := context.Background()
	repo := repository.NewMemory()
	hub := websocket_controller.NewHub(ctx)
	go hub.Run()
	defer hub.Close()

	wsHandler := websocket_controller.NewHandler(hub, repo, nil)
	mockUC := &UseCaseMock{}

	httpServer := server.New(mockUC,
		server.WithWebSocketHandler(wsHandler),
		server.WithNoAuthorization(true),
	)

	testServer := httptest.NewServer(httpServer)
	defer testServer.Close()

	// Test endpoint without auth - should fail at auth level before reaching ticket check
	resp, err := http.Get(testServer.URL + "/ws/chat/ticket/non-existent")
	gt.NoError(t, err)
	defer resp.Body.Close()

	// Should return 401 (missing user ID) - this confirms endpoint routing works
	gt.Value(t, resp.StatusCode).Equal(http.StatusUnauthorized)
}
