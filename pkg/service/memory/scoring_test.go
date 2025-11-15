package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	memoryRepo "github.com/secmon-lab/warren/pkg/repository/memory"
	memoryService "github.com/secmon-lab/warren/pkg/service/memory"
)

func TestScoringConfig_Validate(t *testing.T) {
	t.Run("default config is valid", func(t *testing.T) {
		cfg := memoryService.DefaultScoringConfig()
		gt.NoError(t, cfg.Validate())
	})

	t.Run("invalid EMA alpha", func(t *testing.T) {
		cfg := memoryService.DefaultScoringConfig()
		cfg.EMAAlpha = 1.5
		gt.Error(t, cfg.Validate())

		cfg.EMAAlpha = -0.1
		gt.Error(t, cfg.Validate())
	})

	t.Run("invalid score range", func(t *testing.T) {
		cfg := memoryService.DefaultScoringConfig()
		cfg.ScoreMin = 10.0
		cfg.ScoreMax = -10.0
		gt.Error(t, cfg.Validate())
	})

	t.Run("invalid search parameters", func(t *testing.T) {
		cfg := memoryService.DefaultScoringConfig()
		cfg.SearchMultiplier = 0
		gt.Error(t, cfg.Validate())

		cfg = memoryService.DefaultScoringConfig()
		cfg.SearchMaxCandidates = -1
		gt.Error(t, cfg.Validate())
	})

	t.Run("invalid ranking weights", func(t *testing.T) {
		cfg := memoryService.DefaultScoringConfig()
		cfg.RankSimilarityWeight = -0.1
		gt.Error(t, cfg.Validate())
	})

	t.Run("invalid recency half-life", func(t *testing.T) {
		cfg := memoryService.DefaultScoringConfig()
		cfg.RecencyHalfLifeDays = 0
		gt.Error(t, cfg.Validate())
	})

	t.Run("invalid pruning thresholds order", func(t *testing.T) {
		cfg := memoryService.DefaultScoringConfig()
		cfg.PruneCriticalScore = -3.0
		cfg.PruneHarmfulScore = -5.0
		gt.Error(t, cfg.Validate())
	})

	t.Run("invalid pruning days", func(t *testing.T) {
		cfg := memoryService.DefaultScoringConfig()
		cfg.PruneHarmfulDays = 200
		cfg.PruneModerateDays = 100
		gt.Error(t, cfg.Validate())
	})
}

func TestPruneAgentMemories(t *testing.T) {
	ctx := context.Background()
	repo := memoryRepo.New()
	llm := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
			embeddings := make([][]float64, len(input))
			for i := range input {
				vec := make([]float64, dimension)
				for j := 0; j < dimension; j++ {
					vec[j] = 0.1 * float64(i+j)
				}
				embeddings[i] = vec
			}
			return embeddings, nil
		},
	}
	svc := memoryService.New(llm, repo)

	agentID := "test-agent"
	now := time.Now()

	// Create test memories with various scores and timestamps
	memories := []*memory.AgentMemory{
		// Critical score - should be deleted immediately
		{
			ID:             types.NewAgentMemoryID(),
			AgentID:        agentID,
			TaskQuery:      "critical bad memory",
			QueryEmbedding: []float32{0.1, 0.2, 0.3},
			Timestamp:      now.Add(-10 * 24 * time.Hour),
			QualityScore:   -9.0,
			LastUsedAt:     now.Add(-10 * 24 * time.Hour),
		},
		// Harmful + old - should be deleted
		{
			ID:             types.NewAgentMemoryID(),
			AgentID:        agentID,
			TaskQuery:      "harmful old memory",
			QueryEmbedding: []float32{0.1, 0.2, 0.3},
			Timestamp:      now.Add(-100 * 24 * time.Hour),
			QualityScore:   -6.0,
			LastUsedAt:     now.Add(-100 * 24 * time.Hour),
		},
		// Harmful but recent - should NOT be deleted
		{
			ID:             types.NewAgentMemoryID(),
			AgentID:        agentID,
			TaskQuery:      "harmful recent memory",
			QueryEmbedding: []float32{0.1, 0.2, 0.3},
			Timestamp:      now.Add(-10 * 24 * time.Hour),
			QualityScore:   -6.0,
			LastUsedAt:     now.Add(-10 * 24 * time.Hour),
		},
		// Moderate + very old - should be deleted
		{
			ID:             types.NewAgentMemoryID(),
			AgentID:        agentID,
			TaskQuery:      "moderate very old memory",
			QueryEmbedding: []float32{0.1, 0.2, 0.3},
			Timestamp:      now.Add(-200 * 24 * time.Hour),
			QualityScore:   -4.0,
			LastUsedAt:     now.Add(-200 * 24 * time.Hour),
		},
		// Good memory - should NOT be deleted
		{
			ID:             types.NewAgentMemoryID(),
			AgentID:        agentID,
			TaskQuery:      "good memory",
			QueryEmbedding: []float32{0.1, 0.2, 0.3},
			Timestamp:      now,
			QualityScore:   5.0,
			LastUsedAt:     now,
		},
	}

	// Save all memories
	for _, mem := range memories {
		gt.NoError(t, repo.SaveAgentMemory(ctx, mem))
	}

	// Run pruning
	deleted, err := svc.PruneAgentMemories(ctx, agentID)
	gt.NoError(t, err)

	// Should delete 3 memories: critical, harmful+old, moderate+very old
	gt.Equal(t, deleted, 3)

	// Verify remaining memories
	remaining, err := repo.ListAgentMemories(ctx, agentID)
	gt.NoError(t, err)
	gt.Equal(t, len(remaining), 2) // harmful recent + good memory
}

func TestScoringConfig_CustomValues(t *testing.T) {
	repo := memoryRepo.New()
	llm := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
			embeddings := make([][]float64, len(input))
			for i := range input {
				vec := make([]float64, dimension)
				for j := 0; j < dimension; j++ {
					vec[j] = 0.1 * float64(i+j)
				}
				embeddings[i] = vec
			}
			return embeddings, nil
		},
	}
	svc := memoryService.New(llm, repo)

	// Test custom config
	customConfig := memoryService.ScoringConfig{
		EMAAlpha:             0.5,
		ScoreMin:             -10.0,
		ScoreMax:             10.0,
		SearchMultiplier:     5,
		SearchMaxCandidates:  25,
		FilterMinQuality:     -3.0,
		RankSimilarityWeight: 0.6,
		RankQualityWeight:    0.3,
		RankRecencyWeight:    0.1,
		RecencyHalfLifeDays:  60,
		PruneCriticalScore:   -9.0,
		PruneHarmfulScore:    -6.0,
		PruneHarmfulDays:     60,
		PruneModerateScore:   -4.0,
		PruneModerateDays:    120,
	}

	gt.NoError(t, customConfig.Validate())
	svc.ScoringConfig = customConfig

	// Verify the custom config is applied
	gt.Equal(t, svc.ScoringConfig.EMAAlpha, 0.5)
	gt.Equal(t, svc.ScoringConfig.SearchMultiplier, 5)
	gt.Equal(t, svc.ScoringConfig.RecencyHalfLifeDays, 60.0)
}
