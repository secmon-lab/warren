package bigquery_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	bqagent "github.com/secmon-lab/warren/pkg/agents/bigquery"
	"github.com/secmon-lab/warren/pkg/repository"
	memoryservice "github.com/secmon-lab/warren/pkg/service/memory"
)

// newMockLLMClient creates a mock LLM client for testing
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
			return &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
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
	gt.V(t, result["result"]).NotNil()

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
	gt.V(t, result["result"]).NotNil()

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
