package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	gollemTrace "github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/model/eval"
)

// Evaluate runs the 3-layer evaluation against a trace and expectations.
// Deterministic verifiers run first; LLM judge is used only for Layer A criteria.
func Evaluate(ctx context.Context, trace *eval.Trace, expectations *eval.Expectations, llmClient gollem.LLMClient) (*eval.EvalResult, error) {
	result := &eval.EvalResult{
		Trace: trace,
	}

	if expectations == nil {
		return result, nil
	}

	if expectations.Outcome != nil {
		outcome, err := evaluateOutcome(ctx, trace, expectations.Outcome, llmClient)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to evaluate outcome")
		}
		result.Outcome = outcome
	}

	if expectations.Trajectory != nil {
		result.Trajectory = evaluateTrajectory(trace, expectations.Trajectory)
	}

	if expectations.Efficiency != nil {
		result.Efficiency = evaluateEfficiency(trace, expectations.Efficiency)
	}

	return result, nil
}

// EvaluateWithAgent runs a full evaluation using an LLM agent that reads the
// gollem trace and agent output to produce a comprehensive assessment.
// The agent is given tools to navigate the trace structure without loading
// the entire trace into context at once.
func EvaluateWithAgent(ctx context.Context, trace *eval.Trace, gollemTraceData *gollemTrace.Trace, expectations *eval.Expectations, agentOutput string, llmClient gollem.LLMClient) (*eval.EvalResult, error) {
	// First run deterministic evaluations
	result, err := Evaluate(ctx, trace, expectations, nil) // nil LLM = skip LLM judge in deterministic pass
	if err != nil {
		return nil, err
	}

	if llmClient == nil || expectations == nil {
		return result, nil
	}

	// Build trace reader tool for the agent
	traceToolSet := newTraceReaderToolSet(gollemTraceData, agentOutput)

	// Build evaluation prompt
	criteriaJSON, _ := json.MarshalIndent(expectations, "", "  ")
	systemPrompt := fmt.Sprintf(`You are a security agent evaluator. Your job is to assess whether a security investigation agent performed well.

You have access to tools that let you navigate the agent's execution trace. Use them to understand what the agent did, then evaluate against the criteria.

## Evaluation Criteria
%s

## Agent's Final Output
%s

## Instructions
1. Use get_trace_overview to understand the overall execution structure
2. Use get_span_detail to examine specific spans (planning, tasks, final response)
3. Evaluate each criterion and provide pass/fail with reasoning
4. Respond with a JSON object:
{
  "criteria_results": [
    {"criterion": "...", "pass": true/false, "reasoning": "..."}
  ]
}`, string(criteriaJSON), agentOutput)

	agent := gollem.New(llmClient,
		gollem.WithToolSets(traceToolSet),
		gollem.WithSystemPrompt(systemPrompt),
		gollem.WithContentType(gollem.ContentTypeJSON),
	)

	resp, err := agent.Execute(ctx, gollem.Text("Evaluate the agent's performance against all criteria."))
	if err != nil {
		return nil, goerr.Wrap(err, "evaluation agent failed")
	}

	// Parse agent response
	responseText := strings.Join(resp.Texts, "")
	var agentResult struct {
		CriteriaResults []eval.CriterionResult `json:"criteria_results"`
	}
	if err := json.Unmarshal([]byte(responseText), &agentResult); err != nil {
		// Try to strip code fences
		cleaned := strings.TrimSpace(responseText)
		if strings.HasPrefix(cleaned, "```") {
			lines := strings.Split(cleaned, "\n")
			if len(lines) >= 3 {
				cleaned = strings.Join(lines[1:len(lines)-1], "\n")
			}
		}
		if err2 := json.Unmarshal([]byte(cleaned), &agentResult); err2 != nil {
			return result, nil // Return deterministic results only
		}
	}

	if result.Outcome == nil {
		result.Outcome = &eval.OutcomeResult{}
	}
	result.Outcome.CriteriaResults = agentResult.CriteriaResults

	return result, nil
}

// evaluateOutcome — Layer A: deterministic checks + LLM judge.
func evaluateOutcome(ctx context.Context, trace *eval.Trace, exp *eval.OutcomeExpectation, llmClient gollem.LLMClient) (*eval.OutcomeResult, error) {
	result := &eval.OutcomeResult{}

	// Deterministic: finding keyword checks
	agentOutput := strings.ToLower(trace.AgentOutput)
	for _, keyword := range exp.FindingMustContain {
		if strings.Contains(agentOutput, strings.ToLower(keyword)) {
			result.FindingKeywordsFound = append(result.FindingKeywordsFound, keyword)
		} else {
			result.FindingKeywordsMissed = append(result.FindingKeywordsMissed, keyword)
		}
	}

	// Deterministic: severity match
	if exp.Severity != "" {
		result.SeverityMatch = strings.Contains(agentOutput, strings.ToLower(exp.Severity))
	}

	// LLM judge: criteria evaluation (only if llmClient provided)
	if len(exp.Criteria) > 0 && llmClient != nil {
		criteriaResults, err := evaluateCriteriaWithLLM(ctx, trace.AgentOutput, exp.Criteria, llmClient)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to evaluate criteria with LLM judge")
		}
		result.CriteriaResults = criteriaResults
	}

	return result, nil
}

