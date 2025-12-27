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

func generateTestEmbedding(size int) []float32 {
	embedding := make([]float32, size)
	for i := range embedding {
		embedding[i] = rand.Float32()
	}
	return embedding
}

func TestListAllAgentIDs(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		now := time.Now()

		// Create unique agent IDs using timestamp
		baseTime := time.Now().UnixNano()
		bigqueryID := fmt.Sprintf("bigquery-%d", baseTime)
		slackID := fmt.Sprintf("slack-%d", baseTime)
		virustotalID := fmt.Sprintf("virustotal-%d", baseTime)

		// Create memories for different agents
		agentMemories := []*memory.AgentMemory{
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        bigqueryID,
				Query:          "test query 1",
				QueryEmbedding: generateTestEmbedding(256),
				Claim:          "test claim 1",
				Score:          1.0,
				CreatedAt:      now,
				LastUsedAt:     now,
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        bigqueryID,
				Query:          "test query 2",
				QueryEmbedding: generateTestEmbedding(256),
				Claim:          "test claim 2",
				Score:          2.0,
				CreatedAt:      now.Add(time.Minute),
				LastUsedAt:     now.Add(time.Minute),
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        slackID,
				Query:          "test query 3",
				QueryEmbedding: generateTestEmbedding(256),
				Claim:          "test claim 3",
				Score:          3.0,
				CreatedAt:      now.Add(2 * time.Minute),
				LastUsedAt:     now.Add(2 * time.Minute),
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        virustotalID,
				Query:          "test query 4",
				QueryEmbedding: generateTestEmbedding(256),
				Claim:          "test claim 4",
				Score:          4.0,
				CreatedAt:      now.Add(3 * time.Minute),
				LastUsedAt:     now.Add(3 * time.Minute),
			},
		}

		// Save all memories
		err := repo.BatchSaveAgentMemories(ctx, agentMemories)
		gt.NoError(t, err)

		// Get all agent IDs with counts
		result, err := repo.ListAllAgentIDs(ctx)
		gt.NoError(t, err)

		// Verify counts for our test agents
		gt.Equal(t, result[bigqueryID], 2)
		gt.Equal(t, result[slackID], 1)
		gt.Equal(t, result[virustotalID], 1)
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

func TestListAllAgentIDs_Empty(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()

		// Get all agent IDs when no memories exist
		result, err := repo.ListAllAgentIDs(ctx)
		gt.NoError(t, err)
		gt.V(t, len(result)).Equal(0)
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
