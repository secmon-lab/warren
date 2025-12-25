package session

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
)

// Session represents a chat session with its execution status
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
}

// NewSession creates a new session with running status
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
