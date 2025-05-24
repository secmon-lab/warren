package command_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"

	domain_mock "github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
)

func setupTestService(t *testing.T) (*command.Service, *domain_mock.RepositoryMock, *slack_svc.ThreadService, *slack.User, []*alert.Alert) {
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
	user := &slack.User{
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
			Data:      map[string]interface{}{"color": "blue"},
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
			Data:      map[string]interface{}{"color": "orange"},
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
			Data:      map[string]interface{}{"color": "red"},
			CreatedAt: fixedTime.Add(-3 * time.Hour),
			SlackThread: &slack.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
		},
	}

	alertListMap := map[types.AlertListID]*alert.List{}

	repo.GetAlertWithoutTicketFunc = func(ctx context.Context) (alert.Alerts, error) {
		return alerts, nil
	}

	repo.GetAlertListFunc = func(ctx context.Context, listID types.AlertListID) (*alert.List, error) {
		if l, ok := alertListMap[listID]; ok {
			return l, nil
		}
		return nil, nil
	}

	repo.PutAlertListFunc = func(ctx context.Context, list *alert.List) error {
		alertListMap[list.ID] = list
		return nil
	}

	return svc, repo, threadService, user, alerts
}

func TestService_List(t *testing.T) {
	svc, repo, threadService, user, baseAlerts := setupTestService(t)
	ctx := context.Background()

	t.Run("show alerts with limit", func(t *testing.T) {
		listID, err := svc.List(ctx, threadService, user, "limit 1")
		gt.NoError(t, err)
		gt.Value(t, listID).NotEqual(types.EmptyAlertListID)

		list, err := repo.GetAlertList(ctx, listID)
		gt.NoError(t, err).Required()
		alerts, err := list.GetAlerts(ctx, repo)
		gt.NoError(t, err).Required()
		gt.Array(t, alerts).Length(1)
		gt.Array(t, alerts).Has(baseAlerts[0])
	})

	t.Run("show alerts with offset", func(t *testing.T) {
		listID, err := svc.List(ctx, threadService, user, "offset 1")
		gt.NoError(t, err)
		gt.Value(t, listID).NotEqual(types.EmptyAlertListID)

		list, err := repo.GetAlertList(ctx, listID)
		gt.NoError(t, err).Required()
		alerts, err := list.GetAlerts(ctx, repo)
		gt.NoError(t, err).Required()
		gt.Array(t, alerts).Length(2)
		gt.Value(t, alerts[0].ID).Equal(baseAlerts[1].ID)
		gt.Value(t, alerts[1].ID).Equal(baseAlerts[2].ID)
	})

	t.Run("show alerts with grep filter", func(t *testing.T) {
		listID, err := svc.List(ctx, threadService, user, "grep orange")
		gt.NoError(t, err)
		gt.Value(t, listID).NotEqual(types.EmptyAlertListID)

		list, err := repo.GetAlertList(ctx, listID)
		gt.NoError(t, err).Required()
		alerts, err := list.GetAlerts(ctx, repo)
		gt.NoError(t, err).Required()
		gt.Array(t, alerts).Length(1)
		gt.Value(t, alerts[0].ID).Equal(baseAlerts[1].ID)
	})

	t.Run("show alerts with sort by CreatedAt", func(t *testing.T) {
		listID, err := svc.List(ctx, threadService, user, "sort CreatedAt")
		gt.NoError(t, err)
		gt.Value(t, listID).NotEqual(types.EmptyAlertListID)

		list, err := repo.GetAlertList(ctx, listID)
		gt.NoError(t, err).Required()
		alerts, err := list.GetAlerts(ctx, repo)
		gt.NoError(t, err).Required()
		gt.Array(t, alerts).Length(3)
		gt.Value(t, alerts[0].CreatedAt.Before(alerts[1].CreatedAt)).Equal(true)
		gt.Value(t, alerts[1].CreatedAt.Before(alerts[2].CreatedAt)).Equal(true)
	})

	t.Run("show alerts with multiple pipeline actions", func(t *testing.T) {
		listID, err := svc.List(ctx, threadService, user, "grep orange | sort CreatedAt | limit 1")
		gt.NoError(t, err)
		gt.Value(t, listID).NotEqual(types.EmptyAlertListID)

		list, err := repo.GetAlertList(ctx, listID)
		gt.NoError(t, err).Required()
		alerts, err := list.GetAlerts(ctx, repo)
		gt.NoError(t, err).Required()
		gt.Array(t, alerts).Length(1)
		gt.Value(t, alerts[0].ID).Equal(baseAlerts[1].ID)
	})

	t.Run("error on invalid command", func(t *testing.T) {
		_, err := svc.List(ctx, threadService, user, "invalid_command")
		gt.Error(t, err)
		if !strings.Contains(err.Error(), "unknown command") && !strings.Contains(err.Error(), "unknown action") {
			t.Fatalf("value should contain unknown command or unknown action, actual: %s", err.Error())
		}
	})

	t.Run("error on invalid pipeline action", func(t *testing.T) {
		_, err := svc.List(ctx, threadService, user, "invalid_action")
		gt.Error(t, err)
		if !strings.Contains(err.Error(), "unknown action") {
			t.Fatalf("value should contain unknown action, actual: %s", err.Error())
		}
	})

	t.Run("error on invalid action argument", func(t *testing.T) {
		_, err := svc.List(ctx, threadService, user, "limit invalid")
		gt.Error(t, err)
		if !strings.Contains(err.Error(), "limit: failed to convert limit to int") {
			t.Fatalf("value should contain limit: failed to convert limit to int, actual: %s", err.Error())
		}
	})
}
