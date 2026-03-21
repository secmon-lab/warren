package legacy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/agent"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/llm"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/tool/base"
	"github.com/secmon-lab/warren/pkg/usecase/chat"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
	ssnutil "github.com/secmon-lab/warren/pkg/utils/session"
	"github.com/secmon-lab/warren/pkg/utils/slackctx"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// PlanExecChat implements interfaces.ChatUseCase with the Plan & Execute strategy.
type PlanExecChat struct {
	// required dependencies
	repository   interfaces.Repository
	llmClient    gollem.LLMClient
	policyClient interfaces.PolicyClient

	// optional dependencies
	storageClient interfaces.StorageClient
	slackService  *slackService.Service
	tools         []gollem.ToolSet
	subAgents     []*agent.SubAgent

	// swappable components
	strategyFactory StrategyFactory

	// configuration
	storagePrefix    string
	noAuthorization  bool
	frontendURL      string
	userSystemPrompt string
	traceRepository  trace.Repository
}

// Option configures a PlanExecChat.
type Option func(*PlanExecChat)

// WithStrategyFactory sets the strategy factory for agent execution.
func WithStrategyFactory(f StrategyFactory) Option {
	return func(c *PlanExecChat) {
		c.strategyFactory = f
	}
}

// WithSlackService sets the Slack service for message routing.
func WithSlackService(svc *slackService.Service) Option {
	return func(c *PlanExecChat) {
		c.slackService = svc
	}
}

// WithTools sets the tool sets available to the agent.
func WithTools(tools []gollem.ToolSet) Option {
	return func(c *PlanExecChat) {
		c.tools = append(c.tools, tools...)
	}
}

// WithSubAgents sets the sub-agents available to the agent.
func WithSubAgents(subAgents []*agent.SubAgent) Option {
	return func(c *PlanExecChat) {
		c.subAgents = append(c.subAgents, subAgents...)
	}
}

// WithStorageClient sets the storage client for history persistence.
func WithStorageClient(client interfaces.StorageClient) Option {
	return func(c *PlanExecChat) {
		c.storageClient = client
	}
}

// WithStoragePrefix sets the storage prefix for history paths.
func WithStoragePrefix(prefix string) Option {
	return func(c *PlanExecChat) {
		c.storagePrefix = prefix
	}
}

// WithNoAuthorization disables policy-based authorization checks.
func WithNoAuthorization(noAuthz bool) Option {
	return func(c *PlanExecChat) {
		c.noAuthorization = noAuthz
	}
}

// WithFrontendURL sets the frontend URL for session links.
func WithFrontendURL(url string) Option {
	return func(c *PlanExecChat) {
		c.frontendURL = url
	}
}

// WithUserSystemPrompt sets the user system prompt.
func WithUserSystemPrompt(prompt string) Option {
	return func(c *PlanExecChat) {
		c.userSystemPrompt = prompt
	}
}

// WithTraceRepository sets the trace repository for execution tracing.
func WithTraceRepository(repo trace.Repository) Option {
	return func(c *PlanExecChat) {
		c.traceRepository = repo
	}
}

