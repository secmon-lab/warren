package usecase_test

// chat-session-redesign Phase 2.3 / 2.4:
// Integration tests for ChatFromSlack's Session + Lock + Turn wrapping
// introduced in Phase 2. These run at the usecase layer, use the memory
// repository, and do NOT drive the LLM (ChatUC is replaced by a stub so
// we can assert only on the Session/Lock/Turn bookkeeping around Execute).

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	sessSvcDomain "github.com/secmon-lab/warren/pkg/domain/interfaces"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	slackSvc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/service/slack/testutil"
	"github.com/secmon-lab/warren/pkg/usecase"
)

// stubChatUC lets us assert that ChatFromSlack reaches Execute (or not)
// without pulling in the full LLM stack. It also lets a test pause the
// Execute call so the lock can be inspected while a run is in flight.
type stubChatUC struct {
	mu      sync.Mutex
	runs    int
	release chan struct{}
	err     error
	onExec  func(ctx context.Context)
}

func (s *stubChatUC) Execute(ctx context.Context, message string, chatCtx chatModel.ChatContext) error {
	s.mu.Lock()
	s.runs++
	release := s.release
	onExec := s.onExec
	s.mu.Unlock()
	if onExec != nil {
		onExec(ctx)
	}
	if release != nil {
		<-release
	}
	return s.err
}

// ensure stubChatUC implements the interface expected by usecase.UseCases.
var _ sessSvcDomain.ChatUseCase = (*stubChatUC)(nil)

func setupUseCases(t *testing.T, stub *stubChatUC) (*usecase.UseCases, *testutil.Recorder, *repository.Memory) {
	t.Helper()
	repo := repository.NewMemory()

	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)
	slack, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(slack),
		usecase.WithChatUseCase(stub),
	)
	return uc, rec, repo
}

func TestChatFromSlack_AcquiresLockAndCreatesTurn(t *testing.T) {
	ctx := context.Background()
	stub := &stubChatUC{}
	uc, _, repo := setupUseCases(t, stub)

	// Seed a Ticket so ChatFromSlack takes the "with ticket" path.
	thread := slackModel.Thread{ChannelID: "C1", ThreadID: "t1", TeamID: "T1"}
	tk := ticket.Ticket{
		ID:          types.TicketID("tid_1"),
		Status:      types.TicketStatusOpen,
		SlackThread: &thread,
	}
	gt.NoError(t, repo.PutTicket(ctx, tk)).Required()

	msg := slackModel.NewTestMessage(thread.ChannelID, thread.ThreadID, thread.TeamID, "m1", "U1", "@warren help")
	gt.NoError(t, uc.ChatFromSlack(ctx, &msg, "help"))

	// ChatUC.Execute was invoked exactly once.
	gt.V(t, stub.runs).Equal(1)

	// A Slack Session was created with deterministic ID and linked to the
	// ticket; its Lock was released (lock == nil) on the happy path.
	sessions, err := repo.GetSessionsByTicket(ctx, tk.ID)
	gt.NoError(t, err)
	gt.A(t, sessions).Length(1)
	sess := sessions[0]
	gt.V(t, sess.Source).Equal(sessModel.SessionSourceSlack)
	gt.V(t, sess.Lock == nil).Equal(true)

	// A Turn was recorded for the mention and closed as completed.
	turns, err := repo.GetTurnsBySession(ctx, sess.ID)
	gt.NoError(t, err)
	gt.A(t, turns).Length(1)
	gt.V(t, turns[0].Status).Equal(sessModel.TurnStatusCompleted)
	gt.V(t, turns[0].EndedAt != nil).Equal(true)
}

