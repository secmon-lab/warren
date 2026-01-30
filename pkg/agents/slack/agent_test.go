package slack_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	slackSDK "github.com/slack-go/slack"

	slackagent "github.com/secmon-lab/warren/pkg/agents/slack"
	domainmock "github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/repository"
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

func TestAgent_Name(t *testing.T) {
	ctx := context.Background()
	slackClient := &domainmock.SlackClientMock{}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()

	agent := slackagent.New(ctx, slackClient, llmClient, repo)

	gt.V(t, agent.Name()).Equal("search_slack")
}

func TestAgent_Description(t *testing.T) {
	ctx := context.Background()
	slackClient := &domainmock.SlackClientMock{}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()

	agent := slackagent.New(ctx, slackClient, llmClient, repo)

	description := agent.Description()
	gt.V(t, description).NotEqual("")
	gt.True(t, len(description) > 0)
	gt.True(t, strings.Contains(description, "Slack"))
}

func TestAgent_SubAgent(t *testing.T) {
	ctx := context.Background()
	slackClient := &domainmock.SlackClientMock{}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()

	agent := slackagent.New(ctx, slackClient, llmClient, repo)

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

	// Create memory service with in-memory repository
	repo := repository.NewMemory()

	// Create mock Slack client
	slackClient := &domainmock.SlackClientMock{}

	// Create agent
	agent := slackagent.New(ctx, slackClient, llmClient, repo)

	// Create a session with conversation history containing search results
	session, err := llmClient.NewSession(ctx)
	gt.NoError(t, err)

	// Simulate a conversation with Slack search results
	userQuery := "Find messages about authentication problems in the last week"

	// Add user request and assistant response
	searchResults := `Found 2 messages about authentication problems:

Message 1:
User: @john_doe (U12345)
Channel: #support (C98765)
Time: 2024-11-25T10:30:00Z
Text: "Users are reporting authentication errors when trying to log in. The error message says 'Invalid credentials' even with correct passwords."

Message 2:
User: @jane_smith (U67890)
Channel: #incidents (C54321)
Time: 2024-11-26T14:20:00Z
Text: "Multiple authentication failures detected. Seems to be affecting users in the APAC region primarily."`

	userContent, err := gollem.NewTextContent(userQuery)
	gt.NoError(t, err)
	modelContent, err := gollem.NewTextContent(searchResults)
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

	// Verify that message records have expected fields and values
	firstRecord := records[0]

	// Verify text field contains authentication error message
	text, ok := firstRecord["text"].(string)
	gt.True(t, ok)
	gt.S(t, text).Contains("authentication")

	// Verify user field contains one of the expected users
	user, ok := firstRecord["user"].(string)
	gt.True(t, ok)
	gt.S(t, user).ContainsAny("john_doe", "U12345", "jane_smith", "U67890")

	// Verify channel field contains one of the expected channels
	channel, ok := firstRecord["channel"].(string)
	gt.True(t, ok)
	gt.S(t, channel).ContainsAny("support", "C98765", "incidents", "C54321")

	// Verify timestamp field contains expected date format
	timestamp, ok := firstRecord["timestamp"].(string)
	gt.True(t, ok)
	gt.S(t, timestamp).ContainsAny("2024-11-25", "2024-11-26")
}

// TestAgent_SearchMessagesIntegration tests the agent with real Slack API
func TestAgent_SearchMessagesIntegration(t *testing.T) {
	token := os.Getenv("TEST_SLACK_USER_TOKEN")
	if token == "" {
		t.Skip("TEST_SLACK_USER_TOKEN not set, skipping integration test")
	}

	ctx := context.Background()

	// Create agent with real Slack client
	slackClient := slackSDK.New(token)
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()

	agent := slackagent.New(ctx, slackClient, llmClient, repo)

	// Create SubAgent
	subAgent, err := agent.SubAgent()
	gt.NoError(t, err)
	gt.V(t, subAgent).NotNil()

	t.Logf("Slack Search Agent configured successfully")
}

// TestAgent_Middleware tests the middleware logic
func TestAgent_Middleware(t *testing.T) {
	ctx := context.Background()
	slackClient := &domainmock.SlackClientMock{}
	llmClient := newMockLLMClient()
	repo := repository.NewMemory()

	agent := slackagent.New(ctx, slackClient, llmClient, repo)
	middleware := agent.ExportedCreateMiddleware()

	t.Run("parameter parsing - request and default limit", func(t *testing.T) {
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
			"request": "test search query",
		}

		result, err := handler(ctx, args)
		gt.NoError(t, err)
		gt.V(t, result).NotNil()

		// Check that limit defaults to 50
		gt.V(t, capturedArgs["_limit"]).Equal(50)
		gt.V(t, capturedArgs["_original_request"]).Equal("test search query")
	})

	t.Run("parameter parsing - custom limit", func(t *testing.T) {
		var capturedArgs map[string]any
		nextHandler := func(ctx context.Context, args map[string]any) (gollem.SubAgentResult, error) {
			capturedArgs = make(map[string]any)
			for k, v := range args {
				capturedArgs[k] = v
			}
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
			"request": "test search query",
			"limit":   float64(100),
		}

		result, err := handler(ctx, args)
		gt.NoError(t, err)
		gt.V(t, result).NotNil()

		// Check that limit is set to 100
		gt.V(t, capturedArgs["_limit"]).Equal(100)
	})

	t.Run("parameter parsing - limit exceeds maximum", func(t *testing.T) {
		var capturedArgs map[string]any
		nextHandler := func(ctx context.Context, args map[string]any) (gollem.SubAgentResult, error) {
			capturedArgs = make(map[string]any)
			for k, v := range args {
				capturedArgs[k] = v
			}
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
			"request": "test search query",
			"limit":   float64(500),
		}

		result, err := handler(ctx, args)
		gt.NoError(t, err)
		gt.V(t, result).NotNil()

		// Check that limit is capped at 200
		gt.V(t, capturedArgs["_limit"]).Equal(200)
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
					"response":          "test response",
					"_original_request": "should be removed",
					"_memories":         "should be removed",
					"_memory_context":   "should be removed",
					"_limit":            "should be removed",
				},
				Session: session,
			}, nil
		}

		handler := middleware(nextHandler)
		args := map[string]any{
			"request": "test search query",
		}

		result, err := handler(ctx, args)
		gt.NoError(t, err)
		gt.V(t, result).NotNil()

		// Check that internal fields are removed
		_, hasOriginalRequest := result.Data["_original_request"]
		gt.False(t, hasOriginalRequest)
		_, hasMemories := result.Data["_memories"]
		gt.False(t, hasMemories)
		_, hasMemoryContext := result.Data["_memory_context"]
		gt.False(t, hasMemoryContext)
		_, hasLimit := result.Data["_limit"]
		gt.False(t, hasLimit)

		// Check that response is preserved
		gt.V(t, result.Data["response"]).Equal("test response")
	})

	t.Run("no request parameter - passes through", func(t *testing.T) {
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
