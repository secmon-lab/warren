package types

import (
	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
)

type TicketID string

func (x TicketID) String() string {
	return string(x)
}

func NewTicketID() TicketID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return TicketID(id.String())
}

func (x TicketID) Validate() error {
	if x == EmptyTicketID {
		return goerr.New("empty ticket ID")
	}
	if _, err := uuid.Parse(string(x)); err != nil {
		return goerr.Wrap(err, "invalid ticket ID format", goerr.V("id", x))
	}
	return nil
}

const (
	EmptyTicketID TicketID = ""
)

type CommentID string

func (x CommentID) String() string {
	return string(x)
}

func NewCommentID() CommentID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return CommentID(id.String())
}

type TicketStatus string

const (
	TicketStatusOpen     TicketStatus = "open"
	TicketStatusPending  TicketStatus = "pending"
	TicketStatusResolved TicketStatus = "resolved"
	TicketStatusArchived TicketStatus = "archived"
)

var ticketStatusLabels = map[TicketStatus]string{
	TicketStatusOpen:     "🔍 Open",
	TicketStatusPending:  "🕒 Pending",
	TicketStatusResolved: "✅️ Resolved",
	TicketStatusArchived: "📦 Archived",
}

var ticketStatusIcons = map[TicketStatus]string{
	TicketStatusOpen:     "🔍",
	TicketStatusPending:  "🕒",
	TicketStatusResolved: "✅️",
	TicketStatusArchived: "📦",
}

func (s TicketStatus) String() string {
	return string(s)
}

func (s TicketStatus) Label() string {
	return ticketStatusLabels[s]
}

func (s TicketStatus) Icon() string {
	return ticketStatusIcons[s]
}

func (s TicketStatus) Validate() error {
	switch s {
	case TicketStatusOpen, TicketStatusPending, TicketStatusResolved, TicketStatusArchived:
		return nil
	}
	return goerr.New("invalid ticket Ticketstatus", goerr.V("Ticketstatus", s))
}
