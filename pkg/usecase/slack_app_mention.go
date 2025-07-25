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

// HandleSlackAppMention handles a slack app mention event. It will dispatch a slack action to the alert.
func (uc *UseCases) HandleSlackAppMention(ctx context.Context, slackMsg slack.Message) error {
	logger := logging.From(ctx)
	logger.Debug("slack app mention event", "mention", slackMsg.Mention(), "slack_thread", slackMsg.Thread())

	threadSvc := uc.slackNotifier.NewThread(slackMsg.Thread())
	ctx = msg.WithUpdatable(ctx, threadSvc.Reply, threadSvc.NewStateFunc, threadSvc.NewUpdatableMessage)
	if slackMsg.User() != nil {
		ctx = user.WithUserID(ctx, slackMsg.User().ID)
	}

	// Nothing to do
	for i, mention := range slackMsg.Mention() {
		if !uc.slackNotifier.IsBotUser(mention.UserID) {
			continue
		}

		// Set user ID in context for activity tracking

		// Try to parse message as command when it's first mention.
		if i == 0 && len(mention.Message) > 0 {
			// Only execute commands if Slack is enabled (commands require real Slack service)
			if uc.IsSlackEnabled() {
				// Cast to concrete type for command service (commands require real Slack functionality)
				if adapter, ok := uc.slackNotifier.(*slackNotifierAdapter); ok {
					concreteThreadSvc := adapter.service.NewThread(slackMsg.Thread())
					cmdSvc := command.New(uc.repository, uc.llmClient, concreteThreadSvc)
					if err := cmdSvc.Execute(ctx, &slackMsg, mention.Message); err != nil {
						// If errUnknownCommand, it will be falled thorugh.
						if !errors.Is(err, command.ErrUnknownCommand) {
							return goerr.Wrap(err, "failed to handle slack root command")
						}
					} else {
						// If no error in command processor, the mention has been proceeded.
						continue
					}
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

		// Pass user-enriched context to chat function
		return uc.Chat(ctx, ticket, mention.Message)
	}

	return nil
}
