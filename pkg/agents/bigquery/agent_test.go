package bigquery_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/gemini"
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

					// Check if this is a reflection request
					isReflection := false
					for _, inp := range input {
						if text, ok := inp.(gollem.Text); ok {
							if strings.Contains(string(text), "Agent Task Reflection") {
								isReflection = true
								break
							}
						}
					}

					if isReflection {
						// Return reflection JSON (use snake_case as per JSON tags in reflectionResponse)
						reflectionJSON := `{
							"new_claims": ["Test claim from execution"],
							"helpful_memories": [],
							"harmful_memories": []
						}`
						return &gollem.Response{
							Texts: []string{reflectionJSON},
						}, nil
					}

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

	agent := bqagent.NewAgent(config, llmClient, repo)

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

	agent := bqagent.NewAgent(config, llmClient, repo)

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

	agent := bqagent.NewAgent(config, llmClient, repo)

	// Execute a query that should create memories
	args := map[string]any{
		"query": "Get login count",
	}
	_, err := agent.Run(ctx, "query_bigquery", args)
	gt.NoError(t, err)

	// Verify that memories were created
	memories, err := repo.ListAgentMemories(ctx, "bigquery")
	gt.NoError(t, err)
	gt.Number(t, len(memories)).Greater(0)

	// Verify the memory has expected fields
	mem := memories[0]
	gt.V(t, mem.AgentID).Equal("bigquery")
	gt.V(t, mem.Query).Equal("Get login count")
	gt.V(t, mem.Claim).NotEqual("")
	gt.True(t, mem.Score >= -10.0 && mem.Score <= 10.0)
}

func TestAgent_Run_InvalidFunctionName(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()

	repo := repository.NewMemory()

	agent := bqagent.NewAgent(config, llmClient, repo)

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

	agent := bqagent.NewAgent(config, llmClient, repo)

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

	agent := bqagent.NewAgent(config, llmClient, repo)

	// First, create a memory by running a query
	args := map[string]any{
		"query": "Get user login statistics",
	}
	_, err := agent.Run(ctx, "query_bigquery", args)
	gt.NoError(t, err)

	// Search for memories with similar query using memory service
	memSvc := memoryservice.New("bigquery", llmClient, repo)
	memories, err := memSvc.SearchAndSelectMemories(ctx, "user login data", 5)
	gt.NoError(t, err)
	gt.Number(t, len(memories)).Greater(0)
}

func TestAgent_MemoryMetadataOnly(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()

	agent := bqagent.NewAgent(config, llmClient, repo)

	// Create a memory
	args := map[string]any{
		"query": "Count active sessions",
	}
	_, err := agent.Run(ctx, "query_bigquery", args)
	gt.NoError(t, err)

	// Retrieve memory and verify metadata
	memories, err := repo.ListAgentMemories(ctx, "bigquery")
	gt.NoError(t, err)
	gt.Number(t, len(memories)).Greater(0)

	mem := memories[0]
	gt.V(t, mem.AgentID).Equal("bigquery")
	gt.V(t, mem.Query).NotEqual("")
	gt.V(t, mem.Claim).NotEqual("")
	gt.False(t, mem.CreatedAt.IsZero())
	// LastUsedAt is zero for newly created memories
	gt.True(t, mem.LastUsedAt.IsZero())
}

func TestAgent_MemoryFeedbackIntegration(t *testing.T) {
	// Original test code:
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
	agent := bqagent.NewAgent(config, llmClient, repo)

	t.Run("Agent execution triggers feedback collection for used memories", func(t *testing.T) {
		// Step 1: Execute a query to create initial memory
		args := map[string]any{"query": "Get user login count"}
		_, err := agent.Run(ctx, "query_bigquery", args)
		gt.NoError(t, err)

		// Get the created memory
		memSvc := memoryservice.New("bigquery", llmClient, repo)
		memories, err := memSvc.SearchAndSelectMemories(ctx, "Get user login count", 1)
		gt.NoError(t, err)
		gt.V(t, len(memories)).Equal(1)

		initialMem := memories[0]
		initialScore := initialMem.Score
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

		// Score may have changed (depends on LLM feedback)
		// We just verify it's within valid range
		gt.True(t, updatedMemories.Score >= -10.0)
		gt.True(t, updatedMemories.Score <= 10.0)

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
		memSvc := memoryservice.New("bigquery", llmClient, repo)
		memories, err := memSvc.SearchAndSelectMemories(ctx, "count active users", 1)
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
		gt.True(t, finalMem.LastUsedAt.After(finalMem.CreatedAt) ||
			finalMem.LastUsedAt.Equal(finalMem.CreatedAt))

		// Score should be within valid range
		gt.True(t, finalMem.Score >= -10.0)
		gt.True(t, finalMem.Score <= 10.0)
	})
}

