package repository

import (
	"context"
	"sort"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model"
)

type Memory struct {
	alerts      map[model.AlertID]model.Alert
	comments    map[model.AlertID][]model.AlertComment
	policies    map[string]model.PolicyData
	policyDiffs map[model.PolicyDiffID]model.PolicyDiff
	alertLists  map[model.AlertListID]model.AlertList
}

var _ interfaces.Repository = &Memory{}

func NewMemory() *Memory {
	return &Memory{
		alerts:      make(map[model.AlertID]model.Alert),
		comments:    make(map[model.AlertID][]model.AlertComment),
		policies:    make(map[string]model.PolicyData),
		policyDiffs: make(map[model.PolicyDiffID]model.PolicyDiff),
		alertLists:  make(map[model.AlertListID]model.AlertList),
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

func (r *Memory) GetAlerts(ctx context.Context, duration time.Duration, limit int64, offset int64) ([]model.Alert, error) {
	var alerts []model.Alert
	for _, alert := range r.alerts {
		if alert.CreatedAt.After(time.Now().Add(-duration)) {
			alerts = append(alerts, alert)
		}
	}

	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].CreatedAt.After(alerts[j].CreatedAt)
	})

	if offset > 0 && int(offset) < len(alerts) {
		alerts = alerts[int(offset):]
	}

	if len(alerts) > int(limit) {
		alerts = alerts[:int(limit)]
	}

	return alerts, nil
}

func (r *Memory) GetAlertsBySlackThread(ctx context.Context, thread model.SlackThread) ([]model.Alert, error) {
	var alerts []model.Alert
	for _, alert := range r.alerts {
		if alert.SlackThread != nil && alert.SlackThread.ChannelID == thread.ChannelID && alert.SlackThread.ThreadID == thread.ThreadID {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (r *Memory) GetLatestAlerts(ctx context.Context, oldest time.Time, limit int) ([]model.Alert, error) {
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

func (r *Memory) GetAlertsWithoutStatus(ctx context.Context, status model.AlertStatus) ([]model.Alert, error) {
	var alerts []model.Alert
	for _, alert := range r.alerts {
		if alert.Status != status {
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

func (r *Memory) GetAlertsByParentID(ctx context.Context, parentID model.AlertID) ([]model.Alert, error) {
	var alerts []model.Alert
	for _, alert := range r.alerts {
		if alert.ParentID == parentID {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (r *Memory) GetPolicyDiff(ctx context.Context, id model.PolicyDiffID) (*model.PolicyDiff, error) {
	diff, ok := r.policyDiffs[id]
	if !ok {
		return nil, nil
	}
	return &diff, nil
}

func (r *Memory) PutPolicyDiff(ctx context.Context, diff *model.PolicyDiff) error {
	r.policyDiffs[diff.ID] = *diff
	return nil
}

func (r *Memory) GetAlertListByThread(ctx context.Context, thread model.SlackThread) (*model.AlertList, error) {
	for _, list := range r.alertLists {
		if list.SlackThread.ChannelID == thread.ChannelID && list.SlackThread.ThreadID == thread.ThreadID {
			return &list, nil
		}
	}
	return nil, nil
}

func (r *Memory) GetAlertList(ctx context.Context, listID model.AlertListID) (*model.AlertList, error) {
	list, ok := r.alertLists[listID]
	if !ok {
		return nil, goerr.New("alert list not found", goerr.V("list_id", listID))
	}
	return &list, nil
}

func (r *Memory) PutAlertList(ctx context.Context, list model.AlertList) error {
	r.alertLists[list.ID] = list
	return nil
}

func (r *Memory) GetAlertsBySpan(ctx context.Context, begin, end time.Time) ([]model.Alert, error) {
	var alerts []model.Alert
	for _, alert := range r.alerts {
		if alert.CreatedAt.After(begin) && alert.CreatedAt.Before(end) {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (r *Memory) GetLatestAlertListInThread(ctx context.Context, thread model.SlackThread) (*model.AlertList, error) {
	var latestList *model.AlertList
	var latestTime time.Time

	for _, list := range r.alertLists {
		if list.SlackThread != nil && list.SlackThread.ChannelID == thread.ChannelID && list.SlackThread.ThreadID == thread.ThreadID {
			if latestList == nil || list.CreatedAt.After(latestTime) {
				latestList = &list
				latestTime = list.CreatedAt
			}
		}
	}

	if latestList == nil {
		return nil, nil
	}

	latestList.Alerts = nil
	return latestList, nil
}

func (r *Memory) BatchUpdateAlertStatus(ctx context.Context, alertIDs []model.AlertID, status model.AlertStatus) error {
	for _, alertID := range alertIDs {
		alert, ok := r.alerts[alertID]
		if !ok {
			return goerr.New("alert not found", goerr.V("alert_id", alertID))
		}
		alert.Status = status
		r.alerts[alertID] = alert
	}
	return nil
}

func (r *Memory) BatchUpdateAlertConclusion(ctx context.Context, alertIDs []model.AlertID, conclusion model.AlertConclusion, reason string) error {
	for _, alertID := range alertIDs {
		alert, ok := r.alerts[alertID]
		if !ok {
			return goerr.New("alert not found", goerr.V("alert_id", alertID))
		}
		alert.Conclusion = conclusion
		alert.Reason = reason
		r.alerts[alertID] = alert
	}
	return nil
}
