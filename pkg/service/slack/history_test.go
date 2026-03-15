package slack_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	slackSvc "github.com/secmon-lab/warren/pkg/service/slack"
	slackSDK "github.com/slack-go/slack"
)

func newTestSlackService(t *testing.T, client *mock.SlackClientMock) *slackSvc.Service {
	t.Helper()
	// Provide minimal mock responses for New() initialization
	client.AuthTestFunc = func() (*slackSDK.AuthTestResponse, error) {
		return &slackSDK.AuthTestResponse{
			TeamID: "T123",
			Team:   "testteam",
			UserID: "U_BOT",
			BotID:  "B123",
		}, nil
	}
	client.GetTeamInfoFunc = func() (*slackSDK.TeamInfo, error) {
		return &slackSDK.TeamInfo{
			ID:     "T123",
			Name:   "testteam",
			Domain: "testteam",
		}, nil
	}
	client.GetBotInfoContextFunc = func(ctx context.Context, params slackSDK.GetBotInfoParameters) (*slackSDK.Bot, error) {
		return &slackSDK.Bot{
			ID:     "B123",
			UserID: "U_BOT",
		}, nil
	}

	svc, err := slackSvc.New(client, "C_TEST")
	gt.NoError(t, err).Required()
	return svc
}

func TestGetMessageHistory_RootMessage(t *testing.T) {
	client := &mock.SlackClientMock{
		GetConversationHistoryContextFunc: func(ctx context.Context, params *slackSDK.GetConversationHistoryParameters) (*slackSDK.GetConversationHistoryResponse, error) {
			gt.Value(t, params.ChannelID).Equal("C_TEST_CH")
			gt.Value(t, params.Limit).Equal(100)
			return &slackSDK.GetConversationHistoryResponse{
				Messages: []slackSDK.Message{
					{Msg: slackSDK.Msg{Timestamp: "1700000100.000000", Text: "hello", User: "U001"}},
					{Msg: slackSDK.Msg{Timestamp: "1700000200.000000", Text: "world", User: "U002"}},
				},
			}, nil
		},
		GetUserInfoFunc: func(userID string) (*slackSDK.User, error) {
			return &slackSDK.User{
				ID: userID,
				Profile: slackSDK.UserProfile{
					DisplayName: "user-" + userID,
				},
			}, nil
		},
	}

	svc := newTestSlackService(t, client)

	// Root message (no threadID)
	slackMsg := model.NewTestMessage("C_TEST_CH", "", "T_TEAM", "1700000300.000000", "U003", "test query")

	history, err := svc.GetMessageHistory(context.Background(), &slackMsg)
	gt.NoError(t, err)
	gt.A(t, history).Length(2)
	gt.Value(t, history[0].Text).Equal("hello")
	gt.Value(t, history[0].UserID).Equal("U001")
	gt.Value(t, history[0].IsThread).Equal(false)
	gt.Value(t, history[1].Text).Equal("world")
	gt.Value(t, history[1].IsBot).Equal(false)
}

func TestGetMessageHistory_ThreadMessage(t *testing.T) {
	client := &mock.SlackClientMock{
		GetConversationRepliesContextFunc: func(ctx context.Context, params *slackSDK.GetConversationRepliesParameters) ([]slackSDK.Message, bool, string, error) {
			gt.Value(t, params.ChannelID).Equal("C_TEST_CH")
			gt.Value(t, params.Timestamp).Equal("1700000100.000000")
			return []slackSDK.Message{
				{Msg: slackSDK.Msg{Timestamp: "1700000100.000000", Text: "thread start", User: "U001"}},
				{Msg: slackSDK.Msg{Timestamp: "1700000150.000000", Text: "reply", User: "U002"}},
			}, false, "", nil
		},
		GetConversationHistoryContextFunc: func(ctx context.Context, params *slackSDK.GetConversationHistoryParameters) (*slackSDK.GetConversationHistoryResponse, error) {
			gt.Value(t, params.ChannelID).Equal("C_TEST_CH")
			gt.Value(t, params.Latest).Equal("1700000100.000000")
			return &slackSDK.GetConversationHistoryResponse{
				Messages: []slackSDK.Message{
					{Msg: slackSDK.Msg{Timestamp: "1700000050.000000", Text: "before thread", User: "U003"}},
				},
			}, nil
		},
		GetUserInfoFunc: func(userID string) (*slackSDK.User, error) {
			return &slackSDK.User{
				ID: userID,
				Profile: slackSDK.UserProfile{
					DisplayName: "user-" + userID,
				},
			}, nil
		},
	}

	svc := newTestSlackService(t, client)

	// Thread message (has threadID)
	slackMsg := model.NewTestMessage("C_TEST_CH", "1700000100.000000", "T_TEAM", "1700000200.000000", "U003", "test query")

	history, err := svc.GetMessageHistory(context.Background(), &slackMsg)
	gt.NoError(t, err)

	// Root context (1) + Thread replies (2) = 3
	gt.A(t, history).Length(3)

	// Root context comes first
	gt.Value(t, history[0].Text).Equal("before thread")
	gt.Value(t, history[0].IsThread).Equal(false)

	// Thread messages follow
	gt.Value(t, history[1].Text).Equal("thread start")
	gt.Value(t, history[1].IsThread).Equal(true)
	gt.Value(t, history[2].Text).Equal("reply")
	gt.Value(t, history[2].IsThread).Equal(true)
}

