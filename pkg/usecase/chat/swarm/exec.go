package swarm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/model/agent"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/tool/base"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	ssnutil "github.com/secmon-lab/warren/pkg/utils/session"
)

// executePhase runs all tasks in parallel and waits for all to complete.
// All task messages are posted upfront as "waiting" before any execution begins.
// Each task's result context block is posted immediately upon completion.
func (c *SwarmChat) executePhase(ctx context.Context, tasks []TaskPlan, target *ticket.Ticket, ssn *session.Session, pc *planningContext, storageSvc *storage.Service) []*TaskResult {
	results := make([]*TaskResult, len(tasks))

	// Pre-create all task message routings (posts "waiting" messages to Slack)
	type taskRouting struct {
		ctx           context.Context
		markCompleted func()
	}
	routings := make([]taskRouting, len(tasks))
	for i, task := range tasks {
		taskCtx, markCompleted := c.setupTaskMessageRouting(ctx, ssn, target, task.Title)
		routings[i] = taskRouting{ctx: taskCtx, markCompleted: markCompleted}
	}

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t TaskPlan, r taskRouting) {
			defer wg.Done()
			results[idx] = c.executeTask(ctx, t, target, ssn, r.ctx, r.markCompleted, storageSvc)
			// Post result context block immediately upon task completion
			c.postTaskResult(ctx, t, results[idx], target)
		}(i, task, routings[i])
	}

	wg.Wait()

	return results
}

// postTaskResult posts a single task result context block to Slack.
func (c *SwarmChat) postTaskResult(ctx context.Context, task TaskPlan, result *TaskResult, target *ticket.Ticket) {
	if c.slackService == nil || target.SlackThread == nil {
		return
	}
	if result == nil {
		return
	}
	threadSvc := c.slackService.NewThread(*target.SlackThread)
	escaped := escapeSlackMrkdwn(task.Title)
	var blockText string
	if result.Result == "" {
		blockText = fmt.Sprintf("📋 *[%s]*\n\n_(no result)_", escaped)
	} else {
		const slackTextObjectMaxLen = 3000
		prefix := fmt.Sprintf("📋 *[%s]*\n\n", escaped)
		maxResultLen := slackTextObjectMaxLen - len(prefix)
		if maxResultLen < 0 {
			maxResultLen = 0
		}
		blockText = prefix + truncateResult(escapeSlackMrkdwn(result.Result), maxResultLen)
	}
	if err := threadSvc.PostContextBlock(ctx, blockText); err != nil {
		logging.From(ctx).Error("failed to post task completion context block", "error", err)
	}
}

// postDivider posts a divider to the Slack thread if available.
func (c *SwarmChat) postDivider(ctx context.Context, target *ticket.Ticket) {
	if c.slackService == nil || target.SlackThread == nil {
		return
	}
	threadSvc := c.slackService.NewThread(*target.SlackThread)
	if err := threadSvc.PostDivider(ctx); err != nil {
		logging.From(ctx).Error("failed to post divider", "error", err)
	}
}

