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
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

const (
	// EmbeddingDimension is the dimension of embedding vectors
	EmbeddingDimension = 256
)

//go:embed prompt/execution_memory.md
var executionMemoryPromptTemplate string

//go:embed prompt/ticket_memory.md
var ticketMemoryPromptTemplate string

type Service struct {
	llmClient     gollem.LLMClient
	repository    interfaces.Repository
	ScoringConfig ScoringConfig
}

func New(llmClient gollem.LLMClient, repo interfaces.Repository) *Service {
	return &Service{
		llmClient:     llmClient,
		repository:    repo,
		ScoringConfig: DefaultScoringConfig(),
	}
}

// ScoringConfig holds all configurable parameters for memory scoring
type ScoringConfig struct {
	// EMA parameters
	EMAAlpha float64 // Weight for new feedback (0.0-1.0), default: 0.3
	ScoreMin float64 // Minimum score value, default: -10.0
	ScoreMax float64 // Maximum score value, default: +10.0

	// Search parameters
	SearchMultiplier    int     // Multiplier for initial search limit, default: 10
	SearchMaxCandidates int     // Maximum candidates for re-ranking, default: 50
	FilterMinQuality    float64 // Minimum quality score for filtering, default: -5.0

	// Ranking weights (should sum to 1.0)
	RankSimilarityWeight float64 // Weight for similarity score, default: 0.5
	RankQualityWeight    float64 // Weight for quality score, default: 0.3
	RankRecencyWeight    float64 // Weight for recency score, default: 0.2
	RecencyHalfLifeDays  float64 // Half-life for recency decay in days, default: 30

	// Pruning thresholds
	PruneCriticalScore float64 // Critical score threshold (immediate deletion), default: -8.0
	PruneHarmfulScore  float64 // Harmful score threshold, default: -5.0
	PruneHarmfulDays   int     // Days before pruning harmful memories, default: 90
	PruneModerateScore float64 // Moderate score threshold, default: -3.0
	PruneModerateDays  int     // Days before pruning moderate memories, default: 180
}

// DefaultScoringConfig returns default configuration
func DefaultScoringConfig() ScoringConfig {
	return ScoringConfig{
		EMAAlpha:             0.3,
		ScoreMin:             -10.0,
		ScoreMax:             10.0,
		SearchMultiplier:     10,
		SearchMaxCandidates:  50,
		FilterMinQuality:     -5.0,
		RankSimilarityWeight: 0.5,
		RankQualityWeight:    0.3,
		RankRecencyWeight:    0.2,
		RecencyHalfLifeDays:  30,
		PruneCriticalScore:   -8.0,
		PruneHarmfulScore:    -5.0,
		PruneHarmfulDays:     90,
		PruneModerateScore:   -3.0,
		PruneModerateDays:    180,
	}
}

// Validate checks if the ScoringConfig has valid values
func (c *ScoringConfig) Validate() error {
	// EMA alpha must be between 0 and 1
	if c.EMAAlpha < 0.0 || c.EMAAlpha > 1.0 {
		return goerr.New("EMAAlpha must be between 0.0 and 1.0", goerr.V("value", c.EMAAlpha))
	}

	// Score range must be valid
	if c.ScoreMin >= c.ScoreMax {
		return goerr.New("ScoreMin must be less than ScoreMax",
			goerr.V("min", c.ScoreMin),
			goerr.V("max", c.ScoreMax))
	}

	// Search parameters must be positive
	if c.SearchMultiplier <= 0 {
		return goerr.New("SearchMultiplier must be positive", goerr.V("value", c.SearchMultiplier))
	}
	if c.SearchMaxCandidates <= 0 {
		return goerr.New("SearchMaxCandidates must be positive", goerr.V("value", c.SearchMaxCandidates))
	}

	// Ranking weights should be non-negative and sum to 1.0
	if c.RankSimilarityWeight < 0 || c.RankQualityWeight < 0 || c.RankRecencyWeight < 0 {
		return goerr.New("ranking weights must be non-negative",
			goerr.V("similarity", c.RankSimilarityWeight),
			goerr.V("quality", c.RankQualityWeight),
			goerr.V("recency", c.RankRecencyWeight))
	}
	const epsilon = 0.001
	if totalWeight := c.RankSimilarityWeight + c.RankQualityWeight + c.RankRecencyWeight; totalWeight < 1.0-epsilon || totalWeight > 1.0+epsilon {
		return goerr.New("ranking weights must sum to 1.0", goerr.V("total", totalWeight))
	}

	// Recency half-life must be positive
	if c.RecencyHalfLifeDays <= 0 {
		return goerr.New("RecencyHalfLifeDays must be positive", goerr.V("value", c.RecencyHalfLifeDays))
	}

	// Pruning thresholds should be in order: critical < harmful < moderate < 0
	if c.PruneCriticalScore >= c.PruneHarmfulScore {
		return goerr.New("PruneCriticalScore must be less than PruneHarmfulScore",
			goerr.V("critical", c.PruneCriticalScore),
			goerr.V("harmful", c.PruneHarmfulScore))
	}
	if c.PruneHarmfulScore >= c.PruneModerateScore {
		return goerr.New("PruneHarmfulScore must be less than PruneModerateScore",
			goerr.V("harmful", c.PruneHarmfulScore),
			goerr.V("moderate", c.PruneModerateScore))
	}

	// Pruning days must be positive
	if c.PruneHarmfulDays <= 0 {
		return goerr.New("PruneHarmfulDays must be positive", goerr.V("value", c.PruneHarmfulDays))
	}
	if c.PruneModerateDays <= 0 {
		return goerr.New("PruneModerateDays must be positive", goerr.V("value", c.PruneModerateDays))
	}

	// Harmful days should be less than moderate days
	if c.PruneHarmfulDays >= c.PruneModerateDays {
		return goerr.New("PruneHarmfulDays should be less than PruneModerateDays",
			goerr.V("harmful", c.PruneHarmfulDays),
			goerr.V("moderate", c.PruneModerateDays))
	}

	return nil
}

