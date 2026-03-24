package repository_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func TestQueuedAlertCRUD(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		now := time.Now()

		qa1 := &alert.QueuedAlert{
			ID:        types.NewQueuedAlertID(),
			Schema:    "test.schema",
			Data:      map[string]any{"key": "value1"},
			Title:     "Test Alert 1",
			CreatedAt: now,
			Channel:   "C123",
		}
		qa2 := &alert.QueuedAlert{
			ID:        types.NewQueuedAlertID(),
			Schema:    "test.schema",
			Data:      map[string]any{"key": "value2"},
			Title:     "Test Alert 2",
			CreatedAt: now.Add(time.Second),
			Channel:   "C123",
		}
		qa3 := &alert.QueuedAlert{
			ID:        types.NewQueuedAlertID(),
			Schema:    "other.schema",
			Data:      map[string]any{"key": "value3"},
			Title:     "Another Alert",
			CreatedAt: now.Add(2 * time.Second),
			Channel:   "C456",
		}

		// Put
		gt.NoError(t, repo.PutQueuedAlert(ctx, qa1))
		gt.NoError(t, repo.PutQueuedAlert(ctx, qa2))
		gt.NoError(t, repo.PutQueuedAlert(ctx, qa3))

		// Get
		got, err := repo.GetQueuedAlert(ctx, qa1.ID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(qa1.ID)
		gt.Value(t, got.Schema).Equal(qa1.Schema)
		gt.Value(t, got.Title).Equal(qa1.Title)
		gt.Value(t, got.Channel).Equal(qa1.Channel)

		// Get non-existent
		_, err = repo.GetQueuedAlert(ctx, types.QueuedAlertID("non-existent"))
		gt.Error(t, err)

		// Count
		count, err := repo.CountQueuedAlerts(ctx)
		gt.NoError(t, err)
		gt.Value(t, count).Equal(3)

		// List with pagination (FIFO order)
		listed, err := repo.ListQueuedAlerts(ctx, 0, 2)
		gt.NoError(t, err)
		gt.A(t, listed).Length(2)
		gt.Value(t, listed[0].ID).Equal(qa1.ID) // oldest first
		gt.Value(t, listed[1].ID).Equal(qa2.ID)

		listed, err = repo.ListQueuedAlerts(ctx, 2, 10)
		gt.NoError(t, err)
		gt.A(t, listed).Length(1)
		gt.Value(t, listed[0].ID).Equal(qa3.ID)

		// Delete
		gt.NoError(t, repo.DeleteQueuedAlerts(ctx, []types.QueuedAlertID{qa1.ID, qa2.ID}))

		count, err = repo.CountQueuedAlerts(ctx)
		gt.NoError(t, err)
		gt.Value(t, count).Equal(1)

		// Remaining is qa3
		got, err = repo.GetQueuedAlert(ctx, qa3.ID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(qa3.ID)
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})
}

