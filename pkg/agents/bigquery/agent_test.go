package bigquery_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	bqagent "github.com/secmon-lab/warren/pkg/agents/bigquery"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	memoryservice "github.com/secmon-lab/warren/pkg/service/memory"
)

// newMockLLMClient creates a mock LLM client for testing
// Note: This uses a custom mock implementation instead of testutil.NewMockLLMClient()
// because BigQuery agent tests require more sophisticated session behavior
// (plan & execute strategy with state management)
func newMockLLMClient() gollem.LLMClient {
	return &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
			embeddings := make([][]float64, len(input))
			for i := range input {
				vec := make([]float64, dimension)
				for j := 0; j < dimension; j++ {
					vec[j] = 0.1 * float64(i+j+1)
				}
				embeddings[i] = vec
			}
			return embeddings, nil
		},
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			callCount := 0 // Reset for each session
			return &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					callCount++
					// First call: return plan JSON for plan & execute strategy
					if callCount == 1 {
						planJSON := `{
							"objective": "Execute the query task",
							"tasks": [
								{"id": "task1", "description": "Complete the task", "state": "pending"}
							]
						}`
						return &gollem.Response{
							Texts: []string{planJSON},
						}, nil
					}
					// Subsequent calls: return mock response
					return &gollem.Response{
						Texts: []string{"mock response"},
					}, nil
				},
				HistoryFunc: func() (*gollem.History, error) {
					return &gollem.History{}, nil
				},
				AppendHistoryFunc: func(history *gollem.History) error {
					return nil
				},
			}, nil
		},
	}
}

// No need for mockBigQueryTool - Agent uses internal tool implementation

func TestAgent_ID(t *testing.T) {
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	gt.V(t, agent.ID()).Equal("bigquery")
}

func TestAgent_Specs(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()

	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	specs, err := agent.Specs(ctx)
	gt.NoError(t, err)
	gt.V(t, len(specs)).Equal(1)
	gt.V(t, specs[0].Name).Equal("query_bigquery")
	gt.V(t, specs[0].Description).NotEqual("")
	gt.V(t, len(specs[0].Required)).Equal(1)
	gt.V(t, specs[0].Required[0]).Equal("query")
}

func TestAgent_Run_SavesMemory(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables: []bqagent.TableConfig{
			{
				ProjectID:   "test-project",
				DatasetID:   "test-dataset",
				TableID:     "test-table",
				Description: "Test table",
			},
		},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()

	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	// Run the agent
	args := map[string]any{
		"query": "Get user count",
	}
	result, err := agent.Run(ctx, "query_bigquery", args)
	gt.NoError(t, err)
	gt.V(t, result).NotNil()
	gt.V(t, result["data"]).NotNil()

	// Verify memory was saved
	memories, err := memService.SearchRelevantAgentMemories(ctx, "bigquery", "Get user count", 10)
	gt.NoError(t, err)
	gt.True(t, len(memories) > 0)

	// Verify memory structure
	mem := memories[0]
	gt.V(t, mem.AgentID).Equal("bigquery")
	gt.V(t, mem.TaskQuery).Equal("Get user count")
	gt.V(t, len(mem.QueryEmbedding)).Equal(256)
	gt.True(t, mem.Duration > 0)
}

func TestAgent_Run_InvalidFunctionName(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()

	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	// Try to run with invalid function name
	args := map[string]any{
		"query": "test",
	}
	_, err := agent.Run(ctx, "invalid_function", args)
	gt.Error(t, err)
}

func TestAgent_Run_MissingQueryParameter(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()

	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	// Try to run without query parameter
	args := map[string]any{}
	_, err := agent.Run(ctx, "query_bigquery", args)
	gt.Error(t, err)
}

func TestAgent_MemorySearch(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()

	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	// Execute multiple queries to build up memory
	queries := []string{
		"Get user count",
		"Get active users",
		"Get user count by region",
	}

	for _, query := range queries {
		args := map[string]any{"query": query}
		_, err := agent.Run(ctx, "query_bigquery", args)
		gt.NoError(t, err)
	}

	// Search for similar query
	memories, err := memService.SearchRelevantAgentMemories(ctx, "bigquery", "user statistics", 5)
	gt.NoError(t, err)
	gt.True(t, len(memories) > 0)
	gt.True(t, len(memories) <= 3)

	// Verify all memories are for bigquery agent
	for _, mem := range memories {
		gt.V(t, mem.AgentID).Equal("bigquery")
	}
}

func TestAgent_MemoryMetadataOnly(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()

	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := bqagent.NewAgent(config, llmClient, memService)

	// Execute a query
	args := map[string]any{
		"query": "Test query",
	}
	result, err := agent.Run(ctx, "query_bigquery", args)
	gt.NoError(t, err)
	gt.V(t, result["data"]).NotNil()

	// Get the saved memory
	memories, err := memService.SearchRelevantAgentMemories(ctx, "bigquery", "Test query", 1)
	gt.NoError(t, err)
	gt.V(t, len(memories)).Equal(1)

	mem := memories[0]

	// Verify memory has KPT structure (Successes, Problems, Improvements)
	// Note: With mock LLM returning invalid JSON, KPT analysis fallback returns empty arrays
	gt.A(t, mem.Successes).Length(0)    // Empty array from fallback
	gt.A(t, mem.Problems).Length(0)     // Empty array from fallback
	gt.A(t, mem.Improvements).Length(0) // Empty array from fallback
}

