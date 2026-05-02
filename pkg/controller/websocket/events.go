package websocket

// chat-session-redesign Phase 6: new WebSocket event envelope for
// Session / Turn / SessionMessage activity. The legacy ChatResponse
// shape (message / trace / warning / status) remains supported for the
// existing TicketChat.tsx client; Phase 6 frontend work migrates the UI
// to the events defined here and Phase 7 deletes the old shape.
//
// Events are delivered JSON-encoded over the existing per-ticket Hub
// broadcast channel. Clients discriminate on the "event" field and
// ignore unknown events.

import (
	"encoding/json"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// EventKind enumerates the event types the server emits.
type EventKind string

const (
	EventKindSessionMessageAdded   EventKind = "session_message_added"
	EventKindSessionMessageUpdated EventKind = "session_message_updated"
	EventKindSessionCreated        EventKind = "session_created"
	EventKindTurnStarted           EventKind = "turn_started"
	EventKindTurnEnded             EventKind = "turn_ended"
	EventKindHITLRequestPending    EventKind = "hitl_request_pending"
	EventKindHITLRequestResolved   EventKind = "hitl_request_resolved"
)

// Envelope is the common wire shape for redesign events. Only the
// fields relevant to the event kind are populated; all others are
// omitted to keep payloads compact.
type Envelope struct {
	Event     EventKind          `json:"event"`
	Timestamp time.Time          `json:"timestamp"`
	SessionID types.SessionID    `json:"session_id,omitempty"`
	TurnID    *types.TurnID      `json:"turn_id,omitempty"`
	Message   *MessageView       `json:"message,omitempty"`
	Session   *SessionView       `json:"session,omitempty"`
	Turn      *TurnView          `json:"turn,omitempty"`
	Status    session.TurnStatus `json:"status,omitempty"`
	Intent    string             `json:"intent,omitempty"`
	HITL      *HITLView          `json:"hitl,omitempty"`
}

// HITLView is the wire shape for an in-flight HITL request emitted by
// hitl_request_pending / hitl_request_resolved events. The Web UI uses
// this to render approval/question UI and to clear it once the request
// resolves.
type HITLView struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id"`
	Type      string         `json:"type"`
	Status    string         `json:"status"`
	UserID    string         `json:"user_id,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	Response  map[string]any `json:"response,omitempty"`
	// MessageID optionally binds the HITL prompt to an existing Web
	// progress message so the UI can render approval UI in-place
	// instead of floating it above the timeline.
	MessageID string `json:"message_id,omitempty"`
}

// MessageView is the wire shape for SessionMessage. Uses strings for
// IDs so the TypeScript client can treat everything as opaque.
type MessageView struct {
	ID        string          `json:"id"`
	SessionID string          `json:"session_id"`
	TurnID    *string         `json:"turn_id,omitempty"`
	Type      string          `json:"type"`
	Author    *session.Author `json:"author,omitempty"`
	Content   string          `json:"content"`
	CreatedAt time.Time       `json:"created_at"`
}

// SessionView is the wire shape for Session metadata emitted alongside
// session_created. Lock / TicketIDPtr are intentionally omitted; the
// frontend does not need to see internal state.
type SessionView struct {
	ID           string    `json:"id"`
	TicketID     *string   `json:"ticket_id,omitempty"`
	Source       string    `json:"source"`
	UserID       string    `json:"user_id"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
}