// executionMemoryResponse defines the structure for ExecutionMemory LLM response
type executionMemoryResponse struct {
	Summary string `json:"summary" description:"Concise 1-2 sentence overview of what was accomplished and key learnings, used for semantic search"`
	Keep    string `json:"keep,omitempty" description:"Specific successful execution strategies: exact tools used, query patterns, effective methods"`
	Change  string `json:"change,omitempty" description:"Specific failures and root causes: which tools failed, why they failed, concrete improvements"`
	Notes   string `json:"notes,omitempty" description:"Additional insights: data schema observations, edge cases, contextual information"`
}

// GenerateExecutionMemory generates memory from execution history
func (s *Service) GenerateExecutionMemory(
	ctx context.Context,
	schemaID types.AlertSchema,
	history *gollem.History,
	executionError error,
) (*memory.ExecutionMemory, error) {
	// 1. Define response schema for structured output
	schema, err := gollem.ToSchema(executionMemoryResponse{})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate schema")
	}

	// 2. Create session with history and schema
	session, err := s.llmClient.NewSession(ctx,
		gollem.WithSessionHistory(history),
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(schema),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create session with history")
	}

	// 3. Get existing memory
	existing, err := s.repository.GetExecutionMemory(ctx, schemaID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get existing execution memory")
	}

	// 4. Build prompt
	promptText, err := s.buildExecutionMemoryPrompt(ctx, existing, executionError)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build execution memory prompt")
	}

	// 5. Generate content
	response, err := session.GenerateContent(ctx, gollem.Text(promptText))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate execution memory")
	}

	// 6. Parse response and generate embedding
	mem, err := s.parseExecutionMemoryResponse(ctx, response, schemaID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to parse execution memory response")
	}

	return mem, nil
}

// ticketMemoryResponse defines the structure for TicketMemory LLM response
type ticketMemoryResponse struct {
	Insights string `json:"insights" description:"Key security insights and knowledge gained from ticket resolution"`
}

