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

// GenerateReport creates a Report from an EvalResult.
// For JSON format, it serializes the EvalResult directly.
// For markdown format, it uses the LLM to generate a human-readable summary.
func GenerateReport(ctx context.Context, evalResult *eval.EvalResult, format eval.ReportFormat, llmClient gollem.LLMClient) (*eval.Report, error) {
	report := &eval.Report{
		Format:     format,
		EvalResult: evalResult,
	}

	switch format {
	case eval.ReportFormatJSON:
		data, err := json.MarshalIndent(evalResult, "", "  ")
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal eval result to JSON")
		}
		report.Content = string(data)

	case eval.ReportFormatMarkdown:
		content, err := generateMarkdownReport(ctx, evalResult, llmClient)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to generate markdown report")
		}
		report.Content = content

	default:
		return nil, goerr.New("unknown report format", goerr.V("format", format))
	}

	return report, nil
}

func generateMarkdownReport(ctx context.Context, result *eval.EvalResult, llmClient gollem.LLMClient) (string, error) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Evaluation Report: %s\n\n", result.Trace.ScenarioName)
	fmt.Fprintf(&sb, "- **Run ID**: %s\n", result.Trace.RunID)
	fmt.Fprintf(&sb, "- **Duration**: %s\n", result.Trace.EndTime.Sub(result.Trace.StartTime))
	fmt.Fprintf(&sb, "- **Total Tool Calls**: %d\n", len(result.Trace.ToolCalls))
	fmt.Fprintf(&sb, "- **Total Tokens**: %d\n\n", result.Trace.TotalTokens)

	// Layer A: Outcome
	if result.Outcome != nil {
		sb.WriteString("## Layer A: Outcome\n\n")
		if len(result.Outcome.FindingKeywordsFound) > 0 || len(result.Outcome.FindingKeywordsMissed) > 0 {
			sb.WriteString("### Keywords\n")
			for _, kw := range result.Outcome.FindingKeywordsFound {
				fmt.Fprintf(&sb, "- [x] %s\n", kw)
			}
			for _, kw := range result.Outcome.FindingKeywordsMissed {
				fmt.Fprintf(&sb, "- [ ] %s\n", kw)
			}
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "**Severity Match**: %v\n\n", result.Outcome.SeverityMatch)

		if len(result.Outcome.CriteriaResults) > 0 {
			sb.WriteString("### LLM Judge Criteria\n")
			for _, cr := range result.Outcome.CriteriaResults {
				status := "FAIL"
				if cr.Pass {
					status = "PASS"
				}
				fmt.Fprintf(&sb, "- **%s**: %s\n  - %s\n", status, cr.Criterion, cr.Reasoning)
			}
			sb.WriteString("\n")
		}
	}

	// Layer B: Trajectory
	if result.Trajectory != nil {
		sb.WriteString("## Layer B: Trajectory\n\n")
		sb.WriteString("### Must Call\n")
		for tool, called := range result.Trajectory.MustCallResults {
			mark := "[ ]"
			if called {
				mark = "[x]"
			}
			fmt.Fprintf(&sb, "- %s %s\n", mark, tool)
		}
		sb.WriteString("\n### Must Not Call\n")
		for tool, violated := range result.Trajectory.MustNotCallResults {
			status := "OK"
			if violated {
				status = "VIOLATED"
			}
			fmt.Fprintf(&sb, "- %s: %s\n", tool, status)
		}
		fmt.Fprintf(&sb, "\n**Ordered Calls**: %v — %s\n\n", result.Trajectory.OrderedCallsPass, result.Trajectory.OrderedCallsDetail)
	}

	// Layer C: Efficiency
	if result.Efficiency != nil {
		sb.WriteString("## Layer C: Efficiency\n\n")
		fmt.Fprintf(&sb, "- Total Calls: %d (max pass: %v)\n", result.Efficiency.TotalCalls, result.Efficiency.MaxCallsPass)
		fmt.Fprintf(&sb, "- Duplicate Calls: %d (max pass: %v)\n", result.Efficiency.DuplicateCalls, result.Efficiency.MaxDuplicatePass)
		fmt.Fprintf(&sb, "- Duration: %s\n", result.Efficiency.Duration)
		fmt.Fprintf(&sb, "- Total Tokens: %d\n\n", result.Efficiency.TotalTokens)
	}

	// Tool Call Trace
	sb.WriteString("## Tool Call Trace\n\n")
	sb.WriteString("| # | Tool | Source | Duration |\n")
	sb.WriteString("|---|------|--------|----------|\n")
	for _, tc := range result.Trace.ToolCalls {
		fmt.Fprintf(&sb, "| %d | %s | %s | %s |\n", tc.Sequence, tc.ToolName, tc.Source, tc.Duration)
	}
	sb.WriteString("\n")

	deterministicSection := sb.String()

	// Use LLM to generate a summary if available
	if llmClient != nil {
		summary, err := generateLLMSummary(ctx, result, llmClient)
		if err == nil && summary != "" {
			return deterministicSection + "## Summary\n\n" + summary + "\n", nil
		}
	}

	return deterministicSection, nil
}

func generateLLMSummary(ctx context.Context, result *eval.EvalResult, llmClient gollem.LLMClient) (string, error) {
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal eval result for LLM")
	}

	session, err := llmClient.NewSession(ctx,
		gollem.WithSessionSystemPrompt("You are an evaluation report writer. Summarize the evaluation results concisely in markdown. Focus on what passed, what failed, and actionable insights. Be brief."),
	)
	if err != nil {
		return "", goerr.Wrap(err, "failed to create LLM session for report")
	}

	prompt := fmt.Sprintf("Summarize this evaluation result:\n\n```json\n%s\n```", string(resultJSON))
	resp, err := session.Generate(ctx, []gollem.Input{gollem.Text(prompt)})
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate LLM summary")
	}

	return strings.TrimSpace(strings.Join(resp.Texts, "")), nil
}
