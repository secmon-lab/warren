package usecase

import (
	"context"
	_ "embed"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/tool/base"
	knowledgeTool "github.com/secmon-lab/warren/pkg/tool/knowledge"
	chatpkg "github.com/secmon-lab/warren/pkg/usecase/chat"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/slackctx"
)

// ChatFromWebSocket processes a chat message from a WebSocket connection.
// It fetches the ticket by ID, builds a ChatContext, and delegates to ChatUC.
func (x *UseCases) ChatFromWebSocket(ctx context.Context, ticketID types.TicketID, message string) error {
	t, err := x.repository.GetTicket(ctx, ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket")
	}
	if t == nil {
		return goerr.New("ticket not found")
	}

	chatCtx, err := x.buildChatContext(ctx, t, nil, message)
	if err != nil {
		return err
	}

	return x.ChatUC.Execute(ctx, message, chatCtx)
}

// ChatFromSlack processes a chat message from a Slack app mention.
// It resolves the ticket from the Slack thread (or creates a ticketless context),
// fetches Slack history, builds a ChatContext, and delegates to ChatUC.
func (x *UseCases) ChatFromSlack(ctx context.Context, slackMsg *slack.Message, mentionMessage string) error {
	logger := logging.From(ctx)

	// Get Slack history for conversation context
	var history []slack.HistoryMessage
	if x.slackService != nil {
		var err error
		history, err = x.slackService.GetMessageHistory(ctx, slackMsg)
		if err != nil {
			logger.Warn("failed to get slack history", "error", err)
		}
	}

	// Try to find existing ticket by thread
	existingTicket, err := x.repository.GetTicketByThread(ctx, slackMsg.Thread())
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket by slack thread")
	}

	if existingTicket != nil {
		chatCtx, err := x.buildChatContext(ctx, existingTicket, history, mentionMessage)
		if err != nil {
			return err
		}
		return x.ChatUC.Execute(ctx, mentionMessage, chatCtx)
	}

	// Ticketless chat
	ctx = slackctx.WithThread(ctx, slackMsg.Thread())
	thread := slackMsg.Thread()
	placeholderTicket := &ticket.Ticket{SlackThread: &thread}

	chatCtx, err := x.buildChatContext(ctx, placeholderTicket, history, mentionMessage)
	if err != nil {
		return err
	}
	return x.ChatUC.Execute(ctx, mentionMessage, chatCtx)
}

// ChatFromCLI processes a chat message from the CLI.
// The ticket is already constructed by the CLI layer.
func (x *UseCases) ChatFromCLI(ctx context.Context, t *ticket.Ticket, message string) error {
	chatCtx, err := x.buildChatContext(ctx, t, nil, message)
	if err != nil {
		return err
	}
	return x.ChatUC.Execute(ctx, message, chatCtx)
}

// buildChatContext fetches all data needed for chat execution and assembles a ChatContext.
func (x *UseCases) buildChatContext(ctx context.Context, t *ticket.Ticket, slackHistory []slack.HistoryMessage, _ string) (chatModel.ChatContext, error) {
	ticketless := t.ID == ""

	chatCtx := chatModel.ChatContext{
		Ticket:       t,
		SlackHistory: slackHistory,
	}

	// Fetch alerts (skip for ticketless)
	if !ticketless {
		alerts, err := x.repository.BatchGetAlerts(ctx, t.AlertIDs)
		if err != nil {
			return chatCtx, goerr.Wrap(err, "failed to get alerts")
		}
		chatCtx.Alerts = alerts
	}

	// Agent Memory search is removed — knowledge is now accessed via tool search (knowledge v2).

	// Build tools (convert interfaces.ToolSet to gollem.ToolSet for ChatContext)
	allTools := make([]gollem.ToolSet, 0, len(x.tools)+1)
	for _, ts := range x.tools {
		allTools = append(allTools, ts)
	}
	if !ticketless {
		slackUpdateFunc := func(ctx context.Context, updatedTicket *ticket.Ticket) error {
			if x.slackService == nil || !updatedTicket.HasSlackThread() || updatedTicket.Finding == nil {
				return nil
			}
			threadSvc := x.slackService.NewThread(*updatedTicket.SlackThread)
			return threadSvc.PostFinding(ctx, updatedTicket.Finding)
		}
		baseAction := base.New(x.repository, t.ID, base.WithSlackUpdate(slackUpdateFunc), base.WithLLMClient(x.llmClient))
		allTools = append(allTools, baseAction)
	}
	// Add knowledge search tool if knowledge service is configured
	if x.knowledgeSvc != nil {
		factTool := knowledgeTool.New(x.knowledgeSvc, types.KnowledgeCategoryFact, knowledgeTool.ModeReadOnly)
		allTools = append(allTools, factTool)
	}

	chatCtx.Tools = allTools

	// Collect thread comments (skip for ticketless)
	if !ticketless {
		chatCtx.ThreadComments = chatpkg.CollectThreadComments(ctx, x.repository, t.ID, nil)
	}

	// Load history (skip for ticketless)
	if !ticketless {
		storageSvc := storage.New(x.storageClient, storage.WithPrefix(x.storagePrefix))
		history, err := chatpkg.LoadHistory(ctx, x.repository, t.ID, storageSvc)
		if err != nil {
			return chatCtx, err
		}
		chatCtx.History = history
	}

	return chatCtx, nil
}

//go:embed prompt/ticket_comment.md
var ticketCommentPromptTemplate string

// generateInitialTicketComment generates an LLM-based initial comment for a ticket
func (x *UseCases) generateInitialTicketComment(ctx context.Context, ticketData *ticket.Ticket, alerts alert.Alerts) (string, error) {
	commentPrompt, err := prompt.GenerateWithStruct(ctx, ticketCommentPromptTemplate, map[string]any{
		"ticket": ticketData,
		"alerts": alerts,
		"lang":   lang.From(ctx),
	})
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate comment prompt")
	}

	session, err := x.llmClient.NewSession(ctx)
	if err != nil {
		return "", goerr.Wrap(err, "failed to create LLM session")
	}

	response, err := session.Generate(ctx, []gollem.Input{gollem.Text(commentPrompt)})
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate comment")
	}

	if len(response.Texts) == 0 {
		return "", goerr.New("no comment generated by LLM")
	}

	return response.Texts[0], nil
}
