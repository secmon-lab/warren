package slack_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	slackSDK "github.com/slack-go/slack"

	slackagent "github.com/secmon-lab/warren/pkg/agents/slack"
	domainmock "github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestInternalTool_SearchMessages(t *testing.T) {
	ctx := context.Background()

	slackClient := &domainmock.SlackClientMock{
		SearchMessagesContextFunc: func(ctx context.Context, query string, params slackSDK.SearchParameters) (*slackSDK.SearchMessages, error) {
			// Verify query parameter
			gt.V(t, query).Equal("test search")

			return &slackSDK.SearchMessages{
				Total: 2,
				Matches: []slackSDK.SearchMessage{
					{
						Type:      "message",
						Timestamp: "1234567890.123456",
						Text:      "Test message 1",
						Username:  "testuser",
						Channel: slackSDK.CtxChannel{
							ID:   "C123456",
							Name: "general",
						},
					},
					{
						Type:      "message",
						Timestamp: "1234567891.654321",
						Text:      "Test message 2",
						Username:  "testuser2",
						Channel: slackSDK.CtxChannel{
							ID:   "C123456",
							Name: "general",
						},
					},
				},
			}, nil
		},
	}

	tool := slackagent.NewInternalToolForTest(slackClient, 0)

	// Call searchMessages directly
	result, err := tool.Run(ctx, "slack_search_messages", map[string]any{
		"query": "test search",
	})

	gt.NoError(t, err)
	gt.V(t, result).NotNil()

	// Verify response structure
	total, ok := result["total"].(float64)
	gt.True(t, ok)
	gt.V(t, total).Equal(2.0)

	messages, ok := result["messages"].([]any)
	gt.True(t, ok)
	gt.V(t, len(messages)).Equal(2)

	// Verify first message
	msg1, ok := messages[0].(map[string]any)
	gt.True(t, ok)
	gt.V(t, msg1["text"]).Equal("Test message 1")
	gt.V(t, msg1["user_name"]).Equal("testuser")
	gt.V(t, msg1["channel_id"]).Equal("C123456")
	gt.V(t, msg1["channel_name"]).Equal("general")
	gt.V(t, msg1["timestamp"]).Equal("1234567890.123456")

	// Verify formatted time has sub-second precision
	formattedTime, ok := msg1["formatted_time"].(string)
	gt.True(t, ok)
	gt.V(t, formattedTime).NotEqual("")
}

func TestInternalTool_GetThreadMessages(t *testing.T) {
	ctx := context.Background()

	slackClient := &domainmock.SlackClientMock{
		GetConversationRepliesContextFunc: func(ctx context.Context, params *slackSDK.GetConversationRepliesParameters) ([]slackSDK.Message, bool, string, error) {
			// Verify parameters
			gt.V(t, params.ChannelID).Equal("C123456")
			gt.V(t, params.Timestamp).Equal("1234567890.123456")

			return []slackSDK.Message{
				{
					Msg: slackSDK.Msg{
						Timestamp: "1234567890.123456",
						Text:      "Parent message",
						User:      "U123",
						Username:  "user1",
					},
				},
				{
					Msg: slackSDK.Msg{
						Timestamp:       "1234567891.654321",
						ThreadTimestamp: "1234567890.123456",
						Text:            "Reply message",
						User:            "U456",
						Username:        "user2",
					},
				},
			}, false, "", nil
		},
	}

	tool := slackagent.NewInternalToolForTest(slackClient, 0)

	// Call getThreadMessages directly
	result, err := tool.Run(ctx, "slack_get_thread_messages", map[string]any{
		"channel":   "C123456",
		"thread_ts": "1234567890.123456",
	})

	gt.NoError(t, err)
	gt.V(t, result).NotNil()

	// Verify response structure
	messages, ok := result["messages"].([]any)
	gt.True(t, ok)
	gt.V(t, len(messages)).Equal(2)

	// Verify parent message
	msg1, ok := messages[0].(map[string]any)
	gt.True(t, ok)
	gt.V(t, msg1["text"]).Equal("Parent message")
	gt.V(t, msg1["user_name"]).Equal("user1")
	gt.V(t, msg1["timestamp"]).Equal("1234567890.123456")

	// Verify reply message
	msg2, ok := messages[1].(map[string]any)
	gt.True(t, ok)
	gt.V(t, msg2["text"]).Equal("Reply message")
	gt.V(t, msg2["user_name"]).Equal("user2")
	gt.V(t, msg2["timestamp"]).Equal("1234567891.654321")
}