func TestGetMessageHistory_BotDetection(t *testing.T) {
	client := &mock.SlackClientMock{
		GetConversationHistoryContextFunc: func(ctx context.Context, params *slackSDK.GetConversationHistoryParameters) (*slackSDK.GetConversationHistoryResponse, error) {
			return &slackSDK.GetConversationHistoryResponse{
				Messages: []slackSDK.Message{
					{Msg: slackSDK.Msg{Timestamp: "1700000100.000000", Text: "human msg", User: "U001"}},
					{Msg: slackSDK.Msg{Timestamp: "1700000200.000000", Text: "bot msg", User: "U_BOT", BotID: "B123"}},
					{Msg: slackSDK.Msg{Timestamp: "1700000300.000000", Text: "bot msg 2", SubType: "bot_message"}},
				},
			}, nil
		},
		GetUserInfoFunc: func(userID string) (*slackSDK.User, error) {
			return &slackSDK.User{
				ID: userID,
				Profile: slackSDK.UserProfile{
					DisplayName: "user-" + userID,
				},
			}, nil
		},
	}

	svc := newTestSlackService(t, client)
	slackMsg := model.NewTestMessage("C_TEST_CH", "", "T_TEAM", "1700000400.000000", "U003", "test")

	history, err := svc.GetMessageHistory(context.Background(), &slackMsg)
	gt.NoError(t, err)
	gt.A(t, history).Length(3)

	gt.Value(t, history[0].IsBot).Equal(false)
	gt.Value(t, history[1].IsBot).Equal(true)
	gt.Value(t, history[2].IsBot).Equal(true)
}

func TestGetMessageHistory_HistoryFetchFailure(t *testing.T) {
	client := &mock.SlackClientMock{
		GetConversationRepliesContextFunc: func(ctx context.Context, params *slackSDK.GetConversationRepliesParameters) ([]slackSDK.Message, bool, string, error) {
			return []slackSDK.Message{
				{Msg: slackSDK.Msg{Timestamp: "1700000100.000000", Text: "reply", User: "U001"}},
			}, false, "", nil
		},
		GetConversationHistoryContextFunc: func(ctx context.Context, params *slackSDK.GetConversationHistoryParameters) (*slackSDK.GetConversationHistoryResponse, error) {
			return nil, context.DeadlineExceeded
		},
		GetUserInfoFunc: func(userID string) (*slackSDK.User, error) {
			return &slackSDK.User{
				ID: userID,
				Profile: slackSDK.UserProfile{
					DisplayName: "user-" + userID,
				},
			}, nil
		},
	}

	svc := newTestSlackService(t, client)

	// Thread message where root history fetch fails
	slackMsg := model.NewTestMessage("C_TEST_CH", "1700000100.000000", "T_TEAM", "1700000200.000000", "U001", "test")

	// Should still return thread messages without root context
	history, err := svc.GetMessageHistory(context.Background(), &slackMsg)
	gt.NoError(t, err)
	gt.A(t, history).Length(1)
	gt.Value(t, history[0].Text).Equal("reply")
}
