package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestListAgentMemoriesWithOptions(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()
		now := time.Now()
		baseTime := time.Now().UnixNano()
		agentID := fmt.Sprintf("test-agent-%d", baseTime)

		// Create test memories with different scores and timestamps
		memories := []*memory.AgentMemory{
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        agentID,
				Query:          "find database errors",
				QueryEmbedding: generateTestEmbedding(256),
				Claim:          "Database connection timeout",
				Score:          5.0,
				CreatedAt:      now.Add(-5 * time.Hour),
				LastUsedAt:     now.Add(-1 * time.Hour),
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        agentID,
				Query:          "search network issues",
				QueryEmbedding: generateTestEmbedding(256),
				Claim:          "Network latency detected",
				Score:          3.0,
				CreatedAt:      now.Add(-3 * time.Hour),
				LastUsedAt:     now.Add(-2 * time.Hour),
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        agentID,
				Query:          "check authentication",
				QueryEmbedding: generateTestEmbedding(256),
				Claim:          "Authentication failed",
				Score:          -2.0,
				CreatedAt:      now.Add(-1 * time.Hour),
				LastUsedAt:     now.Add(-30 * time.Minute),
			},
			{
				ID:             types.NewAgentMemoryID(),
				AgentID:        agentID,
				Query:          "find security alerts",
				QueryEmbedding: generateTestEmbedding(256),
				Claim:          "Security breach detected",
				Score:          8.0,
				CreatedAt:      now.Add(-2 * time.Hour),
				LastUsedAt:     now.Add(-15 * time.Minute),
			},
		}

		err := repo.BatchSaveAgentMemories(ctx, memories)
		gt.NoError(t, err)

		t.Run("Filter by keyword", func(t *testing.T) {
			keyword := "database"
			opts := interfaces.AgentMemoryListOptions{
				Offset:  0,
				Limit:   10,
				Keyword: &keyword,
			}

			result, totalCount, err := repo.ListAgentMemoriesWithOptions(ctx, agentID, opts)
			gt.NoError(t, err)
			gt.Equal(t, totalCount, 1)
			gt.Equal(t, len(result), 1)
			gt.Equal(t, result[0].Query, "find database errors")
		})

		t.Run("Filter by score range", func(t *testing.T) {
			minScore := 3.0
			maxScore := 6.0
			opts := interfaces.AgentMemoryListOptions{
				Offset:   0,
				Limit:    10,
				MinScore: &minScore,
				MaxScore: &maxScore,
			}

			result, totalCount, err := repo.ListAgentMemoriesWithOptions(ctx, agentID, opts)
			gt.NoError(t, err)
			gt.Equal(t, totalCount, 2) // Score 3.0 and 5.0
			gt.Equal(t, len(result), 2)
		})

		t.Run("Sort by score descending", func(t *testing.T) {
			opts := interfaces.AgentMemoryListOptions{
				Offset:   0,
				Limit:    10,
				SortBy:   "score",
				SortDesc: true,
			}

			result, totalCount, err := repo.ListAgentMemoriesWithOptions(ctx, agentID, opts)
			gt.NoError(t, err)
			gt.Equal(t, totalCount, 4)
			gt.Equal(t, len(result), 4)
			// First should be highest score (8.0)
			gt.Equal(t, result[0].Score, 8.0)
			// Last should be lowest score (-2.0)
			gt.Equal(t, result[3].Score, -2.0)
		})

		t.Run("Sort by created_at ascending", func(t *testing.T) {
			opts := interfaces.AgentMemoryListOptions{
				Offset:   0,
				Limit:    10,
				SortBy:   "created_at",
				SortDesc: false,
			}

			result, totalCount, err := repo.ListAgentMemoriesWithOptions(ctx, agentID, opts)
			gt.NoError(t, err)
			gt.Equal(t, totalCount, 4)
			gt.Equal(t, len(result), 4)
			// Should be ordered oldest to newest
			for i := 0; i < len(result)-1; i++ {
				gt.True(t, result[i].CreatedAt.Before(result[i+1].CreatedAt) || result[i].CreatedAt.Equal(result[i+1].CreatedAt))
			}
		})

		t.Run("Pagination", func(t *testing.T) {
			// Get first 2 items
			opts := interfaces.AgentMemoryListOptions{
				Offset:   0,
				Limit:    2,
				SortBy:   "score",
				SortDesc: true,
			}

			result1, totalCount, err := repo.ListAgentMemoriesWithOptions(ctx, agentID, opts)
			gt.NoError(t, err)
			gt.Equal(t, totalCount, 4)
			gt.Equal(t, len(result1), 2)

			// Get next 2 items
			opts.Offset = 2
			result2, totalCount2, err := repo.ListAgentMemoriesWithOptions(ctx, agentID, opts)
			gt.NoError(t, err)
			gt.Equal(t, totalCount2, 4)
			gt.Equal(t, len(result2), 2)

			// Should be different items
			gt.NotEqual(t, result1[0].ID, result2[0].ID)
			gt.NotEqual(t, result1[1].ID, result2[1].ID)
		})

		t.Run("Combined filters", func(t *testing.T) {
			keyword := "security"
			minScore := 7.0
			opts := interfaces.AgentMemoryListOptions{
				Offset:   0,
				Limit:    10,
				Keyword:  &keyword,
				MinScore: &minScore,
				SortBy:   "score",
				SortDesc: true,
			}

			result, totalCount, err := repo.ListAgentMemoriesWithOptions(ctx, agentID, opts)
			gt.NoError(t, err)
			gt.Equal(t, totalCount, 1)
			gt.Equal(t, len(result), 1)
			gt.Equal(t, result[0].Score, 8.0)
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
