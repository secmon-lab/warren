package event

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// NotificationEvent represents an event during alert pipeline processing
type NotificationEvent interface {
	isNotificationEvent()
}

// IngestPolicyResultEvent is fired when ingest policy evaluation completes
type IngestPolicyResultEvent struct {
	Schema types.AlertSchema
	Alerts []*alert.Alert
}

func (e *IngestPolicyResultEvent) isNotificationEvent() {}

// EnrichPolicyResultEvent is fired when enrich policy evaluation completes
type EnrichPolicyResultEvent struct {
	TaskCount int
	Policy    *policy.EnrichPolicyResult
}

func (e *EnrichPolicyResultEvent) isNotificationEvent() {}

// TriagePolicyResultEvent is fired when triage policy evaluation completes
type TriagePolicyResultEvent struct {
	Result *policy.TriagePolicyResult
}

func (e *TriagePolicyResultEvent) isNotificationEvent() {}

// EnrichTaskPromptEvent is fired when an enrich task prompt is prepared
type EnrichTaskPromptEvent struct {
	TaskID     string
	PromptText string
}

func (e *EnrichTaskPromptEvent) isNotificationEvent() {}

// EnrichTaskResponseEvent is fired when LLM responds to an enrich task
type EnrichTaskResponseEvent struct {
	TaskID   string
	Response any
}

func (e *EnrichTaskResponseEvent) isNotificationEvent() {}

// ErrorEvent is fired when an error occurs during pipeline processing
type ErrorEvent struct {
	TaskID  string // Optional: only set if error is related to a specific task
	Error   error
	Message string
}

func (e *ErrorEvent) isNotificationEvent() {}

// EventNotifier is a function type for receiving notification events
// Implementations should format events appropriately for their output destination
type EventNotifier func(ctx context.Context, event NotificationEvent)
