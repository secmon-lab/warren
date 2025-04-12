package usecase

import (
	"context"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/action/base"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	session_model "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command/aggr"
	"github.com/secmon-lab/warren/pkg/service/command/list"
	session_svc "github.com/secmon-lab/warren/pkg/service/session"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/ptr"
)

// HandleSlackAppMention handles a slack app mention event. It will dispatch a slack action to the alert.
func (uc *UseCases) HandleSlackAppMention(ctx context.Context, user slack.User, mention slack.Mention, slackMsg *slack.Message) error {
	logger := logging.From(ctx)
	logger.Debug("slack app mention event", "mention", mention, "slack_thread", slackMsg.SlackThread())

	st := uc.slackService.NewThread(slackMsg.SlackThread())
	ctx = msg.With(ctx, st.Reply, st.NewStateFunc)

	// Nothing to do
	if !uc.slackService.IsBotUser(mention.UserID) {
		return nil
	}
	if len(mention.Message) == 0 {
		msg.Notify(ctx, "🤔 No message")
		return nil
	}

	if !slackMsg.InThread() {
		return uc.handleSlackRootCommand(ctx, st, user, mention.Message)
	}

	var targetAlertIDs []types.AlertID

	// If session is not found, starting a new session based on existing alert or list
	if v, err := uc.repository.GetAlertByThread(ctx, slackMsg.SlackThread()); err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	} else if v != nil {
		targetAlertIDs = []types.AlertID{v.ID}
	}

	if v, err := uc.repository.GetAlertListByThread(ctx, slackMsg.SlackThread()); err != nil {
		return goerr.Wrap(err, "failed to get alert list by slack thread")
	} else if v != nil {
		targetAlertIDs = v.AlertIDs
	}

	if len(targetAlertIDs) == 0 {
		msg.Notify(ctx, "🤔 No alerts found in the thread")
		return nil
	}

	// If alerts are found, dispatch the action to the existing alerts
	if err := uc.handleSlackInThreadCommand(ctx, st, user, targetAlertIDs, mention.Message); err == nil {
		// If the command is handled, return nil
		return nil
	} else if err != errUnknownCommand {
		// If the command is not valid, ignore the error and continue.
		return err
	}

	// If session is found, dispatch the action to the existing session
	ssn, err := uc.repository.GetSessionByThread(ctx, slackMsg.SlackThread())
	if err != nil {
		return goerr.Wrap(err, "failed to get session by slack thread")
	}
	if ssn == nil {
		ssn = session_model.New(ctx, &user, ptr.Ref(slackMsg.SlackThread()), targetAlertIDs)
		if err := uc.repository.PutSession(ctx, *ssn); err != nil {
			return goerr.Wrap(err, "failed to put session")
		}
	}

	// If session, alert and alert list are not found, call the command handler
	baseAction := base.New(uc.repository, targetAlertIDs, uc.policyClient.Sources(), ssn.ID)
	actionService, err := uc.actionSvc.With(ctx, baseAction)
	if err != nil {
		return goerr.Wrap(err, "failed to create action service")
	}

	svc := session_svc.New(uc.repository, uc.llmClient, actionService, ssn)
	if err := svc.Chat(ctx, mention.Message); err != nil {
		return goerr.Wrap(err, "failed to run session")
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

func (uc *UseCases) handleSlackRootCommand(ctx context.Context, th *slack_svc.ThreadService, user slack.User, message string) error {
	command, remaining := messageToArgs(message)
	if command == "" {
		return errUnknownCommand
	}

	switch command {
	case "list":
		_, err := list.New(uc.repository, uc.llmClient).Run(ctx, th, &user, remaining)
		if err != nil {
			return goerr.Wrap(err, "failed to run list command")
		}
		return nil

	default:
		msg.Notify(ctx, "🤔 Available commands: `list`")
		return errUnknownCommand
	}
}

// HandleSlackMessage handles a message from a slack user. It saves the message as an alert comment if the message is in the Alert thread.
func (uc *UseCases) HandleSlackMessage(ctx context.Context, slackMsg *slack.Message, text string, user slack.User, ts string) error {
	logger := logging.From(ctx)
	th := uc.slackService.NewThread(slackMsg.SlackThread())
	ctx = msg.With(ctx, th.Reply, th.NewStateFunc)

	// Skip if the message is from the bot
	if uc.slackService.IsBotUser(user.ID) {
		return nil
	}

	baseAlert, err := uc.repository.GetAlertByThread(ctx, slackMsg.SlackThread())
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}
	if baseAlert == nil {
		logger.Info("alert not found", "slack_thread", slackMsg.SlackThread())
		return nil
	}

	comment := alert.AlertComment{
		AlertID:   baseAlert.ID,
		Comment:   text,
		Timestamp: ts,
		User:      user,
	}
	if err := uc.repository.PutAlertComment(ctx, comment); err != nil {
		msg.Trace(ctx, "💥 Failed to insert alert comment\n> %s", err.Error())
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

	return nil
}
