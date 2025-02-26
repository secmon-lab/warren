package repository

import (
	"context"
	"sort"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
)

type Memory struct {
	alerts      map[model.AlertID]model.Alert
	comments    map[model.AlertID][]model.AlertComment
	policies    map[string]model.PolicyData
	alertGroups map[model.AlertGroupID]model.AlertGroup
}

var _ interfaces.Repository = &Memory{}

func NewMemory() *Memory {
	return &Memory{
		alerts:      make(map[model.AlertID]model.Alert),
		comments:    make(map[model.AlertID][]model.AlertComment),
		policies:    make(map[string]model.PolicyData),
		alertGroups: make(map[model.AlertGroupID]model.AlertGroup),
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
	return nil, nil
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

func (r *Memory) InsertAlertComment(ctx context.Context, comment model.AlertComment) error {
	r.comments[comment.AlertID] = append(r.comments[comment.AlertID], comment)
	return nil
}

func (r *Memory) GetAlertComments(ctx context.Context, alertID model.AlertID) ([]model.AlertComment, error) {
	comments, ok := r.comments[alertID]
	if !ok {
		return nil, goerr.New("comments not found", goerr.V("alert_id", alertID))
	}

	// Sort by timestamp in descending order
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].Timestamp > comments[j].Timestamp
	})

	return comments, nil
}

func (r *Memory) GetPolicy(ctx context.Context, hash string) (*model.PolicyData, error) {
	policy, ok := r.policies[hash]
	if !ok {
		return nil, nil
	}
	return &policy, nil
}

func (r *Memory) SavePolicy(ctx context.Context, policy *model.PolicyData) error {
	r.policies[policy.Hash] = *policy
	return nil
}

func (r *Memory) GetAlertsByStatus(ctx context.Context, status model.AlertStatus) ([]model.Alert, error) {
	var alerts []model.Alert
	for _, alert := range r.alerts {
		if alert.Status == status {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (r *Memory) BatchGetAlerts(ctx context.Context, alertIDs []model.AlertID) ([]model.Alert, error) {
	var alerts []model.Alert
	for _, alertID := range alertIDs {
		alert, ok := r.alerts[alertID]
		if !ok {
			return nil, goerr.New("alert not found", goerr.V("alert_id", alertID))
		}
		alerts = append(alerts, alert)
	}
	return alerts, nil
}

func (r *Memory) PutAlertGroups(ctx context.Context, groups []model.AlertGroup) error {
	for _, group := range groups {
		r.alertGroups[group.ID] = group
	}
	return nil
}

func (r *Memory) GetAlertGroup(ctx context.Context, groupID model.AlertGroupID) (*model.AlertGroup, error) {
	group, ok := r.alertGroups[groupID]
	if !ok {
		return nil, nil
	}
	return &group, nil
}
