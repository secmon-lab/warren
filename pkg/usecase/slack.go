package usecase

import (
	"context"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/command/list"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/tool/base"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/ptr"
)

// HandleSlackAppMention handles a slack app mention event. It will dispatch a slack action to the alert.
func (uc *UseCases) HandleSlackAppMention(ctx context.Context, slackMsg *slack.Message) error {
	logger := logging.From(ctx)
	logger.Debug("slack app mention event", "mention", slackMsg.Mention(), "slack_thread", slackMsg.Thread())

	threadSvc := uc.slackService.NewThread(slackMsg.Thread())
	ctx = msg.With(ctx, threadSvc.Reply, threadSvc.NewStateFunc)

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

		return uc.handlePrompt(ctx, threadSvc, ssn, mention.Message)
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

/*
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
*/

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
		return uc.handlePrompt(ctx, threadSvc, ssn, remaining)

	default:
		msg.Notify(ctx, "🤔 Available commands: `list`")
		return errUnknownCommand
	}
}

func (uc *UseCases) handlePrompt(ctx context.Context, threadSvc *slack_svc.ThreadService, ssn *session.Session, prompt string) error {
	baseAction := base.New(uc.repository, ssn.AlertIDs, uc.policyClient.Sources(), ssn.ID)

	agent := gollem.New(uc.llmClient,
		gollem.WithToolSets(baseAction),
		gollem.WithResponseMode(gollem.ResponseModeBlocking),
		gollem.WithLogger(logging.From(ctx)),
	)

	storageSvc := storage.New(uc.storageClient, storage.WithPrefix(uc.storagePrefix))

	historyRecord, err := uc.repository.GetLatestHistory(ctx, ssn.ID)
	if err != nil {
		return goerr.Wrap(err, "failed to get latest history")
	}

	history, err := storageSvc.GetHistory(ctx, ssn.ID, historyRecord.ID)
	if err != nil {
		return goerr.Wrap(err, "failed to get history data")
	}

	newHistory, err := agent.Prompt(ctx, prompt,
		gollem.WithHistory(history),
		gollem.WithMessageHook(func(ctx context.Context, message string) error {
			msg.Notify(ctx, "💬 %s", message)
			return nil
		}),
		gollem.WithToolRequestHook(func(ctx context.Context, tool gollem.FunctionCall) error {
			msg.Trace(ctx, "⚡ Execute Tool: `%s`", tool.Name)
			for k, v := range tool.Arguments {
				msg.Trace(ctx, "  ▶️ `%s`: `%v`", k, v)
			}
			return nil
		}),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to prompt")
	}

	newRecord := session.NewHistory(ctx, ssn.ID)

	if err = storageSvc.PutHistory(ctx, ssn.ID, newRecord.ID, newHistory); err != nil {
		return goerr.Wrap(err, "failed to put history")
	}

	if err = uc.repository.PutHistory(ctx, ssn.ID, newRecord); err != nil {
		return goerr.Wrap(err, "failed to put history")
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
