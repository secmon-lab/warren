package session

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
)

// Session represents a chat session with its execution status.
//
// Field evolution (chat-session-redesign spec): the following fields are
// retained for Phase 1-6 compatibility and will be removed in Phase 7:
//
//	RequestID, Status, Query, SlackURL, Intent, UpdatedAt
//
// The authoritative fields for the new model are Source / ChannelRef /
// LastActiveAt / Lock / TicketIDPtr, plus the existing ID / UserID /
// CreatedAt.
type Session struct {
	ID        types.SessionID     `firestore:"id" json:"id"`
	TicketID  types.TicketID      `firestore:"ticket_id" json:"ticket_id"`
	RequestID string              `firestore:"request_id" json:"request_id"`
	Status    types.SessionStatus `firestore:"status" json:"status"`
	UserID    types.UserID        `firestore:"user_id" json:"user_id"`
	Query     string              `firestore:"query" json:"query"`
	SlackURL  string              `firestore:"slack_url" json:"slack_url"`
	Intent    string              `firestore:"intent" json:"intent"`
	CreatedAt time.Time           `firestore:"created_at" json:"created_at"`
	UpdatedAt time.Time           `firestore:"updated_at" json:"updated_at"`

	// --- chat-session-redesign additions ---

	// Source identifies the channel the Session is bound to. Empty on data
	// written by pre-redesign code; readers should treat the empty value as
	// "unknown" and rely on migration to backfill.
	Source SessionSource `firestore:"source,omitempty" json:"source,omitempty"`

	// ChannelRef carries the external channel identifier for Slack
	// Sessions. nil for Web/CLI.
	ChannelRef *ChannelRef `firestore:"channel_ref,omitempty" json:"channel_ref,omitempty"`

	// TicketIDPtr allows nil (ticketless Slack Session). While the
	// non-nullable TicketID above remains for legacy writers, new code
	// should prefer reading/writing through TicketIDPtr. Phase 7 merges
	// these two into a single nullable field.
	TicketIDPtr *types.TicketID `firestore:"ticket_id_ptr,omitempty" json:"ticket_id_ptr,omitempty"`

	// Lock expresses the Slack Session's activity lock (nil = unlocked).
	// Always nil on Web and CLI Sessions, since those are born per-request.
	Lock *SessionLock `firestore:"lock,omitempty" json:"lock,omitempty"`

	// LastActiveAt records the moment the most recent Turn completed. Pure
	// informational; lock eligibility is judged from Lock.ExpiresAt, not
	// from this field.
	LastActiveAt time.Time `firestore:"last_active_at,omitzero" json:"last_active_at,omitzero"`
}

// NewSession creates a new session with running status.
//
// Deprecated: this constructor is the pre-redesign signature retained for
// Phase 1-6 compatibility. New code should use NewSessionV2, which names the
// redesign fields (Source / ChannelRef / TicketIDPtr). Both APIs populate
// the same Session struct; the only difference is which fields are set.
func NewSession(ctx context.Context, ticketID types.TicketID, userID types.UserID, query string, slackURL string) *Session {
	now := clock.Now(ctx)

	return &Session{
		ID:        types.NewSessionID(),
		TicketID:  ticketID,
		RequestID: request_id.FromContext(ctx),
		Status:    types.SessionStatusRunning,
		UserID:    userID,
		Query:     query,
		SlackURL:  slackURL,
		Intent:    "",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewSessionV2 constructs a Session using the chat-session-redesign model.
//
// The legacy fields (RequestID / Status / Query / SlackURL / Intent /
// UpdatedAt) are set to their zero values. Callers should prefer this
// constructor for new code paths.
//
// id may be empty, in which case a random SessionID is assigned. Providing
// id explicitly is how deterministic Slack Session IDs (see SessionResolver)
// are materialized.
func NewSessionV2(
	ctx context.Context,
	id types.SessionID,
	source SessionSource,
	ticketID *types.TicketID,
	channelRef *ChannelRef,
	userID types.UserID,
) *Session {
	now := clock.Now(ctx)
	if id == "" {
		id = types.NewSessionID()
	}
	s := &Session{
		ID:           id,
		UserID:       userID,
		CreatedAt:    now,
		LastActiveAt: now,
		Source:       source,
		ChannelRef:   channelRef,
		TicketIDPtr:  ticketID,
	}
	if ticketID != nil {
		s.TicketID = *ticketID
	}
	return s
}

// TicketIDOrNil returns the effective TicketID pointer, preferring TicketIDPtr
// but falling back to the legacy TicketID when TicketIDPtr is nil and
// TicketID is populated. This shim smooths the migration path.
func (s *Session) TicketIDOrNil() *types.TicketID {
	if s.TicketIDPtr != nil {
		return s.TicketIDPtr
	}
	if s.TicketID != "" {
		tid := s.TicketID
		return &tid
	}
	return nil
}

// TouchLastActive stamps LastActiveAt to the current clock.
func (s *Session) TouchLastActive(ctx context.Context) {
	s.LastActiveAt = clock.Now(ctx)
}

// UpdateStatus updates the session status
func (s *Session) UpdateStatus(ctx context.Context, status types.SessionStatus) {
	s.Status = status
	s.UpdatedAt = clock.Now(ctx)
}

// UpdateIntent updates the session intent
func (s *Session) UpdateIntent(ctx context.Context, intent string) {
	s.Intent = intent
	s.UpdatedAt = clock.Now(ctx)
}
