package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/chat"
)

// ChatUseCase defines the interface for chat processing.
// Implementations can provide different execution strategies
// (e.g., Plan & Execute, multi-agent parallel execution).
// The ChatContext is built by the caller (e.g., ChatFromWebSocket,
// ChatFromSlack, ChatFromCLI) with all pre-fetched data needed
// for execution.
type ChatUseCase interface {
	Execute(ctx context.Context, message string, chatCtx chat.ChatContext) error
}
