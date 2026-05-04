package bluebell

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	hitlService "github.com/secmon-lab/warren/pkg/service/hitl"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/tool/base"
	knowledgeTool "github.com/secmon-lab/warren/pkg/tool/knowledge"
	"github.com/secmon-lab/warren/pkg/usecase/chat"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	ssnutil "github.com/secmon-lab/warren/pkg/utils/session"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// executePhase runs all tasks in parallel and waits for all to complete.
// All task messages are posted upfront as "waiting" before any execution begins.
// Each task's result context block is posted immediately upon completion.
func (c *BluebellChat) executePhase(ctx context.Context, tasks []TaskPlan, chatCtx *chatModel.ChatContext, ssn *session.Session) []*TaskResult {
	results := make([]*TaskResult, len(tasks))

	// Pre-create all task progress handles (posts the "waiting"
	// display row through the active transport sink).
	type taskRouting struct {
		ctx           context.Context
		markCompleted func()
		progress      chat.ProgressHandle
	}
	routings := make([]taskRouting, len(tasks))
	for i, task := range tasks {
		taskCtx, markCompleted, progress := c.setupTaskMessageRouting(ctx, ssn, chatCtx, task.Title)
		routings[i] = taskRouting{ctx: taskCtx, markCompleted: markCompleted, progress: progress}
	}

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t TaskPlan, r taskRouting) {
			defer wg.Done()
			results[idx] = c.executeTask(ctx, t, chatCtx, ssn, r.ctx, r.markCompleted, r.progress)
			// Post result block immediately upon task completion
			c.postTaskResult(ctx, t, results[idx], chatCtx)
		}(i, task, routings[i])
	}

	wg.Wait()

	return results
}

// postTaskResult posts a single task result block through the active
// transport sink (Slack section block / Web persisted response /
// CLI stdout line).
func (c *BluebellChat) postTaskResult(ctx context.Context, task TaskPlan, result *TaskResult, chatCtx *chatModel.ChatContext) {
	if result == nil {
		return
	}
	sink := chat.ResolveSink(chatCtx, c.slackService, c.repository)
	if sink == nil {
		return
	}
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
	if err := sink.PostSectionBlock(ctx, blockText); err != nil {
		logging.From(ctx).Error("failed to post task completion block", "error", err)
	}
}

// postDivider posts a divider through the active transport sink.
func (c *BluebellChat) postDivider(ctx context.Context, chatCtx *chatModel.ChatContext) {
	sink := chat.ResolveSink(chatCtx, c.slackService, c.repository)
	if sink == nil {
		return
	}
	if err := sink.PostDivider(ctx); err != nil {
		logging.From(ctx).Error("failed to post divider", "error", err)
	}
}