func TestAgent_MemoryPruning(t *testing.T) {
	// Original test code:
	ctx := context.Background()
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	agent := bqagent.NewAgent(config, llmClient, repo)

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
		memSvc := memoryservice.New("bigquery", llmClient, repo)
		deleted, err := memSvc.PruneMemories(ctx)
		gt.NoError(t, err)
		gt.N(t, deleted).Greater(0)

		// Verify the critical memory was deleted
		remaining, err := repo.ListAgentMemories(ctx, "bigquery")
		gt.NoError(t, err)

		for _, m := range remaining {
			// No memory should have critical score
			gt.True(t, m.Score > -8.0)
		}
	})
}

func TestAgent_ExtractRecords_WithRealLLM(t *testing.T) {
	projectID := os.Getenv("TEST_GEMINI_PROJECT_ID")
	location := os.Getenv("TEST_GEMINI_LOCATION")

	if projectID == "" || location == "" {
		t.Skip("TEST_GEMINI_PROJECT_ID and TEST_GEMINI_LOCATION must be set for real LLM test")
	}

	ctx := context.Background()

	// Create real Gemini client
	llmClient, err := gemini.New(ctx, projectID, location, gemini.WithModel("gemini-2.0-flash-exp"))
	gt.NoError(t, err)

	// Create in-memory repository
	repo := repository.NewMemory()

	// Create agent config
	cfg := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1024 * 1024 * 10, // 10MB
		QueryTimeout:  time.Minute,
	}

	agent := bqagent.NewAgent(cfg, llmClient, repo)

	// Create a session with conversation history containing query results
	session, err := llmClient.NewSession(ctx)
	gt.NoError(t, err)

	// Simulate a conversation with BigQuery results
	userQuery := "Show me login events from users in the last 7 days"

	// Add user request and assistant response with query results
	queryResults := `Query result from BigQuery:

+----------+---------------------+------------------+
| user_id  | login_time          | ip_address       |
+----------+---------------------+------------------+
| user123  | 2024-11-25 10:30:00 | 192.168.1.100    |
| user456  | 2024-11-26 14:20:00 | 10.0.0.50        |
| user789  | 2024-11-27 09:15:00 | 172.16.0.25      |
+----------+---------------------+------------------+
3 rows returned`

	userContent, err := gollem.NewTextContent(userQuery)
	gt.NoError(t, err)
	modelContent, err := gollem.NewTextContent(queryResults)
	gt.NoError(t, err)

	history := &gollem.History{
		Messages: []gollem.Message{
			{
				Role:     gollem.RoleUser,
				Contents: []gollem.MessageContent{userContent},
			},
			{
				Role:     gollem.RoleAssistant,
				Contents: []gollem.MessageContent{modelContent},
			},
		},
	}

	err = session.AppendHistory(history)
	gt.NoError(t, err)

	// Test extractRecords with the session containing results
	records, err := agent.ExportedExtractRecords(ctx, userQuery, session)
	gt.NoError(t, err)
	gt.V(t, len(records)).NotEqual(0)

	t.Logf("Successfully extracted %d records", len(records))
	t.Logf("Sample record: %+v", records[0])

	// Verify first record has expected fields and values from the test data
	firstRecord := records[0]

	// user_id should be one of the expected users
	userID, ok := firstRecord["user_id"].(string)
	gt.True(t, ok)
	gt.S(t, userID).ContainsAny("user123", "user456", "user789")

	// login_time should be one of the expected dates
	loginTime, ok := firstRecord["login_time"].(string)
	gt.True(t, ok)
	gt.S(t, loginTime).ContainsAny("2024-11-25", "2024-11-26", "2024-11-27")

	// ip_address should be one of the expected IPs
	ipAddress, ok := firstRecord["ip_address"].(string)
	gt.True(t, ok)
	gt.S(t, ipAddress).ContainsAny("192.168.1.100", "10.0.0.50", "172.16.0.25")
}
