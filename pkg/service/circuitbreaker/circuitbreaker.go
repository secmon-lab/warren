package circuitbreaker

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// Config holds the circuit breaker configuration.
type Config struct {
	Enabled bool
	Window  time.Duration
	Limit   int
}

// Service provides alert throttling with a sliding window rate limiter.
type Service struct {
	config Config
	repo   interfaces.Repository
}

// New creates a new circuit breaker service.
func New(repo interfaces.Repository, config Config) *Service {
	return &Service{
		config: config,
		repo:   repo,
	}
}

// IsEnabled returns whether the circuit breaker is enabled.
func (s *Service) IsEnabled() bool {
	return s.config.Enabled
}

// TryAcquire attempts to acquire a throttle slot.
// Returns the throttle result indicating whether the alert can be processed.
func (s *Service) TryAcquire(ctx context.Context) (*alert.ThrottleResult, error) {
	if !s.config.Enabled {
		return &alert.ThrottleResult{Allowed: true, ShouldNotify: false}, nil
	}

	result, err := s.repo.AcquireAlertThrottleSlot(ctx, s.config.Window, s.config.Limit)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to acquire throttle slot")
	}

	return result, nil
}

// EnqueueAlert saves an alert to the queue when throttled.
func (s *Service) EnqueueAlert(ctx context.Context, schema types.AlertSchema, data any, title, channel string) (*alert.QueuedAlert, error) {
	qa := &alert.QueuedAlert{
		ID:        types.NewQueuedAlertID(),
		Schema:    schema,
		Data:      data,
		Title:     title,
		CreatedAt: clock.Now(ctx),
		Channel:   channel,
	}

	if err := s.repo.PutQueuedAlert(ctx, qa); err != nil {
		return nil, goerr.Wrap(err, "failed to enqueue alert")
	}

	logging.From(ctx).Info("alert queued by circuit breaker",
		"queued_alert_id", qa.ID,
		"schema", schema,
		"title", title)

	return qa, nil
}

// ReprocessAlert creates a background job and processes a queued alert.
// Returns the job immediately; processing happens in the provided callback.
func (s *Service) ReprocessAlert(ctx context.Context, queuedAlertID types.QueuedAlertID, processFunc func(ctx context.Context, schema types.AlertSchema, data any) error) (*alert.ReprocessJob, error) {
	qa, err := s.repo.GetQueuedAlert(ctx, queuedAlertID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get queued alert", goerr.V("id", queuedAlertID))
	}

	now := clock.Now(ctx)
	job := &alert.ReprocessJob{
		ID:            types.NewReprocessJobID(),
		QueuedAlertID: queuedAlertID,
		Status:        types.ReprocessJobStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.PutReprocessJob(ctx, job); err != nil {
		return nil, goerr.Wrap(err, "failed to create reprocess job")
	}

	// Run processing in background goroutine.
	// Use context.Background() intentionally: the HTTP request context may be cancelled
	// after the response is sent, but this background job must continue to completion.
	// Propagate the logger from the request context for observability.
	go func() {
		bgCtx := logging.With(context.Background(), logging.From(ctx))
		logger := logging.From(bgCtx)

		// Update status to running
		job.Status = types.ReprocessJobStatusRunning
		job.UpdatedAt = time.Now()
		if err := s.repo.PutReprocessJob(bgCtx, job); err != nil {
			logger.Error("failed to update reprocess job to running", "error", err, "job_id", job.ID)
			return
		}

		// Execute the reprocessing
		if err := processFunc(bgCtx, qa.Schema, qa.Data); err != nil {
			logger.Error("reprocess job failed", "error", err, "job_id", job.ID, "queued_alert_id", queuedAlertID)
			job.Status = types.ReprocessJobStatusFailed
			job.Error = err.Error()
			job.UpdatedAt = time.Now()
			if putErr := s.repo.PutReprocessJob(bgCtx, job); putErr != nil {
				logger.Error("failed to update reprocess job to failed", "error", putErr, "job_id", job.ID)
			}
			return
		}

		// Success: delete queued alert and update job
		if err := s.repo.DeleteQueuedAlerts(bgCtx, []types.QueuedAlertID{queuedAlertID}); err != nil {
			logger.Error("failed to delete queued alert after reprocess", "error", err, "queued_alert_id", queuedAlertID)
		}

		job.Status = types.ReprocessJobStatusCompleted
		job.UpdatedAt = time.Now()
		if err := s.repo.PutReprocessJob(bgCtx, job); err != nil {
			logger.Error("failed to update reprocess job to completed", "error", err, "job_id", job.ID)
		}

		logger.Info("reprocess job completed", "job_id", job.ID, "queued_alert_id", queuedAlertID)
	}()

	return job, nil
}

// DiscardAlerts removes queued alerts without processing.
func (s *Service) DiscardAlerts(ctx context.Context, ids []types.QueuedAlertID) error {
	if err := s.repo.DeleteQueuedAlerts(ctx, ids); err != nil {
		return goerr.Wrap(err, "failed to discard queued alerts")
	}

	logging.From(ctx).Info("queued alerts discarded", "count", len(ids))
	return nil
}
