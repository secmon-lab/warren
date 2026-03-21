package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestHITLRequest_PutAndGet(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		req := &hitl.Request{
			ID:        types.NewHITLRequestID(),
			SessionID: types.NewSessionID(),
			Type:      hitl.RequestTypeToolApproval,
			Payload: map[string]any{
				"tool_name": "web_fetch",
				"tool_args": map[string]any{"url": "https://example.com"},
			},
			Status:      hitl.StatusPending,
			UserID:      "U12345",
			CreatedAt:   time.Now(),
			SlackThread: newTestThread(),
		}

		gt.NoError(t, repo.PutHITLRequest(ctx, req)).Required()

		got, err := repo.GetHITLRequest(ctx, req.ID)
		gt.NoError(t, err).Required()
		gt.Value(t, got.ID).Equal(req.ID)
		gt.Value(t, got.SessionID).Equal(req.SessionID)
		gt.Value(t, got.Type).Equal(hitl.RequestTypeToolApproval)
		gt.Value(t, got.Status).Equal(hitl.StatusPending)
		gt.Value(t, got.UserID).Equal("U12345")
	}

	t.Run("Memory", func(t *testing.T) {
		testFn(t, repository.NewMemory())
	})

	t.Run("Firestore", func(t *testing.T) {
		testFn(t, newFirestoreClient(t))
	})
}

func TestHITLRequest_UpdateStatus(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		req := &hitl.Request{
			ID:        types.NewHITLRequestID(),
			SessionID: types.NewSessionID(),
			Type:      hitl.RequestTypeToolApproval,
			Payload: map[string]any{
				"tool_name": "web_fetch",
			},
			Status:      hitl.StatusPending,
			UserID:      "U12345",
			CreatedAt:   time.Now(),
			SlackThread: newTestThread(),
		}

		gt.NoError(t, repo.PutHITLRequest(ctx, req)).Required()

		response := map[string]any{"comment": "looks good"}
		gt.NoError(t, repo.UpdateHITLRequestStatus(ctx, req.ID, hitl.StatusApproved, "U67890", response)).Required()

		got, err := repo.GetHITLRequest(ctx, req.ID)
		gt.NoError(t, err).Required()
		gt.Value(t, got.Status).Equal(hitl.StatusApproved)
		gt.Value(t, got.RespondedBy).Equal("U67890")
		gt.Value(t, got.Response["comment"]).Equal("looks good")
		gt.True(t, !got.RespondedAt.IsZero())
	}

	t.Run("Memory", func(t *testing.T) {
		testFn(t, repository.NewMemory())
	})

	t.Run("Firestore", func(t *testing.T) {
		testFn(t, newFirestoreClient(t))
	})
}

func TestHITLRequest_WatchApproved(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		req := &hitl.Request{
			ID:          types.NewHITLRequestID(),
			SessionID:   types.NewSessionID(),
			Type:        hitl.RequestTypeToolApproval,
			Payload:     map[string]any{"tool_name": "web_fetch"},
			Status:      hitl.StatusPending,
			UserID:      "U12345",
			CreatedAt:   time.Now(),
			SlackThread: newTestThread(),
		}

		gt.NoError(t, repo.PutHITLRequest(ctx, req)).Required()

		watchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		ch, errCh := repo.WatchHITLRequest(watchCtx, req.ID)

		// Update status in a separate goroutine
		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = repo.UpdateHITLRequestStatus(ctx, req.ID, hitl.StatusApproved, "U67890", map[string]any{"comment": "ok"})
		}()

		select {
		case updated := <-ch:
			gt.Value(t, updated.Status).Equal(hitl.StatusApproved)
			gt.Value(t, updated.RespondedBy).Equal("U67890")
		case err := <-errCh:
			t.Fatalf("watch error: %v", err)
		case <-watchCtx.Done():
			t.Fatal("watch timed out")
		}
	}

	t.Run("Memory", func(t *testing.T) {
		testFn(t, repository.NewMemory())
	})

	t.Run("Firestore", func(t *testing.T) {
		testFn(t, newFirestoreClient(t))
	})
}

func TestHITLRequest_WatchDenied(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		req := &hitl.Request{
			ID:          types.NewHITLRequestID(),
			SessionID:   types.NewSessionID(),
			Type:        hitl.RequestTypeToolApproval,
			Payload:     map[string]any{"tool_name": "web_fetch"},
			Status:      hitl.StatusPending,
			UserID:      "U12345",
			CreatedAt:   time.Now(),
			SlackThread: newTestThread(),
		}

		gt.NoError(t, repo.PutHITLRequest(ctx, req)).Required()

		watchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		ch, errCh := repo.WatchHITLRequest(watchCtx, req.ID)

		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = repo.UpdateHITLRequestStatus(ctx, req.ID, hitl.StatusDenied, "U67890", map[string]any{"comment": "not allowed"})
		}()

		select {
		case updated := <-ch:
			gt.Value(t, updated.Status).Equal(hitl.StatusDenied)
			gt.Value(t, updated.Response["comment"]).Equal("not allowed")
		case err := <-errCh:
			t.Fatalf("watch error: %v", err)
		case <-watchCtx.Done():
			t.Fatal("watch timed out")
		}
	}

	t.Run("Memory", func(t *testing.T) {
		testFn(t, repository.NewMemory())
	})

	t.Run("Firestore", func(t *testing.T) {
		testFn(t, newFirestoreClient(t))
	})
}

func TestHITLRequest_WatchCancelledContext(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		req := &hitl.Request{
			ID:          types.NewHITLRequestID(),
			SessionID:   types.NewSessionID(),
			Type:        hitl.RequestTypeToolApproval,
			Payload:     map[string]any{"tool_name": "web_fetch"},
			Status:      hitl.StatusPending,
			UserID:      "U12345",
			CreatedAt:   time.Now(),
			SlackThread: newTestThread(),
		}

		gt.NoError(t, repo.PutHITLRequest(ctx, req)).Required()

		watchCtx, cancel := context.WithCancel(ctx)
		ch, _ := repo.WatchHITLRequest(watchCtx, req.ID)

		// Cancel immediately
		cancel()

		// Channel should close without a value
		select {
		case _, ok := <-ch:
			gt.Value(t, ok).Equal(false)
		case <-time.After(5 * time.Second):
			t.Fatal("channel did not close after context cancellation")
		}
	}

	t.Run("Memory", func(t *testing.T) {
		testFn(t, repository.NewMemory())
	})

	t.Run("Firestore", func(t *testing.T) {
		testFn(t, newFirestoreClient(t))
	})
}
