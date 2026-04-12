package eval

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/eval"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/usecase/eval/mock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
)

// Runner executes an evaluation scenario.
type Runner struct {
	scenario    *eval.Scenario
	scenarioDir string
	uc          *usecase.UseCases
	mockAgent   *mock.Agent
	store       *ResponseStore
	recorder    *TraceRecorder
	llmClient   gollem.LLMClient // For eval (LLM judge, report generation)
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithLLMClientForEval sets the LLM client used for evaluation (judge + report).
func WithLLMClientForEval(client gollem.LLMClient) RunnerOption {
	return func(r *Runner) { r.llmClient = client }
}

// NewRunner creates a new evaluation Runner.
// It sets up the ResponseStore, MockAgent, wraps the tools, and constructs the UseCases.
func NewRunner(
	ctx context.Context,
	scenario *eval.Scenario,
	scenarioDir string,
	mockLLM gollem.LLMClient,
	ucOpts []usecase.Option,
	originalTools []interfaces.ToolSet,
	opts ...RunnerOption,
) (*Runner, error) {
	// Setup response store
	store, err := NewResponseStore(ResponsesDir(scenarioDir))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create response store")
	}

	// Setup mock agent
	mockAgent := mock.New(mockLLM, scenario.World)

	// Setup trace recorder
	recorder := NewTraceRecorder()

	// Wrap tools for eval
	wrappedTools := WrapForEval(originalTools, store, mockAgent, recorder)

	// Replace tools in usecase options
	ucOpts = append(ucOpts, usecase.WithTools(wrappedTools))

	// Create the usecase
	uc := usecase.New(ucOpts...)

	r := &Runner{
		scenario:    scenario,
		scenarioDir: scenarioDir,
		uc:          uc,
		mockAgent:   mockAgent,
		store:       store,
		recorder:    recorder,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

// RunCLI executes the scenario in CLI mode:
// 1. HandleAlert → pipeline + ticket creation
// 2. ChatFromCLI → initial_message
// 3. Evaluate + Report
func (r *Runner) RunCLI(ctx context.Context) (*eval.EvalResult, error) {
	logger := logging.From(ctx)
	startTime := time.Now()

	// Generate request ID and use it as both run ID and gollem trace ID
	ctx, runID := request_id.Generate(ctx)

	logger.Info("eval: starting CLI run",
		"scenario", r.scenario.Name,
		"run_id", runID,
	)

	// Step 1: HandleAlert
	alerts, err := r.uc.HandleAlert(ctx, r.scenario.Alert.Schema, r.scenario.Alert.Data)
	if err != nil {
		return nil, goerr.Wrap(err, "HandleAlert failed during eval")
	}

	if len(alerts) == 0 {
		logger.Warn("eval: HandleAlert produced no alerts, skipping chat phase")
		return r.buildEvalResult(ctx, runID, startTime)
	}

	// Find the ticket for the first alert
	firstAlert := alerts[0]
	logger.Info("eval: alert created",
		"alert_id", firstAlert.ID,
		"ticket_id", firstAlert.TicketID,
	)

	// Step 2: ChatFromCLI with initial_message
	if r.scenario.InitialMessage != "" {
		t := &ticket.Ticket{
			ID:       firstAlert.TicketID,
			AlertIDs: []types.AlertID{firstAlert.ID},
		}

		logger.Info("eval: sending initial message",
			"message", r.scenario.InitialMessage,
			"ticket_id", t.ID,
		)

		if err := r.uc.ChatFromCLI(ctx, t, r.scenario.InitialMessage); err != nil {
			return nil, goerr.Wrap(err, "ChatFromCLI failed during eval")
		}
	}

	// Step 3: Build eval result
	return r.buildEvalResult(ctx, runID, startTime)
}

// RunSlack executes the scenario in Slack simulation mode:
// 1. Start HTTP server with Slack event handling
// 2. HandleAlert → alert posted to Slack
// 3. Wait for ticket close
// 4. Evaluate + Report
//
// This mode requires the serve command's HTTP/Slack infrastructure.
// Implementation pending CoreDeps shared setup extraction.
func (r *Runner) RunSlack(_ context.Context) (*eval.EvalResult, error) {
	return nil, goerr.New("slack simulation mode is not yet implemented; requires serve command infrastructure (CoreDeps extraction)")
}

func (r *Runner) buildEvalResult(ctx context.Context, runID string, startTime time.Time) (*eval.EvalResult, error) {
	records := r.recorder.Records()

	var totalTokens int64
	for _, rec := range records {
		totalTokens += rec.TokensUsed
	}

	trace := &eval.Trace{
		ScenarioName: r.scenario.Name,
		RunID:        runID,
		StartTime:    startTime,
		EndTime:      time.Now(),
		ToolCalls:    records,
		TotalTokens:  totalTokens,
	}

	evalResult, err := Evaluate(ctx, trace, r.scenario.Expectations, r.llmClient)
	if err != nil {
		return nil, goerr.Wrap(err, "evaluation failed")
	}

	return evalResult, nil
}
