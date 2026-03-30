package alert

import (
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

// ReprocessBatchJob represents a background job for batch reprocessing of queued alerts.
// It tracks progress of reprocessing multiple alerts at once.
type ReprocessBatchJob struct {
	ID             types.ReprocessBatchJobID `json:"id"`
	Status         types.ReprocessJobStatus  `json:"status"`
	TotalCount     int                       `json:"total_count"`
	CompletedCount int                       `json:"completed_count"`
	FailedCount    int                       `json:"failed_count"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
}
