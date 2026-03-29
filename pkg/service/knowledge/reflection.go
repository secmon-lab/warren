package knowledge

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

//go:embed prompt/reflection_fact.md
var reflectionFactPrompt string

//go:embed prompt/reflection_technique.md
var reflectionTechniquePrompt string

// ReflectionInput holds the input for the knowledge reflection agent.
type ReflectionInput struct {
	Category types.KnowledgeCategory

	// ExecutionSummary is a text summary of the execution history.
	// This is provided instead of raw History to avoid gollem type coupling.
	ExecutionSummary string

	// Set only for ticket resolve reflection
	Ticket   *ticket.Ticket
	TicketID types.TicketID

	// OnComplete is called when the background reflection finishes.
	// ctx is a non-cancelled context safe for I/O operations.
	// traceID is the trace ID used for the reflection execution (empty if tracing is not configured).
	OnComplete func(ctx context.Context, traceID string)
}

// RunReflection launches a background reflection agent that extracts
// knowledge from the given execution summary and saves it to the knowledge store.
// The toolSet must be a knowledge tool in ModeReadWrite for the target category.
// Caller is responsible for creating the tool to avoid import cycles.
func (s *Service) RunReflection(ctx context.Context, llmClient gollem.LLMClient, toolSet gollem.ToolSet, input *ReflectionInput) error {
	logger := logging.From(ctx)

	if input == nil {
		logger.Debug("knowledge reflection skipped: nil input")
		return nil
	}
	if input.ExecutionSummary == "" {
		logger.Debug("knowledge reflection skipped: empty execution summary",
			"category", input.Category,
		)
		return nil
	}

	// Select prompt based on category
	var systemPrompt string
	switch input.Category {
	case types.KnowledgeCategoryFact:
		systemPrompt = reflectionFactPrompt
	case types.KnowledgeCategoryTechnique:
		systemPrompt = reflectionTechniquePrompt
	default:
		return goerr.New("unsupported category for reflection", goerr.V("category", input.Category))
	}

	// Append existing tags to prompt so agent reuses them
	existingTags, err := s.ListTags(ctx)
	if err != nil {
		logger.Warn("failed to list existing tags for reflection prompt", "error", err)
	}
	if len(existingTags) > 0 {
		systemPrompt += "\n\n## Existing Tags\n"
		systemPrompt += "The following tags already exist. **ALWAYS reuse existing tags instead of creating new ones.** Only create a new tag if none of the existing tags are appropriate.\n\n"
		for _, t := range existingTags {
			systemPrompt += "- `" + t.Name + "`"
			if t.Description != "" {
				systemPrompt += " — " + t.Description
			}
			systemPrompt += " (ID: " + t.ID.String() + ")\n"
		}
	}

	// Append language instruction from context
	l := lang.From(ctx)
	if l != "" {
		systemPrompt += "\n\n## Language\nWrite all knowledge claims in **" + l.Name() + "**. Use this language for titles, claims, and tag descriptions.\n"
	}

	// Append ticket context to prompt if available
	if input.Ticket != nil {
		systemPrompt += "\n\n## Ticket Context\n"
		systemPrompt += "This reflection is triggered by a ticket resolve.\n"
		systemPrompt += "Conclusion: " + string(input.Ticket.Conclusion) + "\n"
		systemPrompt += "Reason: " + input.Ticket.Reason + "\n"
		if input.Ticket.Finding != nil {
			systemPrompt += "Finding Severity: " + string(input.Ticket.Finding.Severity) + "\n"
			systemPrompt += "Finding Summary: " + input.Ticket.Finding.Summary + "\n"
			systemPrompt += "Finding Recommendation: " + input.Ticket.Finding.Recommendation + "\n"
		}
	}

	logger.Debug("knowledge reflection: building agent",
		"category", input.Category,
		"system_prompt_length", len(systemPrompt),
		"has_ticket_context", input.Ticket != nil,
	)

	// Build agent options with middleware for logging
	opts := []gollem.Option{
		gollem.WithToolSets(toolSet),
		gollem.WithSystemPrompt(systemPrompt),
		gollem.WithResponseMode(gollem.ResponseModeBlocking),
		gollem.WithToolMiddleware(newReflectionToolMiddleware()),
		gollem.WithContentBlockMiddleware(newReflectionContentBlockMiddleware()),
	}

	// Add trace recorder if trace repository is configured
	var recorder *trace.Recorder
	var traceID string
	if s.traceRepository != nil {
		traceID = uuid.Must(uuid.NewV7()).String()
		recorder = trace.New(
			trace.WithTraceID(traceID),
			trace.WithRepository(s.traceRepository),
			trace.WithStackTrace(),
			trace.WithMetadata(trace.TraceMetadata{
				Labels: map[string]string{
					"type":     "reflection",
					"category": string(input.Category),
				},
			}),
		)
		opts = append(opts, gollem.WithTrace(recorder))
		logger.Info("knowledge reflection trace enabled",
			"trace_id", traceID,
			"category", input.Category,
		)
	}

	// Create agent
	agent := gollem.New(llmClient, opts...)

	summaryLen := len(input.ExecutionSummary)
	hasTicket := input.Ticket != nil

	logger.Info("knowledge reflection started",
		"category", input.Category,
		"summary_length", summaryLen,
		"has_ticket", hasTicket,
	)

	// Execute in background
	go func() {
		bgCtx := context.WithoutCancel(ctx)
		logger.Debug("knowledge reflection: executing agent",
			"category", input.Category,
			"summary_length", summaryLen,
		)

		resp, err := agent.Execute(bgCtx, gollem.Text(input.ExecutionSummary))
		if err != nil {
			logger.Error("knowledge reflection failed",
				"category", input.Category,
				"summary_length", summaryLen,
				"error", err,
			)
		} else {
			var respText string
			if resp != nil && !resp.IsEmpty() {
				respText = resp.String()
			}
			logger.Info("knowledge reflection completed",
				"category", input.Category,
				"summary_length", summaryLen,
				"response_length", len(respText),
			)
			if respText != "" {
				logger.Debug("knowledge reflection: agent response",
					"category", input.Category,
					"response", respText,
				)
			}
		}

		// Finish trace recording
		if recorder != nil {
			if err := recorder.Finish(bgCtx); err != nil {
				logger.Error("knowledge reflection: failed to finish trace", "error", err)
			}
		}

		// Notify caller of completion
		if input.OnComplete != nil {
			input.OnComplete(bgCtx, traceID)
		}
	}()

	return nil
}

