package usecase

import (
	"context"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/action/base"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command/aggr"
	"github.com/secmon-lab/warren/pkg/service/command/list"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/ptr"
)

// HandleSlackAppMention handles a slack app mention event. It will dispatch a slack action to the alert.
func (uc *UseCases) HandleSlackAppMention(ctx context.Context, slackMsg *slack.Message) error {
	logger := logging.From(ctx)
	logger.Debug("slack app mention event", "mention", slackMsg.Mention(), "slack_thread", slackMsg.Thread())

	st := uc.slackService.NewThread(slackMsg.Thread())
	ctx = msg.With(ctx, st.Reply, st.NewStateFunc)

	// Nothing to do
	for _, mention := range slackMsg.Mention() {
		if !uc.slackService.IsBotUser(mention.UserID) {
			continue
		}

		if len(mention.Message) == 0 {
			msg.Notify(ctx, "🤔 No message")
			return nil
		}

		if !slackMsg.InThread() {
			return uc.handleSlackRootCommand(ctx, slackMsg, mention.Message)
		}

		ssn, err := createOrGetSession(ctx, uc.repository, slackMsg)
		if err != nil {
			return goerr.Wrap(err, "failed to create or get session")
		}
		if ssn == nil {
			return nil
		}

		// If session, alert and alert list are not found, call the command handler
		baseAction := base.New(uc.repository, ssn.AlertIDs, uc.policyClient.Sources(), ssn.ID)

		// TODO: call agent
		logger.Info("TODO: call agent", "ssn", ssn, "baseAction", baseAction)
	}

	return nil
}

var (
	errUnknownCommand = goerr.New("unknown command")
)

func messageToArgs(message string) (string, string) {
	args := strings.SplitN(message, " ", 2)
	if len(args) == 0 {
		return "", ""
	}
	if len(args) == 1 {
		return strings.ToLower(strings.TrimSpace(args[0])), ""
	}
	return strings.ToLower(strings.TrimSpace(args[0])), strings.TrimSpace(args[1])
}

func (uc *UseCases) handleSlackInThreadCommand(ctx context.Context, th *slack_svc.ThreadService, user slack.User, alertIDs []types.AlertID, message string) error {
	command, remaining := messageToArgs(message)
	if command == "" {
		return errUnknownCommand
	}

	switch command {
	case "aggr", "aggregate":
		if err := aggr.Run(ctx, uc.repository, th, uc.llmClient, user, alertIDs, remaining); err != nil {
			return goerr.Wrap(err, "failed to run aggregate command")
		}
		return nil

	default:
		return errUnknownCommand
	}
}

func (uc *UseCases) handleSlackRootCommand(ctx context.Context, slackMsg *slack.Message, message string) error {
	command, remaining := messageToArgs(message)
	if command == "" {
		return errUnknownCommand
	}

	threadSvc := uc.slackService.NewThread(slackMsg.Thread())

	switch command {
	case "list":
		_, err := list.New(uc.repository, uc.llmClient).Run(ctx, threadSvc, ptr.Ref(slackMsg.User()), remaining)
		if err != nil {
			return goerr.Wrap(err, "failed to run list command")
		}
		return nil

	case "chat":
		ssn, err := createSession(ctx, uc.repository, slackMsg)
		if err != nil {
			return goerr.Wrap(err, "failed to create or get session")
		}
		return uc.handleSlackChatCommand(ctx, threadSvc, ssn, remaining)

	default:
		msg.Notify(ctx, "🤔 Available commands: `list`")
		return errUnknownCommand
	}
}

func (uc *UseCases) handleSlackChatCommand(ctx context.Context, threadSvc *slack_svc.ThreadService, ssn *session.Session, message string) error {
	if ssn == nil {
		return nil
	}

	return nil
}

// HandleSlackMessage handles a message from a slack user. It saves the message as an alert comment if the message is in the Alert thread.
func (uc *UseCases) HandleSlackMessage(ctx context.Context, slackMsg *slack.Message) error {
	logger := logging.From(ctx)
	th := uc.slackService.NewThread(slackMsg.Thread())
	ctx = msg.With(ctx, th.Reply, th.NewStateFunc)

	// Skip if the message is from the bot
	if uc.slackService.IsBotUser(slackMsg.User().ID) {
		return nil
	}

	baseAlert, err := uc.repository.GetAlertByThread(ctx, slackMsg.Thread())
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}
	if baseAlert == nil {
		logger.Info("alert not found", "slack_thread", slackMsg.Thread())
		return nil
	}

	comment := alert.AlertComment{
		AlertID:   baseAlert.ID,
		Comment:   slackMsg.Text(),
		Timestamp: slackMsg.Timestamp(),
		User:      slackMsg.User(),
	}
	if err := uc.repository.PutAlertComment(ctx, comment); err != nil {
		msg.Trace(ctx, "💥 Failed to insert alert comment\n> %s", err.Error())
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

	return nil
}
