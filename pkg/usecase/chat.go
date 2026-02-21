package usecase

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/strategy/planexec"
	"github.com/m-mizutani/gollem/trace"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/tool/base"
	knowledgeTool "github.com/secmon-lab/warren/pkg/tool/knowledge"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
	ssnutil "github.com/secmon-lab/warren/pkg/utils/session"
	"github.com/secmon-lab/warren/pkg/utils/slackctx"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

//go:embed prompt/chat_system_prompt.md
var chatSystemPromptTemplate string

var (
	// ErrSessionAborted is returned when a session is aborted by user request
	ErrSessionAborted = goerr.New("session aborted by user")
)

func (x *UseCases) setupChatMessageFuncs(ctx context.Context, repo interfaces.Repository, sess *session.Session, target *ticket.Ticket) (msg.NotifyFunc, msg.TraceFunc, msg.TraceFunc, msg.WarnFunc) {
	threadSvc := x.slackService.NewThread(*target.SlackThread)

	notifyFunc := func(ctx context.Context, message string) {
		// Save response message to repository
		m := session.NewMessage(ctx, sess.ID, session.MessageTypeResponse, message)
		if err := repo.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}
		// Post to Slack
		if err := threadSvc.PostComment(ctx, message); err != nil {
			errutil.Handle(ctx, err)
		}
	}

	// createUpdatableMessageFunc is a helper to reduce duplication between trace and plan funcs
	createUpdatableMessageFunc := func(msgType session.MessageType) msg.TraceFunc {
		var updateFunc func(context.Context, string)
		return func(ctx context.Context, message string) {
			// Save message to repository
			m := session.NewMessage(ctx, sess.ID, msgType, message)
			if err := repo.PutSessionMessage(ctx, m); err != nil {
				errutil.Handle(ctx, err)
			}

			// Initialize or update the Slack message
			if updateFunc == nil {
				updateFunc = threadSvc.NewUpdatableMessage(ctx, message)
			} else {
				updateFunc(ctx, message)
			}
		}
	}

	traceFunc := createUpdatableMessageFunc(session.MessageTypeTrace)
	planFunc := createUpdatableMessageFunc(session.MessageTypePlan)

	// warnFunc creates a new message block (like notifyFunc) instead of updating
	warnFunc := func(ctx context.Context, message string) {
		// Save warning message to repository
		m := session.NewMessage(ctx, sess.ID, session.MessageTypeWarning, message)
		if err := repo.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}
		// Post to Slack as a new comment
		if err := threadSvc.PostComment(ctx, message); err != nil {
			errutil.Handle(ctx, err)
		}
	}

	return notifyFunc, traceFunc, planFunc, warnFunc
}

