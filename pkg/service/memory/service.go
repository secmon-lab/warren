package memory

import (
	"context"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
)

const (
	// EmbeddingDimension is the dimension of embedding vectors
	EmbeddingDimension = 256
)

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