func TestChatFromSlack_DoubleMention_BlocksSecondAndPostsBusyNotice(t *testing.T) {
	ctx := context.Background()
	stub := &stubChatUC{release: make(chan struct{})}
	uc, rec, repo := setupUseCases(t, stub)

	thread := slackModel.Thread{ChannelID: "C1", ThreadID: "t1", TeamID: "T1"}
	tk := ticket.Ticket{
		ID:          types.TicketID("tid_1"),
		Status:      types.TicketStatusOpen,
		SlackThread: &thread,
	}
	gt.NoError(t, repo.PutTicket(ctx, tk)).Required()

	// First mention: block inside Execute so the lock is still held when
	// the second mention arrives.
	msg := slackModel.NewTestMessage(thread.ChannelID, thread.ThreadID, thread.TeamID, "m1", "U1", "@warren analyze")
	firstDone := make(chan error, 1)
	go func() { firstDone <- uc.ChatFromSlack(ctx, &msg, "analyze") }()

	// Wait for Execute to be reached on the first mention.
	waitUntil(t, func() bool {
		stub.mu.Lock()
		defer stub.mu.Unlock()
		return stub.runs >= 1
	})

	// Clear recorder of the first mention's AuthTest/GetTeamInfo from
	// service construction (already gone; we Reset'd earlier). The only
	// Slack API calls we care about for the second mention are the busy
	// notice.
	rec.Reset()

	// Second mention: should be rejected by the lock and post a context
	// block, and must NOT call Execute again.
	msg2 := slackModel.NewTestMessage(thread.ChannelID, thread.ThreadID, thread.TeamID, "m2", "U2", "@warren again")
	gt.NoError(t, uc.ChatFromSlack(ctx, &msg2, "again"))

	calls := rec.Calls()
	// The busy path still consults GetConversationRepliesContext (slack
	// history) before reaching the lock, then posts the busy notice.
	// Total: 1 history fetch + 1 PostMessageContext (busy block).
	var postCount int
	for _, c := range calls {
		if c.Method == "PostMessageContext" {
			postCount++
		}
	}
	gt.V(t, postCount).Equal(1)

	stub.mu.Lock()
	runsAfterBusy := stub.runs
	stub.mu.Unlock()
	gt.V(t, runsAfterBusy).Equal(1)

	// Unblock the first Execute and let it finish; the lock is released.
	close(stub.release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first ChatFromSlack returned error: %v", err)
	}
}

func TestChatFromSlack_ConcurrentMentions_OneWinsRestBusy(t *testing.T) {
	ctx := context.Background()
	var execStarted atomic.Int32
	stub := &stubChatUC{release: make(chan struct{})}
	stub.onExec = func(ctx context.Context) { execStarted.Add(1) }
	uc, _, repo := setupUseCases(t, stub)

	thread := slackModel.Thread{ChannelID: "C1", ThreadID: "t1", TeamID: "T1"}
	tk := ticket.Ticket{
		ID:          types.TicketID("tid_1"),
		Status:      types.TicketStatusOpen,
		SlackThread: &thread,
	}
	gt.NoError(t, repo.PutTicket(ctx, tk)).Required()

	msg := slackModel.NewTestMessage(thread.ChannelID, thread.ThreadID, thread.TeamID, "m1", "U1", "@warren analyze")

	// Fire one mention that WILL enter Execute and block.
	first := make(chan error, 1)
	go func() { first <- uc.ChatFromSlack(ctx, &msg, "analyze") }()

	// Wait until the first Execute has started and is blocking.
	waitUntil(t, func() bool { return execStarted.Load() == 1 })

	// Fire N concurrent mentions against the same thread while the lock
	// is held. All must be rejected by the lock (none enter Execute).
	const N = 20
	errs := make([]error, N)
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			errs[i] = uc.ChatFromSlack(ctx, &msg, "analyze")
		}()
	}
	wg.Wait()

	for _, err := range errs {
		gt.NoError(t, err)
	}

	// Still only the first Execute entered.
	gt.V(t, execStarted.Load()).Equal(int32(1))

	// Release the first Execute so the goroutine exits cleanly.
	close(stub.release)
	if err := <-first; err != nil {
		t.Fatalf("first ChatFromSlack returned error: %v", err)
	}
}

// waitUntil polls cond for up to ~2 seconds and fails the test otherwise.
func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("waitUntil: condition not satisfied")
}
