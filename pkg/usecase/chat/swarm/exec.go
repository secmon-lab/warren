package swarm

import (
	"context"
	"fmt"
	"sync"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/model/agent"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/tool/base"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	ssnutil "github.com/secmon-lab/warren/pkg/utils/session"
)

// executePhase runs all tasks in parallel and waits for all to complete.
// All task messages are posted upfront as "waiting" before any execution begins.
func (c *SwarmChat) executePhase(ctx context.Context, tasks []TaskPlan, target *ticket.Ticket, ssn *session.Session, pc *planningContext) []*TaskResult {
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
			results[idx] = c.executeTask(ctx, t, target, ssn, r.ctx, r.markCompleted)
		}(i, task, routings[i])
	}

	wg.Wait()
	return results
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
func (c *SwarmChat) executeTask(ctx context.Context, task TaskPlan, target *ticket.Ticket, ssn *session.Session, taskCtx context.Context, markCompleted func()) *TaskResult {
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
	for i, sa := range filteredSubAgents {
		gollemSubAgents[i] = sa.Inner()
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
		gollem.WithToolMiddleware(newTaskToolMiddleware(taskCtx, traceState)),
	)

	// Create and execute agent
	gollemAgent := gollem.New(c.llmClient, agentOpts...)

	resp, err := gollemAgent.Execute(taskCtx, gollem.Text(task.Description))
	if err != nil {
		result.Error = err
		msg.Trace(taskCtx, "❌ Task failed: %s", err.Error())
		logger.Error("task execution failed", "task_id", task.ID, "error", err)
		return result
	}

	if resp != nil && !resp.IsEmpty() {
		result.Result = resp.String()
	}

	markCompleted()
	msg.Trace(taskCtx, "Completed")

	// Post individual task completion as a context block
	if c.slackService != nil && target.SlackThread != nil && result.Result != "" {
		threadSvc := c.slackService.NewThread(*target.SlackThread)
		summary := truncateResult(result.Result, 200)
		blockText := fmt.Sprintf("📋 *[%s]*\n%s", task.Title, summary)
		if err := threadSvc.PostContextBlock(taskCtx, blockText); err != nil {
			logging.From(taskCtx).Error("failed to post task completion context block", "error", err)
		}
	}

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
	initialMsg := fmt.Sprintf("🕐 *[%s]*\n> Waiting...", taskTitle)
	updateFunc := threadSvc.NewUpdatableMessage(ctx, initialMsg)

	taskTraceFunc := func(ctx context.Context, message string) {
		emoji := "⏳"
		if completed {
			emoji = "✅"
		}
		prefixed := fmt.Sprintf("%s *[%s]*\n> %s", emoji, taskTitle, message)
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

// truncateResult truncates a string to maxLen runes, appending "..." if truncated.
func truncateResult(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
