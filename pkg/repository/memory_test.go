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

func TestMemory(t *testing.T) {
	ctx := context.Background()

	testFn := func(t *testing.T, repo interfaces.Repository) {
		schemaID := types.AlertSchema(fmt.Sprintf("test-schema-%d", time.Now().UnixNano()))

		t.Run("ExecutionMemory round trip", func(t *testing.T) {
			mem := memory.NewExecutionMemory(schemaID)
			mem.Keep = "successful patterns"
			mem.Change = "areas to improve"
			mem.Notes = "other insights"
			// Generate random embedding
			emb := make([]float32, 256)
			for i := range emb {
				emb[i] = rand.Float32()
			}
			mem.Embedding = emb

			// Put
			err := repo.PutExecutionMemory(ctx, mem)
			gt.NoError(t, err)

			// Wait for Firestore to propagate (needed for emulator)
			time.Sleep(100 * time.Millisecond)

			// Get
			retrieved, err := repo.GetExecutionMemory(ctx, schemaID)
			gt.NoError(t, err)
			gt.V(t, retrieved).NotNil()
			gt.Equal(t, retrieved.SchemaID, schemaID)
			gt.S(t, retrieved.Keep).Equal("successful patterns")
			gt.S(t, retrieved.Change).Equal("areas to improve")
			gt.S(t, retrieved.Notes).Equal("other insights")
			gt.Equal(t, retrieved.Version, 1)
		})

		t.Run("ExecutionMemory get nonexistent", func(t *testing.T) {
			nonexistentID := types.AlertSchema(fmt.Sprintf("nonexistent-%d", time.Now().UnixNano()))

			retrieved, err := repo.GetExecutionMemory(ctx, nonexistentID)
			gt.NoError(t, err)
			gt.Nil(t, retrieved)
		})

		t.Run("ExecutionMemory update", func(t *testing.T) {
			mem1 := memory.NewExecutionMemory(schemaID)
			mem1.Keep = "initial keep"
			// Generate random embedding
			emb1 := make([]float32, 256)
			for i := range emb1 {
				emb1[i] = rand.Float32()
			}
			mem1.Embedding = emb1

			err := repo.PutExecutionMemory(ctx, mem1)
			gt.NoError(t, err)

			// Wait for Firestore to propagate
			time.Sleep(100 * time.Millisecond)

			// Update with new memory
			mem2 := memory.NewExecutionMemory(schemaID)
			mem2.Keep = "updated keep"
			mem2.Change = "new change"
			mem2.Version = 2
			// Generate random embedding
			emb2 := make([]float32, 256)
			for i := range emb2 {
				emb2[i] = rand.Float32()
			}
			mem2.Embedding = emb2

			err = repo.PutExecutionMemory(ctx, mem2)
			gt.NoError(t, err)

			// Wait for Firestore to propagate
			time.Sleep(100 * time.Millisecond)

			// Verify update
			retrieved, err := repo.GetExecutionMemory(ctx, schemaID)
			gt.NoError(t, err)
			gt.S(t, retrieved.Keep).Equal("updated keep")
			gt.S(t, retrieved.Change).Equal("new change")
			gt.Equal(t, retrieved.Version, 2)
		})

		t.Run("TicketMemory round trip", func(t *testing.T) {
			mem := memory.NewTicketMemory(schemaID)
			mem.Insights = "organizational knowledge"

			// Put
			err := repo.PutTicketMemory(ctx, mem)
			gt.NoError(t, err)

			// Wait for Firestore to propagate (needed for emulator)
			time.Sleep(100 * time.Millisecond)

			// Get
			retrieved, err := repo.GetTicketMemory(ctx, schemaID)
			gt.NoError(t, err)
			gt.V(t, retrieved).NotNil()
			gt.Equal(t, retrieved.SchemaID, schemaID)
			gt.S(t, retrieved.Insights).Equal("organizational knowledge")
			gt.Equal(t, retrieved.Version, 1)
		})

		t.Run("TicketMemory get nonexistent", func(t *testing.T) {
			nonexistentID := types.AlertSchema(fmt.Sprintf("nonexistent-%d", time.Now().UnixNano()))

			retrieved, err := repo.GetTicketMemory(ctx, nonexistentID)
			gt.NoError(t, err)
			gt.Nil(t, retrieved)
		})

		t.Run("TicketMemory update", func(t *testing.T) {
			mem1 := memory.NewTicketMemory(schemaID)
			mem1.Insights = "initial insights"

			err := repo.PutTicketMemory(ctx, mem1)
			gt.NoError(t, err)

			// Wait for Firestore to propagate
			time.Sleep(100 * time.Millisecond)

			// Update with new memory
			mem2 := memory.NewTicketMemory(schemaID)
			mem2.Insights = "updated insights"
			mem2.Version = 2

			err = repo.PutTicketMemory(ctx, mem2)
			gt.NoError(t, err)

			// Wait for Firestore to propagate
			time.Sleep(100 * time.Millisecond)

			// Verify update
			retrieved, err := repo.GetTicketMemory(ctx, schemaID)
			gt.NoError(t, err)
			gt.S(t, retrieved.Insights).Equal("updated insights")
			gt.Equal(t, retrieved.Version, 2)
		})

		t.Run("SearchExecutionMemoriesByEmbedding", func(t *testing.T) {
			// Create random embeddings
			embedding1 := make([]float32, 256)
			embedding2 := make([]float32, 256)
			embedding3 := make([]float32, 256)
			for i := range embedding1 {
				embedding1[i] = rand.Float32()
				embedding2[i] = rand.Float32()
				embedding3[i] = rand.Float32()
			}

			// Create memories with embeddings
			mem1 := memory.NewExecutionMemory(schemaID)
			mem1.Summary = "First execution summary"
			mem1.Keep = "successful pattern 1"
			mem1.Embedding = embedding1

			mem2 := memory.NewExecutionMemory(schemaID)
			mem2.Summary = "Second execution summary"
			mem2.Keep = "successful pattern 2"
			mem2.Embedding = embedding2

			mem3 := memory.NewExecutionMemory(schemaID)
			mem3.Summary = "Third execution summary"
			mem3.Keep = "successful pattern 3"
			mem3.Embedding = embedding3

			// Save all memories
			err := repo.PutExecutionMemory(ctx, mem1)
			gt.NoError(t, err)
			err = repo.PutExecutionMemory(ctx, mem2)
			gt.NoError(t, err)
			err = repo.PutExecutionMemory(ctx, mem3)
			gt.NoError(t, err)

			// Search using embedding1 (should find similar memories)
			results, err := repo.SearchExecutionMemoriesByEmbedding(ctx, schemaID, embedding1, 2)
			gt.NoError(t, err)
			gt.V(t, results).NotNil()
			gt.N(t, len(results)).GreaterOrEqual(1).Required() // At least mem1 should be found
			gt.N(t, len(results)).LessOrEqual(2)               // Limit is 2
			gt.V(t, results[0].ID).Equal(mem1.ID)
			gt.S(t, results[0].Summary).Equal("First execution summary")
			gt.S(t, results[0].Keep).Equal("successful pattern 1")
		})

		t.Run("SearchExecutionMemoriesByEmbedding no results", func(t *testing.T) {
			nonexistentSchema := types.AlertSchema(fmt.Sprintf("nonexistent-%d", time.Now().UnixNano()))
			embedding := make([]float32, 256)
			for i := range embedding {
				embedding[i] = rand.Float32()
			}

			results, err := repo.SearchExecutionMemoriesByEmbedding(ctx, nonexistentSchema, embedding, 5)
			gt.NoError(t, err)
			gt.N(t, len(results)).Equal(0)
		})

		t.Run("SearchExecutionMemoriesByEmbedding empty embedding", func(t *testing.T) {
			results, err := repo.SearchExecutionMemoriesByEmbedding(ctx, schemaID, []float32{}, 5)
			gt.NoError(t, err)
			gt.N(t, len(results)).Equal(0)
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