// evaluateCriteriaWithLLM uses an LLM to judge each criterion against the agent output.
func evaluateCriteriaWithLLM(ctx context.Context, agentOutput string, criteria []string, llmClient gollem.LLMClient) ([]eval.CriterionResult, error) {
	session, err := llmClient.NewSession(ctx,
		gollem.WithSessionSystemPrompt("You are an evaluation judge for a security analysis agent. Evaluate whether the agent's output meets the given criterion. Respond in JSON."),
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create LLM judge session")
	}

	results := make([]eval.CriterionResult, 0, len(criteria))
	for _, criterion := range criteria {
		prompt := fmt.Sprintf(`Evaluate the following agent output against this criterion.

Criterion: %s

Agent Output:
%s

Respond with a JSON object: {"pass": true/false, "reasoning": "brief explanation"}`, criterion, agentOutput)

		resp, err := session.Generate(ctx, []gollem.Input{gollem.Text(prompt)})
		if err != nil {
			results = append(results, eval.CriterionResult{
				Criterion: criterion,
				Pass:      false,
				Reasoning: fmt.Sprintf("LLM judge error: %v", err),
			})
			continue
		}

		text := strings.TrimSpace(strings.Join(resp.Texts, ""))
		var judgeResult struct {
			Pass      bool   `json:"pass"`
			Reasoning string `json:"reasoning"`
		}
		if err := json.Unmarshal([]byte(text), &judgeResult); err != nil {
			results = append(results, eval.CriterionResult{
				Criterion: criterion,
				Pass:      false,
				Reasoning: fmt.Sprintf("Failed to parse LLM judge response: %s", text),
			})
			continue
		}

		results = append(results, eval.CriterionResult{
			Criterion: criterion,
			Pass:      judgeResult.Pass,
			Reasoning: judgeResult.Reasoning,
		})
	}

	return results, nil
}

// evaluateTrajectory — Layer B: tool call sequence evaluation.
func evaluateTrajectory(trace *eval.Trace, exp *eval.TrajectoryExpectation) *eval.TrajectoryResult {
	result := &eval.TrajectoryResult{
		MustCallResults:    make(map[string]bool),
		MustNotCallResults: make(map[string]bool),
	}

	calledTools := make(map[string]bool)
	var callSequence []string
	for _, tc := range trace.ToolCalls {
		calledTools[tc.ToolName] = true
		callSequence = append(callSequence, tc.ToolName)
	}

	for _, tool := range exp.MustCall {
		result.MustCallResults[tool] = calledTools[tool]
	}

	for _, tool := range exp.MustNotCall {
		result.MustNotCallResults[tool] = calledTools[tool]
	}

	if len(exp.OrderedCalls) > 0 {
		result.OrderedCallsPass = isSubsequence(exp.OrderedCalls, callSequence)
		if result.OrderedCallsPass {
			result.OrderedCallsDetail = "ordered calls found in expected order"
		} else {
			result.OrderedCallsDetail = fmt.Sprintf("expected order %v not found in call sequence %v", exp.OrderedCalls, callSequence)
		}
	} else {
		result.OrderedCallsPass = true
		result.OrderedCallsDetail = "no ordered calls specified"
	}

	return result
}

func isSubsequence(sub, seq []string) bool {
	si := 0
	for _, s := range seq {
		if si < len(sub) && s == sub[si] {
			si++
		}
	}
	return si == len(sub)
}

// evaluateEfficiency — Layer C: quantitative metrics.
func evaluateEfficiency(trace *eval.Trace, exp *eval.EfficiencyExpectation) *eval.EfficiencyResult {
	result := &eval.EfficiencyResult{
		TotalCalls:  len(trace.ToolCalls),
		TotalTokens: trace.TotalTokens,
		Duration:    trace.EndTime.Sub(trace.StartTime),
	}

	type callKey struct {
		tool string
		args string
	}
	seen := make(map[callKey]int)
	for _, tc := range trace.ToolCalls {
		argsJSON, _ := json.Marshal(tc.Args)
		key := callKey{tool: tc.ToolName, args: string(argsJSON)}
		seen[key]++
	}
	for _, count := range seen {
		if count > 1 {
			result.DuplicateCalls += count - 1
		}
	}

	if exp.MaxTotalCalls > 0 {
		result.MaxCallsPass = result.TotalCalls <= exp.MaxTotalCalls
	} else {
		result.MaxCallsPass = true
	}

	if exp.MaxDuplicateCalls > 0 {
		result.MaxDuplicatePass = result.DuplicateCalls <= exp.MaxDuplicateCalls
	} else {
		result.MaxDuplicatePass = true
	}

	return result
}

// traceReaderToolSet provides tools for the evaluation agent to navigate a gollem trace.
type traceReaderToolSet struct {
	trace       *gollemTrace.Trace
	agentOutput string
	spanIndex   map[string]*gollemTrace.Span // spanID -> span
}

