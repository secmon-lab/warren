package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/model/eval"
)

// Agent generates dynamic tool responses with accumulated context.
// It maintains a single LLM session to ensure consistency across
// multiple tool calls within a scenario run.
type Agent struct {
	llmClient gollem.LLMClient
	world     eval.WorldConfig
	session   gollem.Session
	mu        sync.Mutex
}

// New creates a new mock Agent.
func New(llmClient gollem.LLMClient, world eval.WorldConfig) *Agent {
	return &Agent{
		llmClient: llmClient,
		world:     world,
	}
}

// Generate produces a response for the given tool call.
// It accumulates context from all previous calls via the session history.
// Returns the generated response and the number of tokens used.
func (a *Agent) Generate(ctx context.Context, toolName string, args map[string]any) (map[string]any, int64, error) {
	// Strip gollem trace handler to prevent mock agent LLM calls
	// from leaking into the main agent's trace
	ctx = trace.WithHandler(ctx, nil)

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.session == nil {
		session, err := a.llmClient.NewSession(ctx,
			gollem.WithSessionSystemPrompt(a.buildSystemPrompt()),
			gollem.WithSessionContentType(gollem.ContentTypeJSON),
		)
		if err != nil {
			return nil, 0, goerr.Wrap(err, "failed to create mock agent session")
		}
		a.session = session
	}

	argsJSON, err := json.MarshalIndent(args, "", "  ")
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to marshal tool args")
	}

	prompt := fmt.Sprintf(`Tool call: %s
Arguments:
%s

Generate a realistic JSON response that this tool would return.
The response MUST be consistent with all previous tool call responses in this session.
Return ONLY a valid JSON object, no markdown or explanation.`, toolName, string(argsJSON))

	resp, err := a.session.Generate(ctx, []gollem.Input{gollem.Text(prompt)})
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to generate mock response",
			goerr.V("tool", toolName))
	}

	tokens := int64(resp.InputToken + resp.OutputToken)

	text := strings.Join(resp.Texts, "")
	text = strings.TrimSpace(text)
	// Strip markdown code fence if present
	text = stripCodeFence(text)

	result, parseErr := parseJSONResponse(text)
	if parseErr != nil {
		return nil, tokens, goerr.Wrap(parseErr, "failed to parse mock LLM response as JSON",
			goerr.V("tool", toolName),
			goerr.V("raw_response", text))
	}

	return result, tokens, nil
}

func (a *Agent) buildSystemPrompt() string {
	var sb strings.Builder
	sb.WriteString("You are a tool response simulator for a security alert evaluation scenario.\n\n")
	sb.WriteString("## Scenario World\n")
	sb.WriteString(a.world.Description)
	sb.WriteString("\n")

	if len(a.world.ToolHints) > 0 {
		sb.WriteString("\n## Tool-Specific Hints\n")
		for tool, hint := range a.world.ToolHints {
			fmt.Fprintf(&sb, "\n### %s\n%s\n", tool, hint)
		}
	}

	sb.WriteString("\n## Rules\n")
	sb.WriteString("- Generate realistic responses consistent with the scenario world above\n")
	sb.WriteString("- Maintain consistency with ALL prior responses in this conversation\n")
	sb.WriteString("- Return ONLY valid JSON, no markdown or explanation\n")
	sb.WriteString("- JSON objects or arrays are both acceptable\n")

	return sb.String()
}

// parseJSONResponse parses a JSON string into map[string]any.
// If the JSON is an array, it wraps it as {"result": [...]}.
func parseJSONResponse(text string) (map[string]any, error) {
	// Try as object first
	var obj map[string]any
	if err := json.Unmarshal([]byte(text), &obj); err == nil {
		return obj, nil
	}

	// Try as array — wrap in {"result": [...]}
	var arr []any
	if err := json.Unmarshal([]byte(text), &arr); err == nil {
		return map[string]any{"result": arr}, nil
	}

	// Neither object nor array
	return nil, fmt.Errorf("response is neither JSON object nor array")
}

// stripCodeFence removes markdown code fences (```json ... ```) from the text.
// Handles both multi-line and single-line fences.
func stripCodeFence(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "```") {
		return text
	}

	// Single-line: ```json {"key": "value"} ```
	if !strings.Contains(text, "\n") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text, _ = strings.CutPrefix(text, "json")
		return strings.TrimSpace(text)
	}

	// Multi-line
	lines := strings.Split(text, "\n")
	if len(lines) >= 2 {
		start := 1
		end := len(lines)
		if strings.TrimSpace(lines[end-1]) == "```" {
			end--
		}
		text = strings.Join(lines[start:end], "\n")
	}
	return strings.TrimSpace(text)
}
