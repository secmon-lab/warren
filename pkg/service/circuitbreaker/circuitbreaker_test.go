package circuitbreaker_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/circuitbreaker"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func newTestService(limit int) (*circuitbreaker.Service, *repository.Memory) {
	repo := repository.NewMemory()
	svc := circuitbreaker.New(repo, circuitbreaker.Config{
		Enabled: true,
		Window:  10 * time.Minute,
		Limit:   limit,
	})
	return svc, repo
}

// acquireSlot is a helper that calls AcquireSlot and asserts success
func acquireSlot(t *testing.T, svc *circuitbreaker.Service, ctx context.Context) {
	t.Helper()
	result, err := svc.AcquireSlot(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)
}

func TestService_CheckThrottle_Disabled(t *testing.T) {
	repo := repository.NewMemory()
	svc := circuitbreaker.New(repo, circuitbreaker.Config{
		Enabled: false,
		Window:  10 * time.Minute,
		Limit:   1,
	})

	ctx := t.Context()

	for i := 0; i < 100; i++ {
		result, err := svc.CheckThrottle(ctx)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)
		gt.Value(t, result.ShouldNotify).Equal(false)
	}
}

func TestService_CheckThrottle_Enabled(t *testing.T) {
	svc, _ := newTestService(3)
	ctx := t.Context()

	for i := 0; i < 3; i++ {
		result, err := svc.CheckThrottle(ctx)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)
		acquireSlot(t, svc, ctx)
	}

	result, err := svc.CheckThrottle(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
}

func TestService_CheckThrottle_DoesNotConsumeSlot(t *testing.T) {
	svc, _ := newTestService(2)
	ctx := t.Context()

	// Check many times without acquiring — should always be allowed
	for i := 0; i < 10; i++ {
		result, err := svc.CheckThrottle(ctx)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)
	}

	// Now acquire 2 slots atomically
	acquireSlot(t, svc, ctx)
	acquireSlot(t, svc, ctx)

	// Now check should be denied
	result, err := svc.CheckThrottle(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
}

func TestService_AcquireSlot_Disabled(t *testing.T) {
	repo := repository.NewMemory()
	svc := circuitbreaker.New(repo, circuitbreaker.Config{
		Enabled: false,
		Window:  10 * time.Minute,
		Limit:   1,
	})

	// AcquireSlot should always return Allowed when disabled
	result, err := svc.AcquireSlot(t.Context())
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)
}

