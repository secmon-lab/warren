package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
)

// ChatUseCase defines the interface for chat processing.
// Implementations can provide different execution strategies
// (e.g., Plan & Execute, multi-agent parallel execution).
// For ticketless chat, pass a placeholder ticket with empty ID and
// provide Slack history for conversation context. For ticket-bound
// chat, slackHistory should be nil.
type ChatUseCase interface {
	Execute(ctx context.Context, target *ticket.Ticket, slackHistory []slack.HistoryMessage, message string) error
}