// NewPlanExecChat creates a new PlanExecChat with the given dependencies and options.
func NewPlanExecChat(repo interfaces.Repository, llmClient gollem.LLMClient, policyClient interfaces.PolicyClient, opts ...Option) *PlanExecChat {
	c := &PlanExecChat{
		repository:      repo,
		llmClient:       llmClient,
		policyClient:    policyClient,
		strategyFactory: DefaultStrategyFactory(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Execute processes a chat message using the Plan & Execute strategy.
// The ChatContext must be pre-built by the caller with all necessary data.
func (c *PlanExecChat) Execute(ctx context.Context, message string, chatCtx chatModel.ChatContext) error {
	target := chatCtx.Ticket
	logger := logging.From(ctx)

	// Phase 1: Session setup
	ssn, ctx := c.createSession(ctx, target, message)

	// Phase 2: Message routing setup
	planFunc, ctx := c.setupMessageRouting(ctx, ssn, target)

	// Phase 3: Session status tracking
	ctx = c.setupStatusCheck(ctx, ssn)

	finalStatus := types.SessionStatusCompleted
	defer c.finishSession(ctx, ssn, target, &finalStatus, logger)

	// Phase 4: Authorization
	authorized, err := c.authorize(ctx, message)
	if err != nil {
		return err
	}
	if !authorized {
		return nil
	}

	// Phase 5: Agent execution
	return c.executeAgent(ctx, ssn, message, planFunc, &finalStatus, &chatCtx)
}

// createSession creates and persists a new chat session.
func (c *PlanExecChat) createSession(ctx context.Context, target *ticket.Ticket, message string) (*session.Session, context.Context) {
	userID := types.UserID(user.FromContext(ctx))
	slackURL := slackctx.SlackURL(ctx)

	ssn := session.NewSession(ctx, target.ID, userID, message, slackURL)
	if err := c.repository.PutSession(ctx, ssn); err != nil {
		// Log but don't fail - session is important but not blocking
		logging.From(ctx).Error("failed to save session", "error", err)
	}

	logger := logging.From(ctx).With("session_id", ssn.ID)
	ctx = logging.With(ctx, logger)

	if target.SlackThread != nil {
		ctx = slackctx.WithThread(ctx, *target.SlackThread)
	}

	return ssn, ctx
}

// setupMessageRouting configures Slack/CLI message routing functions in the context.
func (c *PlanExecChat) setupMessageRouting(ctx context.Context, ssn *session.Session, target *ticket.Ticket) (func(context.Context, string), context.Context) {
	planFunc := func(ctx context.Context, msg string) {}

	if c.slackService != nil && target.SlackThread != nil {
		notifyFunc, traceFunc, pf, warnFunc := c.setupSlackMessageFuncs(ctx, ssn, target)
		ctx = msg.With(ctx, notifyFunc, traceFunc, warnFunc)
		planFunc = pf

		requestID := request_id.FromContext(ctx)
		if requestID == "" {
			requestID = "unknown"
		}
		planFunc(ctx, fmt.Sprintf("🚀 Thinking... (Request ID: %s)", requestID))
	}

	return planFunc, ctx
}

// setupSlackMessageFuncs creates Slack message routing functions for notify, trace, plan, and warn.
func (c *PlanExecChat) setupSlackMessageFuncs(ctx context.Context, sess *session.Session, target *ticket.Ticket) (msg.NotifyFunc, msg.TraceFunc, msg.TraceFunc, msg.WarnFunc) {
	threadSvc := c.slackService.NewThread(*target.SlackThread)

	notifyFunc := func(ctx context.Context, message string) {
		m := session.NewMessage(ctx, sess.ID, session.MessageTypeResponse, message)
		if err := c.repository.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}
		if err := threadSvc.PostComment(ctx, message); err != nil {
			errutil.Handle(ctx, err)
		}
	}

	createUpdatableMessageFunc := func(msgType session.MessageType) msg.TraceFunc {
		var updateFunc func(context.Context, string)
		return func(ctx context.Context, message string) {
			m := session.NewMessage(ctx, sess.ID, msgType, message)
			if err := c.repository.PutSessionMessage(ctx, m); err != nil {
				errutil.Handle(ctx, err)
			}

			if updateFunc == nil {
				updateFunc = threadSvc.NewUpdatableMessage(ctx, message)
			} else {
				updateFunc(ctx, message)
			}
		}
	}

	traceFunc := createUpdatableMessageFunc(session.MessageTypeTrace)
	planFunc := createUpdatableMessageFunc(session.MessageTypePlan)

	warnFunc := func(ctx context.Context, message string) {
		m := session.NewMessage(ctx, sess.ID, session.MessageTypeWarning, message)
		if err := c.repository.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}
		if err := threadSvc.PostComment(ctx, message); err != nil {
			errutil.Handle(ctx, err)
		}
	}

	return notifyFunc, traceFunc, planFunc, warnFunc
}

// setupStatusCheck embeds a session abort check function in the context.
func (c *PlanExecChat) setupStatusCheck(ctx context.Context, ssn *session.Session) context.Context {
	statusCheckFunc := func(ctx context.Context) error {
		s, err := c.repository.GetSession(ctx, ssn.ID)
		if err != nil {
			return goerr.Wrap(err, "failed to get session status")
		}
		if s != nil && s.Status == types.SessionStatusAborted {
			return ErrSessionAborted
		}
		return nil
	}
	return ssnutil.WithStatusCheck(ctx, statusCheckFunc)
}

// finishSession updates session status and posts session actions on completion.
func (c *PlanExecChat) finishSession(ctx context.Context, ssn *session.Session, target *ticket.Ticket, finalStatus *types.SessionStatus, logger interface{ Error(string, ...any) }) {
	if r := recover(); r != nil {
		*finalStatus = types.SessionStatusAborted
		ssn.UpdateStatus(ctx, *finalStatus)
		if err := c.repository.PutSession(ctx, ssn); err != nil {
			logger.Error("failed to update session status on panic", "error", err, "status", *finalStatus)
		}
		panic(r)
	}

	ssn.UpdateStatus(ctx, *finalStatus)
	if err := c.repository.PutSession(ctx, ssn); err != nil {
		logger.Error("failed to update final session status", "error", err, "status", *finalStatus)
	}

	// Skip session actions for ticketless chat (no ticket to act on)
	if target.ID != "" && *finalStatus == types.SessionStatusCompleted && c.slackService != nil && target.SlackThread != nil {
		threadSvc := c.slackService.NewThread(*target.SlackThread)

		var sessionURL string
		if c.frontendURL != "" {
			sessionURL = fmt.Sprintf("%s/sessions/%s", c.frontendURL, ssn.ID)
		}

		currentTicket, err := c.repository.GetTicket(ctx, target.ID)
		if err != nil {
			logger.Error("failed to get ticket for session actions", "error", err, "ticket_id", target.ID)
		} else if currentTicket != nil {
			if err := threadSvc.PostSessionActions(ctx, target.ID, currentTicket.Status, sessionURL); err != nil {
				logger.Error("failed to post session actions to Slack", "error", err, "session_id", ssn.ID)
			}
		}
	}
}

// authorize checks policy-based authorization for agent execution.
// Returns (true, nil) if authorized, (false, nil) if denied (notification already sent), or (false, err) on error.
func (c *PlanExecChat) authorize(ctx context.Context, message string) (bool, error) {
	if err := chat.AuthorizeAgentRequest(ctx, c.policyClient, c.noAuthorization, message); err != nil {
		if errors.Is(err, chat.ErrAgentAuthPolicyNotDefined) {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nAgent execution policy is not defined. Please configure the `auth.agent` policy or use `--no-authorization` flag for development.\n\nSee: https://docs.warren.secmon-lab.com/policy.md#agent-execution-authorization")
		} else if errors.Is(err, chat.ErrAgentAuthDenied) {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nYou are not authorized to execute agent requests. Please contact your administrator if you believe this is an error.")
		} else {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nFailed to check authorization. Please contact your administrator.")
			return false, goerr.Wrap(err, "failed to evaluate agent auth")
		}
		return false, nil
	}
	return true, nil
}

// executeAgent handles agent construction, execution, and result processing.
func (c *PlanExecChat) executeAgent(ctx context.Context, ssn *session.Session, message string, planFunc func(context.Context, string), finalStatus *types.SessionStatus, chatCtx *chatModel.ChatContext) error {
	target := chatCtx.Ticket
	ticketless := chatCtx.IsTicketless()

	tools := chatCtx.Tools
	history := chatCtx.History
	storageSvc := storage.New(c.storageClient, storage.WithPrefix(c.storagePrefix))

	var systemPrompt string
	if ticketless {
		var err error
		systemPrompt, err = c.buildTicketlessSystemPrompt(ctx, tools, chatCtx.SlackHistory)
		if err != nil {
			return err
		}
	} else {
		var err error
		systemPrompt, err = c.buildSystemPrompt(ctx, target, ssn, tools, chatCtx)
		if err != nil {
			return err
		}
	}

	// Get request ID
	requestID := request_id.FromContext(ctx)
	if requestID == "" {
		requestID = "unknown"
	}

	// Create strategy via factory
	strategy, reporter := c.strategyFactory(ctx, &StrategyParams{
		LLMClient:  c.llmClient,
		Session:    ssn,
		Repository: c.repository,
		RequestID:  requestID,
		PlanFunc:   planFunc,
	})

	// Build and execute agent
	// Skip trace LLM middleware for ticketless to avoid leaking planexec's
	// internal JSON responses (e.g. {needs_plan, direct_response}) to Slack.
	gollemAgent := c.buildAgent(ctx, strategy, history, tools, systemPrompt, requestID, buildAgentOption{skipTraceLLM: ticketless})

	result, executionErr := gollemAgent.Execute(ctx, gollem.Text(message))
	if executionErr != nil {
		*finalStatus = types.SessionStatusAborted

		if errors.Is(executionErr, ErrSessionAborted) {
			msg.Notify(ctx, "🛑 Execution aborted by user request.")
			return nil
		}
		msg.Notify(ctx, "💥 Execution failed: %s", executionErr.Error())
		return goerr.Wrap(executionErr, "failed to execute agent")
	}

	if reporter != nil && reporter.Planned() {
		msg.Trace(ctx, "✅ Execution completed")
	}

	// Handle result
	if ticketless {
		// Ticketless: just notify, no ticket comment
		if result != nil && !result.IsEmpty() {
			msg.Notify(ctx, "💬 %s", result.String())
		}
	} else {
		c.handleResult(ctx, result, target, ssn)
	}

	// Save history (skip for ticketless)
	if !ticketless {
		return c.saveHistory(ctx, gollemAgent, target, storageSvc)
	}
	return nil
}

// buildTicketlessSystemPrompt generates the system prompt for ticketless chat.
func (c *PlanExecChat) buildTicketlessSystemPrompt(ctx context.Context, tools []gollem.ToolSet, slackHistory []slack.HistoryMessage) (string, error) {
	logger := logging.From(ctx)
	userID := types.UserID(user.FromContext(ctx))

	// Collect tool prompts
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

	for _, sa := range c.subAgents {
		if ph := sa.PromptHint(); ph != "" {
			toolPrompts = append(toolPrompts, ph)
		}
	}

	var additionalInstructions string
	if len(toolPrompts) > 0 {
		additionalInstructions = "# Available Tools and Resources\n\n" + strings.Join(toolPrompts, "\n\n")
	}

	return GenerateTicketlessSystemPrompt(ctx, slackHistory, additionalInstructions, nil, string(userID), c.userSystemPrompt)
}

// buildSystemPrompt generates the system prompt with all context from ChatContext.
func (c *PlanExecChat) buildSystemPrompt(ctx context.Context, target *ticket.Ticket, _ *session.Session, tools []gollem.ToolSet, chatCtx *chatModel.ChatContext) (string, error) {
	logger := logging.From(ctx)
	userID := types.UserID(user.FromContext(ctx))

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

	for _, sa := range c.subAgents {
		if ph := sa.PromptHint(); ph != "" {
			toolPrompts = append(toolPrompts, ph)
		}
	}

	var additionalInstructions string
	if len(toolPrompts) > 0 {
		additionalInstructions = "# Available Tools and Resources\n\n" + strings.Join(toolPrompts, "\n\n")
	}

	return GenerateChatSystemPrompt(ctx, target, len(chatCtx.Alerts), additionalInstructions, chatCtx.Knowledges, string(userID), chatCtx.ThreadComments, c.userSystemPrompt, chatCtx.SlackHistory)
}

// buildAgentOption holds optional configuration for buildAgent.
type buildAgentOption struct {
	skipTraceLLM bool
}

// buildAgent constructs the gollem agent with strategy, tools, and middleware.
func (c *PlanExecChat) buildAgent(ctx context.Context, strategy gollem.Strategy, history *gollem.History, tools []gollem.ToolSet, systemPrompt string, requestID string, opts ...buildAgentOption) *gollem.Agent {
	logger := logging.From(ctx)

	var opt buildAgentOption
	if len(opts) > 0 {
		opt = opts[0]
	}

	gollemSubAgents := make([]*gollem.SubAgent, len(c.subAgents))
	for i, sa := range c.subAgents {
		gollemSubAgents[i] = sa.Inner()
	}

	agentOpts := []gollem.Option{
		gollem.WithStrategy(strategy),
		gollem.WithHistory(history),
		gollem.WithToolSets(tools...),
		gollem.WithSubAgents(gollemSubAgents...),
		gollem.WithResponseMode(gollem.ResponseModeBlocking),
		gollem.WithSystemPrompt(systemPrompt),
	}

	if c.traceRepository != nil {
		recorder := trace.New(
			trace.WithTraceID(requestID),
			trace.WithRepository(c.traceRepository),
			trace.WithStackTrace(),
		)
		agentOpts = append(agentOpts, gollem.WithTrace(recorder))
	}

	agentOpts = append(agentOpts,
		gollem.WithContentBlockMiddleware(llm.NewCompactionMiddleware(c.llmClient, logging.From(ctx))),
	)

	// The trace LLM middleware posts every LLM text response (including planexec's
	// internal JSON like {needs_plan, direct_response}) to Slack. Skip it for
	// ticketless chat where those intermediate responses should not be visible.
	if !opt.skipTraceLLM {
		traceMW := newTraceLLMMiddleware()
		agentOpts = append(agentOpts, gollem.WithContentBlockMiddleware(traceMW))
	}

	agentOpts = append(agentOpts,
		gollem.WithToolMiddleware(func(next gollem.ToolHandler) gollem.ToolHandler {
			return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
				if err := ssnutil.CheckStatus(ctx); err != nil {
					return &gollem.ToolExecResponse{
						Error: err,
					}, nil
				}

				if !base.IgnorableTool(req.Tool.Name) {
					message := ToolCallToText(ctx, c.llmClient, req.ToolSpec, req.Tool)
					msg.Trace(ctx, "🤖 %s", message)
					logger.Debug("execute tool", "tool", req.Tool.Name, "args", req.Tool.Arguments)
				}

				resp, err := next(ctx, req)

				if resp != nil && resp.Error != nil {
					msg.Trace(ctx, "❌ Error: %s", resp.Error.Error())
					logger.Error("tool error", "error", resp.Error, "call", req.Tool)
				}

				return resp, err
			}
		}),
	)

	return gollem.New(c.llmClient, agentOpts...)
}