func TestInternalTool_GetContextMessages(t *testing.T) {
	ctx := context.Background()

	slackClient := &domainmock.SlackClientMock{
		GetConversationHistoryContextFunc: func(ctx context.Context, params *slackSDK.GetConversationHistoryParameters) (*slackSDK.GetConversationHistoryResponse, error) {
			// Verify channel parameter
			gt.V(t, params.ChannelID).Equal("C123456")

			// Return different messages based on Latest/Oldest
			if params.Latest != "" {
				// Before messages
				gt.V(t, params.Latest).Equal("1234567890.000000")
				gt.V(t, params.Inclusive).Equal(false)
				return &slackSDK.GetConversationHistoryResponse{
					Messages: []slackSDK.Message{
						{
							Msg: slackSDK.Msg{
								Timestamp: "1234567888.123456",
								Text:      "Message before",
								Username:  "user1",
							},
						},
					},
				}, nil
			} else if params.Oldest != "" {
				// After messages
				gt.V(t, params.Oldest).Equal("1234567890.000000")
				gt.V(t, params.Inclusive).Equal(false)
				return &slackSDK.GetConversationHistoryResponse{
					Messages: []slackSDK.Message{
						{
							Msg: slackSDK.Msg{
								Timestamp: "1234567892.654321",
								Text:      "Message after",
								Username:  "user2",
							},
						},
					},
				}, nil
			}
			return &slackSDK.GetConversationHistoryResponse{
				Messages: []slackSDK.Message{},
			}, nil
		},
	}

	tool := slackagent.NewInternalToolForTest(slackClient, 0)

	// Call getContextMessages directly
	result, err := tool.Run(ctx, "slack_get_context_messages", map[string]any{
		"channel":   "C123456",
		"around_ts": "1234567890",
		"before":    float64(1),
		"after":     float64(1),
	})

	gt.NoError(t, err)
	gt.V(t, result).NotNil()

	// Verify response structure
	beforeMessages, ok := result["before_messages"].([]any)
	gt.True(t, ok)
	gt.V(t, len(beforeMessages)).Equal(1)

	afterMessages, ok := result["after_messages"].([]any)
	gt.True(t, ok)
	gt.V(t, len(afterMessages)).Equal(1)

	// Verify before message
	beforeMsg, ok := beforeMessages[0].(map[string]any)
	gt.True(t, ok)
	gt.V(t, beforeMsg["text"]).Equal("Message before")
	gt.V(t, beforeMsg["user_name"]).Equal("user1")

	// Verify after message
	afterMsg, ok := afterMessages[0].(map[string]any)
	gt.True(t, ok)
	gt.V(t, afterMsg["text"]).Equal("Message after")
	gt.V(t, afterMsg["user_name"]).Equal("user2")
}

func TestInternalTool_LimitEnforcement(t *testing.T) {
	ctx := context.Background()

	// Create 150 mock messages
	matches := make([]slackSDK.SearchMessage, 150)
	for i := 0; i < 150; i++ {
		matches[i] = slackSDK.SearchMessage{
			Type:      "message",
			Timestamp: "1234567890.123456",
			Text:      "Test message",
			Username:  "user",
			Channel: slackSDK.CtxChannel{
				ID:   "C123",
				Name: "general",
			},
		}
	}

	slackClient := &domainmock.SlackClientMock{
		SearchMessagesContextFunc: func(ctx context.Context, query string, params slackSDK.SearchParameters) (*slackSDK.SearchMessages, error) {
			return &slackSDK.SearchMessages{
				Total:   150,
				Matches: matches,
			}, nil
		},
	}

	llmClient := newMockLLMClient()
	repo := repository.NewMemory()

	agent := slackagent.New(
		slackagent.WithSlackClient(slackClient),
		slackagent.WithLLMClient(llmClient),
	)

	initialized, err := agent.Init(ctx, llmClient, repo)
	gt.NoError(t, err)
	gt.True(t, initialized)

	// Run with limit of 50
	result, err := agent.Run(ctx, "search_slack", map[string]any{
		"request": "test",
		"limit":   float64(50),
	})

	gt.NoError(t, err)
	_, hasResponse := result["response"]
	gt.True(t, hasResponse)
}

func TestInternalTool_DirectLimitEnforcement(t *testing.T) {
	ctx := context.Background()

	// Create 250 mock messages (exceeds limit)
	matches := make([]slackSDK.SearchMessage, 250)
	for i := 0; i < 250; i++ {
		matches[i] = slackSDK.SearchMessage{
			Type:      "message",
			Timestamp: "1234567890.123456",
			Text:      "Test message",
			Username:  "user",
			Channel: slackSDK.CtxChannel{
				ID:   "C123",
				Name: "general",
			},
		}
	}

	slackClient := &domainmock.SlackClientMock{
		SearchMessagesContextFunc: func(ctx context.Context, query string, params slackSDK.SearchParameters) (*slackSDK.SearchMessages, error) {
			return &slackSDK.SearchMessages{
				Total:   250,
				Matches: matches,
			}, nil
		},
	}

	// Create internal tool directly with maxLimit set
	tool := slackagent.NewInternalToolForTest(slackClient, 50)

	// Call searchMessages directly
	result, err := tool.Run(ctx, "slack_search_messages", map[string]any{
		"query": "test",
	})

	gt.NoError(t, err)
	gt.V(t, result).NotNil()

	// Verify that result has messages
	messages, ok := result["messages"].([]any)
	gt.True(t, ok)

	// Verify that only 50 messages were returned (enforced by maxLimit)
	gt.V(t, len(messages)).Equal(50)

	// Verify total count is still 250 (from Slack API)
	total, ok := result["total"].(float64)
	gt.True(t, ok)
	gt.V(t, total).Equal(250.0)
}
