package chat_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/chat"
)

// MockNotifier implements interfaces.ChatNotifier for testing
type MockNotifier struct {
	name         string
	shouldFail   bool
	messagesSent []string
	tracesSent   []string
}

func (m *MockNotifier) NotifyMessage(ctx context.Context, ticketID types.TicketID, message string) error {
	if m.shouldFail {
		return goerr.New("mock notifier failed", goerr.V("notifier", m.name))
	}
	m.messagesSent = append(m.messagesSent, message)
	return nil
}

func (m *MockNotifier) NotifyTrace(ctx context.Context, ticketID types.TicketID, message string) error {
	if m.shouldFail {
		return goerr.New("mock notifier failed", goerr.V("notifier", m.name))
	}
	m.tracesSent = append(m.tracesSent, message)
	return nil
}

func TestMultiNotifier_NotifyMessage_Success(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket")
	message := "Test message"

	// Create mock notifiers
	slack := &MockNotifier{name: "slack", shouldFail: false}
	websocket := &MockNotifier{name: "websocket", shouldFail: false}

	// Create MultiNotifier
	multiNotifier := chat.NewMultiNotifier(slack, websocket)

	// Send message
	err := multiNotifier.NotifyMessage(ctx, ticketID, message)
	gt.NoError(t, err)

	// Verify both notifiers received the message
	gt.Value(t, len(slack.messagesSent)).Equal(1)
	gt.Value(t, slack.messagesSent[0]).Equal(message)
	gt.Value(t, len(websocket.messagesSent)).Equal(1)
	gt.Value(t, websocket.messagesSent[0]).Equal(message)
}

func TestMultiNotifier_NotifyMessage_PartialFailure(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket")
	message := "Test message"

	// Create mock notifiers - one will fail
	slack := &MockNotifier{name: "slack", shouldFail: false}
	websocket := &MockNotifier{name: "websocket", shouldFail: true}

	// Create MultiNotifier
	multiNotifier := chat.NewMultiNotifier(slack, websocket)

	// Send message
	err := multiNotifier.NotifyMessage(ctx, ticketID, message)
	gt.Error(t, err)

	// Verify successful notifier still received the message
	gt.Value(t, len(slack.messagesSent)).Equal(1)
	gt.Value(t, slack.messagesSent[0]).Equal(message)

	// Verify failed notifier didn't send anything
	gt.Value(t, len(websocket.messagesSent)).Equal(0)
}

func TestMultiNotifier_NotifyTrace_Success(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket")
	message := "Test trace"

	// Create mock notifiers
	slack := &MockNotifier{name: "slack", shouldFail: false}
	websocket := &MockNotifier{name: "websocket", shouldFail: false}

	// Create MultiNotifier
	multiNotifier := chat.NewMultiNotifier(slack, websocket)

	// Send trace
	err := multiNotifier.NotifyTrace(ctx, ticketID, message)
	gt.NoError(t, err)

	// Verify both notifiers received the trace
	gt.Value(t, len(slack.tracesSent)).Equal(1)
	gt.Value(t, slack.tracesSent[0]).Equal(message)
	gt.Value(t, len(websocket.tracesSent)).Equal(1)
	gt.Value(t, websocket.tracesSent[0]).Equal(message)
}

func TestMultiNotifier_Empty(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket")
	message := "Test message"

	// Create empty MultiNotifier
	multiNotifier := chat.NewMultiNotifier()

	// Send message - should succeed with no notifiers
	err := multiNotifier.NotifyMessage(ctx, ticketID, message)
	gt.NoError(t, err)

	// Send trace - should succeed with no notifiers
	err = multiNotifier.NotifyTrace(ctx, ticketID, message)
	gt.NoError(t, err)
}
