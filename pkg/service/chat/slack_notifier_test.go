package chat_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/chat"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func TestSlackNotifier_NotifyMessage(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket")
	message := "Test message"

	// Track messages sent
	var sentMessages []string
	notifyFunc := func(ctx context.Context, msg string) {
		sentMessages = append(sentMessages, msg)
	}

	// Set up context with msg functions
	ctx = msg.With(ctx, notifyFunc, nil)

	// Create notifier
	notifier := chat.NewSlackNotifier()

	// Send message
	err := notifier.NotifyMessage(ctx, ticketID, message)
	gt.NoError(t, err)

	// Verify message was sent
	gt.Value(t, len(sentMessages)).Equal(1)
	gt.Value(t, sentMessages[0]).Equal(message)
}

func TestSlackNotifier_NotifyTrace(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket")
	message := "Test trace"

	// Track trace messages sent
	var sentTraces []string
	traceFunc := func(ctx context.Context, msg string) func(ctx context.Context, msg string) {
		// The trace function creates an initial trace message
		sentTraces = append(sentTraces, msg)
		// Return function for subsequent trace calls
		return func(ctx context.Context, traceMsg string) {
			sentTraces = append(sentTraces, traceMsg)
		}
	}

	// Set up context with msg functions
	ctx = msg.With(ctx, nil, traceFunc)

	// Create notifier
	notifier := chat.NewSlackNotifier()

	// Send trace
	err := notifier.NotifyTrace(ctx, ticketID, message)
	gt.NoError(t, err)

	// Verify trace was sent - msg.Trace calls the initial trace function
	gt.Value(t, len(sentTraces)).Equal(1)
	gt.Value(t, sentTraces[0]).Equal(message)
}

func TestSlackNotifier_NoContext(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket")
	message := "Test message"

	// Create notifier
	notifier := chat.NewSlackNotifier()

	// Send message without msg context - should not error
	err := notifier.NotifyMessage(ctx, ticketID, message)
	gt.NoError(t, err)

	// Send trace without msg context - should not error
	err = notifier.NotifyTrace(ctx, ticketID, message)
	gt.NoError(t, err)
}

func TestSlackNotifier_Creation(t *testing.T) {
	notifier := chat.NewSlackNotifier()
	gt.Value(t, notifier).NotNil()
}
