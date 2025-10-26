package websocket_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	websocket_ctrl "github.com/secmon-lab/warren/pkg/controller/websocket"
	websocket_model "github.com/secmon-lab/warren/pkg/domain/model/websocket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func setupTestHub(t *testing.T) (*websocket_ctrl.Hub, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := websocket_ctrl.NewHub(ctx)

	// Start hub in goroutine
	go hub.Run()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	return hub, cancel
}

func createMockClient(t *testing.T, hub *websocket_ctrl.Hub, ticketID types.TicketID, userID string) *websocket_ctrl.Client {
	// Create a mock websocket connection (we won't actually use it in tests)
	// In real usage, this would be from websocket.Upgrader.Upgrade()
	return hub.NewClient(nil, ticketID, userID)
}

func TestHub_ClientRegistration(t *testing.T) {
	hub, cancel := setupTestHub(t)
	defer cancel()
	defer func() { _ = hub.Close() }()

	ticketID := types.TicketID("test-ticket")
	userID := "test-user"

	// Create and register client
	client := createMockClient(t, hub, ticketID, userID)
	hub.Register(client)

	// Wait for registration to process
	time.Sleep(10 * time.Millisecond)

	// Verify client is registered
	gt.Value(t, hub.GetClientCount(ticketID)).Equal(1)
	gt.Value(t, hub.GetTotalClientCount()).Equal(1)

	activeTickets := hub.GetActiveTickets()
	gt.Array(t, activeTickets).Length(1)
	gt.Value(t, activeTickets[0]).Equal(ticketID)
}

func TestHub_ClientUnregistration(t *testing.T) {
	hub, cancel := setupTestHub(t)
	defer cancel()
	defer func() { _ = hub.Close() }()

	ticketID := types.TicketID("test-ticket")
	userID := "test-user"

	// Register client
	client := createMockClient(t, hub, ticketID, userID)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Verify client is registered
	gt.Value(t, hub.GetClientCount(ticketID)).Equal(1)

	// Unregister client
	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	// Verify client is unregistered
	gt.Value(t, hub.GetClientCount(ticketID)).Equal(0)
	gt.Value(t, hub.GetTotalClientCount()).Equal(0)
	gt.Array(t, hub.GetActiveTickets()).Length(0)
}

func TestHub_MultipleClients(t *testing.T) {
	hub, cancel := setupTestHub(t)
	defer cancel()
	defer func() { _ = hub.Close() }()

	ticketID := types.TicketID("test-ticket")

	// Register multiple clients for the same ticket
	client1 := createMockClient(t, hub, ticketID, "user1")
	client2 := createMockClient(t, hub, ticketID, "user2")
	client3 := createMockClient(t, hub, ticketID, "user3")

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(10 * time.Millisecond)

	// Verify all clients are registered
	gt.Value(t, hub.GetClientCount(ticketID)).Equal(3)
	gt.Value(t, hub.GetTotalClientCount()).Equal(3)

	// Unregister one client
	hub.Unregister(client2)
	time.Sleep(10 * time.Millisecond)

	// Verify count is updated
	gt.Value(t, hub.GetClientCount(ticketID)).Equal(2)
	gt.Value(t, hub.GetTotalClientCount()).Equal(2)
}

func TestHub_MultipleTickets(t *testing.T) {
	hub, cancel := setupTestHub(t)
	defer cancel()
	defer func() { _ = hub.Close() }()

	ticket1 := types.TicketID("ticket-1")
	ticket2 := types.TicketID("ticket-2")

	// Register clients for different tickets
	client1 := createMockClient(t, hub, ticket1, "user1")
	client2 := createMockClient(t, hub, ticket2, "user2")
	client3 := createMockClient(t, hub, ticket1, "user3")

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(10 * time.Millisecond)

	// Verify client counts per ticket
	gt.Value(t, hub.GetClientCount(ticket1)).Equal(2)
	gt.Value(t, hub.GetClientCount(ticket2)).Equal(1)
	gt.Value(t, hub.GetTotalClientCount()).Equal(3)

	// Verify active tickets
	activeTickets := hub.GetActiveTickets()
	gt.Array(t, activeTickets).Length(2)

	// Check that both tickets are in the active list
	ticketMap := make(map[types.TicketID]bool)
	for _, ticket := range activeTickets {
		ticketMap[ticket] = true
	}
	gt.Value(t, ticketMap[ticket1]).Equal(true)
	gt.Value(t, ticketMap[ticket2]).Equal(true)
}

