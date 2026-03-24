package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

// acquireSlot is a helper that calls AcquireAlertThrottleSlot and asserts it was allowed.
func acquireSlot(t *testing.T, repo interfaces.Repository, ctx context.Context, window time.Duration, limit int) {
	t.Helper()
	result, err := repo.AcquireAlertThrottleSlot(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)
}

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

func TestCheckAlertThrottle_BasicAllowAndDeny(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()
		window := 10 * time.Minute
		limit := 3

		// Check + consume for 3 slots
		for i := 0; i < limit; i++ {
			result, err := repo.CheckAlertThrottle(ctx, window, limit)
			gt.NoError(t, err)
			gt.Value(t, result.Allowed).Equal(true)
			acquireSlot(t, repo, ctx, window, limit)
		}

		// 4th check should be denied with notification
		result, err := repo.CheckAlertThrottle(ctx, window, limit)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(false)
		gt.Value(t, result.ShouldNotify).Equal(true)

		// 5th check should be denied WITHOUT notification (already notified)
		result, err = repo.CheckAlertThrottle(ctx, window, limit)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(false)
		gt.Value(t, result.ShouldNotify).Equal(false)
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})
}

func TestCheckAlertThrottle_ReadOnly(t *testing.T) {
	// CheckAlertThrottle should NOT consume slots
	repo := repository.NewMemory()
	window := 5 * time.Minute
	limit := 1

	ctx := t.Context()

	// Check many times — should always be allowed since no slots are consumed
	for i := 0; i < 10; i++ {
		result, err := repo.CheckAlertThrottle(ctx, window, limit)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)
	}

	// Now consume 1 slot
	acquireSlot(t, repo, ctx, window, limit)

	// Now check should be denied
	result, err := repo.CheckAlertThrottle(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
}

func TestCheckAlertThrottle_SlidingWindowExpiry(t *testing.T) {
	repo := repository.NewMemory()
	window := 5 * time.Minute
	limit := 2

	baseTime := time.Date(2026, 3, 24, 14, 0, 0, 0, time.UTC)

	// T=0: consume 2 slots
	ctx := clock.With(t.Context(), func() time.Time { return baseTime })
	for i := 0; i < limit; i++ {
		acquireSlot(t, repo, ctx, window, limit)
	}

	// T=0: should be denied now
	result, err := repo.CheckAlertThrottle(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)

	// T=3min: still denied (within window)
	ctx3 := clock.With(t.Context(), func() time.Time { return baseTime.Add(3 * time.Minute) })
	result, err = repo.CheckAlertThrottle(ctx3, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)

	// T=6min: old buckets expire, slots available again
	ctx6 := clock.With(t.Context(), func() time.Time { return baseTime.Add(6 * time.Minute) })
	result, err = repo.CheckAlertThrottle(ctx6, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)
}

