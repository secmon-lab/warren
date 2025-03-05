package list_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/mock"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/list"
	"github.com/secmon-lab/warren/pkg/service/source"
)

func TestService_Run(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		alerts   []model.Alert
		expected []model.Alert
		wantErr  bool
	}{
		{
			name: "filter by user",
			args: []string{"user", "<@U123>"},
			alerts: []model.Alert{
				{Assignee: &model.SlackUser{ID: "U123"}},
				{Assignee: &model.SlackUser{ID: "U456"}},
			},
			expected: []model.Alert{
				{Assignee: &model.SlackUser{ID: "U123"}},
			},
		},
		{
			name: "sort by created at",
			args: []string{"sort", "created_at"},
			alerts: []model.Alert{
				{CreatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)},
				{CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
			expected: []model.Alert{
				{CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
				{CreatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name:    "invalid command",
			args:    []string{"invalid"},
			wantErr: true,
		},
		{
			name: "limit alerts",
			args: []string{"limit", "1"},
			alerts: []model.Alert{
				{ID: "1"},
				{ID: "2"},
			},
			expected: []model.Alert{
				{ID: "1"},
			},
		},
		{
			name: "offset alerts",
			args: []string{"offset", "1"},
			alerts: []model.Alert{
				{ID: "1"},
				{ID: "2"},
			},
			expected: []model.Alert{
				{ID: "2"},
			},
		},
		{
			name: "filter by status",
			args: []string{"status", "new", "resolved"},
			alerts: []model.Alert{
				{Status: model.AlertStatusNew},
				{Status: model.AlertStatusAcknowledged},
				{Status: model.AlertStatusResolved},
			},
			expected: []model.Alert{
				{Status: model.AlertStatusNew},
				{Status: model.AlertStatusResolved},
			},
		},
		{
			name:    "invalid status",
			args:    []string{"status", "invalid"},
			wantErr: true,
		},
		{
			name: "status pipeline",
			args: []string{"status", "new", "|", "status", "resolved"},
			alerts: []model.Alert{
				{Status: model.AlertStatusNew},
				{Status: model.AlertStatusAcknowledged},
				{Status: model.AlertStatusResolved},
			},
			expected: nil,
		},
		{
			name: "limit offset pipeline",
			args: []string{"limit", "2", "|", "offset", "1"},
			alerts: []model.Alert{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
			},
			expected: []model.Alert{
				{ID: "2"},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemory()
			svc := list.New(repo)
			var alertList *model.AlertList
			th := mock.SlackThreadServiceMock{
				ReplyFunc: func(ctx context.Context, message string) {},
				PostAlertListFunc: func(ctx context.Context, list *model.AlertList) error {
					alertList = list
					return nil
				},
				ChannelIDFunc: func() string {
					return "C123"
				},
				ThreadIDFunc: func() string {
					return "T123"
				},
			}
			args := append([]string{"|"}, tt.args...)
			err := svc.Run(t.Context(), &th, &model.SlackUser{}, source.Static(tt.alerts), args)
			if tt.wantErr {
				gt.Error(t, err)
				return
			}

			gt.NoError(t, err).Must()
			gt.Equal(t, tt.expected, alertList.Alerts)
		})
	}
}
