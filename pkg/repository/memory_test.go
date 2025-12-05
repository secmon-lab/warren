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

// TestMemory is removed - ExecutionMemory and TicketMemory features have been deleted

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

			mem := &memory.AgentMemory{
				ID:             memID,
				AgentID:        agentID,
				TaskQuery:      "SELECT * FROM table WHERE id = 1",
				QueryEmbedding: embedding,
				Successes:      []string{"Query executed successfully", "Retrieved 1 row"},
				Problems:       []string{},
				Improvements:   []string{"Consider adding index on id field"},
				Timestamp:      time.Now(),
				Duration:       100 * time.Millisecond,
			}

			// Save memory
			err := repo.SaveAgentMemory(ctx, mem)
			gt.NoError(t, err)

			// Wait for Firestore to propagate
			time.Sleep(100 * time.Millisecond)

			// Get memory
			retrieved, err := repo.GetAgentMemory(ctx, agentID, memID)
			gt.NoError(t, err)
			gt.NotNil(t, retrieved)
			gt.V(t, retrieved.ID).Equal(memID)
			gt.V(t, retrieved.AgentID).Equal(agentID)
			gt.V(t, retrieved.TaskQuery).Equal("SELECT * FROM table WHERE id = 1")
			gt.V(t, len(retrieved.QueryEmbedding)).Equal(256)
			gt.A(t, retrieved.Successes).Length(2)
			gt.V(t, retrieved.Successes[0]).Equal("Query executed successfully")
			gt.A(t, retrieved.Problems).Length(0)
			gt.A(t, retrieved.Improvements).Length(1)
		})

		t.Run("SearchMemoriesByEmbedding", func(t *testing.T) {
			agentID := fmt.Sprintf("agent-%d", time.Now().UnixNano())

			// Create base embedding
			baseEmbedding := make([]float32, 256)
			for i := range baseEmbedding {
				baseEmbedding[i] = rand.Float32()
			}

			// Save multiple memories with similar embeddings
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
					TaskQuery:      fmt.Sprintf("SELECT * FROM table WHERE id = %d", i+1),
					QueryEmbedding: embedding,
					Successes:      []string{fmt.Sprintf("Query %d executed", i+1)},
					Problems:       []string{},
					Improvements:   []string{},
					Timestamp:      time.Now().Add(time.Duration(i) * time.Second),
					Duration:       100 * time.Millisecond,
				}

				err := repo.SaveAgentMemory(ctx, mem)
				gt.NoError(t, err)
			}

			// Wait for Firestore to propagate
			time.Sleep(200 * time.Millisecond)

			// Search memories
			results, err := repo.SearchMemoriesByEmbedding(ctx, agentID, baseEmbedding, 2)

			gt.NoError(t, err)
			gt.Number(t, len(results)).GreaterOrEqual(1)
			gt.Number(t, len(results)).LessOrEqual(2)

			// Verify all results belong to the correct agent
			for _, result := range results {
				gt.V(t, result.AgentID).Equal(agentID)
			}
		})

		t.Run("AgentMemoryIsolation", func(t *testing.T) {
			agent1ID := fmt.Sprintf("agent1-%d", time.Now().UnixNano())
			agent2ID := fmt.Sprintf("agent2-%d", time.Now().UnixNano())

			embedding := make([]float32, 256)
			for i := range embedding {
				embedding[i] = rand.Float32()
			}

			// Save memory for agent1
			mem1 := &memory.AgentMemory{
				ID:             types.NewAgentMemoryID(),
				AgentID:        agent1ID,
				TaskQuery:      "Agent 1 query",
				QueryEmbedding: embedding,
				Successes:      []string{"Agent 1 result"},
				Problems:       []string{},
				Improvements:   []string{},
				Timestamp:      time.Now(),
				Duration:       100 * time.Millisecond,
			}
			err := repo.SaveAgentMemory(ctx, mem1)
			gt.NoError(t, err)

			// Save memory for agent2
			mem2 := &memory.AgentMemory{
				ID:             types.NewAgentMemoryID(),
				AgentID:        agent2ID,
				TaskQuery:      "Agent 2 query",
				QueryEmbedding: embedding,
				Successes:      []string{"Agent 2 result"},
				Problems:       []string{},
				Improvements:   []string{},
				Timestamp:      time.Now(),
				Duration:       100 * time.Millisecond,
			}
			err = repo.SaveAgentMemory(ctx, mem2)
			gt.NoError(t, err)

			// Wait for Firestore to propagate
			time.Sleep(100 * time.Millisecond)

			// Search for agent1's memories
			results1, err := repo.SearchMemoriesByEmbedding(ctx, agent1ID, embedding, 10)
			gt.NoError(t, err)

			// Verify only agent1's memories are returned
			for _, result := range results1 {
				gt.V(t, result.AgentID).Equal(agent1ID)
			}

			// Search for agent2's memories
			results2, err := repo.SearchMemoriesByEmbedding(ctx, agent2ID, embedding, 10)
			gt.NoError(t, err)

			// Verify only agent2's memories are returned
			for _, result := range results2 {
				gt.V(t, result.AgentID).Equal(agent2ID)
			}
		})

		t.Run("GetNonExistentAgentMemory", func(t *testing.T) {
			agentID := fmt.Sprintf("agent-%d", time.Now().UnixNano())
			memID := types.NewAgentMemoryID()

			// Try to get non-existent memory
			retrieved, err := repo.GetAgentMemory(ctx, agentID, memID)
			gt.Error(t, err)
			gt.Nil(t, retrieved)
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
