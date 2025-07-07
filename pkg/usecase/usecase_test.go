package usecase

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/service/slack"
	slack_sdk "github.com/slack-go/slack"
)

func TestUseCases_GetUserIcon(t *testing.T) {
	ctx := context.Background()

	// Setup Slack client mock
	slackMock := &mock.SlackClientMock{
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "U123456",
				TeamID: "T123456",
				Team:   "test-team",
				BotID:  "B123456",
			}, nil
		},
		GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
			return &slack_sdk.TeamInfo{
				ID:     "T123456",
				Name:   "test-team",
				Domain: "test-workspace",
			}, nil
		},
		GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
			return &slack_sdk.User{
				ID: userID,
				Profile: slack_sdk.UserProfile{
					Image192: "https://example.com/avatar.jpg",
				},
			}, nil
		},
	}

	// Create slack service
	slackService, err := slack.New(slackMock, "C123456")
	gt.NoError(t, err)

	// Create use case with slack service
	uc := New(WithSlackService(slackService))

	// Test GetUserIcon
	_, _, err = uc.GetUserIcon(ctx, "U123456")
	// We expect an error because example.com won't have the image
	gt.Error(t, err)

	// Verify Slack was called
	gt.Array(t, slackMock.GetUserInfoCalls()).Length(1)
	gt.Value(t, slackMock.GetUserInfoCalls()[0].UserID).Equal("U123456")
}

func TestUseCases_GetUserIcon_NoSlackService(t *testing.T) {
	ctx := context.Background()

	// Create use case without slack service
	uc := New()

	// Test GetUserIcon should return error
	_, _, err := uc.GetUserIcon(ctx, "U123456")
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("slack service not configured")
}

func TestUseCases_GetUserProfile(t *testing.T) {
	ctx := context.Background()

	// Setup Slack client mock
	slackMock := &mock.SlackClientMock{
		AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
			return &slack_sdk.AuthTestResponse{
				UserID: "U123456",
				TeamID: "T123456",
				Team:   "test-team",
				BotID:  "B123456",
			}, nil
		},
		GetTeamInfoFunc: func() (*slack_sdk.TeamInfo, error) {
			return &slack_sdk.TeamInfo{
				ID:     "T123456",
				Name:   "test-team",
				Domain: "test-workspace",
			}, nil
		},
		GetUserInfoFunc: func(userID string) (*slack_sdk.User, error) {
			return &slack_sdk.User{
				ID: userID,
				Profile: slack_sdk.UserProfile{
					DisplayName: "Test User",
				},
			}, nil
		},
	}

	// Create slack service
	slackService, err := slack.New(slackMock, "C123456")
	gt.NoError(t, err)

	// Create use case with slack service
	uc := New(WithSlackService(slackService))

	// Test GetUserProfile
	name, err := uc.GetUserProfile(ctx, "U123456")
	gt.NoError(t, err)
	gt.Value(t, name).Equal("Test User")

	// Verify Slack was called
	gt.Array(t, slackMock.GetUserInfoCalls()).Length(1)
	gt.Value(t, slackMock.GetUserInfoCalls()[0].UserID).Equal("U123456")
}

func TestUseCases_GetUserProfile_NoSlackService(t *testing.T) {
	ctx := context.Background()

	// Create use case without slack service
	uc := New()

	// Test GetUserProfile should return error
	_, err := uc.GetUserProfile(ctx, "U123456")
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("slack service not configured")
}
