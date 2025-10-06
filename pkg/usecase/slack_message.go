package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// HandleSlackMessage handles a message from a slack user. It saves the message as an alert comment if the message is in the Alert thread.
func (uc *UseCases) HandleSlackMessage(ctx context.Context, slackMsg slack.Message) error {
	logger := logging.From(ctx)
	if uc.slackService == nil {
		return goerr.New("slack service not configured")
	}
	th := uc.slackService.NewThread(slackMsg.Thread())
	ctx = msg.With(ctx, th.Reply, th.NewTraceMessage)

	// Set user ID in context for activity tracking and skip if the message is from the bot
	if slackMsg.User() != nil {
		ctx = user.WithUserID(ctx, slackMsg.User().ID)
		if uc.slackService.IsBotUser(slackMsg.User().ID) {
			return nil
		}
	}

	ticket, err := uc.repository.GetTicketByThread(ctx, slackMsg.Thread())
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket by slack thread")
	}
	if ticket == nil {
		logger.Info("ticket not found", "slack_thread", slackMsg.Thread())
		return nil
	}

	comment := ticket.NewComment(ctx, slackMsg.Text(), slackMsg.User(), slackMsg.ID())
	if err := uc.repository.PutTicketComment(ctx, comment); err != nil {
		_ = msg.Trace(ctx, "💥 Failed to insert alert comment\n> %s", err.Error())
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

	return nil
}
