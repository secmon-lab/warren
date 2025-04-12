package list_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/command/list"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func TestShowHelp(t *testing.T) {
	svc := slack_svc.NewTestService(t)

	th, err := svc.PostMessage(t.Context(), "test help")
	gt.NoError(t, err).Required()

	ctx := msg.With(t.Context(), th.Reply, th.NewStateFunc)
	list.ShowHelp(ctx)
}

func TestService_Run(t *testing.T) {
	// Setup test data
	ctx := context.Background()
	repo := repository.NewMemory()
	svc := list.New(repo)
	slackService := slack_svc.NewTestService(t)
	th, err := slackService.PostMessage(ctx, "test message")
	gt.NoError(t, err).Required()
	slackThread := th
	user := &slack.User{
		ID:   "U0123456789",
		Name: "Test User",
	}

	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Create test alerts
	alerts := []*alert.Alert{
		{
			ID:          types.NewAlertID(),
			Title:       "Alert 1",
			Description: "Test alert 1",
			Status:      types.AlertStatusNew,
			CreatedAt:   fixedTime.Add(-1 * time.Hour),
			UpdatedAt:   fixedTime.Add(-1 * time.Hour),
			Data:        map[string]interface{}{"color": "blue"},
			SlackThread: &slack.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
		},
		{
			ID:          types.NewAlertID(),
			Title:       "Alert 2",
			Description: "Test alert 2 with grep match",
			Status:      types.AlertStatusNew,
			CreatedAt:   fixedTime.Add(-2 * time.Hour),
			UpdatedAt:   fixedTime.Add(-2 * time.Hour),
			Data:        map[string]interface{}{"color": "orange"},
			SlackThread: &slack.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
		},
		{
			ID:          types.NewAlertID(),
			Title:       "Alert 3",
			Description: "Test alert 3",
			Status:      types.AlertStatusResolved,
			CreatedAt:   fixedTime.Add(-3 * time.Hour),
			UpdatedAt:   fixedTime.Add(-3 * time.Hour),
			Data:        map[string]interface{}{"color": "red"},
			SlackThread: &slack.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
		},
	}

	// Store alerts in repository
	for _, a := range alerts {
		err := repo.PutAlert(ctx, *a)
		gt.NoError(t, err)
	}

	// Create and store an alert list for testing
	testList := alert.NewList(ctx, slack.Thread{
		ChannelID: "C0123456789",
		ThreadID:  "T0123456789",
	}, user, alert.Alerts{alerts[0], alerts[1]})
	err = repo.PutAlertList(ctx, testList)
	gt.NoError(t, err)

	tests := []struct {
		name          string
		input         string
		expectedError string
		validate      func(*testing.T, *alert.List)
	}{
		{
			name:  "show unresolved alerts",
			input: "unresolved",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(2).Required()
				for _, a := range list.Alerts {
					gt.Value(t, a.Status).Equal(types.AlertStatusNew)
				}
				gt.Array(t, list.AlertIDs).Has(alerts[0].ID)
				gt.Array(t, list.AlertIDs).Has(alerts[1].ID)
			},
		},
		{
			name:  "show alerts with status filter",
			input: "status resolved",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(1)
				gt.Value(t, list.Alerts[0].Status).Equal(types.AlertStatusResolved)
				gt.Value(t, list.Alerts[0].ID).Equal(alerts[2].ID)
			},
		},
		{
			name:  "show alerts with time span",
			input: "between 2024-01-01 2024-01-02",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(3)
			},
		},
		{
			name:  "show no alerts with time span",
			input: "between 2024-01-03 2024-01-04",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(0)
			},
		},
		{
			name:  "show alerts with limit",
			input: "unresolved | limit 1",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(1)
				gt.Array(t, []types.AlertID{alerts[0].ID, alerts[1].ID}).Has(list.AlertIDs[0])
			},
		},
		{
			name:  "show alerts with offset",
			input: "unresolved | offset 1",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(1)
				gt.Array(t, []types.AlertID{alerts[0].ID, alerts[1].ID}).Has(list.AlertIDs[0])
			},
		},
		{
			name:  "show alerts with grep filter",
			input: "unresolved | grep orange",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(1)
				gt.Value(t, list.Alerts[0].ID).Equal(alerts[1].ID)
			},
		},
		{
			name:  "show alerts with sort by CreatedAt",
			input: "unresolved | sort CreatedAt",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(2)
				// Verify alerts are sorted by CreatedAt
				gt.Value(t, list.Alerts[0].CreatedAt.Before(list.Alerts[1].CreatedAt)).Equal(true)
				gt.Array(t, list.AlertIDs).Has(alerts[1].ID)
				gt.Array(t, list.AlertIDs).Has(alerts[0].ID)
			},
		},
		{
			name:  "show alerts with multiple pipeline actions",
			input: "unresolved | grep orange | sort CreatedAt | limit 1",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(1)
				gt.Value(t, list.Alerts[0].ID).Equal(alerts[1].ID)
			},
		},
		{
			name:  "show alerts from alert list ID",
			input: testList.ID.String(),
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(2)
				gt.Array(t, list.AlertIDs).Equal(testList.AlertIDs)
				gt.Value(t, list.CreatedBy).Equal(testList.CreatedBy)
			},
		},
		{
			name:          "error on invalid alert list ID",
			input:         "invalid-id",
			expectedError: "unknown command",
		},
		{
			name:  "empty input shows help",
			input: "",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(0)
			},
		},
		{
			name:  "too many spaces",
			input: "unresolved  | grep orange | sort CreatedAt | limit 1",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(1)
				gt.Value(t, list.Alerts[0].ID).Equal(alerts[1].ID)
			},
		},
		{
			name:  "too many spaces with between",
			input: "  between    2024-01-01    2024-01-02   |    grep orange | sort CreatedAt | limit 1",
			validate: func(t *testing.T, list *alert.List) {
				gt.Array(t, list.Alerts).Length(1)
				gt.Value(t, list.Alerts[0].ID).Equal(alerts[1].ID)
			},
		},
		{
			name:          "error on invalid command",
			input:         "invalid_command",
			expectedError: "unknown command",
		},
		{
			name:          "error on invalid pipeline action",
			input:         "unresolved | invalid_action",
			expectedError: "unknown action",
		},
		{
			name:          "error on invalid action argument",
			input:         "unresolved | limit invalid",
			expectedError: "limit: failed to convert limit to int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run test
			listID, err := svc.Run(ctx, slackThread, user, tt.input)

			// Validate error
			if tt.expectedError != "" {
				gt.Error(t, err)
				gt.String(t, err.Error()).Contains(tt.expectedError)
				return
			}

			// Validate success case
			gt.NoError(t, err)
			gt.Value(t, listID).NotEqual(types.EmptyAlertListID)

			// Get the list from repository
			list, err := repo.GetAlertList(ctx, listID)
			gt.NoError(t, err).Required()

			if tt.validate != nil {
				tt.validate(t, list)
			}
		})
	}
}
