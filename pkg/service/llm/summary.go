package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

type SummaryOption func(*summaryConfig)

type summaryConfig struct {
	maxPartSize int
}

func WithMaxPartSize(maxPartSize int) SummaryOption {
	return func(o *summaryConfig) {
		o.maxPartSize = maxPartSize
	}
}

// Summary generates a summary of the large data by using the LLM.
func Summary[T any](ctx context.Context, llm gollem.LLMClient, prompt string, data []T, opts ...SummaryOption) (string, error) {
	var results []string
	var parts []string
	var partSize = 0

	cfg := &summaryConfig{
		maxPartSize: 1 * 1024 * 1024,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	startIdx := 0
	for idx, d := range data {
		rawData, err := json.Marshal(d)
		if err != nil {
			return "", err
		}
		parts = append(parts, string(rawData))

		partSize += len(rawData)
		if partSize > cfg.maxPartSize {
			msg.Trace(ctx, "✍️ reading data (%d-%d)", startIdx, idx)
			startIdx = idx + 1

			result, err := generatePartSummary(ctx, llm, prompt, parts)
			if err != nil {
				return "", err
			}
			results = append(results, result)
			parts = []string{}
			partSize = 0
		}
	}

	if len(parts) > 0 {
		msg.Trace(ctx, "✍️ reading data (%d-%d, last part)", startIdx, len(data))
		result, err := generatePartSummary(ctx, llm, prompt, parts)
		if err != nil {
			return "", err
		}
		results = append(results, result)
	}

	msg.Trace(ctx, "✍️ generating summary")
	return generateSummary(ctx, llm, prompt, results)
}

func generatePartSummary(ctx context.Context, llm gollem.LLMClient, prompt string, data []string) (string, error) {
	systemPrompt := `
You are a helpful assistant specialized in analyzing and summarizing data for a comprehensive report.
Your task is to analyze and summarize a large dataset.
Due to token limitations, the data is provided to you in multiple parts.

Please understand that the data you receive now is only a portion of the complete dataset. There are other parts of the data that you cannot see.
The summaries you create for each part will be combined later to generate a final, comprehensive summary or conclusion for the entire dataset.

With this in mind, please analyze the current data part and extract/summarize the information that is essential for understanding this specific portion and contributing to the final overall summary (e.g., key trends, significant facts, notable points, relevant figures).
To facilitate the final integration, please present your summary in a concise and structured format (e.g., bullet points or a short paragraph focusing on key findings).

CRITICAL OUTPUT CONSTRAINTS:
- Keep your summary within 8,000 tokens (approximately 6,000 words) to stay within Gemini 2.0's 8,192 token limit
- Focus on the most essential and actionable information
- Use bullet points or structured format for clarity and readability
- Avoid unnecessary repetition or verbose explanations
- Prioritize key insights and actionable data over detailed descriptions
- If dealing with schema/field data, provide concise but precise field descriptions
- For large datasets, group similar items together and summarize patterns rather than listing every individual item
- Include specific field names and data types when relevant for downstream usage

Specific instructions regarding the analysis policy and the type of information to extract will be provided separately in the "Instruction" section.
	`

	userPrompt := fmt.Sprintf(`# Instruction

	%s

# Data

%s`, prompt, strings.Join(data, "\n"))

	ssn, err := llm.NewSession(ctx, gollem.WithSessionSystemPrompt(systemPrompt))
	if err != nil {
		return "", err
	}

	resp, err := ssn.GenerateContent(ctx, gollem.Text(userPrompt))
	if err != nil {
		return "", err
	}

	return resp.Texts[0], nil
}

func generateSummary(ctx context.Context, llm gollem.LLMClient, prompt string, data []string) (string, error) {
	systemPrompt := `
	You are an expert skilled in integrating analysis results from multiple sources to create comprehensive reports.
Your task is to read multiple partial summaries, which were previously created by dividing a large dataset, and integrate them to create a final summary for the entire dataset.

You will now be provided with multiple summaries extracted from different parts of the dataset. Each of these summaries summarizes a different portion of the whole.

Carefully read these partial summaries and integrate them considering the following points:

1.  **Grasp the overall picture:** Synthesize the information from each summary to understand the main trends, patterns, and the overall situation suggested by the entire dataset.
2.  **Identify commonalities and differences:** Identify important points and trends mentioned in common across the summaries, as well as notable points or differences specific to certain parts.
3.  **Extract key insights:** Extract the most important findings or insights that emerge from the data as a whole.
4.  **Organize and structure information:** Eliminate redundant information and organize the information in a logical flow.
5.  **Present conclusions:** Clearly present the conclusions or main messages derived from the entire dataset based on the analysis results.

CRITICAL OUTPUT CONSTRAINTS:
- Keep your final summary within 8,000 tokens (approximately 6,000 words) to stay within Gemini 2.0's 8,192 token limit
- Focus on synthesizing the most important insights across all partial summaries
- Use structured format with clear sections and bullet points for readability
- Eliminate redundant information while preserving essential details
- Prioritize actionable insights and key findings that emerged from the complete dataset

More detailed instructions regarding the report structure or specific analytical perspectives and data trends to focus on will be provided separately in the "Instruction" section.
	`

	userPrompt := fmt.Sprintf(`# Instruction

	%s

# Partial Summaries

%s`, prompt, strings.Join(data, "------------\n"))

	ssn, err := llm.NewSession(ctx, gollem.WithSessionSystemPrompt(systemPrompt))
	if err != nil {
		return "", err
	}

	resp, err := ssn.GenerateContent(ctx, gollem.Text(userPrompt))
	if err != nil {
		return "", err
	}

	return resp.Texts[0], nil
}
