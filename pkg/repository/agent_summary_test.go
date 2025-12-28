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

		// Convert to map for easier verification
		resultMap := make(map[string]*interfaces.AgentSummary)
		for _, summary := range result {
			resultMap[summary.AgentID] = summary
		}

		// Verify counts for our test agents
		gt.V(t, resultMap[bigqueryID]).NotNil()
		gt.Equal(t, resultMap[bigqueryID].Count, 2)
		gt.V(t, resultMap[slackID]).NotNil()
		gt.Equal(t, resultMap[slackID].Count, 1)
		gt.V(t, resultMap[virustotalID]).NotNil()
		gt.Equal(t, resultMap[virustotalID].Count, 1)

		// Verify latest timestamps are set
		gt.False(t, resultMap[bigqueryID].LatestMemoryAt.IsZero())
		gt.False(t, resultMap[slackID].LatestMemoryAt.IsZero())
		gt.False(t, resultMap[virustotalID].LatestMemoryAt.IsZero())
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
	t.Run("Memory", func(t *testing.T) {
		ctx := context.Background()
		repo := repository.NewMemory()

		// Get all agent IDs when no memories exist (Memory implementation starts empty)
		result, err := repo.ListAllAgentIDs(ctx)
		gt.NoError(t, err)
		gt.Equal(t, len(result), 0)
	})

	t.Run("Firestore", func(t *testing.T) {
		ctx := context.Background()
		repo := newFirestoreClient(t)

		// For Firestore, just verify the method works (may have existing data)
		result, err := repo.ListAllAgentIDs(ctx)
		gt.NoError(t, err)
		// Firestore may have existing data from other tests or production use
		// Just verify it returns a valid map (not nil)
		gt.V(t, result).NotNil()
	})
}
