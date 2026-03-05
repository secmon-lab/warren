package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
)

// ChatUseCase defines the interface for chat processing.
// Implementations can provide different execution strategies
// (e.g., Plan & Execute, multi-agent parallel execution).
type ChatUseCase interface {
	Execute(ctx context.Context, target *ticket.Ticket, message string) error
}
