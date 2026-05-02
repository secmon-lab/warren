package session_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	sessSvc "github.com/secmon-lab/warren/pkg/service/session"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
)

func ctxWithRequest(reqID string) context.Context {
	return request_id.With(context.Background(), reqID)
}

func seedSession(t *testing.T, repo *repository.Memory, id types.SessionID) {
	t.Helper()
	sess := &session.Session{ID: id}
	gt.NoError(t, repo.PutSession(context.Background(), sess))
}

func TestLockService_TryAcquire_Succeeds(t *testing.T) {
	repo := repository.NewMemory()
	seedSession(t, repo, "sid_1")
	svc := sessSvc.NewLockService(repo)

	lock, ok, err := svc.TryAcquire(ctxWithRequest("req-1"), "sid_1")
	gt.NoError(t, err)
	gt.V(t, ok).Equal(true)
	gt.V(t, lock == nil).Equal(false)
	gt.V(t, lock.HolderID()).Equal("req-1")
}

func TestLockService_TryAcquire_DeniesSecondCaller(t *testing.T) {
	repo := repository.NewMemory()
	seedSession(t, repo, "sid_1")
	svc := sessSvc.NewLockService(repo)

	_, ok1, err := svc.TryAcquire(ctxWithRequest("req-A"), "sid_1")
	gt.NoError(t, err)
	gt.V(t, ok1).Equal(true)

	_, ok2, err := svc.TryAcquire(ctxWithRequest("req-B"), "sid_1")
	gt.NoError(t, err)
	gt.V(t, ok2).Equal(false)
}

func TestLockService_ReleaseThenReacquire_ImmediateSuccess(t *testing.T) {
	repo := repository.NewMemory()
	seedSession(t, repo, "sid_1")
	svc := sessSvc.NewLockService(repo)

	ctx := ctxWithRequest("req-A")
	lock, ok, err := svc.TryAcquire(ctx, "sid_1")
	gt.NoError(t, err)
	gt.V(t, ok).Equal(true)

	gt.NoError(t, lock.Release(ctx))

	// Next TryAcquire (even with a different holder) succeeds without
	// waiting for the 3-minute TTL.
	_, ok2, err := svc.TryAcquire(ctxWithRequest("req-B"), "sid_1")
	gt.NoError(t, err)
	gt.V(t, ok2).Equal(true)
}

func TestLockService_Refresh_ExtendsExpiry(t *testing.T) {
	repo := repository.NewMemory()
	seedSession(t, repo, "sid_1")
	svc := sessSvc.NewLockService(repo, sessSvc.WithLockTTL(100*time.Millisecond))

	ctx := ctxWithRequest("req-A")
	lock, ok, err := svc.TryAcquire(ctx, "sid_1")
	gt.NoError(t, err)
	gt.V(t, ok).Equal(true)
	oldExpires := lock.ExpiresAt()

	time.Sleep(20 * time.Millisecond)
	gt.NoError(t, lock.Refresh(ctx))

	gt.V(t, lock.ExpiresAt().After(oldExpires)).Equal(true)
}

func TestLockService_TTLExpiry_AllowsTakeover(t *testing.T) {
	repo := repository.NewMemory()
	seedSession(t, repo, "sid_1")
	svc := sessSvc.NewLockService(repo, sessSvc.WithLockTTL(10*time.Millisecond))

	_, ok, err := svc.TryAcquire(ctxWithRequest("req-A"), "sid_1")
	gt.NoError(t, err)
	gt.V(t, ok).Equal(true)

	// Do NOT release; wait for TTL to elapse.
	time.Sleep(20 * time.Millisecond)

	_, ok2, err := svc.TryAcquire(ctxWithRequest("req-B"), "sid_1")
	gt.NoError(t, err)
	gt.V(t, ok2).Equal(true)
}

func TestLockService_AutoGeneratesHolderWhenRequestIDMissing(t *testing.T) {
	repo := repository.NewMemory()
	seedSession(t, repo, "sid_1")
	svc := sessSvc.NewLockService(repo)

	lock, ok, err := svc.TryAcquire(context.Background(), "sid_1")
	gt.NoError(t, err)
	gt.V(t, ok).Equal(true)
	// A holder was auto-generated so Release/Refresh have something to
	// authenticate with.
	gt.V(t, lock.HolderID() != "").Equal(true)
	gt.V(t, lock.HolderID() != "(unknown)").Equal(true)
}

func TestLockService_ConcurrentAcquire_OnlyOneWinner(t *testing.T) {
	repo := repository.NewMemory()
	seedSession(t, repo, "sid_1")
	svc := sessSvc.NewLockService(repo)

	const N = 50
	var wins int64
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, ok, err := svc.TryAcquire(ctxWithRequest(reqName(i)), "sid_1")
			gt.NoError(t, err)
			if ok {
				atomic.AddInt64(&wins, 1)
			}
		}()
	}
	wg.Wait()
	gt.V(t, atomic.LoadInt64(&wins)).Equal(int64(1))
}

func TestLockService_ReleaseThenConcurrentAcquire_OneWinner(t *testing.T) {
	repo := repository.NewMemory()
	seedSession(t, repo, "sid_1")
	svc := sessSvc.NewLockService(repo)

	ctx := ctxWithRequest("req-init")
	lock, ok, err := svc.TryAcquire(ctx, "sid_1")
	gt.NoError(t, err)
	gt.V(t, ok).Equal(true)
	gt.NoError(t, lock.Release(ctx))

	// Release has completed; fire 50 concurrent TryAcquires. Exactly one
	// should win (the others should observe the lock already taken by the
	// winner).
	const N = 50
	var wins int64
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, ok, _ := svc.TryAcquire(ctxWithRequest(reqName(i)), "sid_1")
			if ok {
				atomic.AddInt64(&wins, 1)
			}
		}()
	}
	wg.Wait()
	gt.V(t, atomic.LoadInt64(&wins)).Equal(int64(1))
}

func reqName(i int) string {
	return "req-" + string(rune('a'+i%26)) + "_" + string(rune('0'+i/26%10))
}
