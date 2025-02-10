package repository

import (
	"context"
	"sort"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
)

type Memory struct {
	alerts map[model.AlertID]model.Alert
}

func NewMemory() *Memory {
	return &Memory{
		alerts: make(map[model.AlertID]model.Alert),
	}
}

func (r *Memory) PutAlert(ctx context.Context, alert model.Alert) error {
	r.alerts[alert.ID] = alert
	return nil
}

func (r *Memory) GetAlert(ctx context.Context, alertID model.AlertID) (*model.Alert, error) {
	alert, ok := r.alerts[alertID]
	if !ok {
		return nil, goerr.New("alert not found", goerr.V("alert_id", alertID))
	}
	return &alert, nil
}

func (r *Memory) GetAlertBySlackThread(ctx context.Context, thread model.SlackThread) (*model.Alert, error) {
	for _, alert := range r.alerts {
		if alert.SlackThread != nil && alert.SlackThread.ChannelID == thread.ChannelID && alert.SlackThread.ThreadID == thread.ThreadID {
			return &alert, nil
		}
	}
	return nil, goerr.New("alert not found", goerr.V("slack_thread", thread))
}

func (r *Memory) FetchLatestAlerts(ctx context.Context, oldest time.Time, limit int) ([]model.Alert, error) {
	var alerts []model.Alert
	for _, alert := range r.alerts {
		if alert.CreatedAt.After(oldest) {
			alerts = append(alerts, alert)
		}
	}

	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].CreatedAt.After(alerts[j].CreatedAt)
	})

	if len(alerts) > limit {
		alerts = alerts[:limit]
	}

	return alerts, nil
}
