package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// HandleSlackMessage handles a message from a slack user. It saves the message as an alert comment if the message is in the Alert thread.
func (uc *UseCases) HandleSlackMessage(ctx context.Context, slackMsg slack.Message) error {
	logger := logging.From(ctx)
	th := uc.slackService.NewThread(slackMsg.Thread())
	ctx = msg.With(ctx, th.Reply, th.NewStateFunc)

	ticket, err := uc.repository.GetTicketByThread(ctx, slackMsg.Thread())
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket by slack thread")
	}
	if ticket == nil {
		logger.Info("ticket not found", "slack_thread", slackMsg.Thread())
		return nil
	}

	// For bot messages, only record if it's a Notify message (not Trace message)
	if slackMsg.User() != nil && uc.slackService.IsBotUser(slackMsg.User().ID) {
		// Bot messages should only be recorded if they are NOT Trace messages
		// Trace messages are identified by having ContextBlock in their structure

		logger.Debug("bot message", "slack_msg", slackMsg.LogValue())
		/*
			if slackMsg.IsTraceMessage() {
				return nil
			}
		*/
		return nil
	}

	comment := ticket.NewComment(ctx, slackMsg)
	if err := uc.repository.PutTicketComment(ctx, comment); err != nil {
		_ = msg.Trace(ctx, "💥 Failed to insert alert comment\n> %s", err.Error())
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

	return nil
}
