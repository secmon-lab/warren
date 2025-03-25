package repository

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type Memory struct {
	alerts      map[types.AlertID]alert.Alert
	comments    map[types.AlertID][]alert.AlertComment
	lists       map[types.AlertListID]alert.List
	histories   map[types.SessionID]session.Histories
	policyDiffs map[types.PolicyDiffID]*policy.Diff
	sessions    map[types.SessionID]session.Session
}

func NewMemory() *Memory {
	return &Memory{
		alerts:      make(map[types.AlertID]alert.Alert),
		comments:    make(map[types.AlertID][]alert.AlertComment),
		lists:       make(map[types.AlertListID]alert.List),
		histories:   make(map[types.SessionID]session.Histories),
		policyDiffs: make(map[types.PolicyDiffID]*policy.Diff),
		sessions:    make(map[types.SessionID]session.Session),
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

func (r *Memory) GetHistory(ctx context.Context, sessionID types.SessionID) (session.Histories, error) {
	return r.histories[sessionID], nil
}

func (r *Memory) PutHistory(ctx context.Context, sessionID types.SessionID, histories session.Histories) error {
	r.histories[sessionID] = histories
	return nil
}

func (r *Memory) GetAlertList(ctx context.Context, listID types.AlertListID) (*alert.List, error) {
	list, ok := r.lists[listID]
	if !ok {
		return nil, nil
	}
	return &list, nil
}

func (r *Memory) PutAlertList(ctx context.Context, list alert.List) error {
	r.lists[list.ID] = list
	return nil
}

func (r *Memory) GetAlertListByThread(ctx context.Context, thread slack.Thread) (*alert.List, error) {
	for _, list := range r.lists {
		if list.SlackThread.ChannelID == thread.ChannelID && list.SlackThread.ThreadID == thread.ThreadID {
			return &list, nil
		}
	}
	return nil, nil
}

func (r *Memory) GetLatestAlertListInThread(ctx context.Context, thread slack.Thread) (*alert.List, error) {
	var latestList *alert.List
	var latestTime time.Time

	for _, list := range r.lists {
		if list.SlackThread.ChannelID == thread.ChannelID && list.SlackThread.ThreadID == thread.ThreadID {
			if latestList == nil || list.CreatedAt.After(latestTime) {
				latestList = &list
				latestTime = list.CreatedAt
			}
		}
	}
	return latestList, nil
}

func (r *Memory) GetAlertsByStatus(ctx context.Context, status types.AlertStatus) (alert.Alerts, error) {
	var alerts alert.Alerts
	for _, alert := range r.alerts {
		if alert.Status == status {
			alerts = append(alerts, &alert)
		}
	}
	return alerts, nil
}

func (r *Memory) GetAlertsWithoutStatus(ctx context.Context, status types.AlertStatus) (alert.Alerts, error) {
	var alerts alert.Alerts
	for _, alert := range r.alerts {
		if alert.Status != status {
			alerts = append(alerts, &alert)
		}
	}
	return alerts, nil
}

func (r *Memory) GetAlertsBySpan(ctx context.Context, begin, end time.Time) (alert.Alerts, error) {
	var alerts alert.Alerts
	for _, alert := range r.alerts {
		if alert.CreatedAt.After(begin) && alert.CreatedAt.Before(end) {
			alerts = append(alerts, &alert)
		}
	}
	return alerts, nil
}

func (r *Memory) BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) (alert.Alerts, error) {
	var alerts alert.Alerts
	for _, alertID := range alertIDs {
		if alert, ok := r.alerts[alertID]; ok {
			alerts = append(alerts, &alert)
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
	return diff, nil
}

func (r *Memory) PutPolicyDiff(ctx context.Context, diff *policy.Diff) error {
	r.policyDiffs[types.PolicyDiffID(diff.ID)] = diff
	return nil
}

func (r *Memory) GetSession(ctx context.Context, id types.SessionID) (*session.Session, error) {
	s, ok := r.sessions[id]
	if !ok {
		return nil, nil
	}
	return &s, nil
}

func (r *Memory) GetSessionByThread(ctx context.Context, thread slack.Thread) (*session.Session, error) {
	for _, s := range r.sessions {
		if s.Thread.ChannelID == thread.ChannelID && s.Thread.ThreadID == thread.ThreadID {
			return &s, nil
		}
	}
	return nil, nil
}

func (r *Memory) PutSession(ctx context.Context, s session.Session) error {
	r.sessions[s.ID] = s
	return nil
}