// TurnView is the wire shape for Turn metadata emitted alongside
// turn_started / turn_ended.
type TurnView struct {
	ID        string     `json:"id"`
	SessionID string     `json:"session_id"`
	Status    string     `json:"status"`
	Intent    string     `json:"intent,omitempty"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

// NewSessionMessageAddedEvent builds the envelope for a newly persisted
// Message. created_at / timestamp default to m.CreatedAt if zero.
func NewSessionMessageAddedEvent(m *session.Message) Envelope {
	ts := m.CreatedAt
	if ts.IsZero() {
		ts = time.Now()
	}
	env := Envelope{
		Event:     EventKindSessionMessageAdded,
		Timestamp: ts,
		SessionID: m.SessionID,
		TurnID:    m.TurnID,
		Message: &MessageView{
			ID:        string(m.ID),
			SessionID: string(m.SessionID),
			Type:      string(m.Type),
			Author:    m.Author,
			Content:   m.Content,
			CreatedAt: m.CreatedAt,
		},
	}
	if m.TurnID != nil {
		tid := string(*m.TurnID)
		env.Message.TurnID = &tid
	}
	return env
}

// NewSessionMessageUpdatedEvent builds the envelope for a Message whose
// content was replaced in-place (updatable trace / HITL target). The
// MessageView carries the *current* content; prior revisions live on
// the persisted Message and are not shipped over the wire to keep
// events small.
func NewSessionMessageUpdatedEvent(m *session.Message) Envelope {
	env := NewSessionMessageAddedEvent(m)
	env.Event = EventKindSessionMessageUpdated
	// Use UpdatedAt so ordering against later additions stays correct.
	if !m.UpdatedAt.IsZero() {
		env.Timestamp = m.UpdatedAt
	}
	return env
}

// NewHITLRequestPendingEvent builds the envelope emitted when a
// Web-facing HITL request enters pending state. messageID optionally
// binds the prompt to an existing progress message so the UI can
// render approval UI in-place.
func NewHITLRequestPendingEvent(sessionID types.SessionID, view *HITLView) Envelope {
	return Envelope{
		Event:     EventKindHITLRequestPending,
		Timestamp: time.Now(),
		SessionID: sessionID,
		HITL:      view,
	}
}

// NewHITLRequestResolvedEvent builds the envelope emitted when a HITL
// request transitions out of pending state. The Web UI uses this to
// clear the approval/question prompt.
func NewHITLRequestResolvedEvent(sessionID types.SessionID, view *HITLView) Envelope {
	return Envelope{
		Event:     EventKindHITLRequestResolved,
		Timestamp: time.Now(),
		SessionID: sessionID,
		HITL:      view,
	}
}

// NewSessionCreatedEvent builds the envelope for a newly created Session.
func NewSessionCreatedEvent(s *session.Session) Envelope {
	view := &SessionView{
		ID:           string(s.ID),
		Source:       string(s.Source),
		UserID:       string(s.UserID),
		CreatedAt:    s.CreatedAt,
		LastActiveAt: s.LastActiveAt,
	}
	if s.TicketIDPtr != nil {
		tid := string(*s.TicketIDPtr)
		view.TicketID = &tid
	} else if s.TicketID != "" {
		tid := string(s.TicketID)
		view.TicketID = &tid
	}
	return Envelope{
		Event:     EventKindSessionCreated,
		Timestamp: time.Now(),
		SessionID: s.ID,
		Session:   view,
	}
}

// NewTurnStartedEvent builds the envelope for a Turn that has just
// entered running state.
func NewTurnStartedEvent(t *session.Turn) Envelope {
	tid := t.ID
	return Envelope{
		Event:     EventKindTurnStarted,
		Timestamp: t.StartedAt,
		SessionID: t.SessionID,
		TurnID:    &tid,
		Turn: &TurnView{
			ID:        string(t.ID),
			SessionID: string(t.SessionID),
			Status:    string(t.Status),
			Intent:    t.Intent,
			StartedAt: t.StartedAt,
			EndedAt:   t.EndedAt,
		},
	}
}

// NewTurnEndedEvent builds the envelope for a Turn that has entered a
// terminal state (completed / aborted).
func NewTurnEndedEvent(t *session.Turn) Envelope {
	tid := t.ID
	ts := t.StartedAt
	if t.EndedAt != nil {
		ts = *t.EndedAt
	}
	return Envelope{
		Event:     EventKindTurnEnded,
		Timestamp: ts,
		SessionID: t.SessionID,
		TurnID:    &tid,
		Status:    t.Status,
		Intent:    t.Intent,
	}
}

// Marshal returns the JSON bytes for an envelope. Separate method so
// callers do not need to import encoding/json directly and to allow
// future transport instrumentation.
func (e Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
