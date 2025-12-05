package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	memoryService "github.com/secmon-lab/warren/pkg/service/memory"
)

func createTestRepository(t *testing.T) interfaces.Repository {
	t.Helper()
	return repository.NewMemory()
}

// TestFormatMemoriesForPrompt is removed - ExecutionMemory and TicketMemory features have been deleted
// TestLoadMemoriesForPrompt is removed - ExecutionMemory and TicketMemory features have been deleted
// TestGenerateExecutionMemoryWithRealLLM is removed - ExecutionMemory feature has been deleted
// TestGenerateTicketMemoryWithRealLLM is removed - TicketMemory feature has been deleted

func TestAgentMemory_SaveAndSearch(t *testing.T) {
	repo := createTestRepository(t)
	llmClient := &mock.LLMClientMock{
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
	svc := memoryService.New(llmClient, repo)
	ctx := context.Background()

	// Create test memory
	mem1 := &memory.AgentMemory{
		ID:             types.NewAgentMemoryID(),
		AgentID:        "bigquery",
		TaskQuery:      "get login errors",
		QueryEmbedding: []float32{0.1, 0.2, 0.3},
		Timestamp:      time.Now(),
		Duration:       time.Second,
		Successes:      []string{"Successfully executed query"},
		Problems:       []string{},
		Improvements:   []string{},
	}

	mem2 := &memory.AgentMemory{
		ID:             types.NewAgentMemoryID(),
		AgentID:        "bigquery",
		TaskQuery:      "get authentication failures",
		QueryEmbedding: []float32{0.15, 0.25, 0.35},
		Timestamp:      time.Now(),
		Duration:       2 * time.Second,
		Problems:       []string{"Query timeout"},
		Improvements:   []string{"Add index"},
	}

	// Different agent
	mem3 := &memory.AgentMemory{
		ID:             types.NewAgentMemoryID(),
		AgentID:        "virustotal",
		TaskQuery:      "scan file hash",
		QueryEmbedding: []float32{0.5, 0.6, 0.7},
		Timestamp:      time.Now(),
		Duration:       time.Second,
	}

	// Save memories
	gt.NoError(t, svc.SaveAgentMemory(ctx, mem1))
	gt.NoError(t, svc.SaveAgentMemory(ctx, mem2))
	gt.NoError(t, svc.SaveAgentMemory(ctx, mem3))

	// Search by similar embedding (should find mem1 and mem2, not mem3)
	results, err := svc.SearchRelevantAgentMemories(ctx, "bigquery", "login errors", 2)
	gt.NoError(t, err)
	gt.V(t, len(results)).Equal(2)

	// Verify results are from correct agent
	for _, r := range results {
		gt.V(t, r.AgentID).Equal("bigquery")
	}
}

// newMockLLMClient creates a mock LLM client for integration testing
func newMockLLMClient() gollem.LLMClient {
	return &mock.LLMClientMock{
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
}

func TestMemoryService_E2E_ScoringFlow(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	llm := newMockLLMClient()
	svc := memoryService.New(llm, repo)

	agentID := "e2e-test-agent"

	t.Run("End-to-end scoring flow", func(t *testing.T) {
		// Step 1: Save memories with different quality scores
		now := time.Now()
		memories := []*memory.AgentMemory{
			{
				ID:           types.NewAgentMemoryID(),
				AgentID:      agentID,
				TaskQuery:    "high quality memory",
				Timestamp:    now.Add(-1 * time.Hour),
				QualityScore: 8.0,
				LastUsedAt:   now.Add(-1 * time.Hour),
				Successes:    []string{"Success pattern 1"},
				Problems:     []string{},
				Improvements: []string{"Improvement 1"},
			},
			{
				ID:           types.NewAgentMemoryID(),
				AgentID:      agentID,
				TaskQuery:    "medium quality memory",
				Timestamp:    now.Add(-2 * time.Hour),
				QualityScore: 2.0,
				LastUsedAt:   now.Add(-2 * time.Hour),
				Successes:    []string{"Success pattern 2"},
				Problems:     []string{},
				Improvements: []string{"Improvement 2"},
			},
			{
				ID:           types.NewAgentMemoryID(),
				AgentID:      agentID,
				TaskQuery:    "low quality memory",
				Timestamp:    now.Add(-3 * time.Hour),
				QualityScore: -6.0,
				LastUsedAt:   now.Add(-100 * 24 * time.Hour), // 100 days ago
				Successes:    []string{},
				Problems:     []string{"Problem 1"},
				Improvements: []string{},
			},
			{
				ID:           types.NewAgentMemoryID(),
				AgentID:      agentID,
				TaskQuery:    "critical bad memory",
				Timestamp:    now.Add(-4 * time.Hour),
				QualityScore: -9.0,
				LastUsedAt:   now.Add(-4 * time.Hour),
				Successes:    []string{},
				Problems:     []string{"Critical problem"},
				Improvements: []string{},
			},
		}

		for _, mem := range memories {
			gt.NoError(t, svc.SaveAgentMemory(ctx, mem))
		}

		// Step 2: Search for relevant memories
		// The search should use re-ranking with quality scores
		results, err := svc.SearchRelevantAgentMemories(ctx, agentID, "quality memory", 3)
		gt.NoError(t, err)
		gt.N(t, len(results)).Greater(0)

		// High quality memories should be prioritized
		// (though exact ordering depends on similarity too)
		hasHighQuality := false
		for _, r := range results {
			if r.QualityScore > 5.0 {
				hasHighQuality = true
				break
			}
		}
		gt.True(t, hasHighQuality)

		// Step 3: Test pruning - should delete critical bad memory
		deleted, err := svc.PruneAgentMemories(ctx, agentID)
		gt.NoError(t, err)
		gt.N(t, deleted).Greater(0) // At least the critical one should be deleted

		// Step 4: Verify pruning results
		remaining, err := repo.ListAgentMemories(ctx, agentID)
		gt.NoError(t, err)

		// Critical bad memory (-9.0) should be gone
		hasCriticalBad := false
		for _, r := range remaining {
			if r.QualityScore <= -8.0 {
				hasCriticalBad = true
				break
			}
		}
		gt.False(t, hasCriticalBad)

		// High quality memory should still exist
		hasHighQualityAfterPrune := false
		for _, r := range remaining {
			if r.QualityScore > 5.0 {
				hasHighQualityAfterPrune = true
				break
			}
		}
		gt.True(t, hasHighQualityAfterPrune)
	})

	t.Run("Re-ranking with different weights", func(t *testing.T) {
		// Clean up
		existing, err := repo.ListAgentMemories(ctx, agentID)
		gt.NoError(t, err)
		if len(existing) > 0 {
			ids := make([]types.AgentMemoryID, len(existing))
			for i, m := range existing {
				ids[i] = m.ID
			}
			_, err = repo.DeleteAgentMemoriesBatch(ctx, agentID, ids)
			gt.NoError(t, err)
		}

		// Create memories with different quality and recency
		now := time.Now()
		memories := []*memory.AgentMemory{
			{
				ID:           types.NewAgentMemoryID(),
				AgentID:      agentID,
				TaskQuery:    "old high quality",
				Timestamp:    now.Add(-30 * 24 * time.Hour),
				QualityScore: 9.0,
				LastUsedAt:   now.Add(-30 * 24 * time.Hour),
				Successes:    []string{"Old success"},
				Problems:     []string{},
				Improvements: []string{},
			},
			{
				ID:           types.NewAgentMemoryID(),
				AgentID:      agentID,
				TaskQuery:    "recent medium quality",
				Timestamp:    now.Add(-1 * time.Hour),
				QualityScore: 3.0,
				LastUsedAt:   now.Add(-1 * time.Hour),
				Successes:    []string{"Recent success"},
				Problems:     []string{},
				Improvements: []string{},
			},
		}

		for _, mem := range memories {
			gt.NoError(t, svc.SaveAgentMemory(ctx, mem))
		}

		// Test with quality-heavy weights
		svc.ScoringConfig.RankQualityWeight = 0.7
		svc.ScoringConfig.RankRecencyWeight = 0.1
		svc.ScoringConfig.RankSimilarityWeight = 0.2

		results, err := svc.SearchRelevantAgentMemories(ctx, agentID, "quality", 2)
		gt.NoError(t, err)
		gt.N(t, len(results)).Greater(0)

		// With heavy quality weight, old high quality should rank well
		foundHighQuality := false
		for _, r := range results {
			if r.QualityScore > 5.0 {
				foundHighQuality = true
				break
			}
		}
		gt.True(t, foundHighQuality)
	})
}

func TestMemoryService_E2E_ConfigValidation(t *testing.T) {
	t.Run("Config validation prevents invalid configurations", func(t *testing.T) {
		invalidConfig := memoryService.ScoringConfig{
			EMAAlpha:             1.5, // Invalid: > 1.0
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

		err := invalidConfig.Validate()
		gt.Error(t, err)
	})

	t.Run("Default config is always valid", func(t *testing.T) {
		defaultConfig := memoryService.DefaultScoringConfig()
		gt.NoError(t, defaultConfig.Validate())
	})
}
