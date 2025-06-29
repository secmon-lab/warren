package list_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/slack-go/slack/slackevents"

	mock "github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/command"
	"github.com/secmon-lab/warren/pkg/service/command/core"
	list "github.com/secmon-lab/warren/pkg/service/command/list"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func setupTestService(t *testing.T) (*command.Service, *mock.RepositoryMock, *slack_svc.ThreadService, *slack.User, []*alert.Alert, gollem.LLMClient) {
	ctx := context.Background()
	repo := &mock.RepositoryMock{}
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

	slackService := slack_svc.NewTestService(t)
	threadService, err := slackService.PostMessage(ctx, "test message")
	gt.NoError(t, err).Required()
	svc := command.New(repo, llm, threadService)
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

	repo.GetAlertWithoutTicketFunc = func(ctx context.Context, offset, limit int) (alert.Alerts, error) {
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

	return svc, repo, threadService, user, alerts, llm
}

// createTestSlackMessage creates a test slack.Message for testing purposes
func createTestSlackMessage(user *slack.User) *slack.Message {
	ctx := context.Background()

	// Create a mock event to use with NewMessage
	event := &slackevents.EventsAPIEvent{
		TeamID: "T0123456789",
		InnerEvent: slackevents.EventsAPIInnerEvent{
			Data: &slackevents.MessageEvent{
				TimeStamp:       "1234567890.123456",
				Channel:         "C0123456789",
				ThreadTimeStamp: "1234567890.123456",
				User:            user.ID,
				Text:            "test message",
			},
		},
	}

	return slack.NewMessage(ctx, event)
}

func TestService_List(t *testing.T) {
	_, repo, threadService, user, baseAlerts, llm := setupTestService(t)
	ctx := context.Background()

	// Create core.Clients directly
	clients := core.NewClients(repo, llm, threadService)

	t.Run("show alerts with limit", func(t *testing.T) {
		resp, err := list.Create(ctx, clients, createTestSlackMessage(user), "limit 1")
		gt.NoError(t, err)
		listID := gt.Cast[types.AlertListID](t, resp)
		gt.Value(t, listID).NotEqual(types.EmptyAlertListID)

		list, err := repo.GetAlertList(ctx, listID)
		gt.NoError(t, err).Required()
		alerts, err := list.GetAlerts(ctx, repo)
		gt.NoError(t, err).Required()
		gt.Array(t, alerts).Length(1)
		gt.Array(t, alerts).Has(baseAlerts[0])
	})

	t.Run("show alerts with offset", func(t *testing.T) {
		resp, err := list.Create(ctx, clients, createTestSlackMessage(user), "offset 1")
		gt.NoError(t, err)
		listID := gt.Cast[types.AlertListID](t, resp)
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
		resp, err := list.Create(ctx, clients, createTestSlackMessage(user), "grep orange")
		listID := gt.Cast[types.AlertListID](t, resp)
		gt.NoError(t, err)
		gt.Value(t, listID).NotEqual(types.EmptyAlertListID)

		list, err := repo.GetAlertList(ctx, gt.Cast[types.AlertListID](t, resp))
		gt.NoError(t, err).Required()
		alerts, err := list.GetAlerts(ctx, repo)
		gt.NoError(t, err).Required()
		gt.Array(t, alerts).Length(1)
		gt.Value(t, alerts[0].ID).Equal(baseAlerts[1].ID)
	})

	t.Run("show alerts with sort by CreatedAt", func(t *testing.T) {
		resp, err := list.Create(ctx, clients, createTestSlackMessage(user), "sort CreatedAt")
		gt.NoError(t, err)
		listID := gt.Cast[types.AlertListID](t, resp)
		gt.Value(t, listID).NotEqual(types.EmptyAlertListID)

		list, err := repo.GetAlertList(ctx, gt.Cast[types.AlertListID](t, resp))
		gt.NoError(t, err).Required()
		alerts, err := list.GetAlerts(ctx, repo)
		gt.NoError(t, err).Required()
		gt.Array(t, alerts).Length(3)
		gt.Value(t, alerts[0].CreatedAt.Before(alerts[1].CreatedAt)).Equal(true)
		gt.Value(t, alerts[1].CreatedAt.Before(alerts[2].CreatedAt)).Equal(true)
	})

	t.Run("show alerts with multiple pipeline actions", func(t *testing.T) {
		resp, err := list.Create(ctx, clients, createTestSlackMessage(user), "grep orange | sort CreatedAt | limit 1")
		listID := gt.Cast[types.AlertListID](t, resp)
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
		_, err := list.Create(ctx, clients, createTestSlackMessage(user), "invalid_command")
		gt.Error(t, err)
		if !strings.Contains(err.Error(), "unknown command") && !strings.Contains(err.Error(), "unknown action") {
			t.Fatalf("value should contain unknown command or unknown action, actual: %s", err.Error())
		}
	})

	t.Run("error on invalid pipeline action", func(t *testing.T) {
		_, err := list.Create(ctx, clients, createTestSlackMessage(user), "invalid_action")
		gt.Error(t, err)
		if !strings.Contains(err.Error(), "unknown action") {
			t.Fatalf("value should contain unknown action, actual: %s", err.Error())
		}
	})

	t.Run("error on invalid action argument", func(t *testing.T) {
		_, err := list.Create(ctx, clients, createTestSlackMessage(user), "limit invalid")
		gt.Error(t, err)
		if !strings.Contains(err.Error(), "limit: failed to convert limit to int") {
			t.Fatalf("value should contain limit: failed to convert limit to int, actual: %s", err.Error())
		}
	})

	t.Run("default behavior shows all alerts when no input", func(t *testing.T) {
		resp, err := list.Create(ctx, clients, createTestSlackMessage(user), "")
		gt.NoError(t, err)
		listID := gt.Cast[types.AlertListID](t, resp)
		gt.Value(t, listID).NotEqual(types.EmptyAlertListID)

		list, err := repo.GetAlertList(ctx, listID)
		gt.NoError(t, err).Required()
		alerts, err := list.GetAlerts(ctx, repo)
		gt.NoError(t, err).Required()
		gt.Array(t, alerts).Length(3)
		gt.Value(t, alerts[0].ID).Equal(baseAlerts[0].ID)
		gt.Value(t, alerts[1].ID).Equal(baseAlerts[1].ID)
		gt.Value(t, alerts[2].ID).Equal(baseAlerts[2].ID)
	})
}

func TestTimeFilters(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := clock.With(context.Background(), func() time.Time { return fixedTime })
	alerts := alert.Alerts{
		{
			CreatedAt: fixedTime.Add(-2 * time.Hour),
		},
		{
			CreatedAt: fixedTime.Add(-1 * time.Hour),
		},
		{
			CreatedAt: fixedTime,
		},
	}

	type testCase struct {
		name     string
		cmdName  string
		args     string
		expected int
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			init, err := list.FindMatchedInitFunc(tc.cmdName)
			gt.NoError(t, err)

			action, err := init(tc.args)
			gt.NoError(t, err)

			result, err := action(ctx, alerts)
			gt.NoError(t, err)
			gt.Number(t, len(result)).Equal(tc.expected)
		}
	}

	t.Run("all", runTest(testCase{
		name:     "all",
		cmdName:  "all",
		args:     "",
		expected: 3,
	}))

	t.Run("from to", runTest(testCase{
		name:     "from to",
		cmdName:  "from",
		args:     fixedTime.Add(-2*time.Hour).Format("15:04") + " to " + fixedTime.Add(-90*time.Minute).Format("15:04"),
		expected: 1,
	}))

	t.Run("after", runTest(testCase{
		name:     "after",
		cmdName:  "after",
		args:     fixedTime.Add(-90 * time.Minute).Format("15:04"),
		expected: 2,
	}))

	t.Run("since", runTest(testCase{
		name:     "since",
		cmdName:  "since",
		args:     "90m",
		expected: 2,
	}))
}