// executeTask executes a single task with its own agent and trace context.
func (c *SwarmChat) executeTask(ctx context.Context, task TaskPlan, target *ticket.Ticket, ssn *session.Session, taskCtx context.Context, markCompleted func(), storageSvc *storage.Service) *TaskResult {
	logger := logging.From(ctx)
	result := &TaskResult{
		TaskID: task.ID,
		Title:  task.Title,
	}

	msg.Trace(taskCtx, "Starting...")

	// Generate task system prompt
	taskPrompt, err := generateTaskPrompt(ctx, task)
	if err != nil {
		result.Error = err
		msg.Trace(taskCtx, "❌ Failed to generate prompt: %s", err.Error())
		return result
	}

	// Filter tools for this task (all tools including base must be explicitly planned)
	filteredTools := filterToolSets(ctx, c.tools, task.Tools)

	// Add base action tool only if any warren_* tools are in the task's tool list
	// Note: SlackUpdate (PostFinding) is intentionally omitted in swarm mode.
	// Individual task results are posted as context blocks instead.
	baseAction := base.New(c.repository, target.ID, base.WithLLMClient(c.llmClient))
	if filtered := filterToolSets(ctx, []gollem.ToolSet{baseAction}, task.Tools); len(filtered) > 0 {
		filteredTools = append(filteredTools, baseAction)
	}

	// Filter sub-agents for this task
	filteredSubAgents := filterSubAgents(c.subAgents, task.SubAgents)
	gollemSubAgents := make([]*gollem.SubAgent, len(filteredSubAgents))

	// Setup budget tracker and sub-agent options
	// Note: Task agents and their sub-agents do not get WithHistoryRepository
	// as they are stateless, single-use tools in the swarm. Sharing the main
	// history would cause them to load the parent's conversation, leading to
	// context confusion and API errors like "tool_use without tool_result".
	var tracker *BudgetTracker
	var subAgentOpts []gollem.Option
	if c.budgetStrategy != nil {
		tracker = newBudgetTracker(c.budgetStrategy)
		budgetMW := newBudgetToolMiddleware(tracker)
		subAgentOpts = append(subAgentOpts, gollem.WithToolMiddleware(budgetMW))
	}

	// Inject options into sub-agents' child agents
	for i, sa := range filteredSubAgents {
		inner := sa.Inner()
		gollem.WithSubAgentOptions(subAgentOpts...)(inner)
		gollemSubAgents[i] = inner
	}

	// Build agent options
	agentOpts := []gollem.Option{
		gollem.WithToolSets(filteredTools...),
		gollem.WithSubAgents(gollemSubAgents...),
		gollem.WithResponseMode(gollem.ResponseModeBlocking),
		gollem.WithSystemPrompt(taskPrompt),
	}

	// Setup trace handler for this task
	handler := trace.HandlerFrom(ctx)
	if handler != nil {
		taskHandler := trace.AsChildAgent(handler, task.Title)
		agentOpts = append(agentOpts, gollem.WithTrace(taskHandler))
	}

	// Add content block middleware for tracing LLM responses
	traceState := &taskTraceState{}
	agentOpts = append(agentOpts,
		gollem.WithContentBlockMiddleware(llm.NewCompactionMiddleware(c.llmClient, logging.From(ctx))),
		gollem.WithContentBlockMiddleware(newTaskTraceMW(taskCtx, traceState)),
	)

	// Add budget middleware before task tool middleware (so it can block execution)
	if tracker != nil {
		agentOpts = append(agentOpts, gollem.WithToolMiddleware(newBudgetToolMiddleware(tracker)))
	}
	agentOpts = append(agentOpts, gollem.WithToolMiddleware(newTaskToolMiddleware(taskCtx, traceState)))

	// Create and execute agent
	gollemAgent := gollem.New(c.llmClient, agentOpts...)

	resp, err := gollemAgent.Execute(taskCtx, gollem.Text(task.Description))
	if err != nil {
		result.Error = err
		msg.Trace(taskCtx, "❌ Task failed: %s", err.Error())
		logger.Error("task execution failed", "task_id", task.ID, "error", err)
		return result
	}

	// Check if budget was exceeded and generate handover info
	if tracker != nil && tracker.status() == BudgetHardLimit {
		result.BudgetExceeded = true
		handover := tracker.GenerateHandoverInfo()
		if resp != nil && !resp.IsEmpty() {
			result.Result = resp.String() + "\n\n" + handover
		} else {
			result.Result = handover
		}
		msg.Trace(taskCtx, "⚠️ Budget exceeded — task terminated with handover info")
		markCompleted()
		return result
	}

	if resp != nil && !resp.IsEmpty() {
		result.Result = resp.String()
	}

	markCompleted()
	msg.Trace(taskCtx, "Completed")

	return result
}

// setupTaskMessageRouting creates task-specific msg routing with title-prefixed trace.
// The initial "Waiting..." message is posted immediately so all task messages appear upfront.
// Returns the new context and a function to mark the task as completed (changes emoji).
func (c *SwarmChat) setupTaskMessageRouting(ctx context.Context, ssn *session.Session, target *ticket.Ticket, taskTitle string) (context.Context, func()) {
	if c.slackService == nil || target.SlackThread == nil {
		return ctx, func() {}
	}

	threadSvc := c.slackService.NewThread(*target.SlackThread)

	// Post initial "waiting" message immediately
	completed := false
	escaped := escapeSlackMrkdwn(taskTitle)
	initialMsg := fmt.Sprintf("🕐 *[%s]*\n\nWaiting...", escaped)
	updateFunc := threadSvc.NewUpdatableMessage(ctx, initialMsg)

	taskTraceFunc := func(ctx context.Context, message string) {
		emoji := "⏳"
		if completed {
			emoji = "✅"
		}
		prefixed := fmt.Sprintf("%s *[%s]*\n\n> %s", emoji, escaped, escapeSlackMrkdwn(message))
		m := session.NewMessage(ctx, ssn.ID, session.MessageTypeTrace, prefixed)
		if err := c.repository.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}

		updateFunc(ctx, prefixed)
	}
	markCompleted := func() {
		completed = true
	}

	// Notify and warn use the task trace function to prefix messages
	notifyFunc := func(ctx context.Context, message string) {
		taskTraceFunc(ctx, message)
	}
	warnFunc := func(ctx context.Context, message string) {
		taskTraceFunc(ctx, "⚠️ "+message)
	}

	return msg.With(ctx, notifyFunc, taskTraceFunc, warnFunc), markCompleted
}

