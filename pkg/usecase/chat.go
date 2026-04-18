package usecase

import (
	"context"
	_ "embed"
	"errors"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/tool/base"
	knowledgeTool "github.com/secmon-lab/warren/pkg/tool/knowledge"
	chatpkg "github.com/secmon-lab/warren/pkg/usecase/chat"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
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
//
// chat-session-redesign Phase 2 wrapping: before delegating to ChatUC.Execute,
// the Slack Session is resolved (or created with a deterministic ID so
// repeated mentions on the same thread reuse it), the Session activity lock
// is acquired so simultaneous mentions are serialized, and a Turn is
// recorded for the mention. The existing Execute code path is retained; it
// still creates its internal "session.Session" (legacy) for backwards
// compatibility with aster/bluebell, and Phase 3+ will unify the two.
//
// If the Session lock is already held when a new mention arrives, the
// function posts a short "please wait" context block to the thread and
// returns without starting a second AI run.
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

	var ticketIDPtr *types.TicketID
	if existingTicket != nil {
		tid := existingTicket.ID
		ticketIDPtr = &tid
	}

	// Resolve (or create) the Slack Session + Turn + Lock. This wraps the
	// existing Execute call so that lock contention is enforced; the legacy
	// Session handling inside Execute is untouched.
	slackSess, _, err := x.sessionResolver.ResolveSlackSession(ctx, ticketIDPtr, slackMsg.Thread(), types.UserID(slackUserID(slackMsg)))
	if err != nil {
		logger.Warn("failed to resolve slack session; continuing without session lock", "error", err)
	}

	var releaseLock func()
	if slackSess != nil {
		lock, acquired, lockErr := x.lockService.TryAcquire(ctx, slackSess.ID)
		if lockErr != nil {
			logger.Warn("failed to acquire slack session lock; continuing", "error", lockErr)
		} else if !acquired {
			// Another mention is already running on this thread.
			if x.slackService != nil {
				threadSvc := x.slackService.NewThread(slackMsg.Thread())
				if postErr := threadSvc.PostContextBlock(ctx, "⏳ Another chat is still running on this thread. Please wait for it to finish."); postErr != nil {
					errutil.Handle(ctx, goerr.Wrap(postErr, "failed to post busy notice"))
				}
			}
			return nil
		} else if lock != nil {
			turn := sessModel.NewTurn(ctx, slackSess.ID)
			if putErr := x.repository.PutTurn(ctx, turn); putErr != nil {
				errutil.Handle(ctx, goerr.Wrap(putErr, "failed to persist slack turn"))
			}
			releaseLock = func() {
				// Close the Turn first so observers see terminal state
				// before the lock is released.
				turn.Close(ctx, sessModel.TurnStatusCompleted)
				if updErr := x.repository.UpdateTurnStatus(ctx, turn.ID, turn.Status, turn.EndedAt); updErr != nil {
					errutil.Handle(ctx, goerr.Wrap(updErr, "failed to update slack turn status"))
				}
				if relErr := lock.Release(ctx); relErr != nil {
					if !errors.Is(relErr, interfaces.ErrLockNotHeld) {
						errutil.Handle(ctx, goerr.Wrap(relErr, "failed to release slack session lock"))
					}
				}
			}
		}
	}
	if releaseLock != nil {
		defer releaseLock()
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

// slackUserID extracts the sender's Slack user id for Session attribution.
// Returns an empty string when the message carries no user (rare, e.g.
// system-generated events).
func slackUserID(m *slack.Message) string {
	if m == nil {
		return ""
	}
	if u := m.User(); u != nil {
		return u.ID
	}
	return ""
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
