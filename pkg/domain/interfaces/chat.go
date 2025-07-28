package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

// ChatSessionID represents a unique chat session identifier
type ChatSessionID string

// ChatSource represents the type of chat source
type ChatSource string

const (
	ChatSourceSlack     ChatSource = "slack"
	ChatSourceWebSocket ChatSource = "websocket"
)

// Deprecated: ChatSession and ChatResponder are no longer used
// Chat message routing is now handled via msg.Notify and msg.Trace context functions

// ChatSession represents a specific chat session context
// Deprecated: Use msg.With to setup context-based message routing
type ChatSession struct {
	ID       ChatSessionID
	Source   ChatSource
	TicketID types.TicketID
	UserID   string
	// For WebSocket: specific connection ID
	// For Slack: thread information
	Metadata map[string]interface{}
}

// ChatResponder provides response capabilities for a specific chat session
// Deprecated: Use msg.With to setup context-based message routing instead
type ChatResponder interface {
	// SendMessage sends a message to the specific chat session
	SendMessage(ctx context.Context, message string) error

	// SendTrace sends a trace message to the specific chat session
	SendTrace(ctx context.Context, message string) error

	// GetSession returns the chat session information
	GetSession() *ChatSession
}

// Deprecated: ChatNotifier is no longer used in the new design
// Use ChatResponder for targeted responses to specific chat sessions
// ChatNotifier provides an abstraction for chat notification services
type ChatNotifier interface {
	// NotifyMessage sends a message to a chat room/channel
	NotifyMessage(ctx context.Context, ticketID types.TicketID, message string) error

	// NotifyTrace sends a trace message (debug/info level) to a chat room/channel
	NotifyTrace(ctx context.Context, ticketID types.TicketID, message string) error
}
