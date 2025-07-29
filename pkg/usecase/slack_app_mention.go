package usecase

import (
	"context"
	"errors"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/command"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// Note: slackResponder has been removed in favor of using msg.With context-based approach

// HandleSlackAppMention handles a slack app mention event. It will dispatch a slack action to the alert.
func (uc *UseCases) HandleSlackAppMention(ctx context.Context, slackMsg slack.Message) error {
	logger := logging.From(ctx)
	logger.Debug("slack app mention event", "mention", slackMsg.Mention(), "slack_thread", slackMsg.Thread())

	if uc.slackService == nil {
		return goerr.New("slack service not configured")
	}
	threadSvc := uc.slackService.NewThread(slackMsg.Thread())
	ctx = msg.WithUpdatable(ctx, threadSvc.Reply, threadSvc.NewStateFunc, threadSvc.NewUpdatableMessage)
	if slackMsg.User() != nil {
		ctx = user.WithUserID(ctx, slackMsg.User().ID)
	}

	// Nothing to do
	for i, mention := range slackMsg.Mention() {
		if !uc.slackService.IsBotUser(mention.UserID) {
			continue
		}

		// Set user ID in context for activity tracking

		// Try to parse message as command when it's first mention.
		if i == 0 && len(mention.Message) > 0 {
			// Only execute commands if Slack is enabled
			if uc.IsSlackEnabled() {
				// Create command service using SlackThreadService
				threadSvc := uc.slackService.NewThread(slackMsg.Thread())

				// We need to get access to concrete ThreadService for command package
				// This is the only remaining coupling point
				if err := uc.executeSlackCommand(ctx, &slackMsg, threadSvc, mention.Message); err != nil {
					// If errUnknownCommand, it will be fallen through.
					if !errors.Is(err, command.ErrUnknownCommand) {
						return goerr.Wrap(err, "failed to handle slack root command")
					}
				} else {
					// If no error in command processor, the mention has been proceeded.
					continue
				}
			}
		}

		if len(mention.Message) == 0 {
			msg.Notify(ctx, "Tell me what you want to do. 🙂")
			return nil
		}

		ticket, err := uc.repository.GetTicketByThread(ctx, slackMsg.Thread())
		if err != nil {
			return goerr.Wrap(err, "failed to get ticket by slack thread")
		}
		if ticket == nil {
			msg.Notify(ctx, "😣 Please create a ticket first. I will not work without a ticket.")
			return nil
		}

		// Setup Slack-specific message handlers using msg.With
		threadSvc := uc.slackService.NewThread(slackMsg.Thread())

		notifyFunc := func(ctx context.Context, message string) {
			if err := threadSvc.PostComment(ctx, message); err != nil {
				logging.From(ctx).Error("failed to post message to slack", "error", err)
			}
		}

		traceFunc := func(ctx context.Context, message string) func(context.Context, string) {
			return threadSvc.NewTraceMessage(ctx, message)
		}

		// Setup context with Slack-specific message handlers
		ctx = msg.With(ctx, notifyFunc, traceFunc)

		// Pass user-enriched context to chat function
		return uc.Chat(ctx, ticket, mention.Message)
	}

	return nil
}
