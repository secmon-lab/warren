package amber

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	hitlService "github.com/secmon-lab/warren/pkg/service/hitl"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/service/storage"
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

// AmberChat implements interfaces.ChatUseCase with parallel task execution.
type AmberChat struct {
	repository          interfaces.Repository
	llmClient           gollem.LLMClient
	policyClient        interfaces.PolicyClient
	storageClient       interfaces.StorageClient
	slackService        *slackService.Service
	knowledgeService    *svcknowledge.Service
	tools               []interfaces.ToolSet
	storagePrefix       string
	noAuthorization     bool
	frontendURL         string
	userSystemPrompt    string
	traceRepository     trace.Repository
	maxPhases           int
	monitorPollInterval time.Duration
	budgetStrategy      BudgetStrategy
	hitlTools           []string
}

// Option configures a AmberChat.
type Option func(*AmberChat)

// WithSlackService sets the Slack service for message routing.
func WithSlackService(svc *slackService.Service) Option {
	return func(c *AmberChat) { c.slackService = svc }
}

// WithTools sets the tool sets available to the agent.
func WithTools(tools []interfaces.ToolSet) Option {
	return func(c *AmberChat) { c.tools = append(c.tools, tools...) }
}

// WithStorageClient sets the storage client for history persistence.
func WithStorageClient(client interfaces.StorageClient) Option {
	return func(c *AmberChat) { c.storageClient = client }
}

// WithStoragePrefix sets the storage prefix for history paths.
func WithStoragePrefix(prefix string) Option {
	return func(c *AmberChat) { c.storagePrefix = prefix }
}

// WithNoAuthorization disables policy-based authorization checks.
func WithNoAuthorization(noAuthz bool) Option {
	return func(c *AmberChat) { c.noAuthorization = noAuthz }
}

// WithFrontendURL sets the frontend URL for session links.
func WithFrontendURL(url string) Option {
	return func(c *AmberChat) { c.frontendURL = url }
}

// WithUserSystemPrompt sets the user system prompt.
func WithUserSystemPrompt(prompt string) Option {
	return func(c *AmberChat) { c.userSystemPrompt = prompt }
}

// WithTraceRepository sets the trace repository for execution tracing.
func WithTraceRepository(repo trace.Repository) Option {
	return func(c *AmberChat) { c.traceRepository = repo }
}

// WithKnowledgeService sets the knowledge v2 service for reflection.
func WithKnowledgeService(svc *svcknowledge.Service) Option {
	return func(c *AmberChat) { c.knowledgeService = svc }
}

// WithMaxPhases sets the maximum number of execution phases.
func WithMaxPhases(n int) Option {
	return func(c *AmberChat) { c.maxPhases = n }
}

// WithMonitorPollInterval sets the session monitor polling interval.
func WithMonitorPollInterval(d time.Duration) Option {
	return func(c *AmberChat) { c.monitorPollInterval = d }
}

// WithBudgetStrategy sets the budget strategy for task execution.
// When nil (default), budget tracking is disabled and tools execute without limits.
func WithBudgetStrategy(s BudgetStrategy) Option {
	return func(c *AmberChat) { c.budgetStrategy = s }
}

// WithHITLTools sets the tool names that require human approval before execution.
func WithHITLTools(tools []string) Option {
	return func(c *AmberChat) { c.hitlTools = tools }
}