// GenerateTicketMemory generates memory from ticket resolution
func (s *Service) GenerateTicketMemory(
	ctx context.Context,
	schemaID types.AlertSchema,
	ticketData *ticket.Ticket,
	comments []ticket.Comment,
) (*memory.TicketMemory, error) {
	// 1. Define response schema for structured output
	schema, err := gollem.ToSchema(ticketMemoryResponse{})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate schema")
	}

	// 2. Get existing memory
	existing, err := s.repository.GetTicketMemory(ctx, schemaID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get existing ticket memory")
	}

	// 3. Build prompt
	promptText, err := s.buildTicketMemoryPrompt(ctx, existing, ticketData, comments)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build ticket memory prompt")
	}

	// 4. Create session with schema and generate content
	session, err := s.llmClient.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(schema),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create LLM session")
	}

	response, err := session.GenerateContent(ctx, gollem.Text(promptText))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate ticket memory")
	}

	// 5. Parse response
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
		Summary string `json:"summary"`
		Keep    string `json:"keep"`
		Change  string `json:"change"`
		Notes   string `json:"notes"`
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
func (s *Service) parseExecutionMemoryResponse(ctx context.Context, response *gollem.Response, schemaID types.AlertSchema) (*memory.ExecutionMemory, error) {
	if len(response.Texts) == 0 {
		return nil, goerr.New("no response text from LLM")
	}

	text := response.Texts[0]

	// Extract JSON from markdown code blocks if present
	text = extractJSONFromMarkdown(text)

	var resp struct {
		Summary string `json:"summary"`
		Keep    string `json:"keep"`
		Change  string `json:"change"`
		Notes   string `json:"notes"`
	}

	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal execution memory response", goerr.V("text", text))
	}

	mem := memory.NewExecutionMemory(schemaID)
	mem.Summary = resp.Summary
	mem.Keep = resp.Keep
	mem.Change = resp.Change
	mem.Notes = resp.Notes

	// Generate embedding from summary if summary is not empty
	if mem.Summary != "" {
		embeddings, err := s.llmClient.GenerateEmbedding(ctx, EmbeddingDimension, []string{mem.Summary})
		if err != nil {
			// Log error but continue - embedding is optional
			logging.From(ctx).Warn("failed to generate embedding for execution memory summary", "error", err)
			return mem, nil
		}
		if len(embeddings) > 0 {
			// Convert float64 to float32
			vector32 := make([]float32, len(embeddings[0]))
			for i, v := range embeddings[0] {
				vector32[i] = float32(v)
			}
			mem.Embedding = vector32
		}
	}

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

// SaveAgentMemory saves an agent memory record with automatic embedding generation
func (s *Service) SaveAgentMemory(ctx context.Context, mem *memory.AgentMemory) error {
	if err := mem.Validate(); err != nil {
		return goerr.Wrap(err, "invalid agent memory")
	}

	// Generate embedding for TaskQuery if not already present
	if len(mem.QueryEmbedding) == 0 {
		embeddings, err := s.llmClient.GenerateEmbedding(ctx, EmbeddingDimension, []string{mem.TaskQuery})
		if err != nil {
			return goerr.Wrap(err, "failed to generate embedding", goerr.V("task_query", mem.TaskQuery))
		}
		if len(embeddings) > 0 {
			// Convert float64 to float32
			vector32 := make([]float32, len(embeddings[0]))
			for i, v := range embeddings[0] {
				vector32[i] = float32(v)
			}
			mem.QueryEmbedding = vector32
		}
	}

	if err := s.repository.SaveAgentMemory(ctx, mem); err != nil {
		return goerr.Wrap(err, "failed to save agent memory", goerr.V("id", mem.ID))
	}

	return nil
}

// SearchRelevantAgentMemories searches for similar memories using semantic search with re-ranking
func (s *Service) SearchRelevantAgentMemories(ctx context.Context, agentID, query string, limit int) ([]*memory.AgentMemory, error) {
	// Generate embedding for the query
	embeddings, err := s.llmClient.GenerateEmbedding(ctx, EmbeddingDimension, []string{query})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate embedding for search", goerr.V("query", query))
	}

	if len(embeddings) == 0 {
		return nil, goerr.New("no embedding generated")
	}

	// Convert float64 to float32
	vector32 := make([]float32, len(embeddings[0]))
	for i, v := range embeddings[0] {
		vector32[i] = float32(v)
	}

	// Use the new searchAndRerankMemories method which incorporates quality and recency scoring
	memories, err := s.searchAndRerankMemories(ctx, agentID, vector32, limit)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to search and rerank memories", goerr.V("agent_id", agentID), goerr.V("limit", limit))
	}

	return memories, nil
}

// SearchRelevantExecutionMemories searches for similar execution memories using semantic search
func (s *Service) SearchRelevantExecutionMemories(ctx context.Context, schemaID types.AlertSchema, query string, limit int) ([]*memory.ExecutionMemory, error) {
	// Generate embedding for the query
	embeddings, err := s.llmClient.GenerateEmbedding(ctx, EmbeddingDimension, []string{query})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate embedding for search", goerr.V("query", query))
	}

	if len(embeddings) == 0 {
		return nil, goerr.New("no embedding generated")
	}

	// Convert float64 to float32
	vector32 := make([]float32, len(embeddings[0]))
	for i, v := range embeddings[0] {
		vector32[i] = float32(v)
	}

	// Search by embedding
	memories, err := s.repository.SearchExecutionMemoriesByEmbedding(ctx, schemaID, vector32, limit)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to search execution memories", goerr.V("schema_id", schemaID), goerr.V("limit", limit))
	}

	return memories, nil
}
