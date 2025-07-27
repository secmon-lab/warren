package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/chat"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestChat_SlackAndWebSocketCoexistence(t *testing.T) {
	// Test that both Slack and WebSocket receive notifications when using MultiNotifier

	// Create mock notifiers
	slackNotifier := chat.NewSlackNotifier()
	wsNotifier := &MockChatNotifier{}

	// Create MultiNotifier
	multiNotifier := chat.NewMultiNotifier(slackNotifier, wsNotifier)

	// Create UseCases with MultiNotifier
	uc := usecase.New(
		usecase.WithChatNotifier(multiNotifier),
	)

	// Verify UseCases was created
	gt.Value(t, uc).NotNil()

	// Test notification through the notifier directly
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket")

	// Send message through MultiNotifier
	err := multiNotifier.NotifyMessage(ctx, ticketID, "Test message for both channels")
	gt.NoError(t, err)

	// Verify message was received by WebSocket notifier
	gt.Value(t, len(wsNotifier.messages)).Equal(1)
	gt.Value(t, wsNotifier.messages[0]).Equal("Test message for both channels")

	// Send trace through MultiNotifier
	err = multiNotifier.NotifyTrace(ctx, ticketID, "Test trace for both channels")
	gt.NoError(t, err)

	// Verify trace was received by WebSocket notifier
	gt.Value(t, len(wsNotifier.traces)).Equal(1)
	gt.Value(t, wsNotifier.traces[0]).Equal("Test trace for both channels")
}

func TestChat_WebSocketOnlyMode(t *testing.T) {
	// Test that WebSocket-only mode works when Slack is not configured

	// Create WebSocket notifier only
	wsNotifier := &MockChatNotifier{}

	// Create UseCases with WebSocket notifier only
	uc := usecase.New(
		usecase.WithChatNotifier(wsNotifier),
	)

	// Verify UseCases was created
	gt.Value(t, uc).NotNil()

	// Test notification
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket")

	// Simulate notification (in real use, this would be called from Chat method)
	err := wsNotifier.NotifyMessage(ctx, ticketID, "WebSocket-only message")
	gt.NoError(t, err)

	// Verify message was received
	gt.Value(t, len(wsNotifier.messages)).Equal(1)
	gt.Value(t, wsNotifier.messages[0]).Equal("WebSocket-only message")
}

func TestChat_NotifierFailureHandling(t *testing.T) {
	// Test that failures in one notifier don't affect others

	// Create a failing notifier
	failingNotifier := &FailingChatNotifier{}
	wsNotifier := &MockChatNotifier{}

	// Create MultiNotifier with failing notifier first
	multiNotifier := chat.NewMultiNotifier(failingNotifier, wsNotifier)

	// Create UseCases with MultiNotifier
	uc := usecase.New(
		usecase.WithChatNotifier(multiNotifier),
	)

	// Verify UseCases was created
	gt.Value(t, uc).NotNil()

	// Test notification through the notifier directly
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket")

	// Send message - should fail because one notifier fails
	err := multiNotifier.NotifyMessage(ctx, ticketID, "Message despite failure")
	gt.Error(t, err) // MultiNotifier returns aggregated errors

	// Verify message was still received by working notifier (best effort)
	gt.Value(t, len(wsNotifier.messages)).Equal(1)
	gt.Value(t, wsNotifier.messages[0]).Equal("Message despite failure")
}

// FailingChatNotifier always returns errors
type FailingChatNotifier struct{}

func (f *FailingChatNotifier) NotifyMessage(ctx context.Context, ticketID types.TicketID, message string) error {
	return errors.New("simulated failure")
}

func (f *FailingChatNotifier) NotifyTrace(ctx context.Context, ticketID types.TicketID, message string) error {
	return errors.New("simulated failure")
}
