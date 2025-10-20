package memory

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

//go:embed prompt/execution_memory.md
var executionMemoryPromptTemplate string

//go:embed prompt/ticket_memory.md
var ticketMemoryPromptTemplate string

type Service struct {
	llmClient  gollem.LLMClient
	repository interfaces.Repository
}

func New(llmClient gollem.LLMClient, repo interfaces.Repository) *Service {
	return &Service{
		llmClient:  llmClient,
		repository: repo,
	}
}

// GenerateExecutionMemory generates memory from execution history
func (s *Service) GenerateExecutionMemory(
	ctx context.Context,
	schemaID types.AlertSchema,
	history *gollem.History,
	executionError error,
) (*memory.ExecutionMemory, error) {
	// 1. Create session with history
	session, err := s.llmClient.NewSession(ctx, gollem.WithSessionHistory(history))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create session with history")
	}

	// 2. Get existing memory
	existing, err := s.repository.GetExecutionMemory(ctx, schemaID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get existing execution memory")
	}

	// 3. Build prompt
	promptText, err := s.buildExecutionMemoryPrompt(ctx, existing, executionError)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build execution memory prompt")
	}

	// 4. Generate content
	response, err := session.GenerateContent(ctx, gollem.Text(promptText))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate execution memory")
	}

	// 5. Parse response
	mem, err := s.parseExecutionMemoryResponse(response, schemaID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to parse execution memory response")
	}

	return mem, nil
}

// GenerateTicketMemory generates memory from ticket resolution
func (s *Service) GenerateTicketMemory(
	ctx context.Context,
	schemaID types.AlertSchema,
	ticketData *ticket.Ticket,
	comments []ticket.Comment,
) (*memory.TicketMemory, error) {
	// 1. Get existing memory
	existing, err := s.repository.GetTicketMemory(ctx, schemaID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get existing ticket memory")
	}

	// 2. Build prompt
	promptText, err := s.buildTicketMemoryPrompt(ctx, existing, ticketData, comments)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build ticket memory prompt")
	}

	// 3. Create session and generate content
	session, err := s.llmClient.NewSession(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create LLM session")
	}

	response, err := session.GenerateContent(ctx, gollem.Text(promptText))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate ticket memory")
	}

	// 4. Parse response
	mem, err := s.parseTicketMemoryResponse(response, schemaID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to parse ticket memory response")
	}

	return mem, nil
}

// LoadMemoriesForPrompt loads both types of memories for a schema
func (s *Service) LoadMemoriesForPrompt(
	ctx context.Context,
	schemaID types.AlertSchema,
) (execMem *memory.ExecutionMemory, ticketMem *memory.TicketMemory, err error) {
	execMem, err = s.repository.GetExecutionMemory(ctx, schemaID)
	if err != nil {
		return nil, nil, goerr.Wrap(err, "failed to get execution memory")
	}

	ticketMem, err = s.repository.GetTicketMemory(ctx, schemaID)
	if err != nil {
		return nil, nil, goerr.Wrap(err, "failed to get ticket memory")
	}

	return execMem, ticketMem, nil
}

