package chatnotifier_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/notifier/chatnotifier"
)

func newSession(t *testing.T, source session.SessionSource, ticketID *types.TicketID) *session.Session {
	t.Helper()
	return session.NewSessionV2(t.Context(),
		types.SessionID("sid_test"),
		source, ticketID, nil, types.UserID("user-1"),
	)
}

func TestNoopNotifier_AllOpsNoError(t *testing.T) {
	ctx := context.Background()
	var n interfaces.SessionNotifier = chatnotifier.NoopNotifier{}

	gt.NoError(t, n.Notify(ctx, "x"))
	gt.NoError(t, n.Trace(ctx, "x"))
	gt.NoError(t, n.Warn(ctx, "x"))
	gt.NoError(t, n.Plan(ctx, "x"))
	gt.NoError(t, n.NotifyUser(ctx, "x", &session.Author{UserID: "u"}))
}

func TestCtx_WithAndFromContext(t *testing.T) {
	ctx := context.Background()

	// Default (no value set) returns Noop.
	def := chatnotifier.FromContext(ctx)
	gt.V(t, def).NotNil()

	// Set explicit notifier and recover it.
	var custom interfaces.SessionNotifier = chatnotifier.NoopNotifier{}
	ctx2 := chatnotifier.WithNotifier(ctx, custom)
	got := chatnotifier.FromContext(ctx2)
	gt.V(t, got).NotNil()

	// Nil passed to WithNotifier substitutes NoopNotifier.
	ctx3 := chatnotifier.WithNotifier(ctx, nil)
	gt.V(t, chatnotifier.FromContext(ctx3)).NotNil()
}

func TestCLINotifier_PersistsAndWrites(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	sess := newSession(t, session.SessionSourceCLI, nil)
	turnID := types.TurnID("turn_1")

	var buf bytes.Buffer
	n := chatnotifier.NewCLINotifier(repo, sess, &turnID, &buf)

	gt.NoError(t, n.Notify(ctx, "response line"))
	gt.NoError(t, n.Trace(ctx, "trace line"))
	gt.NoError(t, n.Plan(ctx, "plan line"))
	gt.NoError(t, n.Warn(ctx, "warn line"))

	output := buf.String()
	gt.True(t, bytes.Contains([]byte(output), []byte("response line")))
	gt.True(t, bytes.Contains([]byte(output), []byte("trace line")))
	gt.True(t, bytes.Contains([]byte(output), []byte("plan line")))
	gt.True(t, bytes.Contains([]byte(output), []byte("warn line")))

	msgs, err := repo.GetSessionMessages(ctx, sess.ID)
	gt.NoError(t, err)
	gt.A(t, msgs).Length(4)
	// Each persisted message carries the TurnID supplied at construction.
	for _, m := range msgs {
		gt.V(t, m.TurnID == nil).Equal(false)
		gt.V(t, string(*m.TurnID)).Equal("turn_1")
	}
}

func TestCLINotifier_NotifyUser_PersistsWithAuthor(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	sess := newSession(t, session.SessionSourceCLI, nil)
	var buf bytes.Buffer
	n := chatnotifier.NewCLINotifier(repo, sess, nil, &buf)

	author := &session.Author{UserID: "u-1", DisplayName: "Alice"}
	gt.NoError(t, n.NotifyUser(ctx, "hi", author))

	msgs, err := repo.GetSessionMessages(ctx, sess.ID)
	gt.NoError(t, err)
	gt.A(t, msgs).Length(1)
	gt.V(t, msgs[0].Type).Equal(session.MessageTypeUser)
	gt.V(t, msgs[0].Author == nil).Equal(false)
	gt.V(t, msgs[0].Author.DisplayName).Equal("Alice")
	// User echo is not written to stdout.
	gt.V(t, buf.String()).Equal("")
}

func TestCLINotifier_NotifyUser_RequiresAuthor(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	sess := newSession(t, session.SessionSourceCLI, nil)
	n := chatnotifier.NewCLINotifier(repo, sess, nil, nil)

	err := n.NotifyUser(ctx, "x", nil)
	gt.Error(t, err)
}

