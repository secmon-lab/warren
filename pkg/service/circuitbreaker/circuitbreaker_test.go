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

func TestService_TryAcquire_Disabled(t *testing.T) {
	repo := repository.NewMemory()
	svc := circuitbreaker.New(repo, circuitbreaker.Config{
		Enabled: false,
		Window:  10 * time.Minute,
		Limit:   1,
	})

	ctx := t.Context()

	// Even with limit=1, disabled circuit breaker always allows
	for i := 0; i < 100; i++ {
		result, err := svc.TryAcquire(ctx)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)
		gt.Value(t, result.ShouldNotify).Equal(false)
	}
}

func TestService_TryAcquire_Enabled(t *testing.T) {
	svc, _ := newTestService(3)
	ctx := t.Context()

	// First 3 should be allowed
	for i := 0; i < 3; i++ {
		result, err := svc.TryAcquire(ctx)
		gt.NoError(t, err)
		gt.Value(t, result.Allowed).Equal(true)
	}

	// 4th should be denied
	result, err := svc.TryAcquire(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
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

	// Verify it's in the repository
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

	// Discard one
	gt.NoError(t, svc.DiscardAlerts(ctx, []types.QueuedAlertID{qa1.ID}))

	count, err = repo.CountQueuedAlerts(ctx)
	gt.NoError(t, err)
	gt.Value(t, count).Equal(1)

	// Remaining is qa2
	got, err := repo.GetQueuedAlert(ctx, qa2.ID)
	gt.NoError(t, err)
	gt.Value(t, got.ID).Equal(qa2.ID)
}

func TestService_ReprocessAlert(t *testing.T) {
	svc, repo := newTestService(1)
	ctx := t.Context()

	// Enqueue an alert
	qa, err := svc.EnqueueAlert(ctx, "test.schema", map[string]any{"data": "value"}, "Title", "")
	gt.NoError(t, err)

	// Reprocess with a successful callback
	processed := make(chan bool, 1)
	job, err := svc.ReprocessAlert(ctx, qa.ID, func(bgCtx context.Context, schema types.AlertSchema, data any) error {
		gt.Value(t, schema).Equal(types.AlertSchema("test.schema"))
		processed <- true
		return nil
	})
	gt.NoError(t, err)
	gt.Value(t, job.Status).Equal(types.ReprocessJobStatusPending)

	// Wait for background goroutine to complete
	select {
	case <-processed:
	case <-time.After(5 * time.Second):
		t.Fatal("reprocess callback not called within timeout")
	}

	// Give the goroutine a moment to finish cleanup
	time.Sleep(100 * time.Millisecond)

	// Job should be completed
	gotJob, err := repo.GetReprocessJob(ctx, job.ID)
	gt.NoError(t, err)
	gt.Value(t, gotJob.Status).Equal(types.ReprocessJobStatusCompleted)
	gt.Value(t, gotJob.Error).Equal("")

	// Queued alert should be deleted
	count, err := repo.CountQueuedAlerts(ctx)
	gt.NoError(t, err)
	gt.Value(t, count).Equal(0)
}

func TestService_ReprocessAlert_Failure(t *testing.T) {
	svc, repo := newTestService(1)
	ctx := t.Context()

	qa, err := svc.EnqueueAlert(ctx, "test.schema", nil, "Title", "")
	gt.NoError(t, err)

	// Reprocess with a failing callback
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

	// Job should be failed
	gotJob, err := repo.GetReprocessJob(ctx, job.ID)
	gt.NoError(t, err)
	gt.Value(t, gotJob.Status).Equal(types.ReprocessJobStatusFailed)
	gt.True(t, gotJob.Error != "")

	// Queued alert should NOT be deleted (still in queue for retry)
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

func TestService_EndToEnd_ThrottleAndEnqueue(t *testing.T) {
	svc, repo := newTestService(2)

	baseTime := time.Date(2026, 3, 24, 14, 0, 0, 0, time.UTC)
	ctx := clock.With(t.Context(), func() time.Time { return baseTime })

	// First 2 alerts pass through
	result, err := svc.TryAcquire(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	result, err = svc.TryAcquire(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// 3rd alert throttled — enqueue it
	result, err = svc.TryAcquire(ctx)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(false)
	gt.Value(t, result.ShouldNotify).Equal(true)

	qa, err := svc.EnqueueAlert(ctx, "guardduty", map[string]any{"alert": true}, "Throttled Alert", "C123")
	gt.NoError(t, err)

	// Verify queue has 1 alert
	count, err := repo.CountQueuedAlerts(ctx)
	gt.NoError(t, err)
	gt.Value(t, count).Equal(1)

	// After window expires, slots available again
	ctx11 := clock.With(t.Context(), func() time.Time { return baseTime.Add(11 * time.Minute) })
	result, err = svc.TryAcquire(ctx11)
	gt.NoError(t, err)
	gt.Value(t, result.Allowed).Equal(true)

	// Discard the queued alert
	gt.NoError(t, svc.DiscardAlerts(ctx, []types.QueuedAlertID{qa.ID}))
	count, err = repo.CountQueuedAlerts(ctx)
	gt.NoError(t, err)
	gt.Value(t, count).Equal(0)
}
