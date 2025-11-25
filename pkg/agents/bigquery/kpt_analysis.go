package bigquery

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// Error tags for KPT analysis fallback scenarios
var (
	tagKPTAnalysisFallback = goerr.NewTag("kpt_analysis_fallback") // Tag for non-critical KPT analysis failures
)

//go:embed prompt/kpt_analysis.md
var kptAnalysisPromptTemplate string

var kptAnalysisTmpl *template.Template

func init() {
	kptAnalysisTmpl = template.Must(template.New("kpt_analysis").Parse(kptAnalysisPromptTemplate))
}

// kptAnalysisResponse represents the structured response from LLM for KPT analysis
type kptAnalysisResponse struct {
	Successes    []string `json:"successes"`
	Problems     []string `json:"problems"`
	Improvements []string `json:"improvements"`
}

// kptPromptData represents the data for KPT analysis prompt template
type kptPromptData struct {
	Query            string
	Duration         string
	Status           string
	ExecutionSummary string
	History          string
	Error            string
}

// generateKPTAnalysis generates comprehensive KPT (Keep/Problem/Try) analysis from execution result
// Returns: (Successes, Problems, Improvements, error)
// Note: Returns non-critical errors with tagKPTAnalysisFallback for graceful degradation
func (a *Agent) generateKPTAnalysis(
	ctx context.Context,
	query string,
	resp *gollem.ExecuteResponse,
	execErr error,
	duration time.Duration,
	session gollem.Session,
) ([]string, []string, []string, error) {
	logger := logging.From(ctx)
	logger.Debug("Generating KPT analysis", "query", query, "duration", duration, "has_error", execErr != nil)

	// Build prompt for LLM
	prompt := a.buildKPTPrompt(query, resp, execErr, duration, session)
	logger.Debug("KPT prompt built", "prompt", prompt)

	// Generate analysis using LLM (create new session for KPT analysis)
	kptSession, err := a.llmClient.NewSession(ctx)
	if err != nil {
		return []string{}, []string{}, []string{}, goerr.Wrap(err, "failed to create LLM session for KPT analysis", goerr.Tag(tagKPTAnalysisFallback))
	}

	llmResp, err := kptSession.GenerateContent(ctx, gollem.Text(prompt))
	if err != nil {
		return []string{}, []string{}, []string{}, goerr.Wrap(err, "failed to generate KPT analysis", goerr.Tag(tagKPTAnalysisFallback))
	}

	if llmResp == nil || len(llmResp.Texts) == 0 {
		return []string{}, []string{}, []string{}, goerr.New("empty response from LLM for KPT analysis", goerr.Tag(tagKPTAnalysisFallback))
	}

	// Parse LLM response
	successes, problems, improvements, err := a.parseKPTResponse(llmResp.Texts[0])
	if err != nil {
		return []string{}, []string{}, []string{}, goerr.Wrap(err, "failed to parse KPT analysis response", goerr.Tag(tagKPTAnalysisFallback))
	}

	logger.Debug("KPT analysis generated successfully",
		"successes_count", len(successes),
		"problems_count", len(problems),
		"improvements_count", len(improvements))

	return successes, problems, improvements, nil
}

// buildKPTPrompt builds the prompt for KPT analysis using template
func (a *Agent) buildKPTPrompt(
	query string,
	resp *gollem.ExecuteResponse,
	execErr error,
	duration time.Duration,
	session gollem.Session,
) string {
	// Prepare execution summary
	var executionSummary string
	if resp != nil && !resp.IsEmpty() {
		respText := resp.String()
		// Limit summary to avoid overwhelming the prompt
		if len(respText) > 1000 {
			respText = respText[:1000] + "... (truncated)"
		}
		executionSummary = "Response summary: " + respText
	} else {
		executionSummary = "No response data"
	}

	// Extract conversation history from session
	var historyText string
	if session != nil {
		history, err := session.History()
		if err == nil && history != nil {
			historyText = a.formatHistory(history)
		}
	}

	// Determine status
	status := "success"
	if execErr != nil {
		status = "failure"
	}

	// Prepare error message
	var errorMsg string
	if execErr != nil {
		errorMsg = execErr.Error()
	}

	// Prepare template data
	data := kptPromptData{
		Query:            query,
		Duration:         duration.String(),
		Status:           status,
		ExecutionSummary: executionSummary,
		History:          historyText,
		Error:            errorMsg,
	}

	// Execute template
	var buf bytes.Buffer
	if err := kptAnalysisTmpl.Execute(&buf, data); err != nil {
		// Fallback to empty prompt if template execution fails
		logging.From(context.Background()).Warn("Failed to execute KPT analysis template", "error", err)
		return ""
	}

	return buf.String()
}

// formatHistory converts conversation history into human-readable text
func (a *Agent) formatHistory(history *gollem.History) string {
	if history == nil || len(history.Messages) == 0 {
		return "No conversation history"
	}

	var buf strings.Builder
	buf.WriteString("Conversation History:\n")

	for i, msg := range history.Messages {
		buf.WriteString(strings.Repeat("=", 80))
		buf.WriteString("\n")
		buf.WriteString("Message ")
		buf.WriteString(fmt.Sprintf("%d", i+1))
		buf.WriteString(" (")
		buf.WriteString(string(msg.Role))
		buf.WriteString("):\n")

		for _, content := range msg.Contents {
			switch content.Type {
			case "text":
				textContent, err := content.GetTextContent()
				if err == nil && textContent != nil {
					buf.WriteString(textContent.Text)
					buf.WriteString("\n")
				}
			case "tool_call":
				toolCall, err := content.GetToolCallContent()
				if err == nil && toolCall != nil {
					buf.WriteString("[Tool Call: ")
					buf.WriteString(toolCall.Name)
					buf.WriteString("]\n")
					if len(toolCall.Arguments) > 0 {
						argsJSON, err := json.MarshalIndent(toolCall.Arguments, "", "  ")
						if err == nil {
							buf.Write(argsJSON)
							buf.WriteString("\n")
						}
					}
				}
			case "tool_response":
				toolResp, err := content.GetToolResponseContent()
				if err == nil && toolResp != nil {
					buf.WriteString("[Tool Response: ")
					buf.WriteString(toolResp.Name)
					if toolResp.IsError {
						buf.WriteString(" (ERROR)")
					}
					buf.WriteString("]\n")
					if len(toolResp.Response) > 0 {
						respJSON, err := json.MarshalIndent(toolResp.Response, "", "  ")
						if err == nil {
							buf.Write(respJSON)
							buf.WriteString("\n")
						}
					}
				}
			}
		}
	}

	buf.WriteString(strings.Repeat("=", 80))
	buf.WriteString("\n")

	return buf.String()
}

// parseKPTResponse parses the LLM response into KPT components
func (a *Agent) parseKPTResponse(responseText string) ([]string, []string, []string, error) {
	// Clean up response text (remove markdown code blocks if present)
	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	// Parse JSON
	var analysis kptAnalysisResponse
	if err := json.Unmarshal([]byte(responseText), &analysis); err != nil {
		return nil, nil, nil, goerr.Wrap(err, "failed to parse KPT analysis JSON")
	}

	// Ensure arrays are initialized (not nil)
	if analysis.Successes == nil {
		analysis.Successes = []string{}
	}
	if analysis.Problems == nil {
		analysis.Problems = []string{}
	}
	if analysis.Improvements == nil {
		analysis.Improvements = []string{}
	}

	return analysis.Successes, analysis.Problems, analysis.Improvements, nil
}
