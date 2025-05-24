package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	slack_sdk "github.com/slack-go/slack"

	"github.com/secmon-lab/warren/pkg/domain/mock"
	domain_mock "github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
)

func setupTestAggrService(t *testing.T) (*command.Service, *domain_mock.RepositoryMock, *slack_svc.ThreadService, slack.User, *alert.List) {
	ctx := context.Background()
	repo := &domain_mock.RepositoryMock{}
	llm := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{
							`{"title": "test title", "description": "test description"}`,
						},
					}, nil
				},
			}, nil
		},
	}

	svc := command.New(repo, llm)
	slackService := slack_svc.NewTestService(t)
	threadService, err := slackService.PostMessage(ctx, "test message")
	gt.NoError(t, err).Required()
	user := slack.User{
		ID:   "U0123456789",
		Name: "Test User",
	}

	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	alerts := []*alert.Alert{
		{
			ID: types.NewAlertID(),
			Metadata: alert.Metadata{
				Title:       "Alert 1",
				Description: "Test alert 1",
			},
			Data:      map[string]any{"color": "blue"},
			CreatedAt: fixedTime.Add(-1 * time.Hour),
			SlackThread: &slack.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
		},
		{
			ID: types.NewAlertID(),
			Metadata: alert.Metadata{
				Title:       "Alert 2",
				Description: "Test alert 2 with grep match",
			},
			Data:      map[string]any{"color": "orange"},
			CreatedAt: fixedTime.Add(-2 * time.Hour),
			SlackThread: &slack.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
		},
		{
			ID: types.NewAlertID(),
			Metadata: alert.Metadata{
				Title:       "Alert 3",
				Description: "Test alert 3",
			},
			Data:      map[string]any{"color": "red"},
			CreatedAt: fixedTime.Add(-3 * time.Hour),
			SlackThread: &slack.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
		},
	}

	alertList := alert.NewList(ctx, slack.Thread{
		ChannelID: "C0123456789",
		ThreadID:  "T0123456789",
	}, &user, alerts)

	repo.GetAlertListFunc = func(ctx context.Context, listID types.AlertListID) (*alert.List, error) {
		if listID == alertList.ID {
			return alertList, nil
		}
		return nil, nil
	}

	repo.PutAlertListFunc = func(ctx context.Context, list *alert.List) error {
		return nil
	}

	return svc, repo, threadService, user, alertList
}

func TestService_Aggregate(t *testing.T) {
	svc, _, threadService, user, alertList := setupTestAggrService(t)
	ctx := context.Background()

	t.Run("aggregate with default parameters", func(t *testing.T) {
		err := svc.Aggregate(ctx, threadService, user, alertList, "")
		gt.NoError(t, err)
	})

	t.Run("aggregate with custom threshold", func(t *testing.T) {
		err := svc.Aggregate(ctx, threadService, user, alertList, "threshold 0.8")
		gt.NoError(t, err)
	})

	t.Run("aggregate with custom threshold and topN", func(t *testing.T) {
		err := svc.Aggregate(ctx, threadService, user, alertList, "th 0.8 top 3")
		gt.NoError(t, err)
	})

	t.Run("error on invalid threshold", func(t *testing.T) {
		err := svc.Aggregate(ctx, threadService, user, alertList, "invalid")
		gt.Error(t, err)
	})

	t.Run("error on invalid threshold range", func(t *testing.T) {
		err := svc.Aggregate(ctx, threadService, user, alertList, "threshold 1.5")
		gt.Error(t, err)
	})

	t.Run("error on invalid topN", func(t *testing.T) {
		err := svc.Aggregate(ctx, threadService, user, alertList, "th 0.8 top invalid")
		gt.Error(t, err)
	})
}

func TestAggregate(t *testing.T) {
	type testCase struct {
		name      string
		args      string
		threshold float64
		topN      int
		wantErr   bool
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := t.Context()
			svc := &command.Service{}

			slackClient := &mock.SlackClientMock{
				PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
					return "", "", nil
				},
				AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
					return &slack_sdk.AuthTestResponse{
						UserID: "U0123456789",
					}, nil
				},
			}
			user := slack.User{}
			alertList := &alert.List{}

			slackSvc, err := slack_svc.New(slackClient, "C0123456789")
			gt.NoError(t, err).Required()
			threadService, err := slackSvc.PostMessage(ctx, "test message")
			gt.NoError(t, err).Required()

			err = svc.Aggregate(context.Background(), threadService, user, alertList, tc.args)
			if tc.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}
	}

	t.Run("default values", runTest(testCase{
		name:      "default values",
		args:      "",
		threshold: 0.99,
		topN:      10,
		wantErr:   false,
	}))

	t.Run("with threshold", runTest(testCase{
		name:      "with threshold",
		args:      "threshold 0.95",
		threshold: 0.95,
		topN:      10,
		wantErr:   false,
	}))

	t.Run("with top", runTest(testCase{
		name:      "with top",
		args:      "top 5",
		threshold: 0.99,
		topN:      5,
		wantErr:   false,
	}))

	t.Run("with both threshold and top", runTest(testCase{
		name:      "with both threshold and top",
		args:      "threshold 0.95 top 5",
		threshold: 0.95,
		topN:      5,
		wantErr:   false,
	}))

	t.Run("invalid threshold", runTest(testCase{
		name:    "invalid threshold",
		args:    "threshold 1.5",
		wantErr: true,
	}))

	t.Run("invalid top", runTest(testCase{
		name:    "invalid top",
		args:    "top 0",
		wantErr: true,
	}))

	t.Run("unknown argument", runTest(testCase{
		name:    "unknown argument",
		args:    "unknown",
		wantErr: true,
	}))
}
