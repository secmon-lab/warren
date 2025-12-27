package websocket

import (
	"encoding/json"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
)

// ChatMessage represents a message sent from client to server
type ChatMessage struct {
	Type      string `json:"type"`      // "message", "ping"
	Content   string `json:"content"`   // message content
	Timestamp int64  `json:"timestamp"` // unix timestamp
}

// ChatResponse represents a message sent from server to client
type ChatResponse struct {
	Type      string `json:"type"`                 // "message", "history", "status", "error", "pong", "trace", "warning"
	Content   string `json:"content"`              // message content
	User      *User  `json:"user,omitempty"`       // sender information
	Timestamp int64  `json:"timestamp"`            // unix timestamp
	MessageID string `json:"message_id,omitempty"` // unique message identifier
}

// User represents user information in WebSocket messages
type User struct {
	ID   string `json:"id"`   // user ID
	Name string `json:"name"` // display name
}

// ToBytes converts ChatResponse to JSON bytes
func (r *ChatResponse) ToBytes() ([]byte, error) {
	return json.Marshal(r)
}

// FromBytes parses JSON bytes to ChatMessage
func (m *ChatMessage) FromBytes(data []byte) error {
	return json.Unmarshal(data, m)
}

// NewChatResponse creates a new ChatResponse with current timestamp
func NewChatResponse(msgType, content string) *ChatResponse {
	return &ChatResponse{
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}
}

// NewMessageResponse creates a chat message response
func NewMessageResponse(content string, user *User) *ChatResponse {
	resp := NewChatResponse("message", content)
	resp.User = user
	return resp
}

// NewStatusResponse creates a status response
func NewStatusResponse(content string) *ChatResponse {
	return NewChatResponse("status", content)
}

// NewErrorResponse creates an error response
func NewErrorResponse(content string) *ChatResponse {
	return NewChatResponse("error", content)
}

// NewPongResponse creates a pong response
func NewPongResponse() *ChatResponse {
	return NewChatResponse("pong", "")
}

// NewTraceResponse creates a trace message response
func NewTraceResponse(content string, user *User) *ChatResponse {
	resp := NewChatResponse("trace", content)
	resp.User = user
	return resp
}

// NewWarningResponse creates a warning message response
func NewWarningResponse(content string, user *User) *ChatResponse {
	resp := NewChatResponse("warning", content)
	resp.User = user
	return resp
}

// UserFromSlackUser converts slack.User to websocket.User
func UserFromSlackUser(slackUser *slack.User) *User {
	if slackUser == nil {
		return nil
	}
	return &User{
		ID:   slackUser.ID,
		Name: slackUser.Name,
	}
}

// IsValidMessageType checks if message type is valid
func (m *ChatMessage) IsValidMessageType() bool {
	switch m.Type {
	case "message", "ping":
		return true
	default:
		return false
	}
}

// IsValidResponseType checks if response type is valid
func (r *ChatResponse) IsValidResponseType() bool {
	switch r.Type {
	case "message", "history", "status", "error", "pong", "trace", "warning":
		return true
	default:
		return false
	}
}
