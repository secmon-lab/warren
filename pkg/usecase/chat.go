package usecase

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/strategy/planexec"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/tool/base"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

//go:embed prompt/chat_system_prompt.md
var chatSystemPromptTemplate string

// Chat processes a chat message for the specified ticket
// Message routing is handled via msg.Notify and msg.Trace functions in the context
func (x *UseCases) Chat(ctx context.Context, target *ticket.Ticket, message string) error {
	logger := logging.From(ctx)

	// Setup update function for findings - only depends on SlackNotifier for Slack updates
	slackUpdateFunc := func(ctx context.Context, ticket *ticket.Ticket) error {
		if !x.IsSlackEnabled() {
			return nil // Skip if Slack service is not configured
		}

		if !ticket.HasSlackThread() {
			return nil // Skip if ticket has no Slack thread
		}

		if ticket.Finding == nil {
			return nil // Skip if ticket has no finding
		}

		if x.slackService != nil {
			threadSvc := x.slackService.NewThread(*ticket.SlackThread)
			return threadSvc.PostFinding(ctx, ticket.Finding)
		}
		return nil // No slack service available
	}

	baseAction := base.New(x.repository, target.ID, base.WithSlackUpdate(slackUpdateFunc), base.WithLLMClient(x.llmClient))
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
			msg.Notify(ctx, "⚠️ Failed to load chat history, starting fresh: %s", err.Error())
			logger.Warn("failed to get history data, starting with new history", "error", err)
			history = nil // Start with new history
		} else {
			// Test if history is compatible with current gollem version
			if history != nil {
				// Try to validate history by attempting conversion
				if history.Version <= 0 || history.ToCount() <= 0 {
					msg.Notify(ctx, "⚠️ Chat history incompatible, starting fresh")
					logger.Warn("history version incompatible, starting with new history",
						"error", err,
						"version", history.Version,
						"history_id", historyRecord.ID)
					history = nil // Start with new history
				}
			}
		}
	}

	alerts, err := x.repository.BatchGetAlerts(ctx, target.AlertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}

	// Collect additional prompts from tools
	var toolPrompts []string
	for _, toolSet := range tools {
		if tool, ok := toolSet.(interfaces.Tool); ok {
			additionalPrompt, err := tool.Prompt(ctx)
			if err != nil {
				msg.Notify(ctx, "⚠️ Tool initialization warning: %s", err.Error())
				logger.Warn("failed to get prompt from tool", "tool", tool, "error", err)
				continue
			}
			if additionalPrompt != "" {
				toolPrompts = append(toolPrompts, additionalPrompt)
			}
		}
	}

	// Prepare additional instructions from tool prompts
	var additionalInstructions string
	if len(toolPrompts) > 0 {
		additionalInstructions = "# Available Tools and Resources\n\n" + strings.Join(toolPrompts, "\n\n")
	}

	// Generate system prompt first (before creating agent)
	systemPrompt, err := prompt.Generate(ctx, chatSystemPromptTemplate, map[string]any{
		"ticket":                  target,
		"total":                   len(alerts),
		"additional_instructions": additionalInstructions,
		"lang":                    lang.From(ctx),
	})
	if err != nil {
		return goerr.Wrap(err, "failed to build system prompt")
	}

	// Create hooks for plan progress tracking
	hooks := &chatPlanHooks{
		ctx: ctx,
	}

	// Create Plan & Execute strategy
	strategy := planexec.New(x.llmClient,
		planexec.WithHooks(hooks),
		planexec.WithMaxIterations(30),
	)

	// Get request ID from context
	requestID := request_id.FromContext(ctx)
	if requestID == "" {
		requestID = "unknown"
	}

	ctx = msg.NewTrace(ctx, "🚀 Starting... (Request ID: %s)", requestID)

	// Create agent with Strategy and Middleware
	agent := gollem.New(x.llmClient,
		gollem.WithStrategy(strategy),
		gollem.WithHistory(history),
		gollem.WithToolSets(tools...),
		gollem.WithResponseMode(gollem.ResponseModeBlocking),
		gollem.WithLogger(logging.From(ctx)),
		gollem.WithSystemPrompt(systemPrompt),
		gollem.WithContentBlockMiddleware(func(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
			return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
				resp, err := next(ctx, req)
				if err == nil && len(resp.Texts) > 0 {
					for _, text := range resp.Texts {
						msg.Trace(ctx, "💭 %s", text)
					}
				}
				return resp, err
			}
		}),
		gollem.WithToolMiddleware(func(next gollem.ToolHandler) gollem.ToolHandler {
			return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
				// Pre-execution: ツール呼び出しのトレース
				if !base.IgnorableTool(req.Tool.Name) {
					message := toolCallToText(ctx, x.llmClient, req.ToolSpec, req.Tool)
					msg.Trace(ctx, "🤖 %s", message)
					logger.Debug("execute tool", "tool", req.Tool.Name, "args", req.Tool.Arguments)
				}

				resp, err := next(ctx, req)

				// Post-execution: エラーハンドリング
				if resp != nil && resp.Error != nil {
					msg.Trace(ctx, "❌ Error: %s", resp.Error.Error())
					logger.Error("tool error", "error", resp.Error, "call", req.Tool)
				}

				return resp, err
			}
		}),
	)

	// Execute with Strategy
	result, err := agent.Execute(ctx, gollem.Text(message))
	if err != nil {
		msg.Notify(ctx, "💥 Execution failed: %s", err.Error())
		return goerr.Wrap(err, "failed to execute agent")
	}

	if hooks.planned {
		ctx = msg.Trace(ctx, "✅ Execution completed")
	}

	// Prepare Warren's final response message
	var warrenResponse string
	if result != nil && !result.IsEmpty() {
		warrenResponse = fmt.Sprintf("💬 %s", result.String())

		if x.slackService != nil && target.SlackThread != nil {
			// Set agent context for agent messages
			agentCtx := user.WithAgent(user.WithUserID(ctx, x.slackService.BotID()))

			// Record agent message as TicketComment
			// Create bot user for agent messages
			botUser := &slack.User{
				ID:   x.slackService.BotID(),
				Name: "Warren",
			}

			// Post agent message to Slack and get message ID
			threadSvc := x.slackService.NewThread(*target.SlackThread)
			logging.From(ctx).Debug("message notify", "from", "Agent", "msg", warrenResponse)
			ts, err := threadSvc.PostCommentWithMessageID(ctx, warrenResponse)
			if err != nil {
				errs.Handle(ctx, goerr.Wrap(err, "failed to post agent message to slack"))
			} else {
				comment := target.NewComment(agentCtx, warrenResponse, botUser, ts)

				if err := x.repository.PutTicketComment(agentCtx, comment); err != nil {
					errs.Handle(ctx, goerr.Wrap(err, "failed to save ticket comment"))
				}
			}

		} else {
			msg.Notify(ctx, "%s", warrenResponse)
		}
	}

	// Get the updated history from the agent's session
	session := agent.Session()
	if session == nil {
		logger.Warn("agent session is nil after execution")
		// Skip history saving when session is unavailable
		return nil
	}

	newHistory, err := session.History()
	if err != nil {
		return goerr.Wrap(err, "failed to get history from agent session")
	}
	if newHistory == nil {
		return goerr.New("history is nil after execution")
	}

	// Warren's response is automatically included in the agent session history
	logger.Debug("saving chat history with Warren's response",
		"warren_response", warrenResponse,
		"history_version", newHistory.Version)
	if newHistory.Version > 0 {
		newRecord := ticket.NewHistory(ctx, target.ID)

		if err := storageSvc.PutHistory(ctx, target.ID, newRecord.ID, newHistory); err != nil {
			msg.Notify(ctx, "💥 Failed to save chat history: %s", err.Error())
			return goerr.Wrap(err, "failed to put history")
		}

		if err := x.repository.PutHistory(ctx, target.ID, &newRecord); err != nil {
			msg.Notify(ctx, "💥 Failed to save chat record: %s", err.Error())
			return goerr.Wrap(err, "failed to put history")
		}

		logger.Debug("history saved", "history_id", newRecord.ID, "ticket_id", target.ID)
	}

	return nil
}