// executeTask executes a single task with its own agent and trace context.
func (c *BluebellChat) executeTask(ctx context.Context, task TaskPlan, chatCtx *chatModel.ChatContext, ssn *session.Session, taskCtx context.Context, markCompleted func(), progress chat.ProgressHandle) *TaskResult {
	target := chatCtx.Ticket
	logger := logging.From(ctx)
	result := &TaskResult{
		TaskID: task.ID,
		Title:  task.Title,
	}

	msg.Trace(taskCtx, "Starting...")

	// Resolve the LLM selected by the planner before doing any other work.
	// On failure (unknown id, empty id, not in [agent].task), the task fails
	// fast and the error is surfaced to replan — no fallback to main, since
	// silently swapping LLMs would mask planner bugs and skew cost reports.
	llmEntry, err := c.llmRegistry.Resolve(task.LLMID)
	if err != nil {
		result.Error = goerr.Wrap(err, "failed to resolve LLM for task",
			goerr.V("task_id", task.ID), goerr.V("llm_id", task.LLMID))
		msg.Trace(taskCtx, "❌ LLM resolve failed: %s", err.Error())
		return result
	}
	msg.Trace(taskCtx, "Using LLM: %s (%s/%s)", llmEntry.ID, llmEntry.Provider, llmEntry.Model)

	// Generate task system prompt
	taskPrompt, err := generateTaskPrompt(ctx, task, c.knowledgeService)
	if err != nil {
		result.Error = err
		msg.Trace(taskCtx, "❌ Failed to generate prompt: %s", err.Error())
		return result
	}

	// Filter tools for this task by ToolSet ID
	filteredTools := filterToolSets(c.tools, task.Tools)

	// Add base action tool only if its ToolSet ID is in the task's tool list
	// Note: SlackUpdate (PostFinding) is intentionally omitted in aster mode.
	// Individual task results are posted as context blocks instead.
	baseAction := base.New(c.repository, target.ID, base.WithLLMClient(c.llmClient))
	if filtered := filterToolSets([]interfaces.ToolSet{baseAction}, task.Tools); len(filtered) > 0 {
		filteredTools = append(filteredTools, baseAction)
	}

	// Always include knowledge tool (search-only) for child agents so they can
	// leverage prior knowledge without requiring the root agent to plan it explicitly.
	// knowledgeService is guaranteed non-nil (required by New()).
	kt := knowledgeTool.New(c.knowledgeService, types.KnowledgeCategoryFact, knowledgeTool.ModeSearchOnly)
	filteredTools = append(filteredTools, kt)

	// Collect additional prompts from filtered ToolSets
	var toolPrompts []string
	for _, ts := range filteredTools {
		p, err := ts.Prompt(ctx)
		if err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "failed to get prompt from tool set", goerr.V("id", ts.ID())))
			continue
		}
		if p != "" {
			toolPrompts = append(toolPrompts, p)
		}
	}
	if len(toolPrompts) > 0 {
		taskPrompt += "\n\n# Additional Tool Instructions\n\n" + strings.Join(toolPrompts, "\n\n")
	}

	// Convert filtered interfaces.ToolSet to []gollem.ToolSet for gollem
	gollemToolSets := make([]gollem.ToolSet, len(filteredTools))
	for i, ts := range filteredTools {
		gollemToolSets[i] = ts
	}

	// Setup budget tracker.
	var tracker *BudgetTracker
	if c.budgetStrategy != nil {
		tracker = newBudgetTracker(c.budgetStrategy)
		taskCtx = withBudgetTracker(taskCtx, tracker)
	}

	// Build agent options
	agentOpts := []gollem.Option{
		gollem.WithToolSets(gollemToolSets...),
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

	// Add HITL middleware if configured.
	// The middleware is always added when hitlTools is set, regardless of whether
	// a Slack presenter is available. If no presenter is available, the middleware
	// blocks tool execution to prevent bypassing the approval policy.
	if len(c.hitlTools) > 0 {
		approvalSet := make(map[string]bool, len(c.hitlTools))
		for _, t := range c.hitlTools {
			approvalSet[t] = true
		}
		hitlSvc := hitlService.New(c.repository)
		// Presenter built from ProgressHandle: Slack/Web/CLI render
		// HITL UI on the same display row as task progress (CLI
		// default-denies via PresentHITL error).
		presenter := chat.NewProgressHandlePresenter(progress, task.Title, user.FromContext(ctx))

		var slackThread *slackModel.Thread
		if target.SlackThread != nil {
			slackThread = target.SlackThread
		}

		var onResolved func(*hitl.Request)
		if chatCtx != nil && chatCtx.OnHITLEvent != nil {
			msgID := chat.ProgressMessageID(progress)
			onResolved = func(r *hitl.Request) {
				chatCtx.OnHITLEvent("resolved", r, msgID)
			}
		}
		agentOpts = append(agentOpts, gollem.WithToolMiddleware(newHITLMiddleware(hitlConfig{
			requireApproval: approvalSet,
			service:         hitlSvc,
			presenter:       presenter,
			userID:          user.FromContext(ctx),
			sessionID:       ssn.ID,
			slackThread:     slackThread,
			onResolved:      onResolved,
		})))
	}

	agentOpts = append(agentOpts, gollem.WithToolMiddleware(newTaskToolMiddleware(taskCtx, traceState)))

	// Create and execute agent using the LLM resolved from the planner's selection.
	gollemAgent := gollem.New(llmEntry.Client, agentOpts...)

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

	// Trigger technique knowledge reflection in background
	c.triggerTechniqueReflection(ctx, taskCtx, result)

	return result
}

// triggerTechniqueReflection runs background knowledge reflection for a completed task.
func (c *BluebellChat) triggerTechniqueReflection(ctx context.Context, taskCtx context.Context, result *TaskResult) {
	logger := logging.From(ctx)

	if c.knowledgeService == nil {
		logger.Debug("technique reflection skipped: knowledge service not configured")
		return
	}
	if result == nil {
		logger.Debug("technique reflection skipped: nil task result")
		return
	}
	if result.Result == "" {
		logger.Debug("technique reflection skipped: empty task result",
			"task_id", result.TaskID,
			"task_title", result.Title,
		)
		return
	}

	logger.Info("triggering technique reflection",
		"task_id", result.TaskID,
		"task_title", result.Title,
		"result_length", len(result.Result),
	)

	tool := knowledgeTool.New(c.knowledgeService, types.KnowledgeCategoryTechnique, knowledgeTool.ModeReadWrite)
	input := &svcknowledge.ReflectionInput{
		Category:         types.KnowledgeCategoryTechnique,
		ExecutionSummary: result.Result,
		OnComplete: func(bgCtx context.Context, traceID string) {
			suffix := "reflection done"
			if traceID != "" {
				suffix = fmt.Sprintf("reflection ID `%s`", traceID)
			}
			// Use bgCtx (non-cancelled) with taskCtx's msg routing
			msg.Trace(msg.CopyTo(bgCtx, taskCtx), "Completed (%s)", suffix)
		},
	}

	if err := c.knowledgeService.RunReflection(ctx, c.llmClient, tool, input); err != nil {
		logger.Error("failed to trigger technique reflection", "error", err)
	}
}