// newReflectionToolMiddleware creates a gollem ToolMiddleware that logs all tool calls.
func newReflectionToolMiddleware() gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			logger := logging.From(ctx)
			argsJSON, _ := json.Marshal(req.Tool.Arguments)
			logger.Debug("knowledge reflection: tool call",
				"tool", req.Tool.Name,
				"args", string(argsJSON),
			)

			resp, err := next(ctx, req)
			if err != nil {
				logger.Debug("knowledge reflection: tool call failed",
					"tool", req.Tool.Name,
					"error", err,
				)
			} else if resp != nil {
				resultJSON, _ := json.Marshal(resp.Result)
				logger.Debug("knowledge reflection: tool call result",
					"tool", req.Tool.Name,
					"result", string(resultJSON),
				)
			}
			return resp, err
		}
	}
}

// newReflectionContentBlockMiddleware creates a gollem ContentBlockMiddleware
// that logs LLM messages during reflection.
func newReflectionContentBlockMiddleware() gollem.ContentBlockMiddleware {
	return func(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
		return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
			logger := logging.From(ctx)
			logger.Debug("knowledge reflection: LLM generate",
				"input_count", len(req.Inputs),
			)

			resp, err := next(ctx, req)
			if err != nil {
				logger.Debug("knowledge reflection: LLM generate failed",
					"error", err,
				)
			} else if resp != nil {
				logger.Debug("knowledge reflection: LLM response",
					"texts", resp.Texts,
					"function_calls_count", len(resp.FunctionCalls),
					"input_tokens", resp.InputToken,
					"output_tokens", resp.OutputToken,
				)
			}
			return resp, err
		}
	}
}