func TestQueuedAlertSearch(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		now := time.Now()

		qa1 := &alert.QueuedAlert{
			ID:        types.NewQueuedAlertID(),
			Schema:    "guardduty",
			Data:      map[string]any{"source_ip": "192.168.1.1"},
			Title:     "GuardDuty Finding: SSH Brute Force",
			CreatedAt: now,
		}
		qa2 := &alert.QueuedAlert{
			ID:        types.NewQueuedAlertID(),
			Schema:    "cloudtrail",
			Data:      map[string]any{"event": "DeleteBucket"},
			Title:     "CloudTrail: S3 Bucket Deleted",
			CreatedAt: now.Add(time.Second),
		}
		qa3 := &alert.QueuedAlert{
			ID:        types.NewQueuedAlertID(),
			Schema:    "guardduty",
			Data:      map[string]any{"source_ip": "10.0.0.1"},
			Title:     "GuardDuty Finding: Port Scan",
			CreatedAt: now.Add(2 * time.Second),
		}

		gt.NoError(t, repo.PutQueuedAlert(ctx, qa1))
		gt.NoError(t, repo.PutQueuedAlert(ctx, qa2))
		gt.NoError(t, repo.PutQueuedAlert(ctx, qa3))

		// Search by title keyword
		results, total, err := repo.SearchQueuedAlerts(ctx, "guardduty", 0, 10)
		gt.NoError(t, err)
		gt.Value(t, total).Equal(2)
		gt.A(t, results).Length(2)
		gt.Value(t, results[0].ID).Equal(qa1.ID)
		gt.Value(t, results[1].ID).Equal(qa3.ID)

		// Search by data content
		results, total, err = repo.SearchQueuedAlerts(ctx, "192.168", 0, 10)
		gt.NoError(t, err)
		gt.Value(t, total).Equal(1)
		gt.A(t, results).Length(1)
		gt.Value(t, results[0].ID).Equal(qa1.ID)

		// Search case-insensitive
		results, total, err = repo.SearchQueuedAlerts(ctx, "SSH brute", 0, 10)
		gt.NoError(t, err)
		gt.Value(t, total).Equal(1)
		gt.Value(t, results[0].ID).Equal(qa1.ID)

		// Search with pagination
		results, total, err = repo.SearchQueuedAlerts(ctx, "guardduty", 0, 1)
		gt.NoError(t, err)
		gt.Value(t, total).Equal(2)
		gt.A(t, results).Length(1)
		gt.Value(t, results[0].ID).Equal(qa1.ID)

		results, total, err = repo.SearchQueuedAlerts(ctx, "guardduty", 1, 1)
		gt.NoError(t, err)
		gt.Value(t, total).Equal(2)
		gt.A(t, results).Length(1)
		gt.Value(t, results[0].ID).Equal(qa3.ID)

		// Search no match
		results, total, err = repo.SearchQueuedAlerts(ctx, "nonexistent", 0, 10)
		gt.NoError(t, err)
		gt.Value(t, total).Equal(0)
		gt.A(t, results).Length(0)
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})
}

func TestReprocessJobCRUD(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		now := time.Now()

		job := &alert.ReprocessJob{
			ID:            types.NewReprocessJobID(),
			QueuedAlertID: types.NewQueuedAlertID(),
			Status:        types.ReprocessJobStatusPending,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		// Put
		gt.NoError(t, repo.PutReprocessJob(ctx, job))

		// Get
		got, err := repo.GetReprocessJob(ctx, job.ID)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(job.ID)
		gt.Value(t, got.QueuedAlertID).Equal(job.QueuedAlertID)
		gt.Value(t, got.Status).Equal(types.ReprocessJobStatusPending)
		gt.Value(t, got.Error).Equal("")

		// Update status
		job.Status = types.ReprocessJobStatusRunning
		job.UpdatedAt = now.Add(time.Second)
		gt.NoError(t, repo.PutReprocessJob(ctx, job))

		got, err = repo.GetReprocessJob(ctx, job.ID)
		gt.NoError(t, err)
		gt.Value(t, got.Status).Equal(types.ReprocessJobStatusRunning)

		// Update to failed with error
		job.Status = types.ReprocessJobStatusFailed
		job.Error = "processing failed: timeout"
		job.UpdatedAt = now.Add(2 * time.Second)
		gt.NoError(t, repo.PutReprocessJob(ctx, job))

		got, err = repo.GetReprocessJob(ctx, job.ID)
		gt.NoError(t, err)
		gt.Value(t, got.Status).Equal(types.ReprocessJobStatusFailed)
		gt.Value(t, got.Error).Equal("processing failed: timeout")

		// Get non-existent
		_, err = repo.GetReprocessJob(ctx, types.ReprocessJobID("non-existent"))
		gt.Error(t, err)
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})
}

