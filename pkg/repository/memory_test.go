package repository_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

// TestAgentMemory tests the new AgentMemory model with claim-based structure
func TestAgentMemory(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()

		t.Run("SaveAndGetAgentMemory", func(t *testing.T) {
			agentID := fmt.Sprintf("agent-%d", time.Now().UnixNano())
			memID := types.NewAgentMemoryID()

			// Create test embedding
			embedding := make([]float32, 256)
			for i := range embedding {
				embedding[i] = rand.Float32()
			}

			now := time.Now()
			mem := &memory.AgentMemory{
				ID:             memID,
				AgentID:        agentID,
				Query:          "Find login failures in BigQuery logs",
				QueryEmbedding: embedding,
				Claim:          "Login failures are identified by severity='ERROR' AND action='login'",
				Score:          2.5,
				CreatedAt:      now,
				LastUsedAt:     now.Add(1 * time.Hour),
			}

			// Save memory
			err := repo.SaveAgentMemory(ctx, mem)
			gt.NoError(t, err)

			// Get memory
			retrieved, err := repo.GetAgentMemory(ctx, agentID, memID)
			gt.NoError(t, err)
			gt.NotNil(t, retrieved)
			gt.V(t, retrieved.ID).Equal(memID)
			gt.V(t, retrieved.AgentID).Equal(agentID)
			gt.V(t, retrieved.Query).Equal("Find login failures in BigQuery logs")
			gt.V(t, retrieved.Claim).Equal("Login failures are identified by severity='ERROR' AND action='login'")
			gt.V(t, retrieved.Score).Equal(2.5)
			gt.V(t, len(retrieved.QueryEmbedding)).Equal(256)
			// Check timestamps with tolerance
			gt.True(t, retrieved.CreatedAt.Sub(now) < time.Second)
			gt.True(t, retrieved.LastUsedAt.Sub(now.Add(1*time.Hour)) < time.Second)
		})

		t.Run("BatchSaveAgentMemories", func(t *testing.T) {
			agentID := fmt.Sprintf("agent-%d", time.Now().UnixNano())

			// Create base embedding
			baseEmbedding := make([]float32, 256)
			for i := range baseEmbedding {
				baseEmbedding[i] = rand.Float32()
			}

			// Create multiple memories
			memories := make([]*memory.AgentMemory, 3)
			now := time.Now()
			for i := 0; i < 3; i++ {
				// Create slightly different embedding
				embedding := make([]float32, 256)
				copy(embedding, baseEmbedding)
				for j := 0; j < 10; j++ {
					embedding[j] += float32(i) * 0.01
				}

				memories[i] = &memory.AgentMemory{
					ID:             types.NewAgentMemoryID(),
					AgentID:        agentID,
					Query:          fmt.Sprintf("Query pattern %d", i+1),
					QueryEmbedding: embedding,
					Claim:          fmt.Sprintf("Claim learned from execution %d", i+1),
					Score:          float64(i),
					CreatedAt:      now.Add(time.Duration(i) * time.Second),
					LastUsedAt:     time.Time{}, // Never used
				}
			}

			// Batch save memories
			err := repo.BatchSaveAgentMemories(ctx, memories)
			gt.NoError(t, err)

			// Verify all memories were saved
			for i, mem := range memories {
				retrieved, err := repo.GetAgentMemory(ctx, agentID, mem.ID)
				gt.NoError(t, err)
				gt.NotNil(t, retrieved)
				gt.V(t, retrieved.ID).Equal(mem.ID)
				gt.V(t, retrieved.AgentID).Equal(agentID)
				gt.V(t, retrieved.Query).Equal(fmt.Sprintf("Query pattern %d", i+1))
				gt.V(t, retrieved.Claim).Equal(fmt.Sprintf("Claim learned from execution %d", i+1))
				gt.V(t, retrieved.Score).Equal(float64(i))
			}
		})

		t.Run("SearchMemoriesByEmbedding", func(t *testing.T) {
			agentID := fmt.Sprintf("agent-%d", time.Now().UnixNano())

			// Create base embedding
			baseEmbedding := make([]float32, 256)
			for i := range baseEmbedding {
				baseEmbedding[i] = rand.Float32()
			}

			// Save multiple memories with similar embeddings
			now := time.Now()
			for i := 0; i < 3; i++ {
				// Create slightly different embedding
				embedding := make([]float32, 256)
				copy(embedding, baseEmbedding)
				for j := 0; j < 10; j++ {
					embedding[j] += float32(i) * 0.01
				}

				mem := &memory.AgentMemory{
					ID:             types.NewAgentMemoryID(),
					AgentID:        agentID,
					Query:          fmt.Sprintf("SELECT * FROM table WHERE id = %d", i+1),
					QueryEmbedding: embedding,
					Claim:          fmt.Sprintf("Query %d optimization tip", i+1),
					Score:          float64(i) - 1.0, // -1.0, 0.0, 1.0
					CreatedAt:      now.Add(time.Duration(i) * time.Second),
					LastUsedAt:     time.Time{}, // Never used
				}

				err := repo.SaveAgentMemory(ctx, mem)
				gt.NoError(t, err)
			}

			// Search memories
			results, err := repo.SearchMemoriesByEmbedding(ctx, agentID, baseEmbedding, 2)

			gt.NoError(t, err)
			gt.Number(t, len(results)).GreaterOrEqual(1)
			gt.Number(t, len(results)).LessOrEqual(2)

			// Verify all results belong to the correct agent
			for _, result := range results {
				gt.V(t, result.AgentID).Equal(agentID)
				gt.NotNil(t, result.Claim)
				gt.True(t, result.Score >= -1.0 && result.Score <= 1.0)
			}
		})

		t.Run("UpdateMemoryScoreBatch", func(t *testing.T) {
			agentID := fmt.Sprintf("agent-%d", time.Now().UnixNano())

			// Create test embedding
			embedding := make([]float32, 256)
			for i := range embedding {
				embedding[i] = rand.Float32()
			}

			// Create and save memories
			mem1ID := types.NewAgentMemoryID()
			mem2ID := types.NewAgentMemoryID()
			now := time.Now()

			mem1 := &memory.AgentMemory{
				ID:             mem1ID,
				AgentID:        agentID,
				Query:          "Query 1",
				QueryEmbedding: embedding,
				Claim:          "Claim 1",
				Score:          0.0, // Initial score
				CreatedAt:      now,
				LastUsedAt:     time.Time{},
			}

			mem2 := &memory.AgentMemory{
				ID:             mem2ID,
				AgentID:        agentID,
				Query:          "Query 2",
				QueryEmbedding: embedding,
				Claim:          "Claim 2",
				Score:          0.0, // Initial score
				CreatedAt:      now,
				LastUsedAt:     time.Time{},
			}

			err := repo.SaveAgentMemory(ctx, mem1)
			gt.NoError(t, err)
			err = repo.SaveAgentMemory(ctx, mem2)
			gt.NoError(t, err)

			// Update scores in batch
			newTime := now.Add(2 * time.Hour)
			updates := map[types.AgentMemoryID]struct {
				Score      float64
				LastUsedAt time.Time
			}{
				mem1ID: {Score: 5.0, LastUsedAt: newTime},
				mem2ID: {Score: -3.0, LastUsedAt: newTime},
			}

			err = repo.UpdateMemoryScoreBatch(ctx, agentID, updates)
			gt.NoError(t, err)

			// Verify updates
			updated1, err := repo.GetAgentMemory(ctx, agentID, mem1ID)
			gt.NoError(t, err)
			gt.V(t, updated1.Score).Equal(5.0)
			gt.True(t, updated1.LastUsedAt.Sub(newTime) < time.Second)

			updated2, err := repo.GetAgentMemory(ctx, agentID, mem2ID)
			gt.NoError(t, err)
			gt.V(t, updated2.Score).Equal(-3.0)
			gt.True(t, updated2.LastUsedAt.Sub(newTime) < time.Second)
		})

		t.Run("DeleteAgentMemoriesBatch", func(t *testing.T) {
			agentID := fmt.Sprintf("agent-%d", time.Now().UnixNano())

			// Create test embedding
			embedding := make([]float32, 256)
			for i := range embedding {
				embedding[i] = rand.Float32()
			}

			// Create and save memories
			mem1ID := types.NewAgentMemoryID()
			mem2ID := types.NewAgentMemoryID()
			mem3ID := types.NewAgentMemoryID()
			now := time.Now()

			for i, id := range []types.AgentMemoryID{mem1ID, mem2ID, mem3ID} {
				mem := &memory.AgentMemory{
					ID:             id,
					AgentID:        agentID,
					Query:          fmt.Sprintf("Query %d", i+1),
					QueryEmbedding: embedding,
					Claim:          fmt.Sprintf("Claim %d", i+1),
					Score:          0.0,
					CreatedAt:      now,
					LastUsedAt:     time.Time{},
				}
				err := repo.SaveAgentMemory(ctx, mem)
				gt.NoError(t, err)
			}

			// Delete two memories in batch
			deleteCount, err := repo.DeleteAgentMemoriesBatch(ctx, agentID, []types.AgentMemoryID{mem1ID, mem2ID})
			gt.NoError(t, err)
			gt.V(t, deleteCount).Equal(2)

			// Verify deletion
			_, err = repo.GetAgentMemory(ctx, agentID, mem1ID)
			gt.Error(t, err) // Should be deleted

			_, err = repo.GetAgentMemory(ctx, agentID, mem2ID)
			gt.Error(t, err) // Should be deleted

			// Verify mem3 still exists
			mem3, err := repo.GetAgentMemory(ctx, agentID, mem3ID)
			gt.NoError(t, err)
			gt.NotNil(t, mem3)
			gt.V(t, mem3.ID).Equal(mem3ID)
		})

		t.Run("ListAgentMemories", func(t *testing.T) {
			agentID := fmt.Sprintf("agent-%d", time.Now().UnixNano())

			// Create test embedding
			embedding := make([]float32, 256)
			for i := range embedding {
				embedding[i] = rand.Float32()
			}

			// Create and save memories with different creation times
			now := time.Now()
			memIDs := make([]types.AgentMemoryID, 3)
			for i := 0; i < 3; i++ {
				memIDs[i] = types.NewAgentMemoryID()
				mem := &memory.AgentMemory{
					ID:             memIDs[i],
					AgentID:        agentID,
					Query:          fmt.Sprintf("Query %d", i+1),
					QueryEmbedding: embedding,
					Claim:          fmt.Sprintf("Claim %d", i+1),
					Score:          float64(i),
					CreatedAt:      now.Add(time.Duration(i) * time.Second),
					LastUsedAt:     time.Time{},
				}
				err := repo.SaveAgentMemory(ctx, mem)
				gt.NoError(t, err)
			}

			// List all memories for the agent
			memories, err := repo.ListAgentMemories(ctx, agentID)
			gt.NoError(t, err)
			gt.V(t, len(memories)).Equal(3)

			// Verify order (should be DESC by CreatedAt)
			// Most recent first
			for i := 0; i < len(memories)-1; i++ {
				gt.True(t, memories[i].CreatedAt.After(memories[i+1].CreatedAt) ||
					memories[i].CreatedAt.Equal(memories[i+1].CreatedAt))
			}

			// Verify all memories belong to the correct agent
			for _, mem := range memories {
				gt.V(t, mem.AgentID).Equal(agentID)
			}
		})

		t.Run("AgentMemoryIsolation", func(t *testing.T) {
			agent1ID := fmt.Sprintf("agent1-%d", time.Now().UnixNano())
			agent2ID := fmt.Sprintf("agent2-%d", time.Now().UnixNano())

			embedding := make([]float32, 256)
			for i := range embedding {
				embedding[i] = rand.Float32()
			}

			now := time.Now()

			// Save memory for agent1
			mem1 := &memory.AgentMemory{
				ID:             types.NewAgentMemoryID(),
				AgentID:        agent1ID,
				Query:          "Agent 1 query",
				QueryEmbedding: embedding,
				Claim:          "Agent 1 claim",
				Score:          1.0,
				CreatedAt:      now,
				LastUsedAt:     time.Time{},
			}
			err := repo.SaveAgentMemory(ctx, mem1)
			gt.NoError(t, err)

			// Save memory for agent2
			mem2 := &memory.AgentMemory{
				ID:             types.NewAgentMemoryID(),
				AgentID:        agent2ID,
				Query:          "Agent 2 query",
				QueryEmbedding: embedding,
				Claim:          "Agent 2 claim",
				Score:          2.0,
				CreatedAt:      now,
				LastUsedAt:     time.Time{},
			}
			err = repo.SaveAgentMemory(ctx, mem2)
			gt.NoError(t, err)

			// Search for agent1's memories
			results1, err := repo.SearchMemoriesByEmbedding(ctx, agent1ID, embedding, 10)
			gt.NoError(t, err)

			// Verify only agent1's memories are returned
			for _, result := range results1 {
				gt.V(t, result.AgentID).Equal(agent1ID)
				gt.V(t, result.Claim).Equal("Agent 1 claim")
			}

			// Search for agent2's memories
			results2, err := repo.SearchMemoriesByEmbedding(ctx, agent2ID, embedding, 10)
			gt.NoError(t, err)

			// Verify only agent2's memories are returned
			for _, result := range results2 {
				gt.V(t, result.AgentID).Equal(agent2ID)
				gt.V(t, result.Claim).Equal("Agent 2 claim")
			}

			// List memories for each agent
			list1, err := repo.ListAgentMemories(ctx, agent1ID)
			gt.NoError(t, err)
			gt.V(t, len(list1)).Equal(1)
			gt.V(t, list1[0].AgentID).Equal(agent1ID)

			list2, err := repo.ListAgentMemories(ctx, agent2ID)
			gt.NoError(t, err)
			gt.V(t, len(list2)).Equal(1)
			gt.V(t, list2[0].AgentID).Equal(agent2ID)
		})

		t.Run("GetNonExistentAgentMemory", func(t *testing.T) {
			agentID := fmt.Sprintf("agent-%d", time.Now().UnixNano())
			memID := types.NewAgentMemoryID()

			// Try to get non-existent memory
			retrieved, err := repo.GetAgentMemory(ctx, agentID, memID)
			gt.Error(t, err)
			gt.Nil(t, retrieved)
		})

		t.Run("ScoreValidation", func(t *testing.T) {
			agentID := fmt.Sprintf("agent-%d", time.Now().UnixNano())

			embedding := make([]float32, 256)
			for i := range embedding {
				embedding[i] = rand.Float32()
			}

			now := time.Now()

			// Test valid score range
			testCases := []struct {
				score      float64
				shouldFail bool
			}{
				{-10.0, false},
				{-5.0, false},
				{0.0, false},
				{5.0, false},
				{10.0, false},
				{-10.1, true},  // Below min
				{10.1, true},   // Above max
				{-100.0, true}, // Far below min
				{100.0, true},  // Far above max
			}

			for _, tc := range testCases {
				mem := &memory.AgentMemory{
					ID:             types.NewAgentMemoryID(),
					AgentID:        agentID,
					Query:          "Test query",
					QueryEmbedding: embedding,
					Claim:          "Test claim",
					Score:          tc.score,
					CreatedAt:      now,
					LastUsedAt:     time.Time{},
				}

				err := repo.SaveAgentMemory(ctx, mem)
				if tc.shouldFail {
					gt.Error(t, err)
				} else {
					gt.NoError(t, err)
				}
			}
		})
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("Firestore", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}
