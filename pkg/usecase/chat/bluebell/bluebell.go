package bluebell

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/cli/config"
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
	"github.com/secmon-lab/warren/pkg/utils/user"
)

const defaultMaxPhases = 10

// BluebellChat implements chat.Strategy with intent resolution and parallel task execution.
// Unlike aster, bluebell resolves intent from multiple user-defined prompts before planning,
// and always runs in agent mode (knowledge service is required).
type BluebellChat struct {
	repository          interfaces.Repository
	llmClient           gollem.LLMClient
	storageClient       interfaces.StorageClient
	slackService        *slackService.Service
	knowledgeService    *svcknowledge.Service
	tools               []interfaces.ToolSet
	storagePrefix       string
	traceRepository     trace.Repository
	maxPhases           int
	monitorPollInterval time.Duration
	budgetStrategy      BudgetStrategy
	hitlTools           []string

	userSystemPrompt string // static user system prompt (--user-system-prompt, environment info etc.)

	// bluebell-specific: user-defined prompt entries for intent resolution
	promptEntries []config.PromptEntry
}

// Option configures a BluebellChat.
type Option func(*BluebellChat)

// WithSlackService sets the Slack service for message routing.
func WithSlackService(svc *slackService.Service) Option {
	return func(c *BluebellChat) { c.slackService = svc }
}

// WithTools sets the tool sets available to the agent.
func WithTools(tools []interfaces.ToolSet) Option {
	return func(c *BluebellChat) { c.tools = append(c.tools, tools...) }
}

// WithStorageClient sets the storage client for history persistence.
func WithStorageClient(client interfaces.StorageClient) Option {
	return func(c *BluebellChat) { c.storageClient = client }
}

// WithStoragePrefix sets the storage prefix for history paths.
func WithStoragePrefix(prefix string) Option {
	return func(c *BluebellChat) { c.storagePrefix = prefix }
}

// WithUserSystemPrompt sets the static user system prompt (environment info, etc.).
func WithUserSystemPrompt(prompt string) Option {
	return func(c *BluebellChat) { c.userSystemPrompt = prompt }
}

// WithTraceRepository sets the trace repository for execution tracing.
func WithTraceRepository(repo trace.Repository) Option {
	return func(c *BluebellChat) { c.traceRepository = repo }
}

// WithKnowledgeService sets the knowledge v2 service (required for bluebell).
func WithKnowledgeService(svc *svcknowledge.Service) Option {
	return func(c *BluebellChat) { c.knowledgeService = svc }
}

// WithMaxPhases sets the maximum number of execution phases.
func WithMaxPhases(n int) Option {
	return func(c *BluebellChat) { c.maxPhases = n }
}

// WithMonitorPollInterval sets the session monitor polling interval.
func WithMonitorPollInterval(d time.Duration) Option {
	return func(c *BluebellChat) { c.monitorPollInterval = d }
}

// WithBudgetStrategy sets the budget strategy for task execution.
func WithBudgetStrategy(s BudgetStrategy) Option {
	return func(c *BluebellChat) { c.budgetStrategy = s }
}

// WithHITLTools sets the tool names that require human approval before execution.
func WithHITLTools(tools []string) Option {
	return func(c *BluebellChat) { c.hitlTools = tools }
}

// WithPromptEntries sets the user-defined prompt entries for intent resolution.
func WithPromptEntries(entries []config.PromptEntry) Option {
	return func(c *BluebellChat) { c.promptEntries = entries }
}

// New creates a new BluebellChat with the given dependencies and options.
// Returns error if knowledge service is not configured (required for bluebell).
func New(repo interfaces.Repository, llmClient gollem.LLMClient, opts ...Option) (*BluebellChat, error) {
	c := &BluebellChat{
		repository:          repo,
		llmClient:           llmClient,
		maxPhases:           defaultMaxPhases,
		monitorPollInterval: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}

	if c.knowledgeService == nil {
		return nil, goerr.New("bluebell strategy requires knowledge service to be configured")
	}

	return c, nil
}

