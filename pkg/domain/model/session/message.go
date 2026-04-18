package session

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

// MessageType represents the type of session message
type MessageType string

const (
	MessageTypeUser     MessageType = "user"     // chat-session-redesign: human user input (Slack/Web/CLI)
	MessageTypeTrace    MessageType = "trace"    // Progress messages displayed in context block
	MessageTypePlan     MessageType = "plan"     // Plan messages displayed in context block
	MessageTypeResponse MessageType = "response" // Final response displayed in normal block
	MessageTypeWarning  MessageType = "warning"  // Warning messages displayed in normal block
)

// Valid reports whether the MessageType is a known value.
func (t MessageType) Valid() bool {
	switch t {
	case MessageTypeUser, MessageTypeTrace, MessageTypePlan, MessageTypeResponse, MessageTypeWarning:
		return true
	}
	return false
}

// Message represents a message recorded during a chat session.
//
// chat-session-redesign fields (TicketID / TurnID / Author) are added
// alongside the legacy fields. Existing writers that use NewMessage keep
// working; new writers should prefer NewMessageV2.
type Message struct {
	ID        types.MessageID `firestore:"id" json:"id"`
	SessionID types.SessionID `firestore:"session_id" json:"session_id"`
	Type      MessageType     `firestore:"type" json:"type"`       // Type only affects Slack display method
	Content   string          `firestore:"content" json:"content"` // All stored as plain text
	CreatedAt time.Time       `firestore:"created_at" json:"created_at"`
	UpdatedAt time.Time       `firestore:"updated_at" json:"updated_at"`

	// TicketID denormalizes Session.TicketID onto each Message so Ticket-
	// level timeline queries do not require a per-Message Session join.
	// nil when the owning Session is ticketless.
	TicketID *types.TicketID `firestore:"ticket_id,omitempty" json:"ticket_id,omitempty"`

	// TurnID identifies which Turn (req/res cycle) produced this Message.
	// nil when the Message is not tied to an AI work unit (e.g. Slack
	// thread messages that were not @warren mentions).
	TurnID *types.TurnID `firestore:"turn_id,omitempty" json:"turn_id,omitempty"`

	// Author is required for Type=user and nil otherwise.
	Author *Author `firestore:"author,omitempty" json:"author,omitempty"`
}

// NewMessage creates a new session message using the legacy signature.
//
// Deprecated: prefer NewMessageV2 for new code. This shim sets Type/Content
// only; TicketID, TurnID, and Author remain nil.
func NewMessage(ctx context.Context, sessionID types.SessionID, msgType MessageType, content string) *Message {
	now := clock.Now(ctx)
	return &Message{
		ID:        types.NewMessageID(),
		SessionID: sessionID,
		Type:      msgType,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewMessageV2 creates a new Message with the chat-session-redesign fields.
//
// For Type=user, author must not be nil. For AI-produced types (trace / plan
// / response / warning), author should be nil. turnID may be nil for messages
// that are not tied to a Turn (e.g. Slack non-mention messages).
func NewMessageV2(
	ctx context.Context,
	sessionID types.SessionID,
	ticketID *types.TicketID,
	turnID *types.TurnID,
	msgType MessageType,
	content string,
	author *Author,
) *Message {
	now := clock.Now(ctx)
	return &Message{
		ID:        types.NewMessageID(),
		SessionID: sessionID,
		Type:      msgType,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
		TicketID:  ticketID,
		TurnID:    turnID,
		Author:    author,
	}
}
