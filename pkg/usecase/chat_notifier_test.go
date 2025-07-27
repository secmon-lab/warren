package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/chat"
	"github.com/secmon-lab/warren/pkg/usecase"
)

// MockChatNotifier for testing
type MockChatNotifier struct {
	messages []string
	traces   []string
}

func (m *MockChatNotifier) NotifyMessage(ctx context.Context, ticketID types.TicketID, message string) error {
	m.messages = append(m.messages, message)
	return nil
}

func (m *MockChatNotifier) NotifyTrace(ctx context.Context, ticketID types.TicketID, message string) error {
	m.traces = append(m.traces, message)
	return nil
}

func TestUseCases_WithChatNotifier(t *testing.T) {
	// Create mock chat notifier
	mockNotifier := &MockChatNotifier{}

	// Create UseCases with chat notifier
	uc := usecase.New(
		usecase.WithChatNotifier(mockNotifier),
	)

	// Verify notifier is not nil (no public getter, but we can test functionality)
	gt.Value(t, uc).NotNil()
}

func TestUseCases_DefaultWithoutChatNotifier(t *testing.T) {
	// Create UseCases without chat notifier
	uc := usecase.New()

	// Should not panic or error
	gt.Value(t, uc).NotNil()
}

func TestChat_ChatNotifierIntegration(t *testing.T) {
	// Test that ChatNotifier can be integrated into UseCases properly

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

	err := multiNotifier.NotifyMessage(ctx, ticketID, "Test message")
	gt.NoError(t, err)

	// Verify message was received by WebSocket notifier
	gt.Value(t, len(wsNotifier.messages)).Equal(1)
	gt.Value(t, wsNotifier.messages[0]).Equal("Test message")

	// Test trace notification
	err = multiNotifier.NotifyTrace(ctx, ticketID, "Test trace")
	gt.NoError(t, err)

	// Verify trace was received by WebSocket notifier
	gt.Value(t, len(wsNotifier.traces)).Equal(1)
	gt.Value(t, wsNotifier.traces[0]).Equal("Test trace")
}
