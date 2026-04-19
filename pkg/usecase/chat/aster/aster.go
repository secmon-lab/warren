package aster

import (
	"context"
	"fmt"
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
	"github.com/secmon-lab/warren/pkg/utils/user"
)

const defaultMaxPhases = 10

// AsterChat implements chat.Strategy with parallel task execution.
type AsterChat struct {
	repository          interfaces.Repository
	llmClient           gollem.LLMClient
	storageClient       interfaces.StorageClient
	slackService        *slackService.Service
	knowledgeService    *svcknowledge.Service
	tools               []interfaces.ToolSet
	storagePrefix       string
	userSystemPrompt    string
	traceRepository     trace.Repository
	maxPhases           int
	monitorPollInterval time.Duration
	budgetStrategy      BudgetStrategy
	hitlTools           []string
}

// Option configures a AsterChat.
type Option func(*AsterChat)

// WithSlackService sets the Slack service for message routing.
func WithSlackService(svc *slackService.Service) Option {
	return func(c *AsterChat) { c.slackService = svc }
}

// WithTools sets the tool sets available to the agent.
func WithTools(tools []interfaces.ToolSet) Option {
	return func(c *AsterChat) { c.tools = append(c.tools, tools...) }
}

// WithStorageClient sets the storage client for history persistence.
func WithStorageClient(client interfaces.StorageClient) Option {
	return func(c *AsterChat) { c.storageClient = client }
}

// WithStoragePrefix sets the storage prefix for history paths.
func WithStoragePrefix(prefix string) Option {
	return func(c *AsterChat) { c.storagePrefix = prefix }
}

// WithUserSystemPrompt sets the user system prompt.
func WithUserSystemPrompt(prompt string) Option {
	return func(c *AsterChat) { c.userSystemPrompt = prompt }
}

// WithTraceRepository sets the trace repository for execution tracing.
func WithTraceRepository(repo trace.Repository) Option {
	return func(c *AsterChat) { c.traceRepository = repo }
}

// WithKnowledgeService sets the knowledge v2 service for reflection.
func WithKnowledgeService(svc *svcknowledge.Service) Option {
	return func(c *AsterChat) { c.knowledgeService = svc }
}

// WithMaxPhases sets the maximum number of execution phases.
func WithMaxPhases(n int) Option {
	return func(c *AsterChat) { c.maxPhases = n }
}

// WithMonitorPollInterval sets the session monitor polling interval.
func WithMonitorPollInterval(d time.Duration) Option {
	return func(c *AsterChat) { c.monitorPollInterval = d }
}

// WithBudgetStrategy sets the budget strategy for task execution.
// When nil (default), budget tracking is disabled and tools execute without limits.
func WithBudgetStrategy(s BudgetStrategy) Option {
	return func(c *AsterChat) { c.budgetStrategy = s }
}

// WithHITLTools sets the tool names that require human approval before execution.
func WithHITLTools(tools []string) Option {
	return func(c *AsterChat) { c.hitlTools = tools }
}