// filterToolSets filters tool sets to only include tools matching the given names.
func filterToolSets(ctx context.Context, allTools []gollem.ToolSet, allowedNames []string) []gollem.ToolSet {
	if len(allowedNames) == 0 {
		return nil
	}

	nameSet := make(map[string]bool, len(allowedNames))
	for _, name := range allowedNames {
		nameSet[name] = true
	}

	var filtered []gollem.ToolSet
	for _, ts := range allTools {
		specs, err := ts.Specs(ctx)
		if err != nil {
			continue
		}
		for _, spec := range specs {
			if nameSet[spec.Name] {
				filtered = append(filtered, ts)
				break
			}
		}
	}
	return filtered
}

// filterSubAgents filters sub-agents to only include those matching the given names.
func filterSubAgents(allAgents []*agent.SubAgent, allowedNames []string) []*agent.SubAgent {
	if len(allowedNames) == 0 {
		return nil
	}

	nameSet := make(map[string]bool, len(allowedNames))
	for _, name := range allowedNames {
		nameSet[name] = true
	}

	var filtered []*agent.SubAgent
	for _, sa := range allAgents {
		spec := sa.Inner().Spec()
		if nameSet[spec.Name] {
			filtered = append(filtered, sa)
		}
	}
	return filtered
}

// taskTraceState holds shared state between content block and tool middlewares
// to display LLM thinking text alongside tool execution.
type taskTraceState struct {
	// pendingThinking holds LLM text that was returned together with function calls.
	// It is consumed by the next tool execution trace.
	pendingThinking string
}

// newTaskTraceMW creates a content block middleware that captures LLM text responses.
// When the response contains both text and function calls, the text is stored as
// pending thinking and displayed alongside the next tool execution.
// When the response contains only text (no function calls), it is traced immediately.
func newTaskTraceMW(taskCtx context.Context, state *taskTraceState) gollem.ContentBlockMiddleware {
	return gollem.ContentBlockMiddleware(func(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
		return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
			resp, err := next(ctx, req)
			if err != nil || resp == nil || len(resp.Texts) == 0 {
				return resp, err
			}

			combined := resp.Texts[0]
			for _, t := range resp.Texts[1:] {
				combined += "\n" + t
			}

			if len(resp.FunctionCalls) > 0 {
				// Text came with function calls — store for tool middleware to display
				state.pendingThinking = combined
			} else {
				// Text only — trace immediately
				msg.Trace(taskCtx, "💭 %s", combined)
			}
			return resp, err
		}
	})
}

// newTaskToolMiddleware creates a tool middleware with status check and tracing for a task.
// If there is pending thinking text from the content block middleware, it is prepended
// to the tool execution trace message.
func newTaskToolMiddleware(taskCtx context.Context, state *taskTraceState) gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			if err := ssnutil.CheckStatus(ctx); err != nil {
				return &gollem.ToolExecResponse{
					Error: err,
				}, nil
			}

			if !base.IgnorableTool(req.Tool.Name) {
				if state.pendingThinking != "" {
					msg.Trace(taskCtx, "💭 %s\n🤖 Execute: `%s`", state.pendingThinking, req.Tool.Name)
					state.pendingThinking = ""
				} else {
					msg.Trace(taskCtx, "🤖 Execute: `%s`", req.Tool.Name)
				}
			}

			resp, err := next(ctx, req)

			if resp != nil && resp.Error != nil {
				msg.Trace(taskCtx, "❌ Error: %s", resp.Error.Error())
			}

			return resp, err
		}
	}
}

// truncateResult truncates a string so that its byte length does not exceed maxBytes.
// If truncated, "..." is appended (included within the maxBytes budget).
// Truncation is performed at rune boundaries to avoid breaking multi-byte characters.
func truncateResult(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	const ellipsis = "..."
	budget := maxBytes - len(ellipsis)
	if budget < 0 {
		budget = 0
	}
	used := 0
	for _, r := range s {
		runeBytes := utf8.RuneLen(r)
		if used+runeBytes > budget {
			break
		}
		used += runeBytes
	}
	return s[:used] + ellipsis
}

// escapeSlackMrkdwn escapes special characters for Slack mrkdwn format.
func escapeSlackMrkdwn(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
