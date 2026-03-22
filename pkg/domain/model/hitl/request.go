package hitl

import (
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// RequestType identifies the kind of HITL interaction.
type RequestType string

const (
	// RequestTypeToolApproval is used when an agent tool requires human approval before execution.
	RequestTypeToolApproval RequestType = "tool_approval"
)

// Status represents the current state of a HITL request.
type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusDenied   Status = "denied"
)

// Request represents a human-in-the-loop interaction request.
// The model is generic: Payload and Response carry use-case-specific data.
type Request struct {
	ID          types.HITLRequestID `firestore:"ID"`
	SessionID   types.SessionID     `firestore:"SessionID"`
	Type        RequestType         `firestore:"Type"`
	Payload     map[string]any      `firestore:"Payload"`
	Status      Status              `firestore:"Status"`
	Response    map[string]any      `firestore:"Response"`
	UserID      string              `firestore:"UserID"`
	RespondedBy string              `firestore:"RespondedBy"`
	CreatedAt   time.Time           `firestore:"CreatedAt"`
	RespondedAt time.Time           `firestore:"RespondedAt"`
	SlackThread slack.Thread        `firestore:"SlackThread"`
	SlackMsgTS  string              `firestore:"SlackMsgTS"`
}

// ToolApprovalPayload is the typed payload for RequestTypeToolApproval.
type ToolApprovalPayload struct {
	ToolName string
	ToolArgs map[string]any
}

// NewToolApprovalPayload creates a Payload map for tool approval requests.
func NewToolApprovalPayload(toolName string, toolArgs map[string]any) map[string]any {
	return map[string]any{
		"tool_name": toolName,
		"tool_args": toolArgs,
	}
}

// ToolApproval extracts a ToolApprovalPayload from the request.
// Returns zero value if the payload keys are missing or have wrong types.
func (r *Request) ToolApproval() ToolApprovalPayload {
	p := ToolApprovalPayload{}
	if r.Payload == nil {
		return p
	}
	if v, ok := r.Payload["tool_name"].(string); ok {
		p.ToolName = v
	}
	if v, ok := r.Payload["tool_args"].(map[string]any); ok {
		p.ToolArgs = v
	}
	return p
}

// ResponseComment extracts the comment from the response map.
func (r *Request) ResponseComment() string {
	if r.Response == nil {
		return ""
	}
	if v, ok := r.Response["comment"].(string); ok {
		return v
	}
	return ""
}