type captureWebPublisher struct {
	Calls []publishedEvent
}

type publishedEvent struct {
	TicketID types.TicketID
	Payload  json.RawMessage
}

func (c *captureWebPublisher) PublishToTicket(_ context.Context, tid types.TicketID, payload []byte) error {
	c.Calls = append(c.Calls, publishedEvent{TicketID: tid, Payload: append(json.RawMessage(nil), payload...)})
	return nil
}

func TestWebNotifier_PersistsAndPublishes(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tid := types.TicketID("tid_1")
	sess := newSession(t, session.SessionSourceWeb, &tid)
	turnID := types.TurnID("turn_1")
	pub := &captureWebPublisher{}

	n := chatnotifier.NewWebNotifier(repo, pub, sess, &turnID)

	gt.NoError(t, n.Trace(ctx, "step 1"))
	gt.NoError(t, n.Notify(ctx, "final"))

	msgs, err := repo.GetSessionMessages(ctx, sess.ID)
	gt.NoError(t, err)
	gt.A(t, msgs).Length(2)

	gt.A(t, pub.Calls).Length(2)
	for _, c := range pub.Calls {
		gt.V(t, c.TicketID).Equal(tid)
		var decoded map[string]any
		gt.NoError(t, json.Unmarshal(c.Payload, &decoded))
		gt.V(t, decoded["event"]).Equal("session_message_added")
		gt.V(t, decoded["session_id"]).Equal(string(sess.ID))
	}
}

func TestWebNotifier_TicketlessSkipsPublish(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	sess := newSession(t, session.SessionSourceWeb, nil) // no ticket
	pub := &captureWebPublisher{}

	n := chatnotifier.NewWebNotifier(repo, pub, sess, nil)
	gt.NoError(t, n.Notify(ctx, "x"))

	msgs, err := repo.GetSessionMessages(ctx, sess.ID)
	gt.NoError(t, err)
	gt.A(t, msgs).Length(1) // persisted
	gt.A(t, pub.Calls).Length(0) // no publish without ticket
}

func TestFactory_FromSession_WebReturnsWebNotifier(t *testing.T) {
	repo := repository.NewMemory()
	pub := &captureWebPublisher{}
	f := chatnotifier.NewFactory(repo, chatnotifier.WithWebPublisher(pub))

	tid := types.TicketID("tid_x")
	sess := newSession(t, session.SessionSourceWeb, &tid)
	n := f.FromSession(sess, nil)
	gt.V(t, n).NotNil()

	// Smoke: Notify should route through the publisher.
	gt.NoError(t, n.Notify(context.Background(), "hi"))
	gt.A(t, pub.Calls).Length(1)
}

func TestFactory_FromSession_CLIReturnsCLINotifier(t *testing.T) {
	repo := repository.NewMemory()
	var buf bytes.Buffer
	f := chatnotifier.NewFactory(repo, chatnotifier.WithCLIWriter(&buf))

	sess := newSession(t, session.SessionSourceCLI, nil)
	n := f.FromSession(sess, nil)
	gt.NoError(t, n.Notify(context.Background(), "cli output"))
	gt.True(t, bytes.Contains(buf.Bytes(), []byte("cli output")))
}

func TestFactory_FromSession_UnknownSourceReturnsNoop(t *testing.T) {
	repo := repository.NewMemory()
	f := chatnotifier.NewFactory(repo)

	sess := &session.Session{ID: "x", Source: session.SessionSource("bogus")}
	n := f.FromSession(sess, nil)
	gt.V(t, n).NotNil()
	gt.NoError(t, n.Notify(context.Background(), "nowhere"))
}

func TestFactory_FromSession_NilSessionReturnsNoop(t *testing.T) {
	repo := repository.NewMemory()
	f := chatnotifier.NewFactory(repo)
	n := f.FromSession(nil, nil)
	gt.V(t, n).NotNil()
}
