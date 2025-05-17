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
	TicketStatusUnknown      TicketStatus = "unknown"
	TicketStatusNew          TicketStatus = "new"
	TicketStatusAcknowledged TicketStatus = "acked"
	TicketStatusBlocked      TicketStatus = "blocked"
	TicketStatusResolved     TicketStatus = "resolved"
)

var ticketstatusLabels = map[TicketStatus]string{
	TicketStatusUnknown:      "❓️ Unknown",
	TicketStatusNew:          "🆕 New",
	TicketStatusAcknowledged: "👀 Acknowledged",
	TicketStatusBlocked:      "🚫 Blocked",
	TicketStatusResolved:     "✅️ Resolved",
}

func (s TicketStatus) String() string {
	return string(s)
}

func (s TicketStatus) Label() string {
	return ticketstatusLabels[s]
}

func (s TicketStatus) Validate() error {
	switch s {
	case TicketStatusUnknown, TicketStatusNew, TicketStatusAcknowledged, TicketStatusBlocked, TicketStatusResolved:
		return nil
	}
	return goerr.New("invalid ticket Ticketstatus", goerr.V("Ticketstatus", s))
}