//go:embed prompt/tool_call_to_text.md
var toolCallToTextPromptTemplate string

//go:embed prompt/ticket_comment.md
var ticketCommentPromptTemplate string

func toolCallToText(ctx context.Context, llmClient gollem.LLMClient, spec *gollem.ToolSpec, call *gollem.FunctionCall) string {
	eb := goerr.NewBuilder(
		goerr.V("tool", call.Name),
		goerr.V("spec", spec),
	)
	defaultMsg := fmt.Sprintf("⚡ Execute Tool: `%s`", call.Name)
	if spec == nil {
		errs.Handle(ctx, eb.New("tool not found"))
		return defaultMsg
	}

	prompt, err := prompt.Generate(ctx, toolCallToTextPromptTemplate, map[string]any{
		"spec":      spec,
		"tool_call": call,
		"lang":      lang.From(ctx),
	})
	if err != nil {
		errs.Handle(ctx, eb.Wrap(err, "failed to generate prompt"))
		return defaultMsg
	}

	session, err := llmClient.NewSession(ctx)
	if err != nil {
		errs.Handle(ctx, eb.Wrap(err, "failed to create session"))
		return defaultMsg
	}

	response, err := session.GenerateContent(ctx, gollem.Text(prompt))
	if err != nil {
		errs.Handle(ctx, eb.Wrap(err, "failed to generate content"))
		return defaultMsg
	}

	if len(response.Texts) == 0 {
		errs.Handle(ctx, eb.New("no response"))
		return defaultMsg
	}

	return response.Texts[0]
}

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

	response, err := session.GenerateContent(ctx, gollem.Text(commentPrompt))
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate comment")
	}

	if len(response.Texts) == 0 {
		return "", goerr.New("no comment generated by LLM")
	}

	return response.Texts[0], nil
}