// Chat processes a chat message for the specified ticket
// Message routing is handled via msg.Notify and msg.Trace functions in the context
func (x *UseCases) Chat(ctx context.Context, target *ticket.Ticket, message string) error {
	logger := logging.From(ctx)

	// Get user ID from context
	userID := types.UserID(user.FromContext(ctx))

	// Get slack URL from context using type-safe helper (will be set by HandleSlackAppMention)
	slackURL := slackctx.SlackURL(ctx)

	// Create and save session
	ssn := session.NewSession(ctx, target.ID, userID, message, slackURL)
	if err := x.repository.PutSession(ctx, ssn); err != nil {
		return goerr.Wrap(err, "failed to save session")
	}

	// Embed session ID in logger
	logger = logger.With("session_id", ssn.ID)
	ctx = logging.With(ctx, logger)

	// Setup message functions only if Slack is enabled
	planFunc := func(ctx context.Context, msg string) {
	}
	if x.slackService != nil && target.SlackThread != nil {
		notifyFunc, traceFunc, pf, warnFunc := x.setupChatMessageFuncs(ctx, x.repository, ssn, target)
		ctx = msg.With(ctx, notifyFunc, traceFunc, warnFunc)
		planFunc = pf

		// Post initial request ID message using planFunc
		requestID := request_id.FromContext(ctx)
		if requestID == "" {
			requestID = "unknown"
		}
		planFunc(ctx, fmt.Sprintf("🚀 Thinking... (Request ID: %s)", requestID))
	}

	// Create status check function and embed in context
	statusCheckFunc := func(ctx context.Context) error {
		s, err := x.repository.GetSession(ctx, ssn.ID)
		if err != nil {
			return goerr.Wrap(err, "failed to get session status")
		}
		if s != nil && s.Status == types.SessionStatusAborted {
			return ErrSessionAborted
		}
		return nil
	}
	ctx = ssnutil.WithStatusCheck(ctx, statusCheckFunc)

	// Track final session status (will be updated by execution flow)
	finalStatus := types.SessionStatusCompleted

	// Ensure session status is updated on completion or error
	defer func() {
		if r := recover(); r != nil {
			// If panic occurred, mark as aborted
			finalStatus = types.SessionStatusAborted
			// Update status before re-panicking
			ssn.UpdateStatus(ctx, finalStatus)
			if err := x.repository.PutSession(ctx, ssn); err != nil {
				logger.Error("failed to update session status on panic", "error", err, "status", finalStatus)
			}
			panic(r) // Re-panic
		}

		ssn.UpdateStatus(ctx, finalStatus)
		if err := x.repository.PutSession(ctx, ssn); err != nil {
			logger.Error("failed to update final session status", "error", err, "status", finalStatus)
		}

		// Post session actions (link + Resolve/Edit buttons) to Slack thread when session is completed
		if finalStatus == types.SessionStatusCompleted && x.slackService != nil && target.SlackThread != nil {
			threadSvc := x.slackService.NewThread(*target.SlackThread)

			// Build session URL (may be empty if frontendURL is not set)
			var sessionURL string
			if x.frontendURL != "" {
				sessionURL = fmt.Sprintf("%s/sessions/%s", x.frontendURL, ssn.ID)
			}

			// Get current ticket status to determine which buttons to show
			currentTicket, err := x.repository.GetTicket(ctx, target.ID)
			if err != nil {
				logger.Error("failed to get ticket for session actions", "error", err, "ticket_id", target.ID)
			} else if currentTicket != nil {
				if err := threadSvc.PostSessionActions(ctx, target.ID, currentTicket.Status, sessionURL); err != nil {
					logger.Error("failed to post session actions to Slack", "error", err, "session_id", ssn.ID)
				}
			}
		}
	}()

	// Authorize agent execution
	if err := x.authorizeAgentRequest(ctx, message); err != nil {
		// Provide detailed feedback to user via Slack
		if errors.Is(err, errAgentAuthPolicyNotDefined) {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nAgent execution policy is not defined. Please configure the `auth.agent` policy or use `--no-authorization` flag for development.\n\nSee: https://docs.warren.secmon-lab.com/policy.md#agent-execution-authorization")
		} else if errors.Is(err, errAgentAuthDenied) {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nYou are not authorized to execute agent requests. Please contact your administrator if you believe this is an error.")
		} else {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nFailed to check authorization. Please contact your administrator.")
			return goerr.Wrap(err, "failed to evaluate agent auth")
		}
		return nil
	}

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
				// Validate history: Version must be > 0 AND must have messages
				// Empty Messages array indicates corrupted or incomplete history
				if history.Version <= 0 || history.ToCount() <= 0 {
					msg.Notify(ctx, "⚠️ Chat history incompatible (version=%d, messages=%d), starting fresh", history.Version, history.ToCount())
					logger.Warn("history incompatible, starting with new history",
						"error", err,
						"version", history.Version,
						"message_count", history.ToCount(),
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

	// Fallback topic to schema if empty
	effectiveTopic := target.Topic
	if effectiveTopic == "" && len(alerts) > 0 {
		effectiveTopic = types.KnowledgeTopic(alerts[0].Schema)
		logger.Warn("ticket topic is empty, falling back to schema",
			"ticket_id", target.ID,
			"schema", alerts[0].Schema,
			"topic", effectiveTopic)
		msg.Notify(ctx, "⚠️ Ticket topic is empty, using schema `%s` as topic", alerts[0].Schema)
	}

	// Set topic for knowledge tool in x.tools if present
	for _, tool := range x.tools {
		if kt, ok := tool.(*knowledgeTool.Knowledge); ok {
			kt.SetTopic(effectiveTopic)
			defer kt.SetTopic("") // Reset after use
			logger.Debug("set topic for knowledge tool", "topic", effectiveTopic)
			break
		}
	}

	// Create knowledge tool with ticket topic for this chat session
	knowledgeTool := knowledgeTool.New(x.repository, effectiveTopic)
	tools := append(x.tools, baseAction, knowledgeTool)

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

	// Collect prompt hints from sub-agents
	for _, sa := range x.subAgents {
		if ph := sa.PromptHint(); ph != "" {
			toolPrompts = append(toolPrompts, ph)
		}
	}

	// Prepare additional instructions from tool prompts
	var additionalInstructions string
	if len(toolPrompts) > 0 {
		additionalInstructions = "# Available Tools and Resources\n\n" + strings.Join(toolPrompts, "\n\n")
	}

	// Get knowledges for the topic
	knowledges := []*knowledge.Knowledge{} // Initialize as empty slice, not nil
	if target.Topic != "" {
		var err error
		retrieved, err := x.repository.GetKnowledges(ctx, target.Topic)
		if err != nil {
			logger.Warn("failed to get knowledges", "error", err, "topic", target.Topic)
			// Continue with empty knowledges
		} else if retrieved != nil {
			knowledges = retrieved
		}
	}

	// Generate system prompt first (before creating agent)
	systemPrompt, err := prompt.GenerateWithStruct(ctx, chatSystemPromptTemplate, map[string]any{
		"ticket":                  target,
		"total":                   len(alerts),
		"additional_instructions": additionalInstructions,
		"knowledges":              knowledges,
		"topic":                   target.Topic,
		"lang":                    lang.From(ctx),
	})
	if err != nil {
		return goerr.Wrap(err, "failed to build system prompt")
	}

	// Get request ID from context
	requestID := request_id.FromContext(ctx)
	if requestID == "" {
		requestID = "unknown"
	}

	// Create hooks for plan progress tracking
	hooks := &chatPlanHooks{
		ctx:        ctx,
		planFunc:   planFunc,
		requestID:  requestID,
		session:    ssn,
		repository: x.repository,
	}

	// Create Plan & Execute strategy
	strategy := planexec.New(x.llmClient,
		planexec.WithHooks(hooks),
		planexec.WithMaxIterations(30),
	)

	// Extract inner gollem.SubAgents for passing to gollem API
	gollemSubAgents := make([]*gollem.SubAgent, len(x.subAgents))
	for i, sa := range x.subAgents {
		gollemSubAgents[i] = sa.Inner()
	}

	// Build agent options
	agentOpts := []gollem.Option{
		gollem.WithStrategy(strategy),
		gollem.WithHistory(history),
		gollem.WithToolSets(tools...),
		gollem.WithSubAgents(gollemSubAgents...),
		gollem.WithResponseMode(gollem.ResponseModeBlocking),
		gollem.WithLogger(logging.From(ctx)),
		gollem.WithSystemPrompt(systemPrompt),
	}

	// Configure trace recorder if trace repository is set
	if x.traceRepository != nil {
		recorder := trace.New(
			trace.WithTraceID(requestID),
			trace.WithRepository(x.traceRepository),
		)
		agentOpts = append(agentOpts, gollem.WithTrace(recorder))
	}

	// Add middleware options
	agentOpts = append(agentOpts,
		// Compaction middleware for automatic history compression
		gollem.WithContentBlockMiddleware(llm.NewCompactionMiddleware(x.llmClient, logging.From(ctx))),
		// Trace middleware for message display
		gollem.WithContentBlockMiddleware(
			func(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
				return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
					resp, err := next(ctx, req)
					if err == nil && len(resp.Texts) > 0 {
						for _, text := range resp.Texts {
							msg.Trace(ctx, "💭 %s", text)
						}
					}
					return resp, err
				}
			},
		),
		gollem.WithToolMiddleware(func(next gollem.ToolHandler) gollem.ToolHandler {
			return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
				// Check if session is aborted before executing tool
				if err := ssnutil.CheckStatus(ctx); err != nil {
					return &gollem.ToolExecResponse{
						Error: err,
					}, nil
				}

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

	// Create agent with Strategy and Middleware
	agent := gollem.New(x.llmClient, agentOpts...)

	// Execute with Strategy
	result, executionErr := agent.Execute(ctx, gollem.Text(message))
	if executionErr != nil {
		// Mark session as aborted on any error
		finalStatus = types.SessionStatusAborted

		// Check if error is due to session abort
		if errors.Is(executionErr, ErrSessionAborted) {
			msg.Notify(ctx, "🛑 Execution aborted by user request.")
			return nil // Don't treat abort as error
		}
		msg.Notify(ctx, "💥 Execution failed: %s", executionErr.Error())
		return goerr.Wrap(executionErr, "failed to execute agent")
	}

	if hooks.planned {
		msg.Trace(ctx, "✅ Execution completed")
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

			// Save response message to repository (must happen before Slack post)
			m := session.NewMessage(ctx, ssn.ID, session.MessageTypeResponse, warrenResponse)
			if err := x.repository.PutSessionMessage(ctx, m); err != nil {
				errutil.Handle(ctx, err)
			}

			// Post agent message to Slack and get message ID
			threadSvc := x.slackService.NewThread(*target.SlackThread)
			logging.From(ctx).Debug("message notify", "from", "Agent", "msg", warrenResponse)
			ts, err := threadSvc.PostCommentWithMessageID(ctx, warrenResponse)
			if err != nil {
				errutil.Handle(ctx, goerr.Wrap(err, "failed to post agent message to slack"))
			} else {
				comment := target.NewComment(agentCtx, warrenResponse, botUser, ts)

				if err := x.repository.PutTicketComment(agentCtx, comment); err != nil {
					logger := logging.From(agentCtx)
					if data, jsonErr := json.Marshal(comment); jsonErr == nil {
						logger.Error("failed to save ticket comment", "error", err, "comment", string(data))
					}
					errutil.Handle(ctx, goerr.Wrap(err, "failed to save ticket comment", goerr.V("comment", comment)))
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
		"history_version", newHistory.Version,
		"message_count", newHistory.ToCount())

	// Warn if history is empty but continue saving
	if newHistory.ToCount() <= 0 {
		logger.Warn("history has no messages, but saving anyway to maintain consistency",
			"version", newHistory.Version,
			"message_count", newHistory.ToCount(),
			"ticket_id", target.ID)
	}

	if newHistory.Version > 0 {
		newRecord := ticket.NewHistory(ctx, target.ID)

		if err := storageSvc.PutHistory(ctx, target.ID, newRecord.ID, newHistory); err != nil {
			msg.Notify(ctx, "💥 Failed to save chat history: %s", err.Error())
			return goerr.Wrap(err, "failed to put history")
		}

		if err := x.repository.PutHistory(ctx, target.ID, &newRecord); err != nil {
			logger := logging.From(ctx)
			if data, jsonErr := json.Marshal(&newRecord); jsonErr == nil {
				logger.Error("failed to save history", "error", err, "history", string(data))
			}
			msg.Notify(ctx, "💥 Failed to save chat record: %s", err.Error())
			return goerr.Wrap(err, "failed to put history", goerr.V("history", &newRecord))
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
		errutil.Handle(ctx, eb.New("tool not found"))
		return defaultMsg
	}

	prompt, err := prompt.Generate(ctx, toolCallToTextPromptTemplate, map[string]any{
		"spec":      spec,
		"tool_call": call,
		"lang":      lang.From(ctx),
	})
	if err != nil {
		errutil.Handle(ctx, eb.Wrap(err, "failed to generate prompt"))
		return defaultMsg
	}

	session, err := llmClient.NewSession(ctx)
	if err != nil {
		errutil.Handle(ctx, eb.Wrap(err, "failed to create session"))
		return defaultMsg
	}

	response, err := session.GenerateContent(ctx, gollem.Text(prompt))
	if err != nil {
		errutil.Handle(ctx, eb.Wrap(err, "failed to generate content"))
		return defaultMsg
	}

	if len(response.Texts) == 0 {
		errutil.Handle(ctx, eb.New("no response"))
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
	ctx        context.Context
	planned    bool
	planFunc   func(context.Context, string)
	requestID  string
	session    *session.Session
	repository interfaces.Repository
}

var _ planexec.PlanExecuteHooks = &chatPlanHooks{}

func (h *chatPlanHooks) OnPlanCreated(ctx context.Context, plan *planexec.Plan) error {
	// Check if session is aborted
	if err := ssnutil.CheckStatus(ctx); err != nil {
		return err
	}

	h.planned = len(plan.Tasks) > 0

	// Update session intent with plan goal
	if plan.Goal != "" && h.session != nil && h.repository != nil {
		h.session.UpdateIntent(ctx, plan.Goal)
		if err := h.repository.PutSession(ctx, h.session); err != nil {
			logging.From(ctx).Error("failed to update session intent", "error", err)
		}
	}

	return postPlanProgress(h.ctx, h.planFunc, h.requestID, plan)
}

func (h *chatPlanHooks) OnPlanUpdated(ctx context.Context, plan *planexec.Plan) error {
	// Check if session is aborted
	if err := ssnutil.CheckStatus(ctx); err != nil {
		return err
	}

	h.planned = len(plan.Tasks) > 0
	return postPlanProgress(h.ctx, h.planFunc, h.requestID, plan)
}

func (h *chatPlanHooks) OnTaskDone(ctx context.Context, plan *planexec.Plan, _ *planexec.Task) error {
	// Check if session is aborted
	if err := ssnutil.CheckStatus(ctx); err != nil {
		return err
	}

	h.planned = len(plan.Tasks) > 0
	if len(plan.Tasks) == 0 {
		return nil
	}
	return postPlanProgress(h.ctx, h.planFunc, h.requestID, plan)
}

// postPlanProgress posts the plan progress as a new message (not an update)
func postPlanProgress(ctx context.Context, planFunc func(context.Context, string), requestID string, plan *planexec.Plan) error {
	if len(plan.Tasks) == 0 {
		// Suppress plan/task messages when there are no tasks
		return nil
	}

	completedCount := 0
	inProgressCount := 0
	for _, task := range plan.Tasks {
		switch task.State {
		case planexec.TaskStateCompleted:
			completedCount++
		case planexec.TaskStateInProgress:
			inProgressCount++
		}
	}

	var messageBuilder strings.Builder

	// Show status based on progress
	var statusMessage string
	switch {
	case completedCount == len(plan.Tasks):
		statusMessage = "✅ Completed"
	case inProgressCount > 0:
		statusMessage = "⟳ Working..."
	default:
		statusMessage = "🚀 Thinking..."
	}
	fmt.Fprintf(&messageBuilder, "%s (Request ID: %s)\n", statusMessage, requestID)

	fmt.Fprintf(&messageBuilder, "🎯 Objective *%s*\n", plan.Goal)
	messageBuilder.WriteString("\n")
	fmt.Fprintf(&messageBuilder, "*Progress: %d/%d tasks completed*\n", completedCount, len(plan.Tasks))

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

		fmt.Fprintf(&messageBuilder, "%s %s\n", icon, status)
	}

	planFunc(ctx, messageBuilder.String())
	return nil
}

// authorizeAgentRequest authorizes agent execution request using policy
func (x *UseCases) authorizeAgentRequest(ctx context.Context, message string) error {
	logger := logging.From(ctx)

	// Bypass authorization check if --no-authorization flag is set
	if x.noAuthorization {
		logging.From(ctx).Debug("agent authorization check bypassed due to --no-authorization flag")
		return nil
	}

	// Build auth context using domain model
	authCtx := auth.BuildAgentContext(ctx, message)

	// Query policy
	var result struct {
		Allow bool `json:"allow"`
	}

	query := "data.auth.agent"
	err := x.policyClient.Query(ctx, query, authCtx, &result, opaq.WithPrintHook(func(ctx context.Context, loc opaq.PrintLocation, msg string) error {
		logger.Debug("[rego] "+msg, "loc", loc)
		return nil
	}))
	if err != nil {
		if errors.Is(err, opaq.ErrNoEvalResult) {
			// Policy not defined, deny by default
			logging.From(ctx).Warn("agent authorization policy not defined, denying by default")
			return goerr.Wrap(errAgentAuthPolicyNotDefined, "agent authorization policy not defined")
		}
		return goerr.Wrap(err, "failed to evaluate agent authorization policy")
	}

	logging.From(ctx).Debug("agent authorization result", "input", authCtx, "output", result)

	if !result.Allow {
		logging.From(ctx).Warn("agent authorization failed", "message", message)
		return goerr.Wrap(errAgentAuthDenied, "agent request denied by policy", goerr.V("message", message))
	}

	return nil
}
