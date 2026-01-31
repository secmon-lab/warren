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
	"github.com/secmon-lab/warren/pkg/repository"
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

func TestAgent_Name(t *testing.T) {
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()

	agent := bqagent.NewAgentForTest(config, llmClient, repo)

	gt.V(t, agent.Name()).Equal("query_bigquery")
}

func TestAgent_Description(t *testing.T) {
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()

	agent := bqagent.NewAgentForTest(config, llmClient, repo)

	description := agent.Description()
	gt.V(t, description).NotEqual("")
	gt.True(t, len(description) > 0)
	gt.True(t, strings.Contains(description, "BigQuery"))
}

func TestAgent_SubAgent(t *testing.T) {
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

	agent := bqagent.NewAgentForTest(config, llmClient, repo)

	subAgent, err := agent.SubAgent()
	gt.NoError(t, err)
	gt.V(t, subAgent).NotNil()
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

	agent := bqagent.NewAgentForTest(cfg, llmClient, repo)

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

// TestAgent_Middleware tests the middleware logic
func TestAgent_Middleware(t *testing.T) {
	ctx := context.Background()
	config := &bqagent.Config{
		Tables:        []bqagent.TableConfig{},
		ScanSizeLimit: 1000000,
	}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()

	agent := bqagent.NewAgentForTest(config, llmClient, repo)
	middleware := agent.ExportedCreateMiddleware()

	t.Run("parameter parsing - query parameter", func(t *testing.T) {
		var capturedArgs map[string]any
		nextHandler := func(ctx context.Context, args map[string]any) (gollem.SubAgentResult, error) {
			capturedArgs = make(map[string]any)
			for k, v := range args {
				capturedArgs[k] = v
			}
			// Return minimal valid result
			session := &mock.SessionMock{
				HistoryFunc: func() (*gollem.History, error) {
					return &gollem.History{}, nil
				},
			}
			return gollem.SubAgentResult{
				Data:    map[string]any{"response": "test response"},
				Session: session,
			}, nil
		}

		handler := middleware(nextHandler)
		args := map[string]any{
			"query": "test BigQuery query",
		}

		result, err := handler(ctx, args)
		gt.NoError(t, err)
		gt.V(t, result).NotNil()

		// Check that _original_query is set
		gt.V(t, capturedArgs["_original_query"]).Equal("test BigQuery query")
	})

	t.Run("internal fields cleanup", func(t *testing.T) {
		nextHandler := func(ctx context.Context, args map[string]any) (gollem.SubAgentResult, error) {
			session := &mock.SessionMock{
				HistoryFunc: func() (*gollem.History, error) {
					return &gollem.History{}, nil
				},
			}
			return gollem.SubAgentResult{
				Data: map[string]any{
					"response":        "test response",
					"_original_query": "should be removed",
					"_memories":       "should be removed",
					"_memory_context": "should be removed",
				},
				Session: session,
			}, nil
		}

		handler := middleware(nextHandler)
		args := map[string]any{
			"query": "test BigQuery query",
		}

		result, err := handler(ctx, args)
		gt.NoError(t, err)
		gt.V(t, result).NotNil()

		// Check that internal fields are removed
		_, hasOriginalQuery := result.Data["_original_query"]
		gt.False(t, hasOriginalQuery)
		_, hasMemories := result.Data["_memories"]
		gt.False(t, hasMemories)
		_, hasMemoryContext := result.Data["_memory_context"]
		gt.False(t, hasMemoryContext)

		// Check that response was converted to data (fallback)
		gt.V(t, result.Data["data"]).Equal("test response")
		_, hasResponse := result.Data["response"]
		gt.False(t, hasResponse)
	})

	t.Run("no query parameter - passes through", func(t *testing.T) {
		nextCalled := false
		nextHandler := func(ctx context.Context, args map[string]any) (gollem.SubAgentResult, error) {
			nextCalled = true
			session := &mock.SessionMock{
				HistoryFunc: func() (*gollem.History, error) {
					return &gollem.History{}, nil
				},
			}
			return gollem.SubAgentResult{
				Data:    map[string]any{},
				Session: session,
			}, nil
		}

		handler := middleware(nextHandler)
		args := map[string]any{}

		result, err := handler(ctx, args)
		gt.NoError(t, err)
		gt.V(t, result).NotNil()
		gt.True(t, nextCalled)
	})
}