// handleResult processes the agent execution result and posts to Slack.
func (c *PlanExecChat) handleResult(ctx context.Context, result *gollem.ExecuteResponse, target *ticket.Ticket, ssn *session.Session) {
	if result == nil || result.IsEmpty() {
		return
	}

	warrenResponse := fmt.Sprintf("💬 %s", result.String())

	if c.slackService != nil && target.SlackThread != nil {
		agentCtx := user.WithAgent(user.WithUserID(ctx, c.slackService.BotID()))

		botUser := &slack.User{
			ID:   c.slackService.BotID(),
			Name: "Warren",
		}

		m := session.NewMessage(ctx, ssn.ID, session.MessageTypeResponse, warrenResponse)
		if err := c.repository.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}

		threadSvc := c.slackService.NewThread(*target.SlackThread)
		logging.From(ctx).Debug("message notify", "from", "Agent", "msg", warrenResponse)
		ts, err := threadSvc.PostCommentWithMessageID(ctx, warrenResponse)
		if err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "failed to post agent message to slack"))
		} else {
			comment := target.NewComment(agentCtx, warrenResponse, botUser, ts)

			if err := c.repository.PutTicketComment(agentCtx, comment); err != nil {
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

// saveHistory saves the updated chat history after agent execution.
func (c *PlanExecChat) saveHistory(ctx context.Context, gollemAgent *gollem.Agent, target *ticket.Ticket, storageSvc *storage.Service) error {
	logger := logging.From(ctx)

	agentSession := gollemAgent.Session()
	if agentSession == nil {
		logger.Warn("agent session is nil after execution")
		return nil
	}

	newHistory, err := agentSession.History()
	if err != nil {
		return goerr.Wrap(err, "failed to get history from agent session")
	}
	if newHistory == nil {
		return goerr.New("history is nil after execution")
	}

	logger.Debug("saving chat history",
		"history_version", newHistory.Version,
		"message_count", newHistory.ToCount())

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

		if err := c.repository.PutHistory(ctx, target.ID, &newRecord); err != nil {
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
