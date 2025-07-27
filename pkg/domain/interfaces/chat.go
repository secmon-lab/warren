package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

// ChatNotifier provides an abstraction for chat notification services
type ChatNotifier interface {
	// NotifyMessage sends a message to a chat room/channel
	NotifyMessage(ctx context.Context, ticketID types.TicketID, message string) error

	// NotifyTrace sends a trace message (debug/info level) to a chat room/channel
	NotifyTrace(ctx context.Context, ticketID types.TicketID, message string) error
}
