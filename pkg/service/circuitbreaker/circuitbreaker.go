package circuitbreaker

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/async"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
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

// CheckThrottle checks whether throttle slots are available (read-only).
// Does NOT consume a slot. Call ConsumeSlot after pipeline completion for non-discarded alerts.
func (s *Service) CheckThrottle(ctx context.Context) (*alert.ThrottleResult, error) {
	if !s.config.Enabled {
		return &alert.ThrottleResult{Allowed: true, ShouldNotify: false}, nil
	}

	result, err := s.repo.CheckAlertThrottle(ctx, s.config.Window, s.config.Limit)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to check throttle")
	}

	return result, nil
}

// AcquireSlot atomically checks and consumes a throttle slot.
// Used after pipeline completion for each non-discarded alert.
func (s *Service) AcquireSlot(ctx context.Context) (*alert.ThrottleResult, error) {
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

	async.Dispatch(ctx, func(bgCtx context.Context) error {
		logger := logging.From(bgCtx)

		// Update status to running
		job.Status = types.ReprocessJobStatusRunning
		job.UpdatedAt = time.Now()
		if err := s.repo.PutReprocessJob(bgCtx, job); err != nil {
			return goerr.Wrap(err, "failed to update reprocess job to running", goerr.V("job_id", job.ID))
		}

		// Execute the reprocessing
		if err := processFunc(bgCtx, qa.Schema, qa.Data); err != nil {
			errutil.Handle(bgCtx, goerr.Wrap(err, "reprocess job failed", goerr.V("job_id", job.ID), goerr.V("queued_alert_id", queuedAlertID)))
			job.Status = types.ReprocessJobStatusFailed
			job.Error = err.Error()
			job.UpdatedAt = time.Now()
			if putErr := s.repo.PutReprocessJob(bgCtx, job); putErr != nil {
				errutil.Handle(bgCtx, goerr.Wrap(putErr, "failed to update reprocess job to failed", goerr.V("job_id", job.ID)))
			}
			return nil
		}

		// Success: delete queued alert and update job
		if err := s.repo.DeleteQueuedAlerts(bgCtx, []types.QueuedAlertID{queuedAlertID}); err != nil {
			errutil.Handle(bgCtx, goerr.Wrap(err, "failed to delete queued alert after reprocess", goerr.V("queued_alert_id", queuedAlertID)))
		}

		job.Status = types.ReprocessJobStatusCompleted
		job.UpdatedAt = time.Now()
		if err := s.repo.PutReprocessJob(bgCtx, job); err != nil {
			errutil.Handle(bgCtx, goerr.Wrap(err, "failed to update reprocess job to completed", goerr.V("job_id", job.ID)))
		}

		logger.Info("reprocess job completed", "job_id", job.ID, "queued_alert_id", queuedAlertID)
		return nil
	})

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

// DiscardAlertsByFilter removes all queued alerts matching the keyword filter.
// If keyword is empty, all queued alerts are discarded.
func (s *Service) DiscardAlertsByFilter(ctx context.Context, keyword *string) (int, error) {
	ids, err := s.collectFilteredIDs(ctx, keyword)
	if err != nil {
		return 0, err
	}

	if len(ids) == 0 {
		return 0, nil
	}

	if err := s.repo.DeleteQueuedAlerts(ctx, ids); err != nil {
		return 0, goerr.Wrap(err, "failed to discard queued alerts by filter")
	}

	logging.From(ctx).Info("queued alerts discarded by filter", "count", len(ids))
	return len(ids), nil
}

// ReprocessAlertsByFilter creates a batch job that reprocesses all queued alerts matching the keyword filter.
func (s *Service) ReprocessAlertsByFilter(ctx context.Context, keyword *string, processFunc func(ctx context.Context, schema types.AlertSchema, data any) error) (*alert.ReprocessBatchJob, error) {
	alerts, err := s.collectFilteredAlerts(ctx, keyword)
	if err != nil {
		return nil, err
	}

	now := clock.Now(ctx)
	job := &alert.ReprocessBatchJob{
		ID:         types.NewReprocessBatchJobID(),
		Status:     types.ReprocessJobStatusPending,
		TotalCount: len(alerts),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.PutReprocessBatchJob(ctx, job); err != nil {
		return nil, goerr.Wrap(err, "failed to create reprocess batch job")
	}

	async.Dispatch(ctx, func(bgCtx context.Context) error {
		logger := logging.From(bgCtx)

		job.Status = types.ReprocessJobStatusRunning
		job.UpdatedAt = time.Now()
		if err := s.repo.PutReprocessBatchJob(bgCtx, job); err != nil {
			return goerr.Wrap(err, "failed to update batch job to running")
		}

		for _, qa := range alerts {
			if err := processFunc(bgCtx, qa.Schema, qa.Data); err != nil {
				errutil.Handle(bgCtx, goerr.Wrap(err, "reprocess failed for queued alert",
					goerr.V("queued_alert_id", qa.ID)))
				job.FailedCount++
			} else {
				if err := s.repo.DeleteQueuedAlerts(bgCtx, []types.QueuedAlertID{qa.ID}); err != nil {
					errutil.Handle(bgCtx, goerr.Wrap(err, "failed to delete queued alert after reprocess",
						goerr.V("queued_alert_id", qa.ID)))
				}
				job.CompletedCount++
			}

			job.UpdatedAt = time.Now()
			if err := s.repo.PutReprocessBatchJob(bgCtx, job); err != nil {
				errutil.Handle(bgCtx, goerr.Wrap(err, "failed to update batch job progress"))
			}
		}

		job.Status = types.ReprocessJobStatusCompleted
		job.UpdatedAt = time.Now()
		if err := s.repo.PutReprocessBatchJob(bgCtx, job); err != nil {
			errutil.Handle(bgCtx, goerr.Wrap(err, "failed to update batch job to completed"))
		}

		logger.Info("reprocess batch job completed",
			"job_id", job.ID,
			"total", job.TotalCount,
			"completed", job.CompletedCount,
			"failed", job.FailedCount)
		return nil
	})

	return job, nil
}

// collectFilteredIDs returns IDs of all queued alerts matching the keyword filter.
func (s *Service) collectFilteredIDs(ctx context.Context, keyword *string) ([]types.QueuedAlertID, error) {
	alerts, err := s.collectFilteredAlerts(ctx, keyword)
	if err != nil {
		return nil, err
	}

	ids := make([]types.QueuedAlertID, len(alerts))
	for i, qa := range alerts {
		ids[i] = qa.ID
	}
	return ids, nil
}

// collectFilteredAlerts returns all queued alerts matching the keyword filter.
func (s *Service) collectFilteredAlerts(ctx context.Context, keyword *string) ([]*alert.QueuedAlert, error) {
	if keyword != nil && *keyword != "" {
		// Use search to filter by keyword (fetch all matching, no pagination)
		alerts, _, err := s.repo.SearchQueuedAlerts(ctx, *keyword, 0, 0)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to search queued alerts")
		}
		return alerts, nil
	}

	// No keyword: fetch all
	alerts, err := s.repo.ListQueuedAlerts(ctx, 0, 0)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list queued alerts")
	}
	return alerts, nil
}
