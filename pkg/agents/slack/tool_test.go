package slack_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	slackSDK "github.com/slack-go/slack"

	slackagent "github.com/secmon-lab/warren/pkg/agents/slack"
	domainmock "github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/repository"
	memoryservice "github.com/secmon-lab/warren/pkg/service/memory"
)

func TestInternalTool_SearchMessages(t *testing.T) {
	ctx := context.Background()

	slackClient := &domainmock.SlackClientMock{
		SearchMessagesContextFunc: func(ctx context.Context, query string, params slackSDK.SearchParameters) (*slackSDK.SearchMessages, error) {
			return &slackSDK.SearchMessages{
				Total: 1,
				Matches: []slackSDK.SearchMessage{
					{
						Type:      "message",
						Timestamp: "1234567890.123456",
						Text:      "Test message",
						Username:  "testuser",
						Channel: slackSDK.CtxChannel{
							ID:   "C123456",
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

	// Initialize to set up internal tool
	initialized, err := agent.Init(ctx, llmClient, memService)
	gt.NoError(t, err)
	gt.True(t, initialized)

	specs, err := agent.Specs(ctx)
	gt.NoError(t, err)
	gt.True(t, len(specs) > 0)
}

func TestInternalTool_GetThreadMessages(t *testing.T) {
	ctx := context.Background()

	slackClient := &domainmock.SlackClientMock{
		GetConversationRepliesContextFunc: func(ctx context.Context, params *slackSDK.GetConversationRepliesParameters) ([]slackSDK.Message, bool, string, error) {
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
						Timestamp:       "1234567891.123456",
						ThreadTimestamp: "1234567890.123456",
						Text:            "Reply message",
						User:            "U456",
						Username:        "user2",
					},
				},
			}, false, "", nil
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

	initialized, err := agent.Init(ctx, llmClient, memService)
	gt.NoError(t, err)
	gt.True(t, initialized)
}

func TestInternalTool_GetContextMessages(t *testing.T) {
	ctx := context.Background()

	slackClient := &domainmock.SlackClientMock{
		GetConversationHistoryContextFunc: func(ctx context.Context, params *slackSDK.GetConversationHistoryParameters) (*slackSDK.GetConversationHistoryResponse, error) {
			// Return different messages based on Latest/Oldest
			if params.Latest != "" {
				// Before messages
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
				return &slackSDK.GetConversationHistoryResponse{
					Messages: []slackSDK.Message{
						{
							Msg: slackSDK.Msg{
								Timestamp: "1234567892.123456",
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

	llmClient := newMockLLMClient()
	repo := repository.NewMemory()
	memService := memoryservice.New(llmClient, repo)

	agent := slackagent.New(
		slackagent.WithSlackClient(slackClient),
		slackagent.WithLLMClient(llmClient),
		slackagent.WithMemoryService(memService),
	)

	initialized, err := agent.Init(ctx, llmClient, memService)
	gt.NoError(t, err)
	gt.True(t, initialized)
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
	memService := memoryservice.New(llmClient, repo)

	agent := slackagent.New(
		slackagent.WithSlackClient(slackClient),
		slackagent.WithLLMClient(llmClient),
		slackagent.WithMemoryService(memService),
	)

	initialized, err := agent.Init(ctx, llmClient, memService)
	gt.NoError(t, err)
	gt.True(t, initialized)

	// Run with limit of 50
	result, err := agent.Run(ctx, "search_slack", map[string]any{
		"query": "test",
		"limit": float64(50),
	})

	gt.NoError(t, err)
	_, hasData := result["data"]
	gt.True(t, hasData)
}