// New creates a new AsterChat with the given dependencies and options.
func New(repo interfaces.Repository, llmClient gollem.LLMClient, opts ...Option) *AsterChat {
	c := &AsterChat{
		repository:          repo,
		llmClient:           llmClient,
		maxPhases:           defaultMaxPhases,
		monitorPollInterval: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Execute implements chat.Strategy. It receives a RunContext with a pre-initialized
// Warren session and delegates to the aster execution loop.
func (c *AsterChat) Execute(ctx context.Context, rc *chat.RunContext) error {
	return c.executeAster(ctx, rc.Session, rc.Message, rc.ChatCtx)
}

// executeAster orchestrates the aster execution: plan → parallel exec → replan → loop → final response.
func (c *AsterChat) executeAster(ctx context.Context, ssn *session.Session, message string, chatCtx *chatModel.ChatContext) error {
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
	logger.Debug("aster executeAster: start", "has_trace_repo", c.traceRepository != nil, "request_id", requestID)
	if c.traceRepository != nil {
		recorder = trace.New(
			trace.WithTraceID(requestID),
			trace.WithRepository(c.traceRepository),
			trace.WithStackTrace(),
		)
		ctx = trace.WithHandler(ctx, recorder)
		defer func() {
			traceData := recorder.Trace()
			logger.Debug("aster executeAster: finishing trace",
				"has_trace", traceData != nil,
				"request_id", requestID,
			)
			if err := recorder.Finish(cleanupCtx); err != nil {
				logger.Error("failed to finish trace", "error", err)
			}
			logger.Debug("aster executeAster: trace finished")
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
		sessionMessages:  chatCtx.SessionMessages,
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
		if abortErr := checkAborted(ctx, cleanupCtx); abortErr != nil {
			return abortErr
		}
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
			return c.saveSessionHistory(ctx, planSession, *chatCtx, storageSvc)
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
				questionResult, qErr := c.handleQuestion(ctx, replanResult.Question, chatCtx, ssn)
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
		results := c.executePhase(ctx, currentTasks, chatCtx, ssn)
		allResults = append(allResults, &phaseResult{
			phase:   phase,
			tasks:   currentTasks,
			results: results,
		})

		// Check for context cancellation (abort detected by monitor)
		if abortErr := checkAborted(ctx, cleanupCtx); abortErr != nil {
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
			if abortErr := checkAborted(ctx, cleanupCtx); abortErr != nil {
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
			questionResult, err := c.handleQuestion(ctx, replanResult.Question, chatCtx, ssn)
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
	c.postDivider(ctx, chatCtx)

	// Generate final response
	finalResp, err := c.generateFinalResponse(ctx, planSession, planCtx, allResults, systemPrompt)
	if err != nil {
		if !ticketless {
			c.saveLatestHistory(cleanupCtx, planSession, target.ID, storageSvc)
		}
		if abortErr := checkAborted(ctx, cleanupCtx); abortErr != nil {
			return abortErr
		}
		msg.Notify(ctx, "💥 Failed to generate final response: %s", err.Error())
		return goerr.Wrap(err, "failed to generate final response")
	}
	if !ticketless {
		c.saveLatestHistory(ctx, planSession, target.ID, storageSvc)
	}

	msg.Notify(ctx, "💬 %s", finalResp)

	// Trigger fact knowledge reflection in background
	c.triggerFactReflection(ctx, buildReflectionSummary(allResults), chatCtx)

	if !ticketless {
		logger.Debug("aster executeAster: completed, saving history")
		return c.saveSessionHistory(ctx, planSession, *chatCtx, storageSvc)
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
// The presenter is resolved per Session.Source: Slack builds an
// UpdatableBlockMessage in the thread, Web emits a hitl_request_pending
// envelope tied to a new progress row in the Conversation, CLI rejects
// (default-deny until interactive CLI HITL is implemented).
func (c *AsterChat) handleQuestion(ctx context.Context, q *Question, chatCtx *chatModel.ChatContext, ssn *session.Session) (*questionResult, error) {
	logger := logging.From(ctx)
	logger.Info("asking question to user",
		"question", q.Question,
		"options", q.Options,
		"reason", q.Reason,
	)

	target := chatCtx.Ticket
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
	if target != nil && target.SlackThread != nil {
		hitlReq.SlackThread = *target.SlackThread
	}

	// Build presenter on a fresh ProgressHandle: the handle hosts the
	// question UI (Slack block message / Web session message) and
	// transitions into an answered state once the user responds.
	initialMsg := fmt.Sprintf("❓ *Question*\n\n%s", q.Question)
	progress := chat.NewProgressHandle(ctx, chatCtx, c.slackService, c.repository, initialMsg)
	presenter := chat.NewProgressHandlePresenter(progress, "Correlating ...", user.FromContext(ctx))

	if presenter == nil {
		return nil, goerr.New("question requires a presenter but none is available for this session source")
	}

	msg.Notify(ctx, "❓ %s", q.Reason)

	result, err := hitlSvc.RequestAndWait(ctx, hitlReq, presenter)
	if err != nil {
		if chatCtx != nil && chatCtx.OnHITLEvent != nil {
			chatCtx.OnHITLEvent("resolved", hitlReq, chat.ProgressMessageID(progress))
		}
		return nil, goerr.Wrap(err, "failed to get question answer")
	}
	if chatCtx != nil && chatCtx.OnHITLEvent != nil {
		chatCtx.OnHITLEvent("resolved", result, chat.ProgressMessageID(progress))
	}

	return &questionResult{
		Question: q.Question,
		Options:  q.Options,
		Answer:   result.ResponseAnswer(),
		Comment:  result.ResponseComment(),
	}, nil
}

// saveLatestHistory saves the current planning session history as the latest snapshot.
// Errors are handled via errutil but do not interrupt execution.
func (c *AsterChat) saveLatestHistory(ctx context.Context, planSession gollem.Session, ticketID types.TicketID, storageSvc *storage.Service) {
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

// saveSessionHistory extracts history from a gollem Session and saves it.
//
// saveSessionHistory writes gollem working memory into the
// Session-scoped storage slot. When chatCtx.Session is absent (e.g.
// the resolver failed), working memory is discarded for this turn —
// there is no longer a ticket-scoped fallback (see
// chat-session-redesign Phase 7 confinement).
func (c *AsterChat) saveSessionHistory(ctx context.Context, planSession gollem.Session, chatCtx chatModel.ChatContext, storageSvc *storage.Service) error {
	newHistory, err := planSession.History()
	if err != nil {
		return goerr.Wrap(err, "failed to get history from planning session")
	}
	if chatCtx.Session == nil {
		return nil
	}
	return chat.SaveSessionHistory(ctx, chatCtx.Session.ID, storageSvc, newHistory)
}

// checkAborted checks if the context has been cancelled (e.g. by the session
// monitor detecting an abort) and, if so, notifies via cleanupCtx.
// Returns chat.ErrSessionAborted when aborted, nil otherwise.
func checkAborted(ctx context.Context, cleanupCtx context.Context) error {
	if ctx.Err() != nil {
		msg.Notify(cleanupCtx, "🛑 Execution aborted by user request.")
		return chat.ErrSessionAborted
	}
	return nil
}

// startSessionMonitor starts a background goroutine that polls session status
// and cancels the context when the session is aborted. This enables immediate
// cancellation of in-flight operations (LLM calls, tool executions) when abort
// is requested, complementing the existing checkpoint-based status checks.
func (c *AsterChat) startSessionMonitor(ctx context.Context, sessionID types.SessionID) (context.Context, func()) {
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
