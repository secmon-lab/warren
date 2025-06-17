package ticket

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type Comment struct {
	ID             types.CommentID `json:"id"`
	TicketID       types.TicketID  `json:"ticket_id"`
	CreatedAt      time.Time       `json:"created_at"`
	Comment        string          `json:"comment"`
	User           *slack.User     `json:"user"`
	SlackMessageID string          `json:"slack_message_id"`
	Prompted       bool            `json:"prompted"`
}

func (x *Ticket) NewComment(ctx context.Context, message string, user *slack.User, slackMessageID string) Comment {
	return Comment{
		ID:             types.NewCommentID(),
		TicketID:       x.ID,
		CreatedAt:      clock.Now(ctx),
		Comment:        message,
		User:           user,
		SlackMessageID: slackMessageID,
		Prompted:       false,
	}
}
