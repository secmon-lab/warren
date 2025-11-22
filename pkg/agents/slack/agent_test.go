package slack_test

import (
	"context"
	"os"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	slackSDK "github.com/slack-go/slack"

	slackagent "github.com/secmon-lab/warren/pkg/agents/slack"
	domainmock "github.com/secmon-lab/warren/pkg/domain/mock"
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
						Texts: []string{"mock search result"},
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

func TestAgent_ID(t *testing.T) {
	agent := slackagent.New()
	gt.V(t, agent.ID()).Equal("slack_search")
}

func TestAgent_Specs_NotEnabled(t *testing.T) {
	ctx := context.Background()
	agent := slackagent.New()

	specs, err := agent.Specs(ctx)
	gt.NoError(t, err)
	gt.V(t, len(specs)).Equal(0) // No specs when not enabled
}

func TestAgent_Specs_Enabled(t *testing.T) {
	ctx := context.Background()
	slackClient := &domainmock.SlackClientMock{
		SearchMessagesContextFunc: func(ctx context.Context, query string, params slackSDK.SearchParameters) (*slackSDK.SearchMessages, error) {
			return &slackSDK.SearchMessages{}, nil
		},
	}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := slackagent.New(
		slackagent.WithSlackClient(slackClient),
		slackagent.WithLLMClient(llmClient),
		slackagent.WithMemoryService(memService),
	)

	specs, err := agent.Specs(ctx)
	gt.NoError(t, err)
	gt.V(t, len(specs)).Equal(1)
	gt.V(t, specs[0].Name).Equal("search_slack")
	gt.V(t, specs[0].Description).NotEqual("")
	gt.V(t, len(specs[0].Parameters)).Equal(2) // query and limit
	_, hasQuery := specs[0].Parameters["query"]
	gt.True(t, hasQuery)
	_, hasLimit := specs[0].Parameters["limit"]
	gt.True(t, hasLimit)
}

func TestAgent_Init_NoToken(t *testing.T) {
	ctx := context.Background()
	agent := slackagent.New()
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	initialized, err := agent.Init(ctx, llmClient, memService)
	gt.NoError(t, err)
	gt.False(t, initialized) // Not initialized without token or client
}

func TestAgent_Init_WithClient(t *testing.T) {
	ctx := context.Background()
	slackClient := &domainmock.SlackClientMock{}
	agent := slackagent.New(slackagent.WithSlackClient(slackClient))
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	initialized, err := agent.Init(ctx, llmClient, memService)
	gt.NoError(t, err)
	gt.True(t, initialized)
	gt.True(t, agent.IsEnabled())
}

func TestAgent_Run_BasicSearch(t *testing.T) {
	ctx := context.Background()

	// Setup mock Slack client
	slackClient := &domainmock.SlackClientMock{
		SearchMessagesContextFunc: func(ctx context.Context, query string, params slackSDK.SearchParameters) (*slackSDK.SearchMessages, error) {
			return &slackSDK.SearchMessages{
				Total: 2,
				Matches: []slackSDK.SearchMessage{
					{
						Type:      "message",
						Timestamp: "1234567890.123456",
						Text:      "Test message 1",
						Username:  "user1",
						Channel: slackSDK.CtxChannel{
							ID:   "C123",
							Name: "general",
						},
					},
					{
						Type:      "message",
						Timestamp: "1234567891.123456",
						Text:      "Test message 2",
						Username:  "user2",
						Channel: slackSDK.CtxChannel{
							ID:   "C123",
							Name: "general",
						},
					},
				},
			}, nil
		},
	}

	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := slackagent.New(
		slackagent.WithSlackClient(slackClient),
		slackagent.WithLLMClient(llmClient),
		slackagent.WithMemoryService(memService),
	)

	// Run search
	result, err := agent.Run(ctx, "search_slack", map[string]any{
		"query": "test search",
		"limit": float64(50),
	})

	gt.NoError(t, err)
	_, hasData := result["data"]
	gt.True(t, hasData)
	gt.V(t, result["data"]).NotEqual("")
}

func TestAgent_Run_LimitEnforcement(t *testing.T) {
	ctx := context.Background()

	slackClient := &domainmock.SlackClientMock{
		SearchMessagesContextFunc: func(ctx context.Context, query string, params slackSDK.SearchParameters) (*slackSDK.SearchMessages, error) {
			return &slackSDK.SearchMessages{
				Total: 2,
				Matches: []slackSDK.SearchMessage{
					{
						Type:      "message",
						Timestamp: "1234567890.123456",
						Text:      "Test message 1",
						Username:  "user1",
						Channel: slackSDK.CtxChannel{
							ID:   "C123",
							Name: "general",
						},
					},
					{
						Type:      "message",
						Timestamp: "1234567891.123456",
						Text:      "Test message 2",
						Username:  "user2",
						Channel: slackSDK.CtxChannel{
							ID:   "C123",
							Name: "general",
						},
					},
				},
			}, nil
		},
	}

	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := slackagent.New(
		slackagent.WithSlackClient(slackClient),
		slackagent.WithLLMClient(llmClient),
		slackagent.WithMemoryService(memService),
	)

	// Request 300 messages (should be capped at 200 by agent)
	result, err := agent.Run(ctx, "search_slack", map[string]any{
		"query": "test",
		"limit": float64(300),
	})

	gt.NoError(t, err)
	_, hasData := result["data"]
	gt.True(t, hasData)
	// Note: Limit enforcement is tested directly in TestInternalTool_DirectLimitEnforcement
}

func TestAgent_Run_MissingQuery(t *testing.T) {
	ctx := context.Background()

	slackClient := &domainmock.SlackClientMock{}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := slackagent.New(
		slackagent.WithSlackClient(slackClient),
		slackagent.WithLLMClient(llmClient),
		slackagent.WithMemoryService(memService),
	)

	// Run without query parameter
	_, err := agent.Run(ctx, "search_slack", map[string]any{})

	gt.Error(t, err) // Should return error for missing query
}

func TestAgent_Run_UnknownFunction(t *testing.T) {
	ctx := context.Background()

	slackClient := &domainmock.SlackClientMock{}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := slackagent.New(
		slackagent.WithSlackClient(slackClient),
		slackagent.WithLLMClient(llmClient),
		slackagent.WithMemoryService(memService),
	)

	// Run with unknown function name
	_, err := agent.Run(ctx, "unknown_function", map[string]any{
		"query": "test",
	})

	gt.Error(t, err) // Should return error for unknown function
}

func TestAgent_Configure_WithToken(t *testing.T) {
	ctx := context.Background()
	agent := slackagent.New()
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	// Mock Slack client to simulate enabled state
	slackClient := &domainmock.SlackClientMock{}
	agent.SetSlackClient(slackClient)

	initialized, err := agent.Init(ctx, llmClient, memService)
	gt.NoError(t, err)
	gt.True(t, initialized)

	err = agent.Configure(ctx)
	gt.NoError(t, err)
}

func TestAgent_Configure_WithoutToken(t *testing.T) {
	ctx := context.Background()
	agent := slackagent.New()

	err := agent.Configure(ctx)
	gt.Error(t, err) // Should fail when not enabled
}

// TestAgent_SearchMessagesIntegration tests the agent with real Slack API
func TestAgent_SearchMessagesIntegration(t *testing.T) {
	token := os.Getenv("TEST_SLACK_USER_TOKEN")
	if token == "" {
		t.Skip("TEST_SLACK_USER_TOKEN not set, skipping integration test")
	}

	// Create agent with real Slack client
	slackClient := slackSDK.New(token)

	// Create mock LLM client that actually executes tools
	llmClient := &mock.LLMClientMock{
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
			// This session will return a simple response after tools are called
			return &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"Search completed successfully"},
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

	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := slackagent.New(
		slackagent.WithSlackClient(slackClient),
		slackagent.WithLLMClient(llmClient),
		slackagent.WithMemoryService(memService),
	)

	query := os.Getenv("TEST_SLACK_QUERY")
	if query == "" {
		query = "test"
	}

	ctx := context.Background()

	// Configure the agent
	err := agent.Configure(ctx)
	gt.NoError(t, err)

	// Execute search via agent
	result, err := agent.Run(ctx, "search_slack", map[string]any{
		"query": query,
		"limit": float64(10),
	})

	// Note: search.messages API requires User OAuth token, not Bot token
	// If you get "not_allowed_token_type" error, you need to use a User token
	if err != nil {
		t.Logf("Search failed: %v", err)
		t.Skip("Skipping due to API error - ensure TEST_SLACK_USER_TOKEN is a User OAuth token with search:read scope")
	}

	gt.NoError(t, err)
	gt.NotNil(t, result)

	// Validate response structure
	data, hasData := result["data"]
	gt.True(t, hasData)
	gt.V(t, data).NotEqual("")

	t.Logf("Agent search completed successfully")
}
