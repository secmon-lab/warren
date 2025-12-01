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
	CreatedAt time.Time           `firestore:"created_at" json:"created_at"`
	UpdatedAt time.Time           `firestore:"updated_at" json:"updated_at"`
}

// NewSession creates a new session with running status
func NewSession(ctx context.Context, ticketID types.TicketID) *Session {
	now := clock.Now(ctx)
	requestID := request_id.FromContext(ctx)
	if requestID == "" {
		requestID = "unknown"
	}

	return &Session{
		ID:        types.NewSessionID(),
		TicketID:  ticketID,
		RequestID: requestID,
		Status:    types.SessionStatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// UpdateStatus updates the session status
func (s *Session) UpdateStatus(ctx context.Context, status types.SessionStatus) {
	s.Status = status
	s.UpdatedAt = clock.Now(ctx)
}
