package chat

import (
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
)

// ChatContext holds all the pre-fetched data needed for chat execution.
// It is built by the usecase layer (ChatFromXxx functions) and passed
// to the ChatUseCase.Execute implementation.
type ChatContext struct {
	Ticket         *ticket.Ticket
	Alerts         []*alert.Alert
	MemoryContext  string
	Tools          []gollem.ToolSet
	ThreadComments []ticket.Comment
	Knowledges     []*knowledge.Knowledge
	History        *gollem.History
	SlackHistory   []slack.HistoryMessage
}

// IsTicketless returns true if the chat has no associated ticket.
func (c *ChatContext) IsTicketless() bool {
	return c.Ticket == nil || c.Ticket.ID == ""
}