func TestCheckAlertThrottle_SlidingWindowPartialExpiry(t *testing.T) {
	repo := repository.NewMemory()
	window := 5 * time.Minute
	limit := 3

	baseTime := time.Date(2026, 3, 24, 14, 0, 0, 0, time.UTC)

	// T=0min: consume 1 slot
	ctx0 := clock.With(t.Context(), func() time.Time { return baseTime })
	acquireSlot(t, repo, ctx0, window, limit)

	// T=2min: consume 1 slot (different bucket)
	ctx2 := clock.With(t.Context(), func() time.Time { return baseTime.Add(2 * time.Minute) })
	acquireSlot(t, repo, ctx2, window, limit)

	// T=4min: consume 1 slot (total = 3, at limit)
	ctx4 := clock.With(t.Context(), func() time.Time { return baseTime.Add(4 * time.Minute) })
	acquireSlot(t, repo, ctx4, window, limit)

	// T=4min: should be denied now (3/3 consumed)
	result, err := repo.CheckAlertThrottle(ctx4, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)

	// T=6min: T=0 bucket expired, so now 2/3 consumed, 1 slot available
	ctx6 := clock.With(t.Context(), func() time.Time { return baseTime.Add(6 * time.Minute) })
	result, err = repo.CheckAlertThrottle(ctx6, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// Consume 1 more at T=6min → 3/3 again
	acquireSlot(t, repo, ctx6, window, limit)
	result, err = repo.CheckAlertThrottle(ctx6, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)

	// T=8min: T=2 bucket expired, 2/3 consumed, 1 slot available
	ctx8 := clock.With(t.Context(), func() time.Time { return baseTime.Add(8 * time.Minute) })
	result, err = repo.CheckAlertThrottle(ctx8, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)
}

func TestCheckAlertThrottle_NotificationCooldown(t *testing.T) {
	repo := repository.NewMemory()
	window := 10 * time.Minute
	limit := 1

	baseTime := time.Date(2026, 3, 24, 14, 0, 0, 0, time.UTC)

	// T=0: consume the only slot
	ctx0 := clock.With(t.Context(), func() time.Time { return baseTime })
	acquireSlot(t, repo, ctx0, window, limit)

	// T=0: denied, first notification
	result, err := repo.CheckAlertThrottle(ctx0, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)

	// T=1min: denied, no notification (cooldown = window = 10min)
	ctx1 := clock.With(t.Context(), func() time.Time { return baseTime.Add(1 * time.Minute) })
	result, err = repo.CheckAlertThrottle(ctx1, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(false)

	// T=11min: old bucket expired → slot available, allowed
	ctx11 := clock.With(t.Context(), func() time.Time { return baseTime.Add(11 * time.Minute) })
	result, err = repo.CheckAlertThrottle(ctx11, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// Consume and deny again → should notify (>10min since last)
	acquireSlot(t, repo, ctx11, window, limit)
	result, err = repo.CheckAlertThrottle(ctx11, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)

	// Immediately after: denied, notification already sent
	result, err = repo.CheckAlertThrottle(ctx11, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(false)
}

func TestCheckAlertThrottle_HighLimit(t *testing.T) {
	repo := repository.NewMemory()
	window := 1 * time.Hour
	limit := 60

	ctx := t.Context()

	// Fill up all 60 slots
	for i := 0; i < 60; i++ {
		acquireSlot(t, repo, ctx, window, limit)
	}

	// Should be denied
	result, err := repo.CheckAlertThrottle(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)
}

func TestCheckAlertThrottle_ZeroLimit(t *testing.T) {
	repo := repository.NewMemory()
	window := 5 * time.Minute
	limit := 0

	ctx := t.Context()
	result, err := repo.CheckAlertThrottle(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)
}

func TestCheckAlertThrottle_IndependentWindows(t *testing.T) {
	repo := repository.NewMemory()
	window := 2 * time.Minute
	limit := 1

	baseTime := time.Date(2026, 3, 24, 14, 0, 0, 0, time.UTC)

	// Cycle through multiple windows, each allowing exactly 1 alert
	for i := 0; i < 5; i++ {
		ctxN := clock.With(t.Context(), func() time.Time {
			return baseTime.Add(time.Duration(i*3) * time.Minute)
		})
		result, err := repo.CheckAlertThrottle(ctxN, window, limit)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)

		acquireSlot(t, repo, ctxN, window, limit)

		// Second check in same window should be denied
		result, err = repo.CheckAlertThrottle(ctxN, window, limit)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(false)
	}
}

func TestAcquireAlertThrottleSlot_AtomicCheckAndConsume(t *testing.T) {
	// Verify AcquireAlertThrottleSlot atomically checks AND consumes in one call.
	// This ensures no race between check and consume.
	repo := repository.NewMemory()
	window := 10 * time.Minute
	limit := 2

	ctx := t.Context()

	// First acquire: allowed + slot consumed
	result, err := repo.AcquireAlertThrottleSlot(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// Second acquire: allowed + slot consumed
	result, err = repo.AcquireAlertThrottleSlot(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// Third acquire: denied (both slots consumed atomically)
	result, err = repo.AcquireAlertThrottleSlot(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)

	// Verify check also sees the consumed state
	checkResult, err := repo.CheckAlertThrottle(ctx, window, limit)
	gt.NoError(t, err)
	gt.Value(t, checkResult.Allowed).Equal(false)
}

func TestAcquireAlertThrottleSlot_MultipleAcquiresRespectLimit(t *testing.T) {
	// Simulates fan-out scenario: multiple acquires for one input.
	// With limit=3, acquiring 5 times should yield exactly 3 allowed + 2 denied.
	repo := repository.NewMemory()
	window := 10 * time.Minute
	limit := 3

	ctx := t.Context()

	allowed := 0
	denied := 0
	for i := 0; i < 5; i++ {
		result, err := repo.AcquireAlertThrottleSlot(ctx, window, limit)
		gt.NoError(t, err)
		if result.Allowed {
			allowed++
		} else {
			denied++
		}
	}

	gt.Value(t, allowed).Equal(3)
	gt.Value(t, denied).Equal(2)
}