func newTraceReaderToolSet(t *gollemTrace.Trace, agentOutput string) *traceReaderToolSet {
	ts := &traceReaderToolSet{
		trace:       t,
		agentOutput: agentOutput,
		spanIndex:   make(map[string]*gollemTrace.Span),
	}
	if t != nil && t.RootSpan != nil {
		ts.indexSpans(t.RootSpan)
	}
	return ts
}

func (ts *traceReaderToolSet) indexSpans(span *gollemTrace.Span) {
	ts.spanIndex[span.SpanID] = span
	for _, child := range span.Children {
		ts.indexSpans(child)
	}
}

func (ts *traceReaderToolSet) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "get_trace_overview",
			Description: "Get an overview of the trace execution: root span, direct children (phases/tasks), their kinds, names, statuses, and durations. Does NOT include full content.",
		},
		{
			Name:        "get_span_detail",
			Description: "Get the full detail of a specific span by span_id. Includes LLM call data (prompts, responses), tool execution data (args, results), and child span summaries.",
			Parameters: map[string]*gollem.Parameter{
				"span_id": {Type: gollem.TypeString, Description: "The span_id to inspect", Required: true},
			},
		},
		{
			Name:        "get_agent_output",
			Description: "Get the agent's final output text (the messages sent to the user via msg.Notify).",
		},
	}, nil
}

func (ts *traceReaderToolSet) Run(_ context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "get_trace_overview":
		return ts.getTraceOverview()
	case "get_span_detail":
		spanID, _ := args["span_id"].(string)
		return ts.getSpanDetail(spanID)
	case "get_agent_output":
		return map[string]any{"output": ts.agentOutput}, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (ts *traceReaderToolSet) getTraceOverview() (map[string]any, error) {
	if ts.trace == nil || ts.trace.RootSpan == nil {
		return map[string]any{"error": "no trace data"}, nil
	}

	root := ts.trace.RootSpan
	children := make([]map[string]any, 0, len(root.Children))
	for _, child := range root.Children {
		childInfo := map[string]any{
			"span_id":  child.SpanID,
			"kind":     string(child.Kind),
			"name":     child.Name,
			"status":   string(child.Status),
			"duration": child.Duration.String(),
		}
		if child.Error != "" {
			childInfo["error"] = child.Error
		}
		grandchildren := make([]string, 0, len(child.Children))
		for _, gc := range child.Children {
			grandchildren = append(grandchildren, fmt.Sprintf("%s(%s:%s)", gc.Name, gc.Kind, gc.SpanID))
		}
		if len(grandchildren) > 0 {
			childInfo["children"] = grandchildren
		}
		children = append(children, childInfo)
	}

	return map[string]any{
		"trace_id":  ts.trace.TraceID,
		"root_kind": string(root.Kind),
		"root_name": root.Name,
		"status":    string(root.Status),
		"duration":  root.Duration.String(),
		"children":  children,
	}, nil
}

func (ts *traceReaderToolSet) getSpanDetail(spanID string) (map[string]any, error) {
	span, ok := ts.spanIndex[spanID]
	if !ok {
		return map[string]any{"error": fmt.Sprintf("span %q not found", spanID)}, nil
	}

	result := map[string]any{
		"span_id":  span.SpanID,
		"kind":     string(span.Kind),
		"name":     span.Name,
		"status":   string(span.Status),
		"duration": span.Duration.String(),
	}

	if span.Error != "" {
		result["error"] = span.Error
	}

	if span.LLMCall != nil {
		llmData := map[string]any{
			"model":         span.LLMCall.Model,
			"input_tokens":  span.LLMCall.InputTokens,
			"output_tokens": span.LLMCall.OutputTokens,
		}
		if span.LLMCall.Request != nil {
			llmData["system_prompt_length"] = len(span.LLMCall.Request.SystemPrompt)
			llmData["message_count"] = len(span.LLMCall.Request.Messages)
			llmData["tool_count"] = len(span.LLMCall.Request.Tools)
		}
		if span.LLMCall.Response != nil {
			llmData["response_texts"] = span.LLMCall.Response.Texts
			if len(span.LLMCall.Response.FunctionCalls) > 0 {
				calls := make([]string, 0, len(span.LLMCall.Response.FunctionCalls))
				for _, fc := range span.LLMCall.Response.FunctionCalls {
					calls = append(calls, fc.Name)
				}
				llmData["function_calls"] = calls
			}
		}
		result["llm_call"] = llmData
	}

	if span.ToolExec != nil {
		result["tool_exec"] = map[string]any{
			"tool_name": span.ToolExec.ToolName,
			"args":      span.ToolExec.Args,
			"result":    span.ToolExec.Result,
			"error":     span.ToolExec.Error,
		}
	}

	// Child summaries (not full detail)
	if len(span.Children) > 0 {
		children := make([]map[string]any, 0, len(span.Children))
		for _, child := range span.Children {
			childInfo := map[string]any{
				"span_id": child.SpanID,
				"kind":    string(child.Kind),
				"name":    child.Name,
				"status":  string(child.Status),
			}
			children = append(children, childInfo)
		}
		result["children"] = children
	}

	return result, nil
}