// Execute implements chat.Strategy. It receives a RunContext with a pre-initialized
// Warren session and delegates to the bluebell execution loop.
func (c *BluebellChat) Execute(ctx context.Context, rc *chat.RunContext) error {
	return c.executeBluebell(ctx, rc.Session, rc.Message, rc.ChatCtx)
}

// executeBluebell orchestrates: intent resolution → plan → parallel exec → replan → loop → final response.
func (c *BluebellChat) executeBluebell(ctx context.Context, ssn *session.Session, message string, chatCtx *chatModel.ChatContext) error {
	target := chatCtx.Ticket
	ticketless := chatCtx.IsTicketless()
	logger := logging.From(ctx)

	cleanupCtx := context.WithoutCancel(ctx)

	ctx, stopMonitor := c.startSessionMonitor(ctx, ssn.ID)
	defer stopMonitor()

	// Setup trace recorder
	var recorder *trace.Recorder
	requestID := request_id.FromContext(ctx)
	if requestID == "" {
		requestID = "unknown"
	}
	logger.Debug("bluebell executeBluebell: start", "has_trace_repo", c.traceRepository != nil, "request_id", requestID)
	if c.traceRepository != nil {
		recorder = trace.New(
			trace.WithTraceID(requestID),
			trace.WithRepository(c.traceRepository),
			trace.WithStackTrace(),
		)
		ctx = trace.WithHandler(ctx, recorder)
		defer func() {
			traceData := recorder.Trace()
			logger.Debug("bluebell executeBluebell: finishing trace",
				"has_trace", traceData != nil,
				"request_id", requestID,
			)
			if err := recorder.Finish(cleanupCtx); err != nil {
				logger.Error("failed to finish trace", "error", err)
			}
			logger.Debug("bluebell executeBluebell: trace finished")
		}()
	}

	handler := trace.HandlerFrom(ctx)
	if handler != nil {
		ctx = handler.StartAgentExecute(ctx)
		defer handler.EndAgentExecute(ctx, nil)
	}

	storageSvc := storage.New(c.storageClient, storage.WithPrefix(c.storagePrefix))

	// Intent resolution phase: select prompt and resolve intent
	resolved, err := c.resolveIntent(ctx, message, chatCtx)
	if err != nil {
		logger.Warn("intent resolution failed, proceeding without resolved intent", "error", err)
	}

	var resolvedIntent string
	if resolved != nil {
		resolvedIntent = resolved.Intent
		// Post context blocks showing execution status and intent
		if c.slackService != nil && target.SlackThread != nil {
			threadSvc := c.slackService.NewThread(*target.SlackThread)
			promptLabel := resolved.PromptName
			if promptLabel == "" {
				promptLabel = "(default)"
			}
			requestID := request_id.FromContext(ctx)
			if requestID == "" {
				requestID = "unknown"
			}
			verbs := []string{
				"Executing as", "Running as", "Processing as", "Operating as",
				"Engaging as", "Launching as", "Activating as", "Invoking as",
			}
			verb := verbs[rand.IntN(len(verbs))] // #nosec G404 -- not security-sensitive, just picking a random UI verb
			if postErr := threadSvc.PostContextBlock(ctx, fmt.Sprintf("%s `%s` ... (ID: `%s`)", verb, promptLabel, requestID)); postErr != nil {
				logging.From(ctx).Error("failed to post execution status", "error", postErr)
			}
			if resolvedIntent != "" {
				if postErr := threadSvc.PostContextBlock(ctx, fmt.Sprintf("💬 %s", resolvedIntent)); postErr != nil {
					logging.From(ctx).Error("failed to post intent", "error", postErr)
				}
			}
		}
	}

	// Build planning context
	planCtx := &planningContext{
		message:          message,
		ticket:           target,
		alerts:           chatCtx.Alerts,
		tools:            c.tools,
		userSystemPrompt: c.userSystemPrompt,
		resolvedIntent:   resolvedIntent,
		lang:             lang.From(ctx),
		requesterID:      string(types.UserID(user.FromContext(ctx))),
		threadComments:   chatCtx.ThreadComments,
		slackHistory:     chatCtx.SlackHistory,
		knowledgeService: c.knowledgeService,
	}

	// Generate system prompt once (shared across plan/replan/final sessions)
	systemPrompt, err := generateSystemPrompt(ctx, planCtx)
	if err != nil {
		return goerr.Wrap(err, "failed to generate system prompt")
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

	questionPending := false
	for phase := 1; phase <= c.maxPhases; phase++ {
		if len(currentTasks) == 0 && !questionPending {
			break
		}

		if questionPending {
			questionPending = false
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

		results := c.executePhase(ctx, currentTasks, target, ssn)
		allResults = append(allResults, &phaseResult{
			phase:   phase,
			tasks:   currentTasks,
			results: results,
		})

		if abortErr := checkAborted(ctx, cleanupCtx); abortErr != nil {
			if !ticketless {
				c.saveLatestHistory(cleanupCtx, planSession, target.ID, storageSvc)
			}
			return abortErr
		}

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

		if replanResult.Message != "" {
			msg.Notify(ctx, "💬 %s", replanResult.Message)
		}

		if replanResult.Question != nil {
			questionResult, err := c.handleQuestion(ctx, replanResult.Question, target, ssn)
			if err != nil {
				logger.Error("question failed", "error", err, "phase", phase)
				msg.Warn(ctx, "⚠️ Question failed: %s", err.Error())
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
	}

	if len(currentTasks) > 0 {
		msg.Warn(ctx, "⚠️ Maximum phase limit (%d) reached. Proceeding to final response.", c.maxPhases)
	}

	c.postDivider(ctx, target)

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

	c.triggerFactReflection(ctx, buildReflectionSummary(allResults), target)

	if !ticketless {
		logger.Debug("bluebell executeBluebell: completed, saving history")
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
	questionResult *questionResult
}

// questionResult holds the outcome of a question asked to the user.
type questionResult struct {
	Question string
	Options  []string
	Answer   string
	Comment  string
}

// handleQuestion asks a question to the user via HITL service and returns the result.
func (c *BluebellChat) handleQuestion(ctx context.Context, q *Question, target *ticket.Ticket, ssn *session.Session) (*questionResult, error) {
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

	var presenter hitlService.Presenter
	if c.slackService != nil && target.SlackThread != nil {
		threadSvc := c.slackService.NewThread(*target.SlackThread).(*slackService.ThreadService)
		initialMsg := fmt.Sprintf("❓ *Question*\n\n%s", q.Question)
		ubm := threadSvc.NewUpdatableBlockMessage(ctx, initialMsg)
		presenter = slackService.NewQuestionPresenter(ubm, "Correlating ...", user.FromContext(ctx))
	}

	msg.Notify(ctx, "❓ %s", q.Reason)

	// Use a no-op presenter when Slack is not available, so the HITL request
	// is still saved to the repository and can be answered via Web UI or API.
	if presenter == nil {
		presenter = hitlService.NoOpPresenter()
	}

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

// saveLatestHistory saves the current planning session history as the latest snapshot.
func (c *BluebellChat) saveLatestHistory(ctx context.Context, planSession gollem.Session, ticketID types.TicketID, storageSvc *storage.Service) {
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
func (c *BluebellChat) saveSessionHistory(ctx context.Context, planSession gollem.Session, ticketID types.TicketID, storageSvc *storage.Service) error {
	newHistory, err := planSession.History()
	if err != nil {
		return goerr.Wrap(err, "failed to get history from planning session")
	}
	return chat.SaveHistory(ctx, c.repository, c.storageClient, storageSvc, ticketID, newHistory)
}

// checkAborted checks if the context has been cancelled.
// Returns chat.ErrSessionAborted when aborted, nil otherwise.
func checkAborted(ctx context.Context, cleanupCtx context.Context) error {
	if ctx.Err() != nil {
		msg.Notify(cleanupCtx, "🛑 Execution aborted by user request.")
		return chat.ErrSessionAborted
	}
	return nil
}

// startSessionMonitor starts a background goroutine that polls session status
// and cancels the context when the session is aborted.
func (c *BluebellChat) startSessionMonitor(ctx context.Context, sessionID types.SessionID) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx) // #nosec G118 -- cancel is called in the stop() closure returned below
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
