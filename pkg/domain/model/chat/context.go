package chat

import (
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// ChatContext holds all the pre-fetched data needed for chat execution.
// It is built by the usecase layer (ChatFromXxx functions) and passed
// to the ChatUseCase.Execute implementation.
//
// chat-session-redesign Phase 5: the canonical timeline now lives on
// Session / SessionMessages. The pre-redesign ThreadComments field has
// been removed; prompts read SessionMessages via the buildChatContext
// population path. History / SlackHistory remain because gollem working
// memory and Slack channel history come from different sources and
// have no Session equivalent.
type ChatContext struct {
	Ticket       *ticket.Ticket
	Alerts       []*alert.Alert
	Tools        []gollem.ToolSet
	History      *gollem.History
	SlackHistory []slack.HistoryMessage

	// Session is the pre-resolved Slack / Web / CLI Session for this
	// chat invocation. nil during the migration period when legacy
	// paths have not yet been updated to populate it.
	Session *session.Session

	// SessionMessages is the time-ordered timeline of Messages on the
	// Session EXCLUDING the current user input (which has just been
	// persisted separately). Empty for fresh Web/CLI Sessions.
	SessionMessages []*session.Message

	// CurrentTurnID is the Turn being executed by this chat
	// invocation. When non-nil, the chat pipeline stamps every AI-
	// produced session.Message (response / plan / trace / warning)
	// with it so the Conversation timeline can bucket the entire
	// req/res cycle together.
	CurrentTurnID *types.TurnID

	// OnSessionEvent is invoked whenever the chat pipeline persists
	// or updates an AI-produced session.Message, so the WebSocket
	// handler can publish the matching Envelope to the bound client.
	// kind is one of "session_message_added" / "session_message_updated".
	// nil for callers that do not publish events (CLI, Slack) — the
	// persistence itself still happens.
	OnSessionEvent func(kind string, m *session.Message)

	// OnHITLEvent is invoked when the chat pipeline enters or leaves a
	// HITL pending state from a transport that renders HITL in the
	// WebSocket stream (Web UI). kind is "pending" or "resolved".
	// messageID optionally ties the prompt to an existing progress
	// Message so the UI can render approval/question UI in-place on
	// that row. nil for transports that present HITL out-of-band
	// (Slack uses in-thread blocks; CLI default-denies).
	OnHITLEvent func(kind string, req *hitl.Request, messageID string)
}

// IsTicketless returns true if the chat has no associated ticket.
func (c *ChatContext) IsTicketless() bool {
	return c.Ticket == nil || c.Ticket.ID == ""
}
