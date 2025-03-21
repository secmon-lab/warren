package repository

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type Memory struct {
	alerts      map[types.AlertID]alert.Alert
	comments    map[types.AlertID][]alert.AlertComment
	alertLists  map[types.AlertListID]alert.List
	policyDiffs map[types.PolicyDiffID]policy.Diff
	histories   map[string]*chat.History
}

func NewMemory() *Memory {
	return &Memory{
		alerts:      make(map[types.AlertID]alert.Alert),
		comments:    make(map[types.AlertID][]alert.AlertComment),
		alertLists:  make(map[types.AlertListID]alert.List),
		policyDiffs: make(map[types.PolicyDiffID]policy.Diff),
		histories:   make(map[string]*chat.History),
	}
}

func (r *Memory) PutAlert(ctx context.Context, alert alert.Alert) error {
	alert.UpdatedAt = clock.Now(ctx)
	r.alerts[alert.ID] = alert
	return nil
}

func (r *Memory) GetAlert(ctx context.Context, alertID types.AlertID) (*alert.Alert, error) {
	alert, ok := r.alerts[alertID]
	if !ok {
		return nil, goerr.New("alert not found", goerr.V("alert_id", alertID))
	}
	return &alert, nil
}

func (r *Memory) GetAlertByThread(ctx context.Context, thread slack.Thread) (*alert.Alert, error) {
	for _, alert := range r.alerts {
		if alert.SlackThread.ChannelID == thread.ChannelID && alert.SlackThread.ThreadID == thread.ThreadID {
			return &alert, nil
		}
	}
	return nil, nil
}

func (r *Memory) PutAlertComment(ctx context.Context, comment alert.AlertComment) error {
	r.comments[comment.AlertID] = append(r.comments[comment.AlertID], comment)
	return nil
}

func (r *Memory) GetAlertComments(ctx context.Context, alertID types.AlertID) ([]alert.AlertComment, error) {
	return r.comments[alertID], nil
}

func (r *Memory) GetHistory(ctx context.Context, thread slack.Thread) (*chat.History, error) {
	key := thread.ChannelID + "_" + thread.ThreadID
	return r.histories[key], nil
}

func (r *Memory) PutHistory(ctx context.Context, history chat.History) error {
	key := history.Thread.ChannelID + "_" + history.Thread.ThreadID
	r.histories[key] = &history
	return nil
}

func (r *Memory) GetAlertList(ctx context.Context, listID types.AlertListID) (*alert.List, error) {
	list, ok := r.alertLists[listID]
	if !ok {
		return nil, nil
	}
	return &list, nil
}

func (r *Memory) PutAlertList(ctx context.Context, list alert.List) error {
	r.alertLists[list.ID] = list
	return nil
}

func (r *Memory) GetAlertListByThread(ctx context.Context, thread slack.Thread) (*alert.List, error) {
	for _, list := range r.alertLists {
		if list.SlackThread.ChannelID == thread.ChannelID && list.SlackThread.ThreadID == thread.ThreadID {
			return &list, nil
		}
	}
	return nil, nil
}

func (r *Memory) GetLatestAlertListInThread(ctx context.Context, thread slack.Thread) (*alert.List, error) {
	var latestList *alert.List
	var latestTime time.Time

	for _, list := range r.alertLists {
		if list.SlackThread.ChannelID == thread.ChannelID && list.SlackThread.ThreadID == thread.ThreadID {
			if latestList == nil || list.CreatedAt.After(latestTime) {
				latestList = &list
				latestTime = list.CreatedAt
			}
		}
	}
	return latestList, nil
}

func (r *Memory) GetAlertsByStatus(ctx context.Context, status types.AlertStatus) ([]alert.Alert, error) {
	var alerts []alert.Alert
	for _, alert := range r.alerts {
		if alert.Status == status {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (r *Memory) GetAlertsBySpan(ctx context.Context, begin, end time.Time) ([]alert.Alert, error) {
	var alerts []alert.Alert
	for _, alert := range r.alerts {
		if alert.CreatedAt.After(begin) && alert.CreatedAt.Before(end) {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (r *Memory) BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) ([]alert.Alert, error) {
	var alerts []alert.Alert
	for _, alertID := range alertIDs {
		if alert, ok := r.alerts[alertID]; ok {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (r *Memory) BatchUpdateAlertStatus(ctx context.Context, alertIDs []types.AlertID, status types.AlertStatus, reason string) error {
	for _, alertID := range alertIDs {
		if alert, ok := r.alerts[alertID]; ok {
			alert.Status = status
			alert.Reason = reason
			r.alerts[alertID] = alert
		}
	}
	return nil
}

func (r *Memory) GetPolicyDiff(ctx context.Context, id types.PolicyDiffID) (*policy.Diff, error) {
	diff, ok := r.policyDiffs[id]
	if !ok {
		return nil, nil
	}
	return &diff, nil
}

func (r *Memory) PutPolicyDiff(ctx context.Context, diff *policy.Diff) error {
	r.policyDiffs[types.PolicyDiffID(diff.ID)] = *diff
	return nil
}