func TestAcquireAlertThrottleSlot_BasicAllowAndDeny(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		window := 10 * time.Minute
		limit := 3

		// First 3 requests should be allowed
		for i := 0; i < limit; i++ {
			result, err := repo.AcquireAlertThrottleSlot(ctx, window, limit)
			gt.NoError(t, err)
			gt.Value(t, result.Allowed).Equal(true)
			gt.Value(t, result.ShouldNotify).Equal(false)
		}

		// 4th request should be denied with notification
		result, err := repo.AcquireAlertThrottleSlot(ctx, window, limit)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(false)
		gt.Value(t, result.ShouldNotify).Equal(true)

		// 5th request should be denied WITHOUT notification (already notified)
		result, err = repo.AcquireAlertThrottleSlot(ctx, window, limit)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(false)
		gt.Value(t, result.ShouldNotify).Equal(false)
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})
}

func TestAcquireAlertThrottleSlot_SlidingWindowExpiry(t *testing.T) {
	repo := repository.NewMemory()
	window := 5 * time.Minute
	limit := 2

	baseTime := time.Date(2026, 3, 24, 14, 0, 0, 0, time.UTC)

	// T=0: consume 2 slots
	ctx := clock.With(t.Context(), func() time.Time { return baseTime })
	for i := 0; i < limit; i++ {
		result, err := repo.AcquireAlertThrottleSlot(ctx, window, limit)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)
	}

	// T=0: should be denied now
	result, err := repo.AcquireAlertThrottleSlot(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)

	// T=3min: still denied (within window)
	ctx3 := clock.With(t.Context(), func() time.Time { return baseTime.Add(3 * time.Minute) })
	result, err = repo.AcquireAlertThrottleSlot(ctx3, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)

	// T=6min: old buckets expire, slots available again
	ctx6 := clock.With(t.Context(), func() time.Time { return baseTime.Add(6 * time.Minute) })
	result, err = repo.AcquireAlertThrottleSlot(ctx6, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)
}

