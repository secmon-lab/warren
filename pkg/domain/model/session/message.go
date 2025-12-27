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
	MessageTypeTrace    MessageType = "trace"    // Progress messages displayed in context block
	MessageTypePlan     MessageType = "plan"     // Plan messages displayed in context block
	MessageTypeResponse MessageType = "response" // Final response displayed in normal block
	MessageTypeWarning  MessageType = "warning"  // Warning messages displayed in normal block
)

// Message represents a message recorded during a chat session
type Message struct {
	ID        types.MessageID `firestore:"id" json:"id"`
	SessionID types.SessionID `firestore:"session_id" json:"session_id"`
	Type      MessageType     `firestore:"type" json:"type"`       // Type only affects Slack display method
	Content   string          `firestore:"content" json:"content"` // All stored as plain text
	CreatedAt time.Time       `firestore:"created_at" json:"created_at"`
	UpdatedAt time.Time       `firestore:"updated_at" json:"updated_at"`
}

// NewMessage creates a new session message
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
