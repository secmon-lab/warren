package ticket

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type Comment struct {
	ID        types.CommentID `json:"id"`
	TicketID  types.TicketID  `json:"ticket_id"`
	CreatedAt time.Time       `json:"created_at"`
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
		CreatedAt: clock.Now(ctx),
		Comment:   comment,
		User:      user,
	}
}
