package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/event"
)

// Notifier is an interface for handling notification events from the alert processing pipeline.
// Each event type has a dedicated method for type-safe event handling.
// Implementations can output events to console, Slack, or other notification channels.
type Notifier interface {
	// NotifyAlertPolicyResult is called when alert policy evaluation completes
	NotifyAlertPolicyResult(ctx context.Context, ev *event.AlertPolicyResultEvent)

	// NotifyEnrichPolicyResult is called when enrich policy evaluation completes
	NotifyEnrichPolicyResult(ctx context.Context, ev *event.EnrichPolicyResultEvent)

	// NotifyCommitPolicyResult is called when commit policy evaluation completes
	NotifyCommitPolicyResult(ctx context.Context, ev *event.CommitPolicyResultEvent)

	// NotifyEnrichTaskPrompt is called when an enrichment task prompt is about to be sent to LLM
	NotifyEnrichTaskPrompt(ctx context.Context, ev *event.EnrichTaskPromptEvent)

	// NotifyEnrichTaskResponse is called when an enrichment task response is received from LLM
	NotifyEnrichTaskResponse(ctx context.Context, ev *event.EnrichTaskResponseEvent)

	// NotifyError is called when an error occurs during pipeline processing
	NotifyError(ctx context.Context, ev *event.ErrorEvent)
}
