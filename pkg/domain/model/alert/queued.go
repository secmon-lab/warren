package alert

import (
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

// QueuedAlert represents an alert that has been queued due to circuit breaker throttling.
// It exists in the queue until it is either reprocessed or discarded (then deleted).
type QueuedAlert struct {
	ID        types.QueuedAlertID `json:"id"`
	Schema    types.AlertSchema   `json:"schema"`
	Data      any                 `json:"data"`
	Title     string              `json:"title"`
	CreatedAt time.Time           `json:"created_at"`
	Channel   string              `json:"channel"`
}

// ReprocessJob represents a background job for reprocessing a queued alert.
type ReprocessJob struct {
	ID             types.ReprocessJobID     `json:"id"`
	QueuedAlertID  types.QueuedAlertID      `json:"queued_alert_id"`
	Status         types.ReprocessJobStatus `json:"status"`
	Error          string                   `json:"error,omitempty"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

// AlertThrottle holds the sliding window state for alert rate limiting.
// Stored as a singleton document in Firestore at throttle/alert.
type AlertThrottle struct {
	// Buckets maps time bucket keys (e.g. "2026-03-24T14:05") to the count of alerts processed in that bucket.
	Buckets    map[string]int `json:"buckets"`
	NotifiedAt time.Time      `json:"notified_at"`
}

// ThrottleResult represents the result of a throttle slot acquisition attempt.
type ThrottleResult struct {
	Allowed      bool // true if the alert can be processed
	ShouldNotify bool // true if a Slack @channel notification should be sent
}
