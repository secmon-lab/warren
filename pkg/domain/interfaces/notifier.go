package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/event"
)

// Notifier is an interface for handling notification events
type Notifier interface {
	NotifyAlertPolicyResult(ctx context.Context, ev *event.AlertPolicyResultEvent)
	NotifyEnrichPolicyResult(ctx context.Context, ev *event.EnrichPolicyResultEvent)
	NotifyCommitPolicyResult(ctx context.Context, ev *event.CommitPolicyResultEvent)
	NotifyEnrichTaskPrompt(ctx context.Context, ev *event.EnrichTaskPromptEvent)
	NotifyEnrichTaskResponse(ctx context.Context, ev *event.EnrichTaskResponseEvent)
	NotifyError(ctx context.Context, ev *event.ErrorEvent)
}
