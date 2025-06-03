package usecase

import (
	"context"
	_ "embed"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/tool/base"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

//go:embed prompt/chat_system_prompt.md
var chatSystemPromptTemplate string

func (x *UseCases) chat(ctx context.Context, target *ticket.Ticket, message string) error {
	logger := logging.From(ctx)

	// Create Slack update callback function
	slackUpdateFunc := func(ctx context.Context, ticket *ticket.Ticket) error {
		if x.slackService == nil {
			return nil // Skip if Slack service is not configured
		}

		if ticket.SlackThread == nil {
			return nil // Skip if ticket has no Slack thread
		}

		if ticket.Finding == nil {
			return nil // Skip if ticket has no finding
		}

		threadSvc := x.slackService.NewThread(*ticket.SlackThread)
		return threadSvc.PostFinding(ctx, *ticket.Finding)
	}

	baseAction := base.New(x.repository, x.policyClient, target.ID, base.WithSlackUpdate(slackUpdateFunc))
	tools := append(x.tools, baseAction)

	storageSvc := storage.New(x.storageClient, storage.WithPrefix(x.storagePrefix))

	historyRecord, err := x.repository.GetLatestHistory(ctx, target.ID)
	if err != nil {
		return goerr.Wrap(err, "failed to get latest history")
	}

	var history *gollem.History
	if historyRecord != nil {
		history, err = storageSvc.GetHistory(ctx, target.ID, historyRecord.ID)
		if err != nil {
			return goerr.Wrap(err, "failed to get history data")
		}
	}

	alerts, err := x.repository.BatchGetAlerts(ctx, target.AlertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}

	showAlerts := alerts[:]
	if len(showAlerts) > 3 {
		showAlerts = showAlerts[:3]
	}

	systemPrompt, err := prompt.Generate(ctx, chatSystemPromptTemplate, map[string]any{
		"ticket": target,
		"alerts": showAlerts,
		"total":  len(alerts),
	})
	if err != nil {
		return goerr.Wrap(err, "failed to build system prompt")
	}

	agent := gollem.New(x.llmClient,
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
			ctx = msg.Trace(ctx, "⚡ Execute Tool: `%s`", tool.Name)
			for k, v := range tool.Arguments {
				ctx = msg.Trace(ctx, "  ▶️ `%s`: `%v`", k, v)
			}
			return nil
		}),
	)

	logger.Debug("run prompt", "prompt", message, "history", history, "ticket", target, "history_record", historyRecord)

	newHistory, err := agent.Prompt(ctx, message)
	if err != nil {
		return goerr.Wrap(err, "failed to prompt")
	}

	newRecord := ticket.NewHistory(ctx, target.ID)

	if err = storageSvc.PutHistory(ctx, target.ID, newRecord.ID, newHistory); err != nil {
		return goerr.Wrap(err, "failed to put history")
	}

	if err = x.repository.PutHistory(ctx, target.ID, &newRecord); err != nil {
		return goerr.Wrap(err, "failed to put history")
	}

	return nil
}
