package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// Resolver creates or looks up Sessions for the new chat-session-redesign
// lifecycle. It uses deterministic IDs for Slack Sessions so that a thread
// maps to exactly one Session across all Warren instances; Web/CLI Sessions
// use random IDs since they are ephemeral per-invocation.
//
// Resolver holds no per-Session state; it only carries the Repository
// dependency that it uses to read/write Sessions.
type Resolver struct {
	repo interfaces.Repository
}

// NewResolver constructs a Resolver bound to repo.
func NewResolver(repo interfaces.Repository) *Resolver {
	return &Resolver{repo: repo}
}

// ResolveSlackSession returns the Session associated with the given Slack
// thread, creating it transactionally if it does not exist.
//
// ticketID may be nil for ticketless threads (e.g. @warren mention on a
// thread that has not yet been escalated into a Ticket). The Session
// document's ID is derived from ticketID+thread so repeated calls from any
// instance return the same Session.
//
// The (session, created) tuple distinguishes "found existing" from "created
// new" so callers that need to emit "starting a new investigation" notices
// only do so on first mention.
func (r *Resolver) ResolveSlackSession(
	ctx context.Context,
	ticketID *types.TicketID,
	thread slackModel.Thread,
	userID types.UserID,
) (sess *sessModel.Session, created bool, err error) {
	id := deriveSlackSessionID(ticketID, thread)

	existing, err := r.repo.GetSession(ctx, id)
	if err != nil {
		return nil, false, goerr.Wrap(err, "failed to look up Slack session",
			goerr.V("session_id", id))
	}
	if existing != nil {
		return existing, false, nil
	}

	ch := thread
	newSess := sessModel.NewSessionV2(ctx, id,
		sessModel.SessionSourceSlack,
		ticketID,
		&sessModel.ChannelRef{SlackThread: &ch},
		userID,
	)
	if err := r.repo.CreateSession(ctx, newSess); err != nil {
		if errors.Is(err, interfaces.ErrSessionAlreadyExists) {
			// Another instance won the race; read its write.
			peer, getErr := r.repo.GetSession(ctx, id)
			if getErr != nil {
				return nil, false, goerr.Wrap(getErr, "failed to read Slack session after AlreadyExists",
					goerr.V("session_id", id))
			}
			if peer == nil {
				return nil, false, goerr.New("Slack session disappeared after AlreadyExists",
					goerr.V("session_id", id))
			}
			return peer, false, nil
		}
		return nil, false, goerr.Wrap(err, "failed to create Slack session",
			goerr.V("session_id", id))
	}

	return newSess, true, nil
}

// CreateFreshSession creates a new Session with a random ID. This is the
// Web/CLI path: each invocation is independent, so no deterministic lookup
// is needed. ticketID is required for Web/CLI flows.
func (r *Resolver) CreateFreshSession(
	ctx context.Context,
	ticketID types.TicketID,
	source sessModel.SessionSource,
	userID types.UserID,
) (*sessModel.Session, error) {
	if source == sessModel.SessionSourceSlack {
		return nil, goerr.New("CreateFreshSession should not be used for Slack; call ResolveSlackSession instead")
	}
	tid := ticketID
	sess := sessModel.NewSessionV2(ctx, "", source, &tid, nil, userID)
	if err := r.repo.PutSession(ctx, sess); err != nil {
		return nil, goerr.Wrap(err, "failed to put fresh session",
			goerr.V("session_id", sess.ID))
	}
	return sess, nil
}

// deriveSlackSessionID returns a deterministic Session ID for a Slack
// thread. Different Tickets may share a channel/thread pair (rare, but
// allowed), so ticketID is part of the hash input. Ticketless sessions use
// a separate prefix so they never collide with ticketed ones.
//
// The hash is truncated to 16 hex chars (64 bits). Collision probability
// for a single Warren deployment's thread space is negligibly small.
func deriveSlackSessionID(ticketID *types.TicketID, thread slackModel.Thread) types.SessionID {
	h := sha256.New()
	h.Write([]byte(thread.ChannelID))
	h.Write([]byte{0})
	h.Write([]byte(thread.ThreadID))
	suffix := hex.EncodeToString(h.Sum(nil))[:16]

	if ticketID == nil {
		return types.SessionID("slack_ticketless_" + suffix)
	}
	// Mix ticketID into the hash so the same thread under a different
	// Ticket (e.g. thread re-used) gets a different Session.
	h2 := sha256.New()
	h2.Write([]byte(ticketID.String()))
	h2.Write([]byte{0})
	h2.Write([]byte(thread.ChannelID))
	h2.Write([]byte{0})
	h2.Write([]byte(thread.ThreadID))
	return types.SessionID("slack_" + hex.EncodeToString(h2.Sum(nil))[:16])
}
