package chat

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// SlackNotifier sends notifications via Slack using the msg utility functions
type SlackNotifier struct {
	// SlackNotifier uses the msg package functions that are injected into context
	// This allows it to work with the existing Slack infrastructure
}

// NewSlackNotifier creates a new SlackNotifier
func NewSlackNotifier() interfaces.ChatNotifier {
	return &SlackNotifier{}
}

// NotifyMessage sends a message via Slack using msg.Notify
func (s *SlackNotifier) NotifyMessage(ctx context.Context, ticketID types.TicketID, message string) error {
	// Use the msg.Notify function which will use the Slack context if available
	msg.Notify(ctx, "%s", message)
	return nil
}

// NotifyTrace sends a trace message via Slack using msg.Trace
func (s *SlackNotifier) NotifyTrace(ctx context.Context, ticketID types.TicketID, message string) error {
	// Use the msg.Trace function which will use the Slack context if available
	msg.Trace(ctx, "%s", message)
	return nil
}
