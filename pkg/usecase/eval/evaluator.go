package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
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

	// LLM judge: criteria evaluation
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

	// Collect called tool names
	calledTools := make(map[string]bool)
	var callSequence []string
	for _, tc := range trace.ToolCalls {
		calledTools[tc.ToolName] = true
		callSequence = append(callSequence, tc.ToolName)
	}

	// Check must_call
	for _, tool := range exp.MustCall {
		result.MustCallResults[tool] = calledTools[tool]
	}

	// Check must_not_call (true = violation)
	for _, tool := range exp.MustNotCall {
		result.MustNotCallResults[tool] = calledTools[tool]
	}

	// Check ordered_calls (subsequence match)
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

// isSubsequence checks if 'sub' is a subsequence of 'seq'.
// Elements don't need to be contiguous, but must appear in order.
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

	// Count duplicate calls (same tool + same args)
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

	// Check thresholds
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