// New creates a new AmberChat with the given dependencies and options.
func New(repo interfaces.Repository, llmClient gollem.LLMClient, policyClient interfaces.PolicyClient, opts ...Option) *AmberChat {
	c := &AmberChat{
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

// Execute processes a chat message using parallel task execution.
// The ChatContext must be pre-built by the caller with all necessary data.
func (c *AmberChat) Execute(ctx context.Context, message string, chatCtx chatModel.ChatContext) error {
	target := chatCtx.Ticket
	logger := logging.From(ctx)
	logger.Debug("amber execute: start",
		"ticket_id", target.ID,
		"request_id", request_id.FromContext(ctx),
	)

	// Phase 1: Session setup
	ssn, ctx := c.createSession(ctx, target, message)
	logger = logging.From(ctx) // refresh logger with session_id
	logger.Debug("amber execute: session created", "session_id", ssn.ID)

	// Phase 2: Message routing setup
	ctx = c.setupMessageRouting(ctx, ssn, target)
	logger.Debug("amber execute: message routing set up")

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
		logger.Debug("amber execute: not authorized, returning")
		return nil
	}
	logger.Debug("amber execute: authorized, starting execution")

	// Phase 5: Main execution
	if err := c.executeAmber(ctx, ssn, message, &finalStatus, &chatCtx); err != nil {
		// Session abort and context cancellation are expected outcomes
		// when a user aborts the session, not errors to report.
		if errors.Is(err, ErrSessionAborted) || errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}

// executeAmber orchestrates the amber execution: plan → parallel exec → replan → loop → final response.
func (c *AmberChat) executeAmber(ctx context.Context, ssn *session.Session, message string, finalStatus *types.SessionStatus, chatCtx *chatModel.ChatContext) error {
	target := chatCtx.Ticket
	ticketless := chatCtx.IsTicketless()
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
	logger.Debug("amber executeAmber: start", "has_trace_repo", c.traceRepository != nil, "request_id", requestID)
	if c.traceRepository != nil {
		recorder = trace.New(
			trace.WithTraceID(requestID),
			trace.WithRepository(c.traceRepository),
			trace.WithStackTrace(),
		)
		ctx = trace.WithHandler(ctx, recorder)
		defer func() {
			traceData := recorder.Trace()
			logger.Debug("amber executeAmber: finishing trace",
				"has_trace", traceData != nil,
				"request_id", requestID,
			)
			if err := recorder.Finish(cleanupCtx); err != nil {
				logger.Error("failed to finish trace", "error", err)
			}
			logger.Debug("amber executeAmber: trace finished")
		}()
	}

	// Start root agent execution span
	handler := trace.HandlerFrom(ctx)
	if handler != nil {
		ctx = handler.StartAgentExecute(ctx)
		defer handler.EndAgentExecute(ctx, nil)
	}

	storageSvc := storage.New(c.storageClient, storage.WithPrefix(c.storagePrefix))

	// Build planning context from ChatContext
	planCtx := &planningContext{
		message:          message,
		ticket:           target,
		alerts:           chatCtx.Alerts,
		tools:            c.tools,
		userPrompt:       c.userSystemPrompt,
		lang:             lang.From(ctx),
		requesterID:      string(types.UserID(user.FromContext(ctx))),
		threadComments:   chatCtx.ThreadComments,
		slackHistory:     chatCtx.SlackHistory,
		knowledgeService: c.knowledgeService,
	}

	// Generate system prompt once (shared across plan/replan/final sessions)
	var systemPrompt string
	if ticketless {
		tlpc := &ticketlessPlanningContext{
			message:          message,
			tools:            c.tools,
			userPrompt:       c.userSystemPrompt,
			lang:             lang.From(ctx),
			requesterID:      string(types.UserID(user.FromContext(ctx))),
			history:          chatCtx.SlackHistory,
			knowledgeService: c.knowledgeService,
		}
		var err error
		systemPrompt, err = generateTicketlessSystemPrompt(ctx, tlpc)
		if err != nil {
			return goerr.Wrap(err, "failed to generate ticketless system prompt")
		}
	} else {
		var err error
		systemPrompt, err = generateSystemPrompt(ctx, planCtx)
		if err != nil {
			return goerr.Wrap(err, "failed to generate system prompt")
		}
	}

	// Create planning session with history
	planSession, err := c.llmClient.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(planSchema),
		gollem.WithSessionSystemPrompt(systemPrompt),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to create planning session")
	}
	if chatCtx.History != nil {
		if err := planSession.AppendHistory(chatCtx.History); err != nil {
			logger.Warn("failed to append history to planning session", "error", err)
		}
	}

	// Planning phase
	planResult, err := c.plan(ctx, planSession, planCtx, systemPrompt)
	if err != nil {
		if abortErr := checkAborted(ctx, cleanupCtx, finalStatus); abortErr != nil {
			return abortErr
		}
		*finalStatus = types.SessionStatusAborted
		msg.Notify(ctx, "💥 Planning failed: %s", err.Error())
		return goerr.Wrap(err, "planning failed")
	}
	if !ticketless {
		c.saveLatestHistory(ctx, planSession, target.ID, storageSvc)
	}

	// Post initial message
	if planResult.Message != "" {
		msg.Notify(ctx, "💬 %s", planResult.Message)
	}

	// Direct response (no tasks)
	if len(planResult.Tasks) == 0 {
		if !ticketless {
			return c.saveSessionHistory(ctx, planSession, target.ID, storageSvc)
		}
		return nil
	}

	// Execute phases
	var allResults []*phaseResult
	currentTasks := planResult.Tasks

	questionPending := false // true when we need to replan after a question
	for phase := 1; phase <= c.maxPhases; phase++ {
		if len(currentTasks) == 0 && !questionPending {
			break
		}

		// Skip task execution if we're replanning after a question
		if questionPending {
			questionPending = false
			// Replan with the question answer already in allResults
			replanResult, err := c.replan(ctx, planSession, planCtx, allResults, phase, systemPrompt)
			if err != nil {
				if !ticketless {
					c.saveLatestHistory(cleanupCtx, planSession, target.ID, storageSvc)
				}
				logger.Error("replan after question failed", "error", err, "phase", phase)
				break
			}
			if !ticketless {
				c.saveLatestHistory(ctx, planSession, target.ID, storageSvc)
			}
			if replanResult.Message != "" {
				msg.Notify(ctx, "💬 %s", replanResult.Message)
			}

			// Handle question again if needed (recursive questions)
			if replanResult.Question != nil {
				questionResult, qErr := c.handleQuestion(ctx, replanResult.Question, target, ssn)
				if qErr != nil {
					logger.Error("question failed", "error", qErr, "phase", phase)
					msg.Warn(ctx, "⚠️ Question failed: %s", qErr.Error())
					break
				}
				allResults = append(allResults, &phaseResult{
					phase:          phase,
					questionResult: questionResult,
				})
				questionPending = true
				continue
			}

			currentTasks = replanResult.Tasks
			continue
		}

		// Execute all tasks in parallel
		results := c.executePhase(ctx, currentTasks, target, ssn)
		allResults = append(allResults, &phaseResult{
			phase:   phase,
			tasks:   currentTasks,
			results: results,
		})

		// Check for context cancellation (abort detected by monitor)
		if abortErr := checkAborted(ctx, cleanupCtx, finalStatus); abortErr != nil {
			if !ticketless {
				c.saveLatestHistory(cleanupCtx, planSession, target.ID, storageSvc)
			}
			return abortErr
		}

		// Replan
		replanResult, err := c.replan(ctx, planSession, planCtx, allResults, phase, systemPrompt)
		if err != nil {
			if !ticketless {
				c.saveLatestHistory(cleanupCtx, planSession, target.ID, storageSvc)
			}
			if abortErr := checkAborted(ctx, cleanupCtx, finalStatus); abortErr != nil {
				return abortErr
			}
			logger.Error("replan failed", "error", err, "phase", phase)
			break
		}
		if !ticketless {
			c.saveLatestHistory(ctx, planSession, target.ID, storageSvc)
		}

		// Post replan message if present
		if replanResult.Message != "" {
			msg.Notify(ctx, "💬 %s", replanResult.Message)
		}

		// Handle question (takes priority over tasks)
		if replanResult.Question != nil {
			questionResult, err := c.handleQuestion(ctx, replanResult.Question, target, ssn)
			if err != nil {
				logger.Error("question failed", "error", err, "phase", phase)
				msg.Warn(ctx, "⚠️ Question failed: %s", err.Error())
				break
			}

			// Add question result to allResults for next replan
			allResults = append(allResults, &phaseResult{
				phase:          phase,
				questionResult: questionResult,
			})

			// Continue to next replan iteration with the answer
			questionPending = true
			continue
		}

		currentTasks = replanResult.Tasks
	}

	if len(currentTasks) > 0 {
		msg.Warn(ctx, "⚠️ Maximum phase limit (%d) reached. Proceeding to final response.", c.maxPhases)
	}

	// Post divider before final response
	c.postDivider(ctx, target)

	// Generate final response
	finalResp, err := c.generateFinalResponse(ctx, planSession, planCtx, allResults, systemPrompt)
	if err != nil {
		if !ticketless {
			c.saveLatestHistory(cleanupCtx, planSession, target.ID, storageSvc)
		}
		if abortErr := checkAborted(ctx, cleanupCtx, finalStatus); abortErr != nil {
			return abortErr
		}
		*finalStatus = types.SessionStatusAborted
		msg.Notify(ctx, "💥 Failed to generate final response: %s", err.Error())
		return goerr.Wrap(err, "failed to generate final response")
	}
	if !ticketless {
		c.saveLatestHistory(ctx, planSession, target.ID, storageSvc)
	}

	msg.Notify(ctx, "💬 %s", finalResp)

	// Trigger fact knowledge reflection in background
	c.triggerFactReflection(ctx, buildReflectionSummary(allResults), target)

	if !ticketless {
		logger.Debug("amber executeAmber: completed, saving history")
		return c.saveSessionHistory(ctx, planSession, target.ID, storageSvc)
	}
	return nil
}

// buildReflectionSummary aggregates all task results into a text summary for reflection.
func buildReflectionSummary(allResults []*phaseResult) string {
	var sb strings.Builder
	for _, pr := range allResults {
		for i, r := range pr.results {
			if r == nil || r.Result == "" {
				continue
			}
			title := ""
			if i < len(pr.tasks) {
				title = pr.tasks[i].Title
			}
			fmt.Fprintf(&sb, "## Task: %s\n%s\n\n", title, r.Result)
		}
	}
	return sb.String()
}

// phaseResult stores the results of a single phase execution.
type phaseResult struct {
	phase          int
	tasks          []TaskPlan
	results        []*TaskResult
	questionResult *questionResult // non-nil if this phase was a question
}

// questionResult holds the outcome of a question asked to the user.
type questionResult struct {
	Question string
	Options  []string
	Answer   string
	Comment  string
}

// handleQuestion asks a question to the user via HITL service and returns the result.
func (c *AmberChat) handleQuestion(ctx context.Context, q *Question, target *ticket.Ticket, ssn *session.Session) (*questionResult, error) {
	logger := logging.From(ctx)
	logger.Info("asking question to user",
		"question", q.Question,
		"options", q.Options,
		"reason", q.Reason,
	)

	hitlSvc := hitlService.New(c.repository)

	hitlReq := &hitl.Request{
		ID:        types.NewHITLRequestID(),
		SessionID: ssn.ID,
		Type:      hitl.RequestTypeQuestion,
		Payload:   hitl.NewQuestionPayload(q.Question, q.Options),
		Status:    hitl.StatusPending,
		UserID:    user.FromContext(ctx),
		CreatedAt: time.Now(),
	}
	if target.SlackThread != nil {
		hitlReq.SlackThread = *target.SlackThread
	}

	// Build presenter
	var presenter hitlService.Presenter
	if c.slackService != nil && target.SlackThread != nil {
		threadSvc := c.slackService.NewThread(*target.SlackThread).(*slackService.ThreadService)
		// Use the session title or a generic title for the question message
		initialMsg := fmt.Sprintf("❓ *Question*\n\n%s", q.Question)
		ubm := threadSvc.NewUpdatableBlockMessage(ctx, initialMsg)
		presenter = slackService.NewQuestionPresenter(ubm, "Correlating ...", user.FromContext(ctx))
	}

	if presenter == nil {
		return nil, goerr.New("question requires a presenter but none is available")
	}

	msg.Notify(ctx, "❓ %s", q.Reason)

	result, err := hitlSvc.RequestAndWait(ctx, hitlReq, presenter)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get question answer")
	}

	return &questionResult{
		Question: q.Question,
		Options:  q.Options,
		Answer:   result.ResponseAnswer(),
		Comment:  result.ResponseComment(),
	}, nil
}

