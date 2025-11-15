package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	memoryRepo "github.com/secmon-lab/warren/pkg/repository/memory"
)

func TestMemoryRepository_ScoringIntegration(t *testing.T) {
	ctx := context.Background()
	repo := memoryRepo.New()
	agentID := "test-agent"

	t.Run("UpdateMemoryScore updates score and timestamp", func(t *testing.T) {
		// Create a memory
		mem := &memory.AgentMemory{
			ID:             types.NewAgentMemoryID(),
			AgentID:        agentID,
			TaskQuery:      "test query",
			QueryEmbedding: []float32{0.1, 0.2, 0.3},
			Timestamp:      time.Now(),
			QualityScore:   0.0,
		}
		gt.NoError(t, repo.SaveAgentMemory(ctx, mem))

		// Update score
		newScore := 5.0
		now := time.Now()
		gt.NoError(t, repo.UpdateMemoryScore(ctx, agentID, mem.ID, newScore, now))

		// Verify update
		retrieved, err := repo.GetAgentMemory(ctx, agentID, mem.ID)
		gt.NoError(t, err)
		gt.Equal(t, retrieved.QualityScore, newScore)
		gt.True(t, retrieved.LastUsedAt.After(time.Time{}))
	})

	t.Run("DeleteAgentMemoriesBatch deletes multiple memories", func(t *testing.T) {
		// Create multiple memories
		ids := make([]types.AgentMemoryID, 3)
		for i := 0; i < 3; i++ {
			mem := &memory.AgentMemory{
				ID:             types.NewAgentMemoryID(),
				AgentID:        agentID,
				TaskQuery:      "batch delete test",
				QueryEmbedding: []float32{0.1, 0.2, 0.3},
				Timestamp:      time.Now(),
			}
			gt.NoError(t, repo.SaveAgentMemory(ctx, mem))
			ids[i] = mem.ID
		}

		// Delete in batch
		deleted, err := repo.DeleteAgentMemoriesBatch(ctx, agentID, ids)
		gt.NoError(t, err)
		gt.Equal(t, deleted, 3)

		// Verify deletion
		for _, id := range ids {
			_, err := repo.GetAgentMemory(ctx, agentID, id)
			gt.Error(t, err) // Should not exist
		}
	})

	t.Run("ListAgentMemories returns all memories for agent", func(t *testing.T) {
		// Clean up first
		existing, _ := repo.ListAgentMemories(ctx, agentID)
		if len(existing) > 0 {
			ids := make([]types.AgentMemoryID, len(existing))
			for i, m := range existing {
				ids[i] = m.ID
			}
			_, _ = repo.DeleteAgentMemoriesBatch(ctx, agentID, ids)
		}

		// Create memories with different timestamps
		times := []time.Time{
			time.Now().Add(-3 * time.Hour),
			time.Now().Add(-2 * time.Hour),
			time.Now().Add(-1 * time.Hour),
		}

		for _, ts := range times {
			mem := &memory.AgentMemory{
				ID:             types.NewAgentMemoryID(),
				AgentID:        agentID,
				TaskQuery:      "list test",
				QueryEmbedding: []float32{0.1, 0.2, 0.3},
				Timestamp:      ts,
			}
			gt.NoError(t, repo.SaveAgentMemory(ctx, mem))
		}

		// List all memories
		memories, err := repo.ListAgentMemories(ctx, agentID)
		gt.NoError(t, err)
		gt.Equal(t, len(memories), 3)

		// Verify ordering (should be DESC by Timestamp)
		for i := 0; i < len(memories)-1; i++ {
			gt.True(t, memories[i].Timestamp.After(memories[i+1].Timestamp) ||
				memories[i].Timestamp.Equal(memories[i+1].Timestamp))
		}
	})

	t.Run("Score-based filtering and ranking", func(t *testing.T) {
		// Clean up
		existing, _ := repo.ListAgentMemories(ctx, agentID)
		if len(existing) > 0 {
			ids := make([]types.AgentMemoryID, len(existing))
			for i, m := range existing {
				ids[i] = m.ID
			}
			_, _ = repo.DeleteAgentMemoriesBatch(ctx, agentID, ids)
		}

		// Create memories with different scores
		scores := []float64{-8.0, -3.0, 0.0, 5.0, 8.0}
		memIDs := make([]types.AgentMemoryID, len(scores))

		for i, score := range scores {
			mem := &memory.AgentMemory{
				ID:             types.NewAgentMemoryID(),
				AgentID:        agentID,
				TaskQuery:      "ranking test",
				QueryEmbedding: []float32{0.1, 0.2, 0.3},
				Timestamp:      time.Now(),
				QualityScore:   score,
			}
			gt.NoError(t, repo.SaveAgentMemory(ctx, mem))
			memIDs[i] = mem.ID
		}

		// List and verify scores
		memories, err := repo.ListAgentMemories(ctx, agentID)
		gt.NoError(t, err)
		gt.Equal(t, len(memories), 5)

		// Verify all scores are preserved
		foundScores := make(map[float64]bool)
		for _, mem := range memories {
			foundScores[mem.QualityScore] = true
		}
		for _, score := range scores {
			if !foundScores[score] {
				t.Errorf("score %f not found", score)
			}
		}
	})
}
