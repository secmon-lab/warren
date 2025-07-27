package websocket_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	websocket_controller "github.com/secmon-lab/warren/pkg/controller/websocket"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestWebSocketErrorHandling_HubCreation(t *testing.T) {
	// Test that Hub can be created and closed without errors
	ctx := context.Background()

	hub := websocket_controller.NewHub(ctx)
	gt.Value(t, hub).NotNil()

	go hub.Run()

	// Close should not cause any errors
	err := hub.Close()
	gt.NoError(t, err)
}

func TestWebSocketErrorHandling_HandlerCreation(t *testing.T) {
	// Test that Handler can be created with nil UseCases
	ctx := context.Background()

	repo := repository.NewMemory()
	hub := websocket_controller.NewHub(ctx)
	defer hub.Close()

	// Should not panic with nil UseCases
	handler := websocket_controller.NewHandler(hub, repo, nil)
	gt.Value(t, handler).NotNil()
}

func TestWebSocketErrorHandling_RepositoryOperations(t *testing.T) {
	// Test repository operations for error handling
	ctx := context.Background()
	repo := repository.NewMemory()

	// Try to get non-existent ticket
	_, err := repo.GetTicket(ctx, types.TicketID("non-existent"))
	gt.Error(t, err) // Should return an error

	// Create a ticket
	testTicket := ticket.Ticket{
		ID:     types.TicketID("test-ticket"),
		Status: types.TicketStatusOpen,
		Metadata: ticket.Metadata{
			Title:       "Test Ticket",
			Description: "Test Description",
		},
	}
	err = repo.PutTicket(ctx, testTicket)
	gt.NoError(t, err)

	// Now it should be retrievable
	retrievedTicket, err := repo.GetTicket(ctx, types.TicketID("test-ticket"))
	gt.NoError(t, err)
	gt.Value(t, retrievedTicket.ID).Equal(testTicket.ID)
}

func TestWebSocketErrorHandling_HubOperations(t *testing.T) {
	// Test Hub operations and error scenarios
	ctx := context.Background()

	hub := websocket_controller.NewHub(ctx)
	go hub.Run()
	defer hub.Close()

	// Try to get client count for non-existent ticket
	count := hub.GetClientCount(types.TicketID("non-existent"))
	gt.Value(t, count).Equal(0) // Should return 0 for non-existent tickets

	// Try to broadcast to non-existent ticket
	// This should not cause a panic
	hub.BroadcastToTicket(types.TicketID("non-existent"), []byte("test message"))
}

func TestWebSocketErrorHandling_ConcurrentOperations(t *testing.T) {
	// Test concurrent operations on Hub
	ctx := context.Background()

	hub := websocket_controller.NewHub(ctx)
	go hub.Run()
	defer hub.Close()

	// Start multiple goroutines doing operations concurrently
	done := make(chan bool, 3)

	// Goroutine 1: Try to get client counts
	go func() {
		for i := 0; i < 10; i++ {
			hub.GetClientCount(types.TicketID("test-ticket"))
		}
		done <- true
	}()

	// Goroutine 2: Try to broadcast
	go func() {
		for i := 0; i < 10; i++ {
			hub.BroadcastToTicket(types.TicketID("test-ticket"), []byte("test message"))
		}
		done <- true
	}()

	// Goroutine 3: Try to send status messages
	go func() {
		for i := 0; i < 10; i++ {
			_ = hub.SendStatusToTicket(types.TicketID("test-ticket"), "test status")
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Hub should still be functional
	gt.Value(t, hub).NotNil()
}
