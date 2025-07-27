package chat_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/controller/websocket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/chat"
)

func TestWebSocketNotifier_NotifyMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create hub
	hub := websocket.NewHub(ctx)
	go hub.Run()
	defer hub.Close()

	// Allow hub to start
	time.Sleep(10 * time.Millisecond)

	// Create notifier
	notifier := chat.NewWebSocketNotifier(hub)

	ticketID := types.TicketID("test-ticket")
	message := "Test message"

	// Send message - should not error even with no connected clients
	err := notifier.NotifyMessage(ctx, ticketID, message)
	gt.NoError(t, err)
}

func TestWebSocketNotifier_NotifyTrace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create hub
	hub := websocket.NewHub(ctx)
	go hub.Run()
	defer hub.Close()

	// Allow hub to start
	time.Sleep(10 * time.Millisecond)

	// Create notifier
	notifier := chat.NewWebSocketNotifier(hub)

	ticketID := types.TicketID("test-ticket")
	message := "Test trace"

	// Send trace - should not error even with no connected clients
	err := notifier.NotifyTrace(ctx, ticketID, message)
	gt.NoError(t, err)
}

func TestWebSocketNotifier_Creation(t *testing.T) {
	ctx := context.Background()
	hub := websocket.NewHub(ctx)

	// Test notifier creation
	notifier := chat.NewWebSocketNotifier(hub)
	gt.Value(t, notifier).NotNil()
}