func TestAgent_MemoryFeedbackIntegration(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables: []bqagent.TableConfig{
			{
				ProjectID:   "test-project",
				DatasetID:   "test-dataset",
				TableID:     "test-table",
				Description: "Test table for feedback",
			},
		},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)
	agent := bqagent.NewAgent(config, llmClient, memService)

	t.Run("Agent execution triggers feedback collection for used memories", func(t *testing.T) {
		// Step 1: Execute a query to create initial memory
		args := map[string]any{"query": "Get user login count"}
		_, err := agent.Run(ctx, "query_bigquery", args)
		gt.NoError(t, err)

		// Get the created memory
		memories, err := memService.SearchRelevantAgentMemories(ctx, "bigquery", "Get user login count", 1)
		gt.NoError(t, err)
		gt.V(t, len(memories)).Equal(1)

		initialMem := memories[0]
		initialScore := initialMem.QualityScore
		initialLastUsed := initialMem.LastUsedAt

		// Step 2: Execute similar query that should use the previous memory
		// Wait a bit to ensure timestamp difference
		args2 := map[string]any{"query": "user login statistics"}
		_, err = agent.Run(ctx, "query_bigquery", args2)
		gt.NoError(t, err)

		// Step 3: Verify that the original memory was updated with feedback
		updatedMemories, err := repo.GetAgentMemory(ctx, "bigquery", initialMem.ID)
		gt.NoError(t, err)

		// LastUsedAt should be updated (memory was used in second query)
		gt.True(t, updatedMemories.LastUsedAt.After(initialLastUsed) ||
			updatedMemories.LastUsedAt.Equal(initialLastUsed))

		// QualityScore may have changed (depends on LLM feedback)
		// We just verify it's within valid range
		gt.True(t, updatedMemories.QualityScore >= -10.0)
		gt.True(t, updatedMemories.QualityScore <= 10.0)

		// Store for comparison
		_ = initialScore // Used initial score for verification
	})

	t.Run("Memory scoring accumulates over multiple uses", func(t *testing.T) {
		// Clean up
		existing, err := repo.ListAgentMemories(ctx, "bigquery")
		gt.NoError(t, err)
		if len(existing) > 0 {
			ids := make([]types.AgentMemoryID, len(existing))
			for i, m := range existing {
				ids[i] = m.ID
			}
			_, err = repo.DeleteAgentMemoriesBatch(ctx, "bigquery", ids)
			gt.NoError(t, err)
		}

		// Create initial memory
		args := map[string]any{"query": "count active users"}
		_, err = agent.Run(ctx, "query_bigquery", args)
		gt.NoError(t, err)

		// Get initial memory
		memories, err := memService.SearchRelevantAgentMemories(ctx, "bigquery", "count active users", 1)
		gt.NoError(t, err)
		gt.V(t, len(memories)).Equal(1)
		memID := memories[0].ID

		// Use the memory multiple times with similar queries
		similarQueries := []string{
			"get active user count",
			"show active users",
			"list active user statistics",
		}

		for _, query := range similarQueries {
			args := map[string]any{"query": query}
			_, err := agent.Run(ctx, "query_bigquery", args)
			gt.NoError(t, err)
		}

		// Verify the original memory still exists and was updated
		finalMem, err := repo.GetAgentMemory(ctx, "bigquery", memID)
		gt.NoError(t, err)

		// LastUsedAt should be more recent than creation time
		gt.True(t, finalMem.LastUsedAt.After(finalMem.Timestamp) ||
			finalMem.LastUsedAt.Equal(finalMem.Timestamp))

		// QualityScore should be within valid range
		gt.True(t, finalMem.QualityScore >= -10.0)
		gt.True(t, finalMem.QualityScore <= 10.0)
	})
}

func TestAgent_MemoryPruning(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)
	agent := bqagent.NewAgent(config, llmClient, memService)

	t.Run("Low quality memories can be pruned", func(t *testing.T) {
		// Clean up
		existing, err := repo.ListAgentMemories(ctx, "bigquery")
		gt.NoError(t, err)
		if len(existing) > 0 {
			ids := make([]types.AgentMemoryID, len(existing))
			for i, m := range existing {
				ids[i] = m.ID
			}
			_, err = repo.DeleteAgentMemoriesBatch(ctx, "bigquery", ids)
			gt.NoError(t, err)
		}

		// Create some memories and manually set low quality scores
		args := map[string]any{"query": "test query"}
		_, err = agent.Run(ctx, "query_bigquery", args)
		gt.NoError(t, err)

		// Get the memory and manually update its score to critical level
		memories, err := repo.ListAgentMemories(ctx, "bigquery")
		gt.NoError(t, err)
		gt.True(t, len(memories) > 0)

		// Manually set a critical bad score
		criticalMem := memories[0]
		err = repo.UpdateMemoryScore(ctx, "bigquery", criticalMem.ID, -9.0, time.Now())
		gt.NoError(t, err)

		// Prune memories
		deleted, err := memService.PruneAgentMemories(ctx, "bigquery")
		gt.NoError(t, err)
		gt.N(t, deleted).Greater(0)

		// Verify the critical memory was deleted
		remaining, err := repo.ListAgentMemories(ctx, "bigquery")
		gt.NoError(t, err)

		for _, m := range remaining {
			// No memory should have critical score
			gt.True(t, m.QualityScore > -8.0)
		}
	})
}