func TestParseTime(t *testing.T) {
	type testCase struct {
		input    string
		expected time.Time
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := clock.With(context.Background(), func() time.Time { return tc.expected })
			result, err := list.ParseTime(ctx, tc.input)
			gt.NoError(t, err)
			gt.Value(t, result.Format("15:04")).Equal(tc.expected.Format("15:04"))
		}
	}

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	t.Run("time format", runTest(testCase{
		input:    "14:30",
		expected: time.Date(now.Year(), now.Month(), now.Day(), 14, 30, 0, 0, now.Location()),
	}))

	t.Run("date format", runTest(testCase{
		input:    "2024-01-01",
		expected: time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local),
	}))
}

func TestParseDuration(t *testing.T) {
	type testCase struct {
		input    string
		expected time.Duration
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			result, err := list.ParseDuration(tc.input)
			gt.NoError(t, err)
			gt.Value(t, result).Equal(tc.expected)
		}
	}

	t.Run("minutes", runTest(testCase{
		input:    "30m",
		expected: 30 * time.Minute,
	}))

	t.Run("hours", runTest(testCase{
		input:    "2h",
		expected: 2 * time.Hour,
	}))

	t.Run("days", runTest(testCase{
		input:    "1d",
		expected: 24 * time.Hour,
	}))
}