// chatPlanHooks implements planexec.PlanExecuteHooks for chat progress tracking
type chatPlanHooks struct {
	ctx     context.Context
	planned bool
}

var _ planexec.PlanExecuteHooks = &chatPlanHooks{}

func (h *chatPlanHooks) OnPlanCreated(ctx context.Context, plan *planexec.Plan) error {
	h.planned = true
	return postPlanProgress(h.ctx, plan, "Plan created")
}

func (h *chatPlanHooks) OnPlanUpdated(ctx context.Context, plan *planexec.Plan) error {
	h.planned = true
	return postPlanProgress(h.ctx, plan, "Plan updated")
}

func (h *chatPlanHooks) OnTaskDone(ctx context.Context, plan *planexec.Plan, _ *planexec.Task) error {
	h.planned = true
	return postPlanProgress(h.ctx, plan, "Task done")
}

// postPlanProgress posts the plan progress as a new message (not an update)
func postPlanProgress(ctx context.Context, plan *planexec.Plan, action string) error {
	if len(plan.Tasks) == 0 {
		msg.Trace(ctx, "🤖 *%s* (no tasks yet)", action)
		return nil
	}

	completedCount := 0
	for _, task := range plan.Tasks {
		if task.State == planexec.TaskStateCompleted {
			completedCount++
		}
	}

	var messageBuilder strings.Builder
	messageBuilder.WriteString(fmt.Sprintf("🤖 *%s*\n\n", action))
	messageBuilder.WriteString(fmt.Sprintf("*Progress: %d/%d tasks completed*\n\n", completedCount, len(plan.Tasks)))

	for _, task := range plan.Tasks {
		var icon string
		var status string

		switch task.State {
		case planexec.TaskStatePending:
			icon = "☑️"
			status = task.Description
		case planexec.TaskStateInProgress:
			icon = "⟳"
			status = task.Description
		case planexec.TaskStateCompleted:
			icon = "✅"
			status = fmt.Sprintf("~%s~", task.Description)
		default:
			icon = "?"
			status = task.Description
		}

		messageBuilder.WriteString(fmt.Sprintf("%s %s\n", icon, status))
	}

	msg.Trace(ctx, "%s", messageBuilder.String())
	return nil
}
