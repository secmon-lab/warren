package types

import "github.com/google/uuid"

// QueuedAlertID represents a unique identifier for a queued alert
type QueuedAlertID string

func (x QueuedAlertID) String() string {
	return string(x)
}

func NewQueuedAlertID() QueuedAlertID {
	return QueuedAlertID(uuid.New().String())
}

// ReprocessJobID represents a unique identifier for a reprocess job
type ReprocessJobID string

func (x ReprocessJobID) String() string {
	return string(x)
}

func NewReprocessJobID() ReprocessJobID {
	return ReprocessJobID(uuid.New().String())
}

// ReprocessJobStatus represents the status of a reprocess job
type ReprocessJobStatus string

const (
	ReprocessJobStatusPending   ReprocessJobStatus = "pending"
	ReprocessJobStatusRunning   ReprocessJobStatus = "running"
	ReprocessJobStatusCompleted ReprocessJobStatus = "completed"
	ReprocessJobStatusFailed    ReprocessJobStatus = "failed"
)

// ReprocessBatchJobID represents a unique identifier for a reprocess batch job
type ReprocessBatchJobID string

func (x ReprocessBatchJobID) String() string {
	return string(x)
}

func NewReprocessBatchJobID() ReprocessBatchJobID {
	return ReprocessBatchJobID(uuid.New().String())
}
