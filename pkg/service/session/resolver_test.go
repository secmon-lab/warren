package session_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/m-mizutani/gt"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	sessSvc "github.com/secmon-lab/warren/pkg/service/session"
)

func newThread() slackModel.Thread {
	return slackModel.Thread{
		TeamID:    "T1",
		ChannelID: "C1",
		ThreadID:  "1700.0001",
	}
}

func TestResolver_ResolveSlackSession_CreatesDeterministicID(t *testing.T) {
	repo := repository.NewMemory()
	r := sessSvc.NewResolver(repo)

	tid := types.TicketID("tid_1")
	s1, created1, err := r.ResolveSlackSession(context.Background(), &tid, newThread(), types.UserID("u1"))
	gt.NoError(t, err)
	gt.V(t, created1).Equal(true)

	s2, created2, err := r.ResolveSlackSession(context.Background(), &tid, newThread(), types.UserID("u2"))
	gt.NoError(t, err)
	gt.V(t, created2).Equal(false)
	gt.V(t, s2.ID).Equal(s1.ID)
}

func TestResolver_ResolveSlackSession_TicketlessDistinctFromTicketed(t *testing.T) {
	repo := repository.NewMemory()
	r := sessSvc.NewResolver(repo)

	tid := types.TicketID("tid_1")
	tl, _, err := r.ResolveSlackSession(context.Background(), nil, newThread(), types.UserID("u1"))
	gt.NoError(t, err)

	tk, _, err := r.ResolveSlackSession(context.Background(), &tid, newThread(), types.UserID("u1"))
	gt.NoError(t, err)

	gt.V(t, tl.ID).NotEqual(tk.ID)
}

func TestResolver_ResolveSlackSession_DifferentThreadsDifferentIDs(t *testing.T) {
	repo := repository.NewMemory()
	r := sessSvc.NewResolver(repo)
	tid := types.TicketID("tid_1")

	s1, _, err := r.ResolveSlackSession(context.Background(), &tid, slackModel.Thread{ChannelID: "C1", ThreadID: "t1"}, "u1")
	gt.NoError(t, err)
	s2, _, err := r.ResolveSlackSession(context.Background(), &tid, slackModel.Thread{ChannelID: "C1", ThreadID: "t2"}, "u1")
	gt.NoError(t, err)
	s3, _, err := r.ResolveSlackSession(context.Background(), &tid, slackModel.Thread{ChannelID: "C2", ThreadID: "t1"}, "u1")
	gt.NoError(t, err)

	gt.V(t, s1.ID).NotEqual(s2.ID)
	gt.V(t, s1.ID).NotEqual(s3.ID)
	gt.V(t, s2.ID).NotEqual(s3.ID)
}

func TestResolver_Parallel_ResolveSlackSession_OneCreation(t *testing.T) {
	repo := repository.NewMemory()
	r := sessSvc.NewResolver(repo)
	tid := types.TicketID("tid_1")
	thread := newThread()

	const N = 50
	var createdCount int64
	var firstID atomic.Value
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			sess, created, err := r.ResolveSlackSession(context.Background(), &tid, thread, types.UserID("u"))
			gt.NoError(t, err)
			if created {
				atomic.AddInt64(&createdCount, 1)
			}
			firstID.Store(sess.ID)
		}()
	}
	wg.Wait()

	// Despite 50 concurrent callers, exactly one Session was created.
	gt.V(t, atomic.LoadInt64(&createdCount)).Equal(int64(1))

	// All callers observe the same Session ID.
	id := firstID.Load().(types.SessionID)
	sess, err := repo.GetSession(context.Background(), id)
	gt.NoError(t, err)
	gt.V(t, sess == nil).Equal(false)
}

func TestResolver_CreateFreshSession_RejectsSlackSource(t *testing.T) {
	repo := repository.NewMemory()
	r := sessSvc.NewResolver(repo)

	_, err := r.CreateFreshSession(context.Background(), "tid_1", sessModel.SessionSourceSlack, "u")
	gt.Error(t, err)
}

func TestResolver_CreateFreshSession_WebAndCLI(t *testing.T) {
	repo := repository.NewMemory()
	r := sessSvc.NewResolver(repo)
	tid := types.TicketID("tid_1")

	s1, err := r.CreateFreshSession(context.Background(), tid, sessModel.SessionSourceWeb, "u")
	gt.NoError(t, err)
	gt.V(t, s1.Source).Equal(sessModel.SessionSourceWeb)
	gt.V(t, s1.TicketIDPtr != nil && *s1.TicketIDPtr == tid).Equal(true)

	s2, err := r.CreateFreshSession(context.Background(), tid, sessModel.SessionSourceCLI, "u")
	gt.NoError(t, err)
	gt.V(t, s2.Source).Equal(sessModel.SessionSourceCLI)
	gt.V(t, s1.ID).NotEqual(s2.ID) // fresh random IDs
}
