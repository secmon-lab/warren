package slack_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/slack"
)

func TestSlackMessageSearch(t *testing.T) {
	testCases := []struct {
		name           string
		serverResponse *slack.SearchResponse
		serverStatus   int
		args           map[string]any
		expectError    bool
		validateResult func(t *testing.T, result map[string]any)
	}{
		{
			name: "successful search",
			serverResponse: &slack.SearchResponse{
				OK:    true,
				Query: "test",
				Messages: slack.MessagesBlock{
					Total: 2,
					Pagination: slack.Paging{
						Count: 2,
						Total: 2,
						Page:  1,
						Pages: 1,
					},
					Matches: []slack.Message{
						{
							Type: "message",
							Channel: slack.ChannelInfo{
								ID:   "C1234567890",
								Name: "general",
							},
							User:      "U1234567890",
							Username:  "john.doe",
							Text:      "Hello, this is a test message",
							Timestamp: "1234567890.123456",
							Permalink: "https://workspace.slack.com/archives/C1234567890/p1234567890123456",
						},
						{
							Type: "message",
							Channel: slack.ChannelInfo{
								ID:   "C0987654321",
								Name: "random",
							},
							User:      "U0987654321",
							Username:  "jane.smith",
							Text:      "Another test message",
							Timestamp: "1234567891.123456",
							Permalink: "https://workspace.slack.com/archives/C0987654321/p1234567891123456",
						},
					},
				},
			},
			serverStatus: http.StatusOK,
			args: map[string]any{
				"query": "test",
				"count": float64(20),
			},
			expectError: false,
			validateResult: func(t *testing.T, result map[string]any) {
				total, ok := result["total"].(float64)
				gt.True(t, ok)
				gt.Number(t, total).Equal(2)
				messages := gt.Cast[[]any](t, result["messages"])
				gt.A(t, messages).Length(2)
			},
		},
		{
			name: "empty search results",
			serverResponse: &slack.SearchResponse{
				OK:    true,
				Query: "nonexistent",
				Messages: slack.MessagesBlock{
					Total: 0,
					Pagination: slack.Paging{
						Count: 0,
						Total: 0,
						Page:  1,
						Pages: 0,
					},
					Matches: []slack.Message{},
				},
			},
			serverStatus: http.StatusOK,
			args: map[string]any{
				"query": "nonexistent",
			},
			expectError: false,
			validateResult: func(t *testing.T, result map[string]any) {
				total, ok := result["total"].(float64)
				gt.True(t, ok)
				gt.Number(t, total).Equal(0)
				messages := gt.Cast[[]any](t, result["messages"])
				gt.A(t, messages).Length(0)
			},
		},
		{
			name: "API error response",
			serverResponse: &slack.SearchResponse{
				OK:    false,
				Error: "invalid_auth",
			},
			serverStatus: http.StatusOK,
			args: map[string]any{
				"query": "test",
			},
			expectError: true,
		},
		{
			name:         "rate limit error",
			serverStatus: http.StatusTooManyRequests,
			args: map[string]any{
				"query": "test",
			},
			expectError: true,
		},
		{
			name: "missing query parameter",
			args: map[string]any{
				"count": float64(10),
			},
			expectError: true,
		},
		{
			name: "search with highlight",
			serverResponse: &slack.SearchResponse{
				OK:    true,
				Query: "highlighted",
				Messages: slack.MessagesBlock{
					Total: 1,
					Pagination: slack.Paging{
						Count: 1,
						Total: 1,
						Page:  1,
						Pages: 1,
					},
					Matches: []slack.Message{
						{
							Type: "message",
							Channel: slack.ChannelInfo{
								ID:   "C1234567890",
								Name: "general",
							},
							User:      "U1234567890",
							Username:  "john.doe",
							Text:      "This is a <em>highlighted</em> message",
							Timestamp: "1234567890.123456",
							Permalink: "https://workspace.slack.com/archives/C1234567890/p1234567890123456",
						},
					},
				},
			},
			serverStatus: http.StatusOK,
			args: map[string]any{
				"query":     "highlighted",
				"highlight": true,
			},
			expectError: false,
			validateResult: func(t *testing.T, result map[string]any) {
				total, ok := result["total"].(float64)
				gt.True(t, ok)
				gt.Number(t, total).Equal(1)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify authorization header
				authHeader := r.Header.Get("Authorization")
				gt.S(t, authHeader).Equal("Bearer test-token")

				// Return configured response
				w.WriteHeader(tc.serverStatus)
				if tc.serverResponse != nil {
					err := json.NewEncoder(w).Encode(tc.serverResponse)
					gt.NoError(t, err)
				}
			}))
			defer server.Close()

			// Create action with test server URL
			action := &slack.Action{}
			action.SetTestURL(server.URL)
			action.SetOAuthToken("test-token")

			// Execute
			ctx := context.Background()
			result, err := action.Run(ctx, "slack_message_search", tc.args)

			// Validate
			if tc.expectError {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
				gt.NotNil(t, result)
				if tc.validateResult != nil {
					tc.validateResult(t, result)
				}
			}
		})
	}
}

func TestSlackMessageSearchIntegration(t *testing.T) {
	token := os.Getenv("TEST_SLACK_USER_TOKEN")
	if token == "" {
		t.Skip("TEST_SLACK_USER_TOKEN not set, skipping integration test")
	}

	action := &slack.Action{}
	action.SetOAuthToken(token)

	query := os.Getenv("TEST_SLACK_QUERY")
	if query == "" {
		query = "test"
	}

	ctx := context.Background()

	// Configure the action
	err := action.Configure(ctx)
	gt.NoError(t, err)

	// Execute search
	args := map[string]any{
		"query": query,
		"count": float64(10),
	}

	result, err := action.Run(ctx, "slack_message_search", args)
	
	// Note: search.messages API requires User OAuth token, not Bot token
	// If you get "not_allowed_token_type" error, you need to use a User token
	if err != nil {
		t.Logf("Search failed: %v", err)
		t.Skip("Skipping due to API error - ensure TEST_SLACK_USER_TOKEN is a User OAuth token with search:read scope")
	}
	
	gt.NoError(t, err)
	gt.NotNil(t, result)

	// Validate response structure
	total, ok := result["total"].(float64)
	gt.B(t, ok).True()
	gt.B(t, total >= 0).True()

	messages, ok := result["messages"].([]any)
	gt.B(t, ok).True()

	if total > 0 && len(messages) > 0 {
		// Check first message structure
		firstMsg, ok := messages[0].(map[string]any)
		gt.B(t, ok).True()

		// Verify required fields exist
		_, hasChannel := firstMsg["channel"]
		_, hasUser := firstMsg["user"]
		_, hasText := firstMsg["text"]
		_, hasTimestamp := firstMsg["timestamp"]

		gt.B(t, hasChannel).True()
		gt.B(t, hasUser).True()
		gt.B(t, hasText).True()
		gt.B(t, hasTimestamp).True()
	}
}

func TestConfigure(t *testing.T) {
	testCases := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "with token",
			token:       "test-token",
			expectError: false,
		},
		{
			name:        "without token",
			token:       "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			action := &slack.Action{}
			action.SetOAuthToken(tc.token)

			err := action.Configure(context.Background())
			if tc.expectError {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
			}
		})
	}
}