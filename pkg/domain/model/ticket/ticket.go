package ticket

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type Ticket struct {
	ID          types.TicketID  `json:"id"`
	AlertIDs    []types.AlertID `json:"alert_ids"`
	SlackThread *slack.Thread   `json:"slack_thread"`

	Title       string                `json:"title"`
	Description string                `json:"description"`
	Status      types.TicketStatus    `json:"status"`
	Severity    types.AlertSeverity   `json:"severity"`
	Conclusion  types.AlertConclusion `json:"conclusion"`
	Reason      string                `json:"reason"`

	Finding  *Finding    `json:"finding"`
	Assignee *slack.User `json:"assignee"`
}

func New(ctx context.Context, alertIDs []types.AlertID, slackThread *slack.Thread) Ticket {
	return Ticket{
		ID:          types.NewTicketID(),
		AlertIDs:    alertIDs,
		SlackThread: slackThread,
	}
}

type Comment struct {
	ID        types.CommentID `json:"id"`
	TicketID  types.TicketID  `json:"ticket_id"`
	Timestamp time.Time       `json:"timestamp"`
	Comment   string          `json:"comment"`
	User      slack.User      `json:"user"`
}

func (x *Ticket) Validate() error {
	if err := x.Status.Validate(); err != nil {
		return goerr.Wrap(err, "invalid status")
	}
	return nil
}

func (x *Ticket) NewComment(ctx context.Context, comment string, user slack.User) Comment {
	return Comment{
		ID:        types.NewCommentID(),
		TicketID:  x.ID,
		Timestamp: clock.Now(ctx),
		Comment:   comment,
		User:      user,
	}
}

// Finding is the conclusion of the alert. This is set by the AI.
type Finding struct {
	Severity       types.AlertSeverity `json:"severity"`
	Summary        string              `json:"summary"`
	Reason         string              `json:"reason"`
	Recommendation string              `json:"recommendation"`
}

func (x *Finding) Validate() error {
	if err := x.Severity.Validate(); err != nil {
		return goerr.Wrap(err, "invalid severity")
	}
	return nil
}
