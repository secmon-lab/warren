package knowledge

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

//go:embed prompt/introspection_fact.md
var introspectionFactPrompt string

//go:embed prompt/introspection_technique.md
var introspectionTechniquePrompt string

// IntrospectionInput holds the input for the knowledge introspection agent.
type IntrospectionInput struct {
	Category types.KnowledgeCategory

	// ExecutionSummary is a text summary of the execution history.
	// This is provided instead of raw History to avoid gollem type coupling.
	ExecutionSummary string

	// Set only for ticket resolve introspection
	Ticket   *ticket.Ticket
	TicketID types.TicketID
}

// RunIntrospection launches a background introspection agent that extracts
// knowledge from the given execution summary and saves it to the knowledge store.
// The toolSet must be a knowledge tool in ModeReadWrite for the target category.
// Caller is responsible for creating the tool to avoid import cycles.
func (s *Service) RunIntrospection(ctx context.Context, llmClient gollem.LLMClient, toolSet gollem.ToolSet, input *IntrospectionInput) error {
	logger := logging.From(ctx)

	if input == nil {
		logger.Debug("knowledge introspection skipped: nil input")
		return nil
	}
	if input.ExecutionSummary == "" {
		logger.Debug("knowledge introspection skipped: empty execution summary",
			"category", input.Category,
		)
		return nil
	}

	// Select prompt based on category
	var systemPrompt string
	switch input.Category {
	case types.KnowledgeCategoryFact:
		systemPrompt = introspectionFactPrompt
	case types.KnowledgeCategoryTechnique:
		systemPrompt = introspectionTechniquePrompt
	default:
		return goerr.New("unsupported category for introspection", goerr.V("category", input.Category))
	}

	// Append existing tags to prompt so agent reuses them
	existingTags, err := s.ListTags(ctx)
	if err != nil {
		logger.Warn("failed to list existing tags for introspection prompt", "error", err)
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
		systemPrompt += "This introspection is triggered by a ticket resolve.\n"
		systemPrompt += "Conclusion: " + string(input.Ticket.Conclusion) + "\n"
		systemPrompt += "Reason: " + input.Ticket.Reason + "\n"
		if input.Ticket.Finding != nil {
			systemPrompt += "Finding Severity: " + string(input.Ticket.Finding.Severity) + "\n"
			systemPrompt += "Finding Summary: " + input.Ticket.Finding.Summary + "\n"
			systemPrompt += "Finding Recommendation: " + input.Ticket.Finding.Recommendation + "\n"
		}
	}

	logger.Debug("knowledge introspection: building agent",
		"category", input.Category,
		"system_prompt_length", len(systemPrompt),
		"has_ticket_context", input.Ticket != nil,
	)

	// Build agent options with middleware for logging
	opts := []gollem.Option{
		gollem.WithToolSets(toolSet),
		gollem.WithSystemPrompt(systemPrompt),
		gollem.WithResponseMode(gollem.ResponseModeBlocking),
		gollem.WithToolMiddleware(newIntrospectionToolMiddleware()),
		gollem.WithContentBlockMiddleware(newIntrospectionContentBlockMiddleware()),
	}

	// Create agent
	agent := gollem.New(llmClient, opts...)

	summaryLen := len(input.ExecutionSummary)
	hasTicket := input.Ticket != nil

	logger.Info("knowledge introspection started",
		"category", input.Category,
		"summary_length", summaryLen,
		"has_ticket", hasTicket,
	)

	// Execute in background
	go func() {
		bgCtx := context.WithoutCancel(ctx)
		logger.Debug("knowledge introspection: executing agent",
			"category", input.Category,
			"summary_length", summaryLen,
		)

		resp, err := agent.Execute(bgCtx, gollem.Text(input.ExecutionSummary))
		if err != nil {
			logger.Error("knowledge introspection failed",
				"category", input.Category,
				"summary_length", summaryLen,
				"error", err,
			)
		} else {
			var respText string
			if resp != nil && !resp.IsEmpty() {
				respText = resp.String()
			}
			logger.Info("knowledge introspection completed",
				"category", input.Category,
				"summary_length", summaryLen,
				"response_length", len(respText),
			)
			if respText != "" {
				logger.Debug("knowledge introspection: agent response",
					"category", input.Category,
					"response", respText,
				)
			}
		}
	}()

	return nil
}

// newIntrospectionToolMiddleware creates a gollem ToolMiddleware that logs all tool calls.
func newIntrospectionToolMiddleware() gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			logger := logging.From(ctx)
			argsJSON, _ := json.Marshal(req.Tool.Arguments)
			logger.Debug("knowledge introspection: tool call",
				"tool", req.Tool.Name,
				"args", string(argsJSON),
			)

			resp, err := next(ctx, req)
			if err != nil {
				logger.Debug("knowledge introspection: tool call failed",
					"tool", req.Tool.Name,
					"error", err,
				)
			} else if resp != nil {
				resultJSON, _ := json.Marshal(resp.Result)
				logger.Debug("knowledge introspection: tool call result",
					"tool", req.Tool.Name,
					"result", string(resultJSON),
				)
			}
			return resp, err
		}
	}
}

// newIntrospectionContentBlockMiddleware creates a gollem ContentBlockMiddleware
// that logs LLM messages during introspection.
func newIntrospectionContentBlockMiddleware() gollem.ContentBlockMiddleware {
	return func(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
		return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
			logger := logging.From(ctx)
			logger.Debug("knowledge introspection: LLM generate",
				"input_count", len(req.Inputs),
			)

			resp, err := next(ctx, req)
			if err != nil {
				logger.Debug("knowledge introspection: LLM generate failed",
					"error", err,
				)
			} else if resp != nil {
				logger.Debug("knowledge introspection: LLM response",
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