// FormatMemoriesForPrompt formats memories as markdown for system prompt
func (s *Service) FormatMemoriesForPrompt(
	execMem *memory.ExecutionMemory,
	ticketMem *memory.TicketMemory,
) string {
	if (execMem == nil || execMem.IsEmpty()) && (ticketMem == nil || ticketMem.IsEmpty()) {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("# Accumulated Knowledge\n\n")
	builder.WriteString("The following knowledge has been accumulated from past executions and ticket resolutions:\n\n")

	if execMem != nil && !execMem.IsEmpty() {
		builder.WriteString("## Execution Learnings\n\n")
		if execMem.Keep != "" {
			builder.WriteString("**Keep (Successful Patterns):**\n")
			builder.WriteString(execMem.Keep)
			builder.WriteString("\n\n")
		}
		if execMem.Change != "" {
			builder.WriteString("**Change (Areas for Improvement):**\n")
			builder.WriteString(execMem.Change)
			builder.WriteString("\n\n")
		}
		if execMem.Notes != "" {
			builder.WriteString("**Notes (Other Insights):**\n")
			builder.WriteString(execMem.Notes)
			builder.WriteString("\n\n")
		}
	}

	if ticketMem != nil && !ticketMem.IsEmpty() {
		builder.WriteString("## Organizational Security Knowledge\n\n")
		builder.WriteString(ticketMem.Insights)
		builder.WriteString("\n\n")
	}

	return builder.String()
}

// buildExecutionMemoryPrompt builds the prompt for execution memory generation
func (s *Service) buildExecutionMemoryPrompt(ctx context.Context, existing *memory.ExecutionMemory, executionError error) (string, error) {
	// Generate JSON schema
	type ExecutionMemoryResponse struct {
		Keep   string `json:"keep"`
		Change string `json:"change"`
		Notes  string `json:"notes"`
	}

	schema := prompt.ToSchema(ExecutionMemoryResponse{})
	jsonSchema, err := schema.Stringify()
	if err != nil {
		return "", goerr.Wrap(err, "failed to stringify JSON schema")
	}

	params := map[string]any{
		"json_schema": jsonSchema,
	}

	if existing != nil && !existing.IsEmpty() {
		params["existing_memory"] = existing
	}

	if executionError != nil {
		params["error"] = executionError.Error()
	}

	return prompt.Generate(ctx, executionMemoryPromptTemplate, params)
}

// parseExecutionMemoryResponse parses LLM response into ExecutionMemory
func (s *Service) parseExecutionMemoryResponse(response *gollem.Response, schemaID types.AlertSchema) (*memory.ExecutionMemory, error) {
	if len(response.Texts) == 0 {
		return nil, goerr.New("no response text from LLM")
	}

	text := response.Texts[0]

	// Extract JSON from markdown code blocks if present
	text = extractJSONFromMarkdown(text)

	var resp struct {
		Keep   string `json:"keep"`
		Change string `json:"change"`
		Notes  string `json:"notes"`
	}

	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal execution memory response", goerr.V("text", text))
	}

	mem := memory.NewExecutionMemory(schemaID)
	mem.Keep = resp.Keep
	mem.Change = resp.Change
	mem.Notes = resp.Notes

	return mem, nil
}

// buildTicketMemoryPrompt builds the prompt for ticket memory generation
func (s *Service) buildTicketMemoryPrompt(ctx context.Context, existing *memory.TicketMemory, ticketData *ticket.Ticket, comments []ticket.Comment) (string, error) {
	// Generate JSON schema
	type TicketMemoryResponse struct {
		Insights string `json:"insights"`
	}

	schema := prompt.ToSchema(TicketMemoryResponse{})
	jsonSchema, err := schema.Stringify()
	if err != nil {
		return "", goerr.Wrap(err, "failed to stringify JSON schema")
	}

	params := map[string]any{
		"ticket":      ticketData,
		"comments":    comments,
		"json_schema": jsonSchema,
	}

	if existing != nil && !existing.IsEmpty() {
		params["existing_memory"] = existing
	}

	return prompt.Generate(ctx, ticketMemoryPromptTemplate, params)
}

// parseTicketMemoryResponse parses LLM response into TicketMemory
func (s *Service) parseTicketMemoryResponse(response *gollem.Response, schemaID types.AlertSchema) (*memory.TicketMemory, error) {
	if len(response.Texts) == 0 {
		return nil, goerr.New("no response text from LLM")
	}

	text := response.Texts[0]

	// Extract JSON from markdown code blocks if present
	text = extractJSONFromMarkdown(text)

	var resp struct {
		Insights string `json:"insights"`
	}

	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal ticket memory response", goerr.V("text", text))
	}

	mem := memory.NewTicketMemory(schemaID)
	mem.Insights = resp.Insights

	return mem, nil
}

// extractJSONFromMarkdown extracts JSON content from markdown code blocks
func extractJSONFromMarkdown(text string) string {
	text = strings.TrimSpace(text)

	// Check if wrapped in markdown code block
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	return text
}