func TestService_AcquireSlot_Atomic(t *testing.T) {
	// Verify AcquireSlot atomically checks AND consumes
	svc, _ := newTestService(2)
	ctx := t.Context()

	// First 2 acquires should succeed
	result, err := svc.AcquireSlot(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	result, err = svc.AcquireSlot(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// 3rd should fail — slots consumed atomically
	result, err = svc.AcquireSlot(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)
}

func TestService_IsEnabled(t *testing.T) {
	repo := repository.NewMemory()

	enabled := circuitbreaker.New(repo, circuitbreaker.Config{Enabled: true, Window: time.Minute, Limit: 1})
	gt.Value(t, enabled.IsEnabled()).Equal(true)

	disabled := circuitbreaker.New(repo, circuitbreaker.Config{Enabled: false, Window: time.Minute, Limit: 1})
	gt.Value(t, disabled.IsEnabled()).Equal(false)
}

func TestService_EnqueueAlert(t *testing.T) {
	svc, repo := newTestService(1)
	ctx := t.Context()

	qa, err := svc.EnqueueAlert(ctx, "test.schema", map[string]any{"key": "val"}, "Test Title", "C123")
	gt.NoError(t, err)
	gt.Value(t, qa.Schema).Equal(types.AlertSchema("test.schema"))
	gt.Value(t, qa.Title).Equal("Test Title")
	gt.Value(t, qa.Channel).Equal("C123")

	got, err := repo.GetQueuedAlert(ctx, qa.ID)
	gt.NoError(t, err)
	gt.Value(t, got.ID).Equal(qa.ID)
	gt.Value(t, got.Title).Equal("Test Title")
}

func TestService_DiscardAlerts(t *testing.T) {
	svc, repo := newTestService(1)
	ctx := t.Context()

	qa1, err := svc.EnqueueAlert(ctx, "s1", nil, "t1", "")
	gt.NoError(t, err)
	qa2, err := svc.EnqueueAlert(ctx, "s2", nil, "t2", "")
	gt.NoError(t, err)

	count, err := repo.CountQueuedAlerts(ctx)
	gt.NoError(t, err)
	gt.Value(t, count).Equal(2)

	gt.NoError(t, svc.DiscardAlerts(ctx, []types.QueuedAlertID{qa1.ID}))

	count, err = repo.CountQueuedAlerts(ctx)
	gt.NoError(t, err)
	gt.Value(t, count).Equal(1)

	got, err := repo.GetQueuedAlert(ctx, qa2.ID)
	gt.NoError(t, err)
	gt.Value(t, got.ID).Equal(qa2.ID)
}

func TestService_ReprocessAlert(t *testing.T) {
	svc, repo := newTestService(1)
	ctx := t.Context()

	qa, err := svc.EnqueueAlert(ctx, "test.schema", map[string]any{"data": "value"}, "Title", "")
	gt.NoError(t, err)

	processed := make(chan bool, 1)
	job, err := svc.ReprocessAlert(ctx, qa.ID, func(bgCtx context.Context, schema types.AlertSchema, data any) error {
		gt.Value(t, schema).Equal(types.AlertSchema("test.schema"))
		processed <- true
		return nil
	})
	gt.NoError(t, err)
	gt.Value(t, job.Status).Equal(types.ReprocessJobStatusPending)

	select {
	case <-processed:
	case <-time.After(5 * time.Second):
		t.Fatal("reprocess callback not called within timeout")
	}

	time.Sleep(100 * time.Millisecond)

	gotJob, err := repo.GetReprocessJob(ctx, job.ID)
	gt.NoError(t, err)
	gt.Value(t, gotJob.Status).Equal(types.ReprocessJobStatusCompleted)
	gt.Value(t, gotJob.Error).Equal("")

	count, err := repo.CountQueuedAlerts(ctx)
	gt.NoError(t, err)
	gt.Value(t, count).Equal(0)
}

func TestService_ReprocessAlert_Failure(t *testing.T) {
	svc, repo := newTestService(1)
	ctx := t.Context()

	qa, err := svc.EnqueueAlert(ctx, "test.schema", nil, "Title", "")
	gt.NoError(t, err)

	failed := make(chan bool, 1)
	job, err := svc.ReprocessAlert(ctx, qa.ID, func(bgCtx context.Context, schema types.AlertSchema, data any) error {
		failed <- true
		return context.DeadlineExceeded
	})
	gt.NoError(t, err)

	select {
	case <-failed:
	case <-time.After(5 * time.Second):
		t.Fatal("reprocess callback not called within timeout")
	}

	time.Sleep(100 * time.Millisecond)

	gotJob, err := repo.GetReprocessJob(ctx, job.ID)
	gt.NoError(t, err)
	gt.Value(t, gotJob.Status).Equal(types.ReprocessJobStatusFailed)
	gt.True(t, gotJob.Error != "")

	count, err := repo.CountQueuedAlerts(ctx)
	gt.NoError(t, err)
	gt.Value(t, count).Equal(1)
}

func TestService_ReprocessAlert_NonExistent(t *testing.T) {
	svc, _ := newTestService(1)
	ctx := t.Context()

	_, err := svc.ReprocessAlert(ctx, types.QueuedAlertID("non-existent"), func(bgCtx context.Context, schema types.AlertSchema, data any) error {
		return nil
	})
	gt.Error(t, err)
}

func TestService_EndToEnd_CheckThenAcquire(t *testing.T) {
	svc, repo := newTestService(2)

	baseTime := time.Date(2026, 3, 24, 14, 0, 0, 0, time.UTC)
	ctx := clock.With(t.Context(), func() time.Time { return baseTime })

	// Check + acquire for 2 alerts
	for i := 0; i < 2; i++ {
		result, err := svc.CheckThrottle(ctx)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)
		acquireSlot(t, svc, ctx)
	}

	// 3rd: check denied → enqueue
	result, err := svc.CheckThrottle(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)

	qa, err := svc.EnqueueAlert(ctx, "guardduty", map[string]any{"alert": true}, "Throttled Alert", "C123")
	gt.NoError(t, err)

	count, err := repo.CountQueuedAlerts(ctx)
	gt.NoError(t, err)
	gt.Value(t, count).Equal(1)

	// After window expires, slots available again
	ctx11 := clock.With(t.Context(), func() time.Time { return baseTime.Add(11 * time.Minute) })
	result, err = svc.CheckThrottle(ctx11)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	gt.NoError(t, svc.DiscardAlerts(ctx, []types.QueuedAlertID{qa.ID}))
	count, err = repo.CountQueuedAlerts(ctx)
	gt.NoError(t, err)
	gt.Value(t, count).Equal(0)
}

func TestService_DiscardedAlerts_DontConsumeSlots(t *testing.T) {
	svc, _ := newTestService(2)
	ctx := t.Context()

	// Check 5 times without acquiring (simulating 5 discarded alerts)
	for i := 0; i < 5; i++ {
		result, err := svc.CheckThrottle(ctx)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)
	}

	// Slots should still be available since nothing was acquired
	result, err := svc.CheckThrottle(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// Now acquire 2 slots (non-discarded alerts)
	acquireSlot(t, svc, ctx)
	acquireSlot(t, svc, ctx)

	// Now should be throttled
	result, err = svc.CheckThrottle(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
}

func TestService_AcquireSlot_AtomicUnderFanOut(t *testing.T) {
	// Issue #2: One input can fan out to multiple alerts via ingest policy.
	// Each non-discarded alert must individually acquire a slot.
	// With limit=2 and 4 acquires, exactly 2 should succeed.
	svc, _ := newTestService(2)
	ctx := t.Context()

	allowed := 0
	denied := 0
	for i := 0; i < 4; i++ {
		result, err := svc.AcquireSlot(ctx)
		gt.NoError(t, err)
		if result.Allowed {
			allowed++
		} else {
			denied++
		}
	}

	gt.Value(t, allowed).Equal(2)
	gt.Value(t, denied).Equal(2)
}

func TestService_AcquireSlot_DeniedReturnsNotifyFlag(t *testing.T) {
	// When acquire is denied, the result includes ShouldNotify for Slack @channel.
	svc, _ := newTestService(1)
	ctx := t.Context()

	// Consume the only slot
	acquireSlot(t, svc, ctx)

	// Next acquire: denied + first notification
	result, err := svc.AcquireSlot(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)

	// Next acquire: denied but no notification (cooldown)
	result, err = svc.AcquireSlot(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(false)
}

func TestService_AcquireSlot_QueueOnDeny(t *testing.T) {
	// Issue #3: When AcquireSlot is denied after pipeline, the alert should be queued.
	// This tests the pattern used by tryAcquireOrQueue in usecase.
	svc, repo := newTestService(1)
	ctx := t.Context()

	// Consume the only slot
	acquireSlot(t, svc, ctx)

	// Simulate pipeline completed for 2nd alert, now try to acquire
	result, err := svc.AcquireSlot(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)

	// Queue the alert since acquire was denied
	qa, err := svc.EnqueueAlert(ctx, "test.schema", map[string]any{"key": "val"}, "Post-Pipeline Alert", "C123")
	gt.NoError(t, err)

	// Verify it's in the queue with title from pipeline
	got, err := repo.GetQueuedAlert(ctx, qa.ID)
	gt.NoError(t, err)
	gt.Value(t, got.Title).Equal("Post-Pipeline Alert")

	count, err := repo.CountQueuedAlerts(ctx)
	gt.NoError(t, err)
	gt.Value(t, count).Equal(1)
}
