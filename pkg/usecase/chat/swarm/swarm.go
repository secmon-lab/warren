package swarm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/agent"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/memory"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
	ssnutil "github.com/secmon-lab/warren/pkg/utils/session"
	"github.com/secmon-lab/warren/pkg/utils/slackctx"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

const defaultMaxPhases = 10

// SwarmChat implements interfaces.ChatUseCase with parallel task execution.
type SwarmChat struct {
	repository       interfaces.Repository
	llmClient        gollem.LLMClient
	policyClient     interfaces.PolicyClient
	storageClient    interfaces.StorageClient
	slackService     *slackService.Service
	memoryService    *memory.Service
	tools            []gollem.ToolSet
	subAgents        []*agent.SubAgent
	storagePrefix    string
	noAuthorization  bool
	frontendURL      string
	userSystemPrompt string
	traceRepository  trace.Repository
	maxPhases        int
}

// Option configures a SwarmChat.
type Option func(*SwarmChat)

// WithSlackService sets the Slack service for message routing.
func WithSlackService(svc *slackService.Service) Option {
	return func(c *SwarmChat) { c.slackService = svc }
}

// WithTools sets the tool sets available to the agent.
func WithTools(tools []gollem.ToolSet) Option {
	return func(c *SwarmChat) { c.tools = append(c.tools, tools...) }
}

// WithSubAgents sets the sub-agents available to the agent.
func WithSubAgents(subAgents []*agent.SubAgent) Option {
	return func(c *SwarmChat) { c.subAgents = append(c.subAgents, subAgents...) }
}

// WithStorageClient sets the storage client for history persistence.
func WithStorageClient(client interfaces.StorageClient) Option {
	return func(c *SwarmChat) { c.storageClient = client }
}

// WithStoragePrefix sets the storage prefix for history paths.
func WithStoragePrefix(prefix string) Option {
	return func(c *SwarmChat) { c.storagePrefix = prefix }
}

// WithNoAuthorization disables policy-based authorization checks.
func WithNoAuthorization(noAuthz bool) Option {
	return func(c *SwarmChat) { c.noAuthorization = noAuthz }
}

// WithFrontendURL sets the frontend URL for session links.
func WithFrontendURL(url string) Option {
	return func(c *SwarmChat) { c.frontendURL = url }
}

// WithUserSystemPrompt sets the user system prompt.
func WithUserSystemPrompt(prompt string) Option {
	return func(c *SwarmChat) { c.userSystemPrompt = prompt }
}

// WithTraceRepository sets the trace repository for execution tracing.
func WithTraceRepository(repo trace.Repository) Option {
	return func(c *SwarmChat) { c.traceRepository = repo }
}

// WithMemoryService sets the memory service for agent memory integration.
func WithMemoryService(svc *memory.Service) Option {
	return func(c *SwarmChat) { c.memoryService = svc }
}

// WithMaxPhases sets the maximum number of execution phases.
func WithMaxPhases(n int) Option {
	return func(c *SwarmChat) { c.maxPhases = n }
}