func TestHub_SendMessageToTicket(t *testing.T) {
	hub, cancel := setupTestHub(t)
	defer cancel()
	defer func() { _ = hub.Close() }()

	ticketID := types.TicketID("test-ticket")
	user := &websocket_model.User{
		ID:   "user123",
		Name: "Test User",
	}

	// Send message (should not error even with no clients)
	err := hub.SendMessageToTicket(ticketID, "Hello, World!", user)
	gt.NoError(t, err)

	// Send status message
	err = hub.SendStatusToTicket(ticketID, "System ready")
	gt.NoError(t, err)

	// Send error message
	err = hub.SendErrorToTicket(ticketID, "Something went wrong")
	gt.NoError(t, err)
}

func TestHub_BroadcastToTicket(t *testing.T) {
	hub, cancel := setupTestHub(t)
	defer cancel()
	defer func() { _ = hub.Close() }()

	ticketID := types.TicketID("test-ticket")
	message := []byte(`{"type":"message","content":"broadcast test"}`)

	// Broadcast message (should not error even with no clients)
	hub.BroadcastToTicket(ticketID, message)

	// No error expected, even with no clients
	gt.Value(t, hub.GetClientCount(ticketID)).Equal(0)
}

func TestHub_Close(t *testing.T) {
	hub, cancel := setupTestHub(t)
	defer cancel()

	ticketID := types.TicketID("test-ticket")

	// Register some clients
	client1 := createMockClient(t, hub, ticketID, "user1")
	client2 := createMockClient(t, hub, ticketID, "user2")

	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)

	gt.Value(t, hub.GetTotalClientCount()).Equal(2)

	// Close hub
	err := hub.Close()
	gt.NoError(t, err)

	// Give close time to process
	time.Sleep(10 * time.Millisecond)

	// After close, client count should still show previous state
	// (the hub doesn't clear its internal state on close, just stops processing)
	gt.Value(t, hub.GetTotalClientCount()).Equal(2)
}

func TestHub_GetNonExistentTicket(t *testing.T) {
	hub, cancel := setupTestHub(t)
	defer cancel()
	defer func() { _ = hub.Close() }()

	nonExistentTicket := types.TicketID("non-existent")

	// Should return 0 for non-existent ticket
	gt.Value(t, hub.GetClientCount(nonExistentTicket)).Equal(0)
}

func TestHub_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := websocket_ctrl.NewHub(ctx)

	// Start hub
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Cancel context
	cancel()

	// Give hub time to shutdown
	time.Sleep(20 * time.Millisecond)

	// Hub should handle context cancellation gracefully
	// (no panic expected)
	gt.Value(t, true).Equal(true) // Test passes if no panic occurs
}

func TestHub_ErrorHandling_Creation(t *testing.T) {
	// Test that Hub can be created and closed without errors
	ctx := context.Background()

	hub := websocket_ctrl.NewHub(ctx)
	gt.Value(t, hub).NotNil()

	go hub.Run()

	// Close should not cause any errors
	err := hub.Close()
	gt.NoError(t, err)
}

func TestHub_ErrorHandling_Operations(t *testing.T) {
	// Test Hub operations and error scenarios
	ctx := context.Background()

	hub := websocket_ctrl.NewHub(ctx)
	go hub.Run()
	defer func() { _ = hub.Close() }()

	// Try to get client count for non-existent ticket
	count := hub.GetClientCount(types.TicketID("non-existent"))
	gt.Value(t, count).Equal(0) // Should return 0 for non-existent tickets

	// Try to broadcast to non-existent ticket
	// This should not cause a panic
	hub.BroadcastToTicket(types.TicketID("non-existent"), []byte("test message"))
}

func TestHub_ConcurrentOperations(t *testing.T) {
	// Test concurrent operations on Hub
	ctx := context.Background()

	hub := websocket_ctrl.NewHub(ctx)
	go hub.Run()
	defer func() { _ = hub.Close() }()

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
