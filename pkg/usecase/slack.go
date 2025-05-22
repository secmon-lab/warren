package usecase

import (
	"context"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/prompt"
	"github.com/secmon-lab/warren/pkg/service/command/list"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/tool/base"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/ptr"
)

// HandleSlackAppMention handles a slack app mention event. It will dispatch a slack action to the alert.
func (uc *UseCases) HandleSlackAppMention(ctx context.Context, slackMsg slack.Message) error {
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

		ticket, err := uc.repository.GetTicketByThread(ctx, slackMsg.Thread())
		if err != nil {
			return goerr.Wrap(err, "failed to get ticket by slack thread")
		}
		if ticket == nil {
			msg.Notify(ctx, "😣 Please create a ticket first. I will not work without a ticket.")
			return nil
		}

		input := uc.buildHandlePromptInput(ticket, mention.Message)
		return handlePrompt(ctx, input)
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
func (uc *UseCases) handleSlackInThreadCommand(ctx context.Context, th *slack.Thread, user slack.User, alertIDs []types.AlertID, message string) error {
	command, remaining := messageToArgs(message)
	if command == "" {
		return errUnknownCommand
	}

	switch command {
	case "resolve":
		if err := uc.resolveTicket(ctx, user, th, remaining); err != nil {
			return goerr.Wrap(err, "failed to run resolve command")
		}
		return nil

	default:
		return errUnknownCommand
	}
}
*/

func (uc *UseCases) handleSlackRootCommand(ctx context.Context, slackMsg slack.Message, message string) error {
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

	default:
		msg.Notify(ctx, "🤔 Available commands: `list`")
		return errUnknownCommand
	}
}

type handlePromptInput struct {
	Ticket        *ticket.Ticket
	Prompt        string
	LLMClient     interfaces.LLMClient
	Repo          interfaces.Repository
	StorageClient interfaces.StorageClient
	StoragePrefix string
	Tools         []gollem.ToolSet
	PolicyClient  interfaces.PolicyClient
}

func (uc *UseCases) buildHandlePromptInput(ticket *ticket.Ticket, p string) handlePromptInput {
	return handlePromptInput{
		Ticket:        ticket,
		Prompt:        p,
		LLMClient:     uc.llmClient,
		Repo:          uc.repository,
		StorageClient: uc.storageClient,
		StoragePrefix: uc.storagePrefix,
		Tools:         uc.tools,
		PolicyClient:  uc.policyClient,
	}
}

func handlePrompt(ctx context.Context, input handlePromptInput) error {
	logger := logging.From(ctx)

	baseAction := base.New(input.Repo, input.Ticket.AlertIDs, input.PolicyClient.Sources(), input.Ticket.ID)
	tools := append(input.Tools, baseAction)

	storageSvc := storage.New(input.StorageClient, storage.WithPrefix(input.StoragePrefix))

	historyRecord, err := input.Repo.GetLatestHistory(ctx, input.Ticket.ID)
	if err != nil {
		return goerr.Wrap(err, "failed to get latest history")
	}

	var history *gollem.History
	if historyRecord != nil {
		history, err = storageSvc.GetHistory(ctx, input.Ticket.ID, historyRecord.ID)
		if err != nil {
			return goerr.Wrap(err, "failed to get history data")
		}
	}

	alerts, err := input.Repo.BatchGetAlerts(ctx, input.Ticket.AlertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}

	systemPrompt, err := prompt.BuildSessionInitPrompt(ctx, alerts)
	if err != nil {
		return goerr.Wrap(err, "failed to build system prompt")
	}

	agent := gollem.New(input.LLMClient,
		gollem.WithHistory(history),
		gollem.WithToolSets(tools...),
		gollem.WithSystemPrompt(systemPrompt),
		gollem.WithResponseMode(gollem.ResponseModeBlocking),
		gollem.WithLogger(logging.From(ctx)),
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

	logger.Debug("run prompt", "prompt", input.Prompt, "history", history, "ticket", input.Ticket, "history_record", historyRecord)

	newHistory, err := agent.Prompt(ctx, input.Prompt)
	if err != nil {
		return goerr.Wrap(err, "failed to prompt")
	}

	newRecord := ticket.NewHistory(ctx, input.Ticket.ID)

	if err = storageSvc.PutHistory(ctx, input.Ticket.ID, newRecord.ID, newHistory); err != nil {
		return goerr.Wrap(err, "failed to put history")
	}

	if err = input.Repo.PutHistory(ctx, input.Ticket.ID, &newRecord); err != nil {
		return goerr.Wrap(err, "failed to put history")
	}

	return nil
}

// HandleSlackMessage handles a message from a slack user. It saves the message as an alert comment if the message is in the Alert thread.
func (uc *UseCases) HandleSlackMessage(ctx context.Context, slackMsg slack.Message) error {
	logger := logging.From(ctx)
	th := uc.slackService.NewThread(slackMsg.Thread())
	ctx = msg.With(ctx, th.Reply, th.NewStateFunc)

	// Skip if the message is from the bot
	if uc.slackService.IsBotUser(slackMsg.User().ID) {
		return nil
	}

	ticket, err := uc.repository.GetTicketByThread(ctx, slackMsg.Thread())
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket by slack thread")
	}
	if ticket == nil {
		logger.Info("ticket not found", "slack_thread", slackMsg.Thread())
		return nil
	}

	comment := ticket.NewComment(ctx, slackMsg)
	if err := uc.repository.PutTicketComment(ctx, comment); err != nil {
		msg.Trace(ctx, "💥 Failed to insert alert comment\n> %s", err.Error())
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

	return nil
}
