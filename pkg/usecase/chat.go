package usecase

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
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
	"github.com/secmon-lab/warren/pkg/utils/user"
)

//go:embed prompt/chat_system_prompt.md
var chatSystemPromptTemplate string

func (x *UseCases) Chat(ctx context.Context, target *ticket.Ticket, message string) error {
	logger := logging.From(ctx)

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

		threadSvc := x.slackNotifier.NewThread(*ticket.SlackThread)
		return threadSvc.PostFinding(ctx, ticket.Finding)
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

	// Collect additional prompts from tools
	var toolPrompts []string
	for _, toolSet := range tools {
		if tool, ok := toolSet.(interfaces.Tool); ok {
			additionalPrompt, err := tool.Prompt(ctx)
			if err != nil {
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

	postWarrenMessage := func(ctx context.Context, message string) {
		if !x.IsSlackEnabled() || !target.HasSlackThread() {
			return
		}
		if strings.TrimSpace(message) == "" {
			return
		}

		// Set agent context for agent messages
		agentCtx := user.WithAgent(user.WithUserID(ctx, x.slackNotifier.BotID()))

		// Record agent message as TicketComment
		// Create bot user for agent messages
		botUser := &slack.User{
			ID:   x.slackNotifier.BotID(),
			Name: "Warren",
		}

		// Post agent message to Slack and get message ID
		threadSvc := x.slackNotifier.NewThread(*target.SlackThread)
		logging.From(ctx).Debug("message notify", "from", "MessageHook", "msg", message)
		ts, err := threadSvc.PostCommentWithMessageID(ctx, "💬 "+message)
		if err != nil {
			errs.Handle(ctx, goerr.Wrap(err, "failed to post agent message to slack"))
			return
		}

		comment := target.NewComment(agentCtx, message, botUser, ts)

		if err := x.repository.PutTicketComment(agentCtx, comment); err != nil {
			logger.Error("failed to record agent message as comment", "error", err)
			// Continue execution even if comment recording fails
		}
	}

	agent := gollem.New(x.llmClient,
		gollem.WithHistory(history),
		gollem.WithToolSets(tools...),
		gollem.WithResponseMode(gollem.ResponseModeBlocking),
		gollem.WithLogger(logging.From(ctx)),
		gollem.WithMessageHook(func(ctx context.Context, message string) error {
			msg.Trace(ctx, "💭 %s", message)
			return nil
		}),
		gollem.WithToolErrorHook(func(ctx context.Context, err error, call gollem.FunctionCall) error {
			msg.Trace(ctx, "❌ Error: %s", err.Error())
			logger.Error("tool error", "error", err, "call", call)
			return nil
		}),
		gollem.WithToolRequestHook(func(ctx context.Context, call gollem.FunctionCall) error {
			if base.IgnorableTool(call.Name) {
				return nil
			}

			message := toolCallToText(ctx, x.llmClient, findTool(ctx, tools, call.Name), &call)
			msg.Trace(ctx, "🤖 %s", message)
			logger.Debug("execute tool", "tool", call.Name, "args", call.Arguments)
			return nil
		}),
	)

	systemPrompt, err := prompt.Generate(ctx, chatSystemPromptTemplate, map[string]any{
		"ticket":                  target,
		"alerts":                  showAlerts,
		"total":                   len(alerts),
		"additional_instructions": additionalInstructions,
		"lang":                    lang.From(ctx),
		"exit_tool_name":          agent.Facilitator().Spec().Name,
	})
	if err != nil {
		return goerr.Wrap(err, "failed to build system prompt")
	}

	// logger.Debug("run prompt", "prompt", message, "history", history, "ticket", target, "history_record", historyRecord)

	// Create updatable message function for plan progress tracking
	progressUpdate := msg.NewUpdatable(ctx, "Initializing AI plan...")

	// Always use plan mode for comprehensive task handling
	plan, err := agent.Plan(ctx, message,
		gollem.WithPlanLanguage(lang.From(ctx).Name()),
		gollem.WithPlanSystemPrompt(systemPrompt),
		gollem.WithPlanPhaseSystemPrompt(func(_ context.Context, _ gollem.PlanPhaseType, _ *gollem.Plan) string {
			return "Use Slack style markdown format in message if you need to decorate text"
		}),
		gollem.WithPlanCreatedHook(func(ctx context.Context, plan *gollem.Plan) error {
			return updatePlanProgress(progressUpdate, plan, "Plan created")
		}),
		gollem.WithPlanToDoStartHook(func(ctx context.Context, plan *gollem.Plan, todo gollem.PlanToDo) error {
			msg.Trace(ctx, "🚀 Starting: %s", todo.Description)
			return nil
		}),
		gollem.WithPlanToDoCompletedHook(func(ctx context.Context, plan *gollem.Plan, todo gollem.PlanToDo) error {
			return updatePlanProgress(progressUpdate, plan, fmt.Sprintf("Completed: %s", todo.Description))
		}),
		gollem.WithPlanToDoUpdatedHook(func(ctx context.Context, plan *gollem.Plan, changes []gollem.PlanToDoChange) error {
			if len(changes) == 0 {
				return nil
			}

			msg.Trace(ctx, "📝 Plan updated (%d todos)", len(changes))
			return nil
		}),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to create plan")
	}

	ctx = msg.NewTrace(ctx, "🚀 Executing plan...")

	execResp, err := plan.Execute(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to execute plan")
	}

	if len(execResp) > 0 {
		postWarrenMessage(ctx, execResp)
	} else {
		msg.Notify(ctx, "✅ All task has been done")
	}

	// Count completed tasks
	todos := plan.GetToDos()
	completedCount := 0
	for _, todo := range todos {
		if todo.Completed {
			completedCount++
		}
	}
	ctx = msg.Trace(ctx, "✅ Plan execution completed (%d/%d tasks)", completedCount, len(todos))

	// Get the updated history from the plan's session
	session := plan.Session()
	if session == nil {
		logger.Warn("plan session is nil after execution")
		// Skip history saving when session is unavailable
		return nil
	}

	newHistory := session.History()
	if newHistory == nil {
		return goerr.New("failed to get history from plan session")
	}

	newRecord := ticket.NewHistory(ctx, target.ID)

	if err := storageSvc.PutHistory(ctx, target.ID, newRecord.ID, newHistory); err != nil {
		return goerr.Wrap(err, "failed to put history")
	}

	if err := x.repository.PutHistory(ctx, target.ID, &newRecord); err != nil {
		return goerr.Wrap(err, "failed to put history")
	}

	logger.Debug("history saved", "history_id", newRecord.ID, "ticket_id", target.ID)

	return nil
}

func findTool(ctx context.Context, toolSets []gollem.ToolSet, name string) *gollem.ToolSpec {
	for _, toolSet := range toolSets {
		specs, err := toolSet.Specs(ctx)
		if err != nil {
			continue
		}

		for _, tool := range specs {
			if tool.Name == name {
				return &tool
			}
		}
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

// updatePlanProgress formats and updates the plan progress message using an updatable message function
func updatePlanProgress(updateFunc func(ctx context.Context, msg string), plan *gollem.Plan, action string) error {
	todos := plan.GetToDos()
	if len(todos) == 0 {
		updateFunc(context.Background(), fmt.Sprintf("🤖 *%s* (no tasks yet)", action))
		return nil
	}

	// Count completed tasks
	completedCount := 0
	for _, todo := range todos {
		if todo.Completed {
			completedCount++
		}
	}

	// Build complete message with all task details
	var messageBuilder strings.Builder
	messageBuilder.WriteString(fmt.Sprintf("🤖 *%s*\n\n", action))
	messageBuilder.WriteString(fmt.Sprintf("*Progress: %d/%d tasks completed*\n\n", completedCount, len(todos)))

	// Add task list with status indicators
	for _, todo := range todos {
		var status string
		var icon string

		switch todo.Status {
		case "Pending":
			status = todo.Description
			icon = "☑️"
		case "Executing":
			status = todo.Description
			icon = "⟳"
		case "Completed":
			// Strike-through for completed tasks
			status = fmt.Sprintf("~~%s~~", todo.Description)
			icon = "✅"
		case "Failed":
			status = fmt.Sprintf("%s (FAILED)", todo.Description)
			icon = "❌"
		case "Skipped":
			status = fmt.Sprintf("~~%s~~ (skipped)", todo.Description)
			icon = "⏭"
		default:
			status = todo.Description
			icon = "?"
		}

		messageBuilder.WriteString(fmt.Sprintf("%s %s\n", icon, status))
	}

	updateFunc(context.Background(), messageBuilder.String())
	return nil
}