func TestAcquireAlertThrottleSlot_SlidingWindowPartialExpiry(t *testing.T) {
	repo := repository.NewMemory()
	window := 5 * time.Minute
	limit := 3

	baseTime := time.Date(2026, 3, 24, 14, 0, 0, 0, time.UTC)

	// T=0min: consume 1 slot
	ctx0 := clock.With(t.Context(), func() time.Time { return baseTime })
	result, err := repo.AcquireAlertThrottleSlot(ctx0, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// T=2min: consume 1 slot (different bucket)
	ctx2 := clock.With(t.Context(), func() time.Time { return baseTime.Add(2 * time.Minute) })
	result, err = repo.AcquireAlertThrottleSlot(ctx2, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// T=4min: consume 1 slot (total = 3, at limit)
	ctx4 := clock.With(t.Context(), func() time.Time { return baseTime.Add(4 * time.Minute) })
	result, err = repo.AcquireAlertThrottleSlot(ctx4, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// T=4min: should be denied now (3/3 consumed)
	result, err = repo.AcquireAlertThrottleSlot(ctx4, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)

	// T=6min: T=0 bucket expired, so now 2/3 consumed, 1 slot available
	ctx6 := clock.With(t.Context(), func() time.Time { return baseTime.Add(6 * time.Minute) })
	result, err = repo.AcquireAlertThrottleSlot(ctx6, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// T=6min: now 3/3 again, denied
	result, err = repo.AcquireAlertThrottleSlot(ctx6, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)

	// T=8min: T=2 bucket expired, 2/3 consumed, 1 slot available
	ctx8 := clock.With(t.Context(), func() time.Time { return baseTime.Add(8 * time.Minute) })
	result, err = repo.AcquireAlertThrottleSlot(ctx8, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)
}

func TestAcquireAlertThrottleSlot_NotificationCooldown(t *testing.T) {
	repo := repository.NewMemory()
	window := 10 * time.Minute
	limit := 1

	baseTime := time.Date(2026, 3, 24, 14, 0, 0, 0, time.UTC)

	// T=0: consume the only slot
	ctx0 := clock.With(t.Context(), func() time.Time { return baseTime })
	result, err := repo.AcquireAlertThrottleSlot(ctx0, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// T=0: denied, first notification
	result, err = repo.AcquireAlertThrottleSlot(ctx0, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)

	// T=1min: denied, no notification (cooldown = window = 10min)
	ctx1 := clock.With(t.Context(), func() time.Time { return baseTime.Add(1 * time.Minute) })
	result, err = repo.AcquireAlertThrottleSlot(ctx1, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(false)

	// T=5min: denied, still no notification (5min < 10min cooldown)
	ctx5 := clock.With(t.Context(), func() time.Time { return baseTime.Add(5 * time.Minute) })
	result, err = repo.AcquireAlertThrottleSlot(ctx5, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(false)

	// T=11min: old bucket expired → slot available, allowed
	ctx11 := clock.With(t.Context(), func() time.Time { return baseTime.Add(11 * time.Minute) })
	result, err = repo.AcquireAlertThrottleSlot(ctx11, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// T=11min: consume the slot, denied again → should notify (>10min since last)
	result, err = repo.AcquireAlertThrottleSlot(ctx11, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)

	// T=11min: denied, but notification already sent
	result, err = repo.AcquireAlertThrottleSlot(ctx11, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(false)
}

func TestAcquireAlertThrottleSlot_HighLimit(t *testing.T) {
	repo := repository.NewMemory()
	window := 1 * time.Hour
	limit := 60

	ctx := t.Context()

	// Fill up all 60 slots
	for i := 0; i < 60; i++ {
		result, err := repo.AcquireAlertThrottleSlot(ctx, window, limit)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)
	}

	// 61st should be denied
	result, err := repo.AcquireAlertThrottleSlot(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)
}

func TestAcquireAlertThrottleSlot_BucketGranularity(t *testing.T) {
	// Verify that requests within the same minute go to the same bucket
	repo := repository.NewMemory()
	window := 5 * time.Minute
	limit := 2

	baseTime := time.Date(2026, 3, 24, 14, 0, 30, 0, time.UTC) // 14:00:30

	// Two requests within the same minute should go to same bucket
	ctx1 := clock.With(t.Context(), func() time.Time { return baseTime })
	result, err := repo.AcquireAlertThrottleSlot(ctx1, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	ctx2 := clock.With(t.Context(), func() time.Time { return baseTime.Add(20 * time.Second) }) // still 14:00
	result, err = repo.AcquireAlertThrottleSlot(ctx2, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// 3rd request in same minute — denied (limit=2)
	result, err = repo.AcquireAlertThrottleSlot(ctx2, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
}

func TestAcquireAlertThrottleSlot_ZeroLimit(t *testing.T) {
	// With limit=0, no requests should be allowed
	repo := repository.NewMemory()
	window := 5 * time.Minute
	limit := 0

	ctx := t.Context()
	result, err := repo.AcquireAlertThrottleSlot(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)
}

func TestAcquireAlertThrottleSlot_IndependentWindows(t *testing.T) {
	// Test that different time windows work correctly over longer periods
	repo := repository.NewMemory()
	window := 2 * time.Minute
	limit := 1

	baseTime := time.Date(2026, 3, 24, 14, 0, 0, 0, time.UTC)

	// Cycle through multiple windows, each allowing exactly 1 alert
	for i := 0; i < 5; i++ {
		ctxN := clock.With(t.Context(), func() time.Time {
			return baseTime.Add(time.Duration(i*3) * time.Minute)
		})
		result, err := repo.AcquireAlertThrottleSlot(ctxN, window, limit)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)

		// Second request in same window should be denied
		result, err = repo.AcquireAlertThrottleSlot(ctxN, window, limit)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(false)
	}
}
