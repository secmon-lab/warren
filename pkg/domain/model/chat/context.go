package chat

import (
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
)

// ChatContext holds all the pre-fetched data needed for chat execution.
// It is built by the usecase layer (ChatFromXxx functions) and passed
// to the ChatUseCase.Execute implementation.
//
// chat-session-redesign Phase 5 (additive): the new Session /
// SessionMessages fields carry the pre-resolved Session and its Message
// timeline. Pre-redesign fields (ThreadComments / History / SlackHistory)
// stay populated by the existing buildChatContext so aster/bluebell
// prompts continue to work unchanged; Phase 5's prompt rewrite will
// migrate the templates to read from SessionMessages and drop the
// legacy fields.
type ChatContext struct {
	Ticket         *ticket.Ticket
	Alerts         []*alert.Alert
	Tools          []gollem.ToolSet
	ThreadComments []ticket.Comment
	History        *gollem.History
	SlackHistory   []slack.HistoryMessage

	// Session is the pre-resolved Slack / Web / CLI Session for this
	// chat invocation. nil during the migration period when legacy
	// paths have not yet been updated to populate it.
	Session *session.Session

	// SessionMessages is the time-ordered timeline of Messages on the
	// Session, excluding the current user input (which has just been
	// persisted separately). Empty for fresh Web/CLI Sessions.
	SessionMessages []*session.Message
}

// IsTicketless returns true if the chat has no associated ticket.
func (c *ChatContext) IsTicketless() bool {
	return c.Ticket == nil || c.Ticket.ID == ""
}
