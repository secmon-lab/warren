package swarm

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/agent"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/memory"
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

const defaultMaxPhases = 10

// SwarmChat implements interfaces.ChatUseCase with parallel task execution.
type SwarmChat struct {
	repository          interfaces.Repository
	llmClient           gollem.LLMClient
	policyClient        interfaces.PolicyClient
	storageClient       interfaces.StorageClient
	slackService        *slackService.Service
	memoryService       *memory.Service
	tools               []gollem.ToolSet
	subAgents           []*agent.SubAgent
	storagePrefix       string
	noAuthorization     bool
	frontendURL         string
	userSystemPrompt    string
	traceRepository     trace.Repository
	maxPhases           int
	monitorPollInterval time.Duration
	budgetStrategy      BudgetStrategy
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

// WithMonitorPollInterval sets the session monitor polling interval.
func WithMonitorPollInterval(d time.Duration) Option {
	return func(c *SwarmChat) { c.monitorPollInterval = d }
}

// WithBudgetStrategy sets the budget strategy for task execution.
// When nil (default), budget tracking is disabled and tools execute without limits.
func WithBudgetStrategy(s BudgetStrategy) Option {
	return func(c *SwarmChat) { c.budgetStrategy = s }
}

// New creates a new SwarmChat with the given dependencies and options.
func New(repo interfaces.Repository, llmClient gollem.LLMClient, policyClient interfaces.PolicyClient, opts ...Option) *SwarmChat {
	c := &SwarmChat{
		repository:          repo,
		llmClient:           llmClient,
		policyClient:        policyClient,
		maxPhases:           defaultMaxPhases,
		monitorPollInterval: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Execute processes a chat message for the specified ticket using parallel task execution.
func (c *SwarmChat) Execute(ctx context.Context, target *ticket.Ticket, message string) error {
	logger := logging.From(ctx)
	logger.Debug("swarm execute: start",
		"ticket_id", target.ID,
		"request_id", request_id.FromContext(ctx),
	)

	// Phase 1: Session setup
	ssn, ctx := c.createSession(ctx, target, message)
	logger = logging.From(ctx) // refresh logger with session_id
	logger.Debug("swarm execute: session created", "session_id", ssn.ID)

	// Phase 2: Message routing setup
	ctx = c.setupMessageRouting(ctx, ssn, target)
	logger.Debug("swarm execute: message routing set up")

	// Phase 3: Session status tracking
	ctx = c.setupStatusCheck(ctx, ssn)

	finalStatus := types.SessionStatusCompleted
	defer c.finishSession(ctx, ssn, target, &finalStatus)

	// Phase 4: Authorization
	authorized, err := c.authorize(ctx, message)
	if err != nil {
		return err
	}
	if !authorized {
		logger.Debug("swarm execute: not authorized, returning")
		return nil
	}
	logger.Debug("swarm execute: authorized, starting swarm")

	// Phase 5: Swarm execution
	if err := c.executeSwarm(ctx, target, ssn, message, &finalStatus); err != nil {
		// Session abort and context cancellation are expected outcomes
		// when a user aborts the session, not errors to report.
		if errors.Is(err, ErrSessionAborted) || errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}

// executeSwarm orchestrates the swarm execution: plan → parallel exec → replan → loop → final response.
func (c *SwarmChat) executeSwarm(ctx context.Context, target *ticket.Ticket, ssn *session.Session, message string, finalStatus *types.SessionStatus) error {
	logger := logging.From(ctx)

	// Preserve a non-cancelled context for finishSession cleanup (Slack posts, DB updates)
	cleanupCtx := context.WithoutCancel(ctx)

	// Start background session monitor for abort detection
	ctx, stopMonitor := c.startSessionMonitor(ctx, ssn.ID)
	defer stopMonitor()

	// Setup trace recorder
	var recorder *trace.Recorder
	requestID := request_id.FromContext(ctx)
	if requestID == "" {
		requestID = "unknown"
	}
	logger.Debug("swarm executeSwarm: start", "has_trace_repo", c.traceRepository != nil, "request_id", requestID)
	if c.traceRepository != nil {
		recorder = trace.New(
			trace.WithTraceID(requestID),
			trace.WithRepository(c.traceRepository),
			trace.WithStackTrace(),
		)
		ctx = trace.WithHandler(ctx, recorder)
		defer func() {
			traceData := recorder.Trace()
			logger.Debug("swarm executeSwarm: finishing trace",
				"has_trace", traceData != nil,
				"request_id", requestID,
			)
			if err := recorder.Finish(cleanupCtx); err != nil {
				logger.Error("failed to finish trace", "error", err)
			}
			logger.Debug("swarm executeSwarm: trace finished")
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
	history, err := chat.LoadHistory(ctx, c.repository, target.ID, storageSvc)
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

	// Build planning context (include base tool specs so planner can assign them)
	baseAction := base.New(c.repository, target.ID)
	allTools := make([]gollem.ToolSet, 0, len(c.tools)+1)
	allTools = append(allTools, c.tools...)
	allTools = append(allTools, baseAction)

	planCtx := &planningContext{
		message:       message,
		ticket:        target,
		alerts:        alerts,
		tools:         allTools,
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
		if abortErr := checkAborted(ctx, cleanupCtx, finalStatus); abortErr != nil {
			return abortErr
		}
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
		return c.saveSessionHistory(ctx, planSession, target.ID, storageSvc)
	}

	// Execute phases
	var allResults []*phaseResult
	currentTasks := planResult.Tasks

	for phase := 1; phase <= c.maxPhases; phase++ {
		if len(currentTasks) == 0 {
			break
		}

		// Execute all tasks in parallel
		results := c.executePhase(ctx, currentTasks, target, ssn, planCtx)
		allResults = append(allResults, &phaseResult{
			phase:   phase,
			tasks:   currentTasks,
			results: results,
		})

		// Check for context cancellation (abort detected by monitor)
		if abortErr := checkAborted(ctx, cleanupCtx, finalStatus); abortErr != nil {
			return abortErr
		}

		// Replan
		replanResult, err := c.replan(ctx, planSession, planCtx, allResults, phase)
		if err != nil {
			if abortErr := checkAborted(ctx, cleanupCtx, finalStatus); abortErr != nil {
				return abortErr
			}
			logger.Error("replan failed", "error", err, "phase", phase)
			break
		}

		currentTasks = replanResult.Tasks
	}

	if len(currentTasks) > 0 {
		msg.Warn(ctx, "⚠️ Maximum phase limit (%d) reached. Proceeding to final response.", c.maxPhases)
	}

	// Post divider before final response
	c.postDivider(ctx, target)

	// Generate final response
	finalResp, err := c.generateFinalResponse(ctx, planSession, planCtx, allResults)
	if err != nil {
		if abortErr := checkAborted(ctx, cleanupCtx, finalStatus); abortErr != nil {
			return abortErr
		}
		*finalStatus = types.SessionStatusAborted
		msg.Notify(ctx, "💥 Failed to generate final response: %s", err.Error())
		return goerr.Wrap(err, "failed to generate final response")
	}

	msg.Notify(ctx, "💬 %s", finalResp)

	logger.Debug("swarm executeSwarm: completed, saving history")
	return c.saveSessionHistory(ctx, planSession, target.ID, storageSvc)
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

	logger := logging.From(ctx).With("session_id", ssn.ID, "request_id", request_id.FromContext(ctx))
	ctx = logging.With(ctx, logger)

	if target.SlackThread != nil {
		ctx = slackctx.WithThread(ctx, *target.SlackThread)
	}

	return ssn, ctx
}

// setupMessageRouting configures Slack/CLI message routing functions in the context.
func (c *SwarmChat) setupMessageRouting(ctx context.Context, ssn *session.Session, target *ticket.Ticket) context.Context {
	if c.slackService != nil && target.SlackThread != nil {
		notifyFunc, traceFunc, warnFunc := c.setupSlackMessageFuncs(ctx, ssn, target)
		ctx = msg.With(ctx, notifyFunc, traceFunc, warnFunc)

		// Post request ID as a context block immediately
		requestID := request_id.FromContext(ctx)
		if requestID == "" {
			requestID = "unknown"
		}
		verbs := []string{
			"Investigating", "Analyzing", "Processing", "Inspecting",
			"Examining", "Scanning", "Assessing", "Evaluating",
			"Reviewing", "Probing", "Surveying", "Diagnosing",
			"Exploring", "Scrutinizing", "Correlating", "Parsing",
			"Decoding", "Interpreting", "Triaging", "Resolving",
		}
		verb := verbs[rand.IntN(len(verbs))]
		threadSvc := c.slackService.NewThread(*target.SlackThread)
		if err := threadSvc.PostContextBlock(ctx, fmt.Sprintf("%s ... (ID: `%s`)", verb, requestID)); err != nil {
			logging.From(ctx).Error("failed to post request ID", "error", err)
		}
	}

	return ctx
}

// setupSlackMessageFuncs creates Slack message routing functions for notify, trace, and warn.
func (c *SwarmChat) setupSlackMessageFuncs(ctx context.Context, sess *session.Session, target *ticket.Ticket) (msg.NotifyFunc, msg.TraceFunc, msg.WarnFunc) {
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

	var traceUpdateFunc func(context.Context, string)
	traceFunc := func(ctx context.Context, message string) {
		m := session.NewMessage(ctx, sess.ID, session.MessageTypeTrace, message)
		if err := c.repository.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}

		if traceUpdateFunc == nil {
			traceUpdateFunc = threadSvc.NewUpdatableMessage(ctx, message)
		} else {
			traceUpdateFunc(ctx, message)
		}
	}

	warnFunc := func(ctx context.Context, message string) {
		m := session.NewMessage(ctx, sess.ID, session.MessageTypeWarning, message)
		if err := c.repository.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}
		if err := threadSvc.PostComment(ctx, message); err != nil {
			errutil.Handle(ctx, err)
		}
	}

	return notifyFunc, traceFunc, warnFunc
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
func (c *SwarmChat) finishSession(ctx context.Context, ssn *session.Session, target *ticket.Ticket, finalStatus *types.SessionStatus) {
	logger := logging.From(ctx)
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

// saveSessionHistory extracts history from a gollem Session and saves it via the shared SaveHistory function.
func (c *SwarmChat) saveSessionHistory(ctx context.Context, planSession gollem.Session, ticketID types.TicketID, storageSvc *storage.Service) error {
	newHistory, err := planSession.History()
	if err != nil {
		return goerr.Wrap(err, "failed to get history from planning session")
	}
	return chat.SaveHistory(ctx, c.repository, c.storageClient, storageSvc, ticketID, newHistory)
}

// checkAborted checks if the context has been cancelled (e.g. by the session
// monitor detecting an abort) and, if so, sets finalStatus and notifies via
// cleanupCtx. Returns ErrSessionAborted when aborted, nil otherwise.
func checkAborted(ctx context.Context, cleanupCtx context.Context, finalStatus *types.SessionStatus) error {
	if ctx.Err() != nil {
		*finalStatus = types.SessionStatusAborted
		msg.Notify(cleanupCtx, "🛑 Execution aborted by user request.")
		return ErrSessionAborted
	}
	return nil
}

// startSessionMonitor starts a background goroutine that polls session status
// and cancels the context when the session is aborted. This enables immediate
// cancellation of in-flight operations (LLM calls, tool executions) when abort
// is requested, complementing the existing checkpoint-based status checks.
func (c *SwarmChat) startSessionMonitor(ctx context.Context, sessionID types.SessionID) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)
		ticker := time.NewTicker(c.monitorPollInterval)
		defer ticker.Stop()

		logger := logging.From(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s, err := c.repository.GetSession(ctx, sessionID)
				if err != nil {
					logger.Warn("session monitor: failed to get session status", "error", err, "session_id", sessionID)
					continue
				}
				if s != nil && s.Status == types.SessionStatusAborted {
					logger.Info("session monitor: abort detected, cancelling context", "session_id", sessionID)
					cancel()
					return
				}
			}
		}
	}()

	stop := func() {
		cancel()
		<-done
	}
	return ctx, stop
}