// createSession creates and persists a new chat session.
func (c *AmberChat) createSession(ctx context.Context, target *ticket.Ticket, message string) (*session.Session, context.Context) {
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
func (c *AmberChat) setupMessageRouting(ctx context.Context, ssn *session.Session, target *ticket.Ticket) context.Context {
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
		verb := verbs[rand.IntN(len(verbs))] // #nosec G404 -- not security-sensitive, just picking a random UI verb
		threadSvc := c.slackService.NewThread(*target.SlackThread)
		if err := threadSvc.PostContextBlock(ctx, fmt.Sprintf("%s ... (ID: `%s`)", verb, requestID)); err != nil {
			logging.From(ctx).Error("failed to post request ID", "error", err)
		}
	}

	return ctx
}

// setupSlackMessageFuncs creates Slack message routing functions for notify, trace, and warn.
func (c *AmberChat) setupSlackMessageFuncs(ctx context.Context, sess *session.Session, target *ticket.Ticket) (msg.NotifyFunc, msg.TraceFunc, msg.WarnFunc) {
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
func (c *AmberChat) setupStatusCheck(ctx context.Context, ssn *session.Session) context.Context {
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
func (c *AmberChat) finishSession(ctx context.Context, ssn *session.Session, target *ticket.Ticket, finalStatus *types.SessionStatus) {
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
func (c *AmberChat) authorize(ctx context.Context, message string) (bool, error) {
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

// saveLatestHistory saves the current planning session history as the latest snapshot.
// Errors are handled via errutil but do not interrupt execution.
func (c *AmberChat) saveLatestHistory(ctx context.Context, planSession gollem.Session, ticketID types.TicketID, storageSvc *storage.Service) {
	history, err := planSession.History()
	if err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to get history for latest save"))
		return
	}
	if history == nil {
		return
	}
	if err := storageSvc.PutLatestHistory(ctx, ticketID, history); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to save latest history", goerr.V("ticket_id", ticketID)))
	}
}

// saveSessionHistory extracts history from a gollem Session and saves it via the shared SaveHistory function.
func (c *AmberChat) saveSessionHistory(ctx context.Context, planSession gollem.Session, ticketID types.TicketID, storageSvc *storage.Service) error {
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
func (c *AmberChat) startSessionMonitor(ctx context.Context, sessionID types.SessionID) (context.Context, func()) {
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
