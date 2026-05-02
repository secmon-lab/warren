package session

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
)

// TurnStatus enumerates the lifecycle states of a Turn.
type TurnStatus string

const (
	// TurnStatusRunning means the AI is currently processing the request.
	TurnStatusRunning TurnStatus = "running"
	// TurnStatusCompleted means the Turn ended normally.
	TurnStatusCompleted TurnStatus = "completed"
	// TurnStatusAborted means the Turn ended abnormally (panic, cancellation,
	// lock release on crash recovery, etc.).
	TurnStatusAborted TurnStatus = "aborted"
)

// Valid reports whether the TurnStatus is a known value.
func (s TurnStatus) Valid() bool {
	switch s {
	case TurnStatusRunning, TurnStatusCompleted, TurnStatusAborted:
		return true
	}
	return false
}

// Turn represents a single AI request/response cycle inside a Session. A
// Session may contain many Turns (each Slack mention, each WebSocket
// chat_message, each CLI line in interactive mode).
//
// Messages that are produced as part of the AI work belonging to this Turn
// carry its TurnID. Messages that are not AI-driven (e.g. Slack thread
// messages that were not @warren mentions) carry a nil TurnID.
type Turn struct {
	ID        types.TurnID    `firestore:"id" json:"id"`
	SessionID types.SessionID `firestore:"session_id" json:"session_id"`
	RequestID string          `firestore:"request_id" json:"request_id"`
	Status    TurnStatus      `firestore:"status" json:"status"`
	Intent    string          `firestore:"intent" json:"intent"`
	StartedAt time.Time       `firestore:"started_at" json:"started_at"`
	EndedAt   *time.Time      `firestore:"ended_at,omitempty" json:"ended_at,omitempty"`
}

// NewTurn constructs a running Turn. RequestID is taken from ctx so log
// correlation works out of the box.
func NewTurn(ctx context.Context, sessionID types.SessionID) *Turn {
	return &Turn{
		ID:        types.NewTurnID(),
		SessionID: sessionID,
		RequestID: request_id.FromContext(ctx),
		Status:    TurnStatusRunning,
		StartedAt: clock.Now(ctx),
	}
}

// Close transitions the Turn to the given terminal status and sets EndedAt.
//
// Calling Close on an already-closed Turn is a no-op so that defer-based
// cleanup paths can safely call Close even when the success path has
// already closed the Turn.
func (t *Turn) Close(ctx context.Context, status TurnStatus) {
	if t.EndedAt != nil {
		return
	}
	if !status.Valid() || status == TurnStatusRunning {
		return
	}
	now := clock.Now(ctx)
	t.Status = status
	t.EndedAt = &now
}

// UpdateIntent sets the AI-judged intent for this Turn. Can be called
// multiple times; the latest value wins.
func (t *Turn) UpdateIntent(intent string) {
	t.Intent = intent
}
