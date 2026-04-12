package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	gollemTrace "github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/eval"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/usecase/eval/mock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
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

	// Capture msg.Notify/Trace/Warn output for evaluation
	capture := newMessageCapture()
	ctx = msg.With(ctx, capture.notifyFunc(), capture.traceFunc(), capture.warnFunc())

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
		return r.buildEvalResult(ctx, runID, startTime, capture.agentOutput())
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
	return r.buildEvalResult(ctx, runID, startTime, capture.agentOutput())
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

func (r *Runner) buildEvalResult(ctx context.Context, runID string, startTime time.Time, agentOutput string) (*eval.EvalResult, error) {
	logger := logging.From(ctx)
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
		AgentOutput:  agentOutput,
		TotalTokens:  totalTokens,
	}

	// Try to load gollem trace for agent-based evaluation
	gollemTraceData := loadGollemTrace(filepath.Join(r.scenarioDir, "traces"), runID)
	if gollemTraceData != nil {
		logger.Info("eval: using agent-based evaluation with gollem trace", "trace_id", runID)
		return EvaluateWithAgent(ctx, trace, gollemTraceData, r.scenario.Expectations, agentOutput, r.llmClient)
	}

	// Fallback to basic evaluation
	logger.Info("eval: gollem trace not found, using basic evaluation")
	return Evaluate(ctx, trace, r.scenario.Expectations, r.llmClient)
}

// loadGollemTrace loads a gollem trace JSON file from the traces directory.
func loadGollemTrace(tracesDir, traceID string) *gollemTrace.Trace {
	filePath := filepath.Join(tracesDir, traceID+".json")
	data, err := os.ReadFile(filePath) // #nosec G304 -- path from scenario dir + trace ID
	if err != nil {
		return nil
	}

	var t gollemTrace.Trace
	if err := json.Unmarshal(data, &t); err != nil {
		return nil
	}

	return &t
}

// messageCapture collects msg.Notify/Trace/Warn messages during eval execution.
type messageCapture struct {
	mu       sync.Mutex
	messages []string
}

func newMessageCapture() *messageCapture {
	return &messageCapture{}
}

func (c *messageCapture) append(prefix, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, fmt.Sprintf("[%s] %s", prefix, message))
}

func (c *messageCapture) notifyFunc() msg.NotifyFunc {
	return func(_ context.Context, message string) {
		c.append("notify", message)
	}
}

func (c *messageCapture) traceFunc() msg.TraceFunc {
	return func(_ context.Context, message string) {
		c.append("trace", message)
	}
}

func (c *messageCapture) warnFunc() msg.WarnFunc {
	return func(_ context.Context, message string) {
		c.append("warn", message)
	}
}

// agentOutput returns all captured messages joined as a single string.
func (c *messageCapture) agentOutput() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.Join(c.messages, "\n")
}
