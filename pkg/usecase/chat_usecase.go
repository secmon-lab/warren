package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/agent"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/llm"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
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

// ChatUseCase encapsulates the chat processing workflow.
// It can be configured with different strategies and middleware via ChatOption.
type ChatUseCase struct {
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

// ChatOption configures a ChatUseCase.
type ChatOption func(*ChatUseCase)

// WithChatStrategyFactory sets the strategy factory for agent execution.
func WithChatStrategyFactory(f StrategyFactory) ChatOption {
	return func(c *ChatUseCase) {
		c.strategyFactory = f
	}
}

// WithChatSlackService sets the Slack service for message routing.
func WithChatSlackService(svc *slackService.Service) ChatOption {
	return func(c *ChatUseCase) {
		c.slackService = svc
	}
}

// WithChatTools sets the tool sets available to the agent.
func WithChatTools(tools []gollem.ToolSet) ChatOption {
	return func(c *ChatUseCase) {
		c.tools = append(c.tools, tools...)
	}
}

// WithChatSubAgents sets the sub-agents available to the agent.
func WithChatSubAgents(subAgents []*agent.SubAgent) ChatOption {
	return func(c *ChatUseCase) {
		c.subAgents = append(c.subAgents, subAgents...)
	}
}

// WithChatStorageClient sets the storage client for history persistence.
func WithChatStorageClient(client interfaces.StorageClient) ChatOption {
	return func(c *ChatUseCase) {
		c.storageClient = client
	}
}

// WithChatStoragePrefix sets the storage prefix for history paths.
func WithChatStoragePrefix(prefix string) ChatOption {
	return func(c *ChatUseCase) {
		c.storagePrefix = prefix
	}
}

// WithChatNoAuthorization disables policy-based authorization checks.
func WithChatNoAuthorization(noAuthz bool) ChatOption {
	return func(c *ChatUseCase) {
		c.noAuthorization = noAuthz
	}
}

// WithChatFrontendURL sets the frontend URL for session links.
func WithChatFrontendURL(url string) ChatOption {
	return func(c *ChatUseCase) {
		c.frontendURL = url
	}
}

// WithChatUserSystemPrompt sets the user system prompt.
func WithChatUserSystemPrompt(prompt string) ChatOption {
	return func(c *ChatUseCase) {
		c.userSystemPrompt = prompt
	}
}

// WithChatTraceRepository sets the trace repository for execution tracing.
func WithChatTraceRepository(repo trace.Repository) ChatOption {
	return func(c *ChatUseCase) {
		c.traceRepository = repo
	}
}

// NewChatUseCase creates a new ChatUseCase with the given dependencies and options.
func NewChatUseCase(repo interfaces.Repository, llmClient gollem.LLMClient, policyClient interfaces.PolicyClient, opts ...ChatOption) *ChatUseCase {
	uc := &ChatUseCase{
		repository:      repo,
		llmClient:       llmClient,
		policyClient:    policyClient,
		strategyFactory: DefaultStrategyFactory(),
	}

	for _, opt := range opts {
		opt(uc)
	}

	return uc
}

// Execute processes a chat message for the specified ticket.
// This is the main orchestrator that coordinates all phases of chat processing.
func (c *ChatUseCase) Execute(ctx context.Context, target *ticket.Ticket, message string) error {
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

	// Phase 5: Context preparation & agent execution
	return c.executeAgent(ctx, target, ssn, message, planFunc, &finalStatus)
}

// createSession creates and persists a new chat session.
func (c *ChatUseCase) createSession(ctx context.Context, target *ticket.Ticket, message string) (*session.Session, context.Context) {
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
func (c *ChatUseCase) setupMessageRouting(ctx context.Context, ssn *session.Session, target *ticket.Ticket) (func(context.Context, string), context.Context) {
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
func (c *ChatUseCase) setupSlackMessageFuncs(ctx context.Context, sess *session.Session, target *ticket.Ticket) (msg.NotifyFunc, msg.TraceFunc, msg.TraceFunc, msg.WarnFunc) {
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
func (c *ChatUseCase) setupStatusCheck(ctx context.Context, ssn *session.Session) context.Context {
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
func (c *ChatUseCase) finishSession(ctx context.Context, ssn *session.Session, target *ticket.Ticket, finalStatus *types.SessionStatus, logger interface{ Error(string, ...any) }) {
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

	if *finalStatus == types.SessionStatusCompleted && c.slackService != nil && target.SlackThread != nil {
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
func (c *ChatUseCase) authorize(ctx context.Context, message string) (bool, error) {
	if err := c.authorizeAgentRequest(ctx, message); err != nil {
		if errors.Is(err, errAgentAuthPolicyNotDefined) {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nAgent execution policy is not defined. Please configure the `auth.agent` policy or use `--no-authorization` flag for development.\n\nSee: https://docs.warren.secmon-lab.com/policy.md#agent-execution-authorization")
		} else if errors.Is(err, errAgentAuthDenied) {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nYou are not authorized to execute agent requests. Please contact your administrator if you believe this is an error.")
		} else {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nFailed to check authorization. Please contact your administrator.")
			return false, goerr.Wrap(err, "failed to evaluate agent auth")
		}
		return false, nil
	}
	return true, nil
}

// authorizeAgentRequest checks policy-based authorization.
func (c *ChatUseCase) authorizeAgentRequest(ctx context.Context, message string) error {
	logger := logging.From(ctx)

	if c.noAuthorization {
		logger.Debug("agent authorization check bypassed due to --no-authorization flag")
		return nil
	}

	authCtx := auth.BuildAgentContext(ctx, message)

	var result struct {
		Allow bool `json:"allow"`
	}

	query := "data.auth.agent"
	err := c.policyClient.Query(ctx, query, authCtx, &result, opaq.WithPrintHook(func(ctx context.Context, loc opaq.PrintLocation, msg string) error {
		logger.Debug("[rego] "+msg, "loc", loc)
		return nil
	}))
	if err != nil {
		if errors.Is(err, opaq.ErrNoEvalResult) {
			logger.Warn("agent authorization policy not defined, denying by default")
			return goerr.Wrap(errAgentAuthPolicyNotDefined, "agent authorization policy not defined")
		}
		return goerr.Wrap(err, "failed to evaluate agent authorization policy")
	}

	logger.Debug("agent authorization result", "input", authCtx, "output", result)

	if !result.Allow {
		logger.Warn("agent authorization failed", "message", message)
		return goerr.Wrap(errAgentAuthDenied, "agent request denied by policy", goerr.V("message", message))
	}

	return nil
}

// executeAgent handles context preparation, agent construction, execution, and result processing.
func (c *ChatUseCase) executeAgent(ctx context.Context, target *ticket.Ticket, ssn *session.Session, message string, planFunc func(context.Context, string), finalStatus *types.SessionStatus) error {
	logger := logging.From(ctx)

	// Setup finding update function
	slackUpdateFunc := func(ctx context.Context, t *ticket.Ticket) error {
		if c.slackService == nil || !t.HasSlackThread() || t.Finding == nil {
			return nil
		}
		threadSvc := c.slackService.NewThread(*t.SlackThread)
		return threadSvc.PostFinding(ctx, t.Finding)
	}

	baseAction := base.New(c.repository, target.ID, base.WithSlackUpdate(slackUpdateFunc), base.WithLLMClient(c.llmClient))

	// Load history
	storageSvc := storage.New(c.storageClient, storage.WithPrefix(c.storagePrefix))
	history, err := c.loadHistory(ctx, target, storageSvc)
	if err != nil {
		return err
	}

	// Load alerts and prepare tools
	alerts, err := c.repository.BatchGetAlerts(ctx, target.AlertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}

	effectiveTopic := c.resolveEffectiveTopic(ctx, target, alerts)

	// Set topic for existing knowledge tool if present
	for _, tool := range c.tools {
		if kt, ok := tool.(*knowledgeTool.Knowledge); ok {
			kt.SetTopic(effectiveTopic)
			defer kt.SetTopic("")
			logger.Debug("set topic for knowledge tool", "topic", effectiveTopic)
			break
		}
	}

	kt := knowledgeTool.New(c.repository, effectiveTopic)
	tools := append(c.tools, baseAction, kt)

	// Build system prompt
	systemPrompt, err := c.buildSystemPrompt(ctx, target, ssn, tools, alerts, effectiveTopic)
	if err != nil {
		return err
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
	agent := c.buildAgent(ctx, strategy, history, tools, systemPrompt, requestID)

	result, executionErr := agent.Execute(ctx, gollem.Text(message))
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
	c.handleResult(ctx, result, target, ssn)

	// Save history
	return c.saveHistory(ctx, agent, target, storageSvc)
}

// loadHistory loads the chat history for the ticket.
func (c *ChatUseCase) loadHistory(ctx context.Context, target *ticket.Ticket, storageSvc *storage.Service) (*gollem.History, error) {
	logger := logging.From(ctx)

	historyRecord, err := c.repository.GetLatestHistory(ctx, target.ID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get latest history")
	}

	if historyRecord == nil {
		return nil, nil
	}

	history, err := storageSvc.GetHistory(ctx, target.ID, historyRecord.ID)
	if err != nil {
		msg.Notify(ctx, "⚠️ Failed to load chat history, starting fresh: %s", err.Error())
		logger.Warn("failed to get history data, starting with new history", "error", err)
		return nil, nil
	}

	if history != nil && (history.Version <= 0 || history.ToCount() <= 0) {
		msg.Notify(ctx, "⚠️ Chat history incompatible (version=%d, messages=%d), starting fresh", history.Version, history.ToCount())
		logger.Warn("history incompatible, starting with new history",
			"version", history.Version,
			"message_count", history.ToCount(),
			"history_id", historyRecord.ID)
		return nil, nil
	}

	return history, nil
}

// resolveEffectiveTopic determines the effective topic, falling back to schema if needed.
func (c *ChatUseCase) resolveEffectiveTopic(ctx context.Context, target *ticket.Ticket, alerts []*alert.Alert) types.KnowledgeTopic {
	logger := logging.From(ctx)
	effectiveTopic := target.Topic

	if effectiveTopic == "" && len(alerts) > 0 {
		effectiveTopic = types.KnowledgeTopic(alerts[0].Schema)
		logger.Warn("ticket topic is empty, falling back to schema",
			"ticket_id", target.ID,
			"schema", alerts[0].Schema,
			"topic", effectiveTopic)
		msg.Notify(ctx, "⚠️ Ticket topic is empty, using schema `%s` as topic", alerts[0].Schema)
	}

	return effectiveTopic
}

// buildSystemPrompt generates the system prompt with all context.
func (c *ChatUseCase) buildSystemPrompt(ctx context.Context, target *ticket.Ticket, ssn *session.Session, tools []gollem.ToolSet, alerts []*alert.Alert, effectiveTopic types.KnowledgeTopic) (string, error) {
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

	// Get knowledges
	knowledges := []*knowledge.Knowledge{}
	if target.Topic != "" {
		retrieved, err := c.repository.GetKnowledges(ctx, target.Topic)
		if err != nil {
			logger.Warn("failed to get knowledges", "error", err, "topic", target.Topic)
		} else if retrieved != nil {
			knowledges = retrieved
		}
	}

	// Collect thread comments
	threadComments := c.collectThreadComments(ctx, target.ID, ssn)

	return generateChatSystemPrompt(ctx, target, len(alerts), additionalInstructions, knowledges, string(userID), threadComments, c.userSystemPrompt)
}

// collectThreadComments retrieves thread comments posted between the previous session and the current session.
func (c *ChatUseCase) collectThreadComments(ctx context.Context, ticketID types.TicketID, currentSession *session.Session) []ticket.Comment {
	logger := logging.From(ctx)

	const maxThreadComments = 50

	sessions, err := c.repository.GetSessionsByTicket(ctx, ticketID)
	if err != nil {
		logger.Warn("failed to get sessions for thread comments", "error", err, "ticket_id", ticketID)
		return nil
	}

	var prevSessionCreatedAt time.Time
	for _, s := range sessions {
		if s.ID == currentSession.ID {
			continue
		}
		if s.Status != types.SessionStatusCompleted {
			continue
		}
		if s.CreatedAt.Before(currentSession.CreatedAt) && s.CreatedAt.After(prevSessionCreatedAt) {
			prevSessionCreatedAt = s.CreatedAt
		}
	}

	logger.Debug("collectThreadComments",
		"ticket_id", ticketID,
		"total_sessions", len(sessions),
		"prev_session_created_at", prevSessionCreatedAt,
		"current_session_created_at", currentSession.CreatedAt,
	)

	comments, err := c.repository.GetTicketComments(ctx, ticketID)
	if err != nil {
		logger.Warn("failed to get ticket comments for thread context", "error", err, "ticket_id", ticketID)
		return nil
	}

	var filtered []ticket.Comment
	for _, co := range comments {
		if co.CreatedAt.After(prevSessionCreatedAt) && co.CreatedAt.Before(currentSession.CreatedAt) {
			filtered = append(filtered, co)
		}
	}

	logger.Debug("collectThreadComments filtered",
		"total_comments", len(comments),
		"filtered_count", len(filtered),
	)

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	if len(filtered) > maxThreadComments {
		filtered = filtered[len(filtered)-maxThreadComments:]
	}

	return filtered
}

// buildAgent constructs the gollem agent with strategy, tools, and middleware.
func (c *ChatUseCase) buildAgent(ctx context.Context, strategy gollem.Strategy, history *gollem.History, tools []gollem.ToolSet, systemPrompt string, requestID string) *gollem.Agent {
	logger := logging.From(ctx)

	gollemSubAgents := make([]*gollem.SubAgent, len(c.subAgents))
	for i, sa := range c.subAgents {
		gollemSubAgents[i] = sa.Inner()
	}

	traceMW := newTraceLLMMiddleware()

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
		)
		agentOpts = append(agentOpts, gollem.WithTrace(recorder))
	}

	agentOpts = append(agentOpts,
		gollem.WithContentBlockMiddleware(llm.NewCompactionMiddleware(c.llmClient, logging.From(ctx))),
		gollem.WithContentBlockMiddleware(traceMW),
		gollem.WithToolMiddleware(func(next gollem.ToolHandler) gollem.ToolHandler {
			return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
				if err := ssnutil.CheckStatus(ctx); err != nil {
					return &gollem.ToolExecResponse{
						Error: err,
					}, nil
				}

				if !base.IgnorableTool(req.Tool.Name) {
					message := toolCallToText(ctx, c.llmClient, req.ToolSpec, req.Tool)
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
func (c *ChatUseCase) handleResult(ctx context.Context, result *gollem.ExecuteResponse, target *ticket.Ticket, ssn *session.Session) {
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
func (c *ChatUseCase) saveHistory(ctx context.Context, agent *gollem.Agent, target *ticket.Ticket, storageSvc *storage.Service) error {
	logger := logging.From(ctx)

	agentSession := agent.Session()
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