// triggerFactReflection runs background knowledge reflection for a completed session.
// The OnComplete hook posts the reflection ID through the active chat
// sink — for Slack this becomes a context block in the thread, for Web
// it persists as a trace session.Message.
func (c *BluebellChat) triggerFactReflection(ctx context.Context, summary string, chatCtx *chatModel.ChatContext) {
	logger := logging.From(ctx)

	if c.knowledgeService == nil {
		logger.Debug("fact reflection skipped: knowledge service not configured")
		return
	}
	if summary == "" {
		logger.Debug("fact reflection skipped: empty session summary")
		return
	}

	logger.Info("triggering fact reflection",
		"summary_length", len(summary),
		"has_ticket", chatCtx != nil && chatCtx.Ticket != nil,
	)

	tool := knowledgeTool.New(c.knowledgeService, types.KnowledgeCategoryFact, knowledgeTool.ModeReadWrite)
	input := &svcknowledge.ReflectionInput{
		Category:         types.KnowledgeCategoryFact,
		ExecutionSummary: summary,
		OnComplete: func(bgCtx context.Context, traceID string) {
			sink := chat.ResolveSink(chatCtx, c.slackService, c.repository)
			if sink == nil {
				return
			}
			suffix := "reflection done"
			if traceID != "" {
				suffix = fmt.Sprintf("reflection ID `%s`", traceID)
			}
			if err := sink.PostContextBlock(bgCtx, fmt.Sprintf("📝 Fact knowledge %s", suffix)); err != nil {
				logging.From(bgCtx).Warn("failed to post fact reflection result", "error", err)
			}
		},
	}
	if chatCtx != nil && chatCtx.Ticket != nil {
		input.Ticket = chatCtx.Ticket
		input.TicketID = chatCtx.Ticket.ID
	}

	if err := c.knowledgeService.RunReflection(ctx, c.llmClient, tool, input); err != nil {
		logger.Error("failed to trigger fact reflection", "error", err)
	}
}

// setupTaskMessageRouting creates task-specific msg routing with title-prefixed trace.
// The initial "Waiting..." message is posted immediately through the active transport sink.
// Returns the new context, a function to mark the task as completed (changes emoji),
// and a ProgressHandle for HITL integration (nil if no sink is available).
func (c *BluebellChat) setupTaskMessageRouting(ctx context.Context, ssn *session.Session, chatCtx *chatModel.ChatContext, taskTitle string) (context.Context, func(), chat.ProgressHandle) {
	escaped := escapeSlackMrkdwn(taskTitle)
	initialMsg := fmt.Sprintf("🕐 *[%s]*\n\nWaiting...", escaped)

	progress := chat.NewProgressHandle(ctx, chatCtx, c.slackService, c.repository, initialMsg)
	if progress == nil {
		return ctx, func() {}, nil
	}

	completed := false
	taskTraceFunc := func(ctx context.Context, message string) {
		emoji := "⏳"
		if completed {
			emoji = "✅"
		}
		prefixed := fmt.Sprintf("%s *[%s]*\n\n> %s", emoji, escaped, escapeSlackMrkdwn(message))
		// Persist the trace line against the Warren ssn regardless of
		// transport so the Slack Session timeline retains per-step
		// trace rows (Web persistence is folded into progress.UpdateText).
		m := session.NewMessageV2(ctx, ssn.ID, nil, nil, session.MessageTypeTrace, prefixed, nil)
		if err := c.repository.PutSessionMessage(ctx, m); err != nil {
			errutil.Handle(ctx, err)
		}
		progress.UpdateText(ctx, prefixed)
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

	return msg.With(ctx, notifyFunc, taskTraceFunc, warnFunc), markCompleted, progress
}

// filterToolSets filters tool sets to only include those matching the given IDs.
func filterToolSets(allTools []interfaces.ToolSet, allowedIDs []string) []interfaces.ToolSet {
	if len(allowedIDs) == 0 {
		return nil
	}

	idSet := make(map[string]bool, len(allowedIDs))
	for _, id := range allowedIDs {
		idSet[id] = true
	}

	var filtered []interfaces.ToolSet
	for _, ts := range allTools {
		if idSet[ts.ID()] {
			filtered = append(filtered, ts)
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