// New creates a new SwarmChat with the given dependencies and options.
func New(repo interfaces.Repository, llmClient gollem.LLMClient, policyClient interfaces.PolicyClient, opts ...Option) *SwarmChat {
	c := &SwarmChat{
		repository:   repo,
		llmClient:    llmClient,
		policyClient: policyClient,
		maxPhases:    defaultMaxPhases,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Execute processes a chat message for the specified ticket using parallel task execution.
func (c *SwarmChat) Execute(ctx context.Context, target *ticket.Ticket, message string) error {
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

	// Phase 5: Swarm execution
	return c.executeSwarm(ctx, target, ssn, message, planFunc, &finalStatus)
}

// executeSwarm orchestrates the swarm execution: plan → parallel exec → replan → loop → final response.
func (c *SwarmChat) executeSwarm(ctx context.Context, target *ticket.Ticket, ssn *session.Session, message string, planFunc func(context.Context, string), finalStatus *types.SessionStatus) error {
	logger := logging.From(ctx)

	// Setup trace recorder
	var recorder *trace.Recorder
	requestID := request_id.FromContext(ctx)
	if requestID == "" {
		requestID = "unknown"
	}
	if c.traceRepository != nil {
		recorder = trace.New(
			trace.WithTraceID(requestID),
			trace.WithRepository(c.traceRepository),
		)
		ctx = trace.WithHandler(ctx, recorder)
		defer func() {
			if err := recorder.Finish(ctx); err != nil {
				logger.Error("failed to finish trace", "error", err)
			}
		}()
	}

	// Start root agent execution span
	handler := trace.HandlerFrom(ctx)
	if handler != nil {
		ctx = handler.StartAgentExecute(ctx)
		defer handler.EndAgentExecute(ctx, nil)
	}

	// Load history
	storageSvc := storage.New(c.storageClient, storage.WithPrefix(c.storagePrefix))
	history, err := c.loadHistory(ctx, target, storageSvc)
	if err != nil {
		return err
	}

	// Load alerts for planning context
	alerts, err := c.repository.BatchGetAlerts(ctx, target.AlertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}

	// Search agent memories
	var memoryContext string
	if c.memoryService != nil {
		memories, memErr := c.memoryService.SearchAndSelectMemories(ctx, message, 16)
		if memErr != nil {
			logger.Warn("failed to search agent memories", "error", memErr)
		} else if len(memories) > 0 {
			memoryContext = formatMemories(memories)
		}
	}

	// Build planning context
	planCtx := &planningContext{
		message:       message,
		ticket:        target,
		alerts:        alerts,
		tools:         c.tools,
		subAgents:     c.subAgents,
		memoryContext: memoryContext,
		userPrompt:    c.userSystemPrompt,
	}

	// Create planning session with history
	planSession, err := c.llmClient.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(planSchema),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to create planning session")
	}
	if history != nil {
		if err := planSession.AppendHistory(history); err != nil {
			logger.Warn("failed to append history to planning session", "error", err)
		}
	}

	// Planning phase
	planResult, err := c.plan(ctx, planSession, planCtx)
	if err != nil {
		*finalStatus = types.SessionStatusAborted
		msg.Notify(ctx, "💥 Planning failed: %s", err.Error())
		return goerr.Wrap(err, "planning failed")
	}

	// Post initial message
	if planResult.Message != "" {
		msg.Notify(ctx, "💬 %s", planResult.Message)
	}

	// Direct response (no tasks)
	if len(planResult.Tasks) == 0 {
		return c.saveHistory(ctx, planSession, target, storageSvc)
	}

	// Execute phases
	var allResults []*phaseResult
	currentTasks := planResult.Tasks

	for phase := 1; phase <= c.maxPhases; phase++ {
		if len(currentTasks) == 0 {
			break
		}

		// Update plan progress
		planFunc(ctx, c.formatPlanProgress(requestID, phase, currentTasks, allResults))

		// Execute all tasks in parallel
		results := c.executePhase(ctx, currentTasks, target, ssn, planCtx)
		allResults = append(allResults, &phaseResult{
			phase:   phase,
			tasks:   currentTasks,
			results: results,
		})

		// Update plan progress with completion
		planFunc(ctx, c.formatPlanProgress(requestID, phase, currentTasks, allResults))

		// Replan
		replanResult, err := c.replan(ctx, planSession, planCtx, allResults, phase)
		if err != nil {
			logger.Error("replan failed", "error", err, "phase", phase)
			break
		}

		currentTasks = replanResult.Tasks
	}

	if len(currentTasks) > 0 {
		msg.Warn(ctx, "⚠️ Maximum phase limit (%d) reached. Proceeding to final response.", c.maxPhases)
	}

	// Generate final response
	finalResp, err := c.generateFinalResponse(ctx, planSession, planCtx, allResults)
	if err != nil {
		*finalStatus = types.SessionStatusAborted
		msg.Notify(ctx, "💥 Failed to generate final response: %s", err.Error())
		return goerr.Wrap(err, "failed to generate final response")
	}

	msg.Notify(ctx, "💬 %s", finalResp)
	planFunc(ctx, fmt.Sprintf("✅ Completed (Request ID: %s)", requestID))

	return c.saveHistory(ctx, planSession, target, storageSvc)
}

// phaseResult stores the results of a single phase execution.
type phaseResult struct {
	phase   int
	tasks   []TaskPlan
	results []*TaskResult
}

// createSession creates and persists a new chat session.
func (c *SwarmChat) createSession(ctx context.Context, target *ticket.Ticket, message string) (*session.Session, context.Context) {
	userID := types.UserID(user.FromContext(ctx))
	slackURL := slackctx.SlackURL(ctx)

	ssn := session.NewSession(ctx, target.ID, userID, message, slackURL)
	if err := c.repository.PutSession(ctx, ssn); err != nil {
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
func (c *SwarmChat) setupMessageRouting(ctx context.Context, ssn *session.Session, target *ticket.Ticket) (func(context.Context, string), context.Context) {
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
func (c *SwarmChat) setupSlackMessageFuncs(ctx context.Context, sess *session.Session, target *ticket.Ticket) (msg.NotifyFunc, msg.TraceFunc, msg.TraceFunc, msg.WarnFunc) {
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
func (c *SwarmChat) setupStatusCheck(ctx context.Context, ssn *session.Session) context.Context {
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
func (c *SwarmChat) finishSession(ctx context.Context, ssn *session.Session, target *ticket.Ticket, finalStatus *types.SessionStatus, logger interface{ Error(string, ...any) }) {
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
func (c *SwarmChat) authorize(ctx context.Context, message string) (bool, error) {
	if err := authorizeAgentRequest(ctx, c.policyClient, c.noAuthorization, message); err != nil {
		if errors.Is(err, ErrAgentAuthPolicyNotDefined) {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nAgent execution policy is not defined. Please configure the `auth.agent` policy or use `--no-authorization` flag for development.\n\nSee: https://docs.warren.secmon-lab.com/policy.md#agent-execution-authorization")
		} else if errors.Is(err, ErrAgentAuthDenied) {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nYou are not authorized to execute agent requests. Please contact your administrator if you believe this is an error.")
		} else {
			msg.Notify(ctx, "🚫 *Authorization Failed*\n\nFailed to check authorization. Please contact your administrator.")
			return false, goerr.Wrap(err, "failed to evaluate agent auth")
		}
		return false, nil
	}
	return true, nil
}

// authorizeAgentRequest checks policy-based authorization for agent execution.
func authorizeAgentRequest(ctx context.Context, policyClient interfaces.PolicyClient, noAuthz bool, message string) error {
	logger := logging.From(ctx)

	if noAuthz {
		logger.Debug("agent authorization check bypassed due to --no-authorization flag")
		return nil
	}

	authCtx := auth.BuildAgentContext(ctx, message)

	var result struct {
		Allow bool `json:"allow"`
	}

	query := "data.auth.agent"
	err := policyClient.Query(ctx, query, authCtx, &result, opaq.WithPrintHook(func(ctx context.Context, loc opaq.PrintLocation, msg string) error {
		logger.Debug("[rego] "+msg, "loc", loc)
		return nil
	}))
	if err != nil {
		if errors.Is(err, opaq.ErrNoEvalResult) {
			logger.Warn("agent authorization policy not defined, denying by default")
			return goerr.Wrap(ErrAgentAuthPolicyNotDefined, "agent authorization policy not defined")
		}
		return goerr.Wrap(err, "failed to evaluate agent authorization policy")
	}

	logger.Debug("agent authorization result", "input", authCtx, "output", result)

	if !result.Allow {
		logger.Warn("agent authorization failed", "message", message)
		return goerr.Wrap(ErrAgentAuthDenied, "agent request denied by policy", goerr.V("message", message))
	}

	return nil
}

// loadHistory loads the chat history for the ticket.
func (c *SwarmChat) loadHistory(ctx context.Context, target *ticket.Ticket, storageSvc *storage.Service) (*gollem.History, error) {
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

// saveHistory saves the updated chat history after execution.
func (c *SwarmChat) saveHistory(ctx context.Context, planSession gollem.Session, target *ticket.Ticket, storageSvc *storage.Service) error {
	logger := logging.From(ctx)

	newHistory, err := planSession.History()
	if err != nil {
		return goerr.Wrap(err, "failed to get history from planning session")
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

	if newHistory.Version > 0 && c.storageClient != nil {
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

// formatPlanProgress formats the plan progress for display.
func (c *SwarmChat) formatPlanProgress(requestID string, currentPhase int, currentTasks []TaskPlan, completedPhases []*phaseResult) string {
	var b []byte
	b = fmt.Appendf(b, "⟳ Working... (Request ID: %s)\n", requestID)
	b = fmt.Appendf(b, "*Phase %d*\n\n", currentPhase)

	// Show completed phases
	for _, pr := range completedPhases {
		b = fmt.Appendf(b, "*Phase %d - Completed*\n", pr.phase)
		for _, r := range pr.results {
			if r.Error != nil {
				b = fmt.Appendf(b, "❌ ~%s~ (error)\n", r.Title)
			} else {
				b = fmt.Appendf(b, "✅ ~%s~\n", r.Title)
			}
		}
		b = append(b, '\n')
	}

	// Show current phase tasks (if not already in completedPhases)
	isCurrentPhaseCompleted := false
	for _, pr := range completedPhases {
		if pr.phase == currentPhase {
			isCurrentPhaseCompleted = true
			break
		}
	}
	if !isCurrentPhaseCompleted {
		for _, t := range currentTasks {
			b = fmt.Appendf(b, "⟳ %s\n", t.Title)
		}
	}

	return string(b)
}
