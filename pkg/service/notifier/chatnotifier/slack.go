package chatnotifier

import (
	"context"
	"fmt"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// SlackChatNotifier persists Messages and mirrors AI-produced content into
// the bound Slack thread. Incoming user Messages (type=user) are persisted
// but NOT posted back to Slack; the user's own message is already visible
// in the thread by construction.
type SlackChatNotifier struct {
	repo     interfaces.Repository
	thread   interfaces.SlackThreadService
	session  *session.Session
	turnID   *types.TurnID
	ticketID *types.TicketID

	// traceOnce lazily initializes an updatable trace message on first
	// Trace call, mirroring the existing chat/usecase.go trace behavior
	// of editing a single context block in place.
	traceOnce sync.Once
	traceFn   func(ctx context.Context, msg string)
}

// NewSlackChatNotifier constructs a SlackChatNotifier. thread may be nil if
// the Session has no bound Slack thread, in which case Slack delivery is
// silently skipped (Message persistence still happens).
func NewSlackChatNotifier(
	repo interfaces.Repository,
	thread interfaces.SlackThreadService,
	sess *session.Session,
	turnID *types.TurnID,
) *SlackChatNotifier {
	return &SlackChatNotifier{
		repo:     repo,
		thread:   thread,
		session:  sess,
		turnID:   turnID,
		ticketID: sess.TicketIDOrNil(),
	}
}

func (s *SlackChatNotifier) persist(ctx context.Context, msgType session.MessageType, content string, author *session.Author) {
	msg := session.NewMessageV2(ctx, s.session.ID, s.ticketID, s.turnID, msgType, content, author)
	if err := s.repo.PutSessionMessage(ctx, msg); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to persist Slack session message",
			goerr.V("session_id", s.session.ID),
			goerr.V("type", msgType),
		))
	}
}

func (s *SlackChatNotifier) Notify(ctx context.Context, content string) error {
	s.persist(ctx, session.MessageTypeResponse, content, nil)
	if s.thread == nil {
		return nil
	}
	if err := s.thread.PostComment(ctx, content); err != nil {
		return goerr.Wrap(err, "failed to post response to slack")
	}
	return nil
}

func (s *SlackChatNotifier) Trace(ctx context.Context, content string) error {
	s.persist(ctx, session.MessageTypeTrace, content, nil)
	if s.thread == nil {
		return nil
	}
	s.traceOnce.Do(func() {
		s.traceFn = s.thread.NewTraceMessage(ctx, content)
	})
	if s.traceFn != nil {
		s.traceFn(ctx, content)
	}
	return nil
}

func (s *SlackChatNotifier) Warn(ctx context.Context, content string) error {
	s.persist(ctx, session.MessageTypeWarning, content, nil)
	if s.thread == nil {
		return nil
	}
	if err := s.thread.PostComment(ctx, fmt.Sprintf("⚠️ %s", content)); err != nil {
		return goerr.Wrap(err, "failed to post warning to slack")
	}
	return nil
}

func (s *SlackChatNotifier) Plan(ctx context.Context, content string) error {
	s.persist(ctx, session.MessageTypePlan, content, nil)
	if s.thread == nil {
		return nil
	}
	if err := s.thread.PostContextBlock(ctx, content); err != nil {
		return goerr.Wrap(err, "failed to post plan to slack")
	}
	return nil
}

// NotifyUser records a user-authored Message. The originating Slack message
// is already visible in the thread (the user sent it), so no post-back is
// issued; we only persist for Session history.
func (s *SlackChatNotifier) NotifyUser(ctx context.Context, content string, author *session.Author) error {
	if author == nil {
		return goerr.New("NotifyUser requires author")
	}
	s.persist(ctx, session.MessageTypeUser, content, author)
	return nil
}
