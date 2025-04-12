package repository

import (
	"context"
	"math"
	"slices"
	"sort"
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
	histories   map[types.SessionID][]*session.History
	policyDiffs map[types.PolicyDiffID]*policy.Diff
	sessions    map[types.SessionID]session.Session
	notes       map[types.SessionID][]*session.Note
}

func NewMemory() *Memory {
	return &Memory{
		alerts:      make(map[types.AlertID]alert.Alert),
		comments:    make(map[types.AlertID][]alert.AlertComment),
		lists:       make(map[types.AlertListID]alert.List),
		histories:   make(map[types.SessionID][]*session.History),
		policyDiffs: make(map[types.PolicyDiffID]*policy.Diff),
		sessions:    make(map[types.SessionID]session.Session),
		notes:       make(map[types.SessionID][]*session.Note),
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

func (r *Memory) GetLatestHistory(ctx context.Context, sessionID types.SessionID) (*session.History, error) {
	histories := r.histories[sessionID]
	if len(histories) == 0 {
		return nil, nil
	}
	return histories[len(histories)-1], nil
}

func (r *Memory) PutHistory(ctx context.Context, sessionID types.SessionID, history *session.History) error {
	r.histories[sessionID] = append(r.histories[sessionID], history)
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

func (r *Memory) GetAlertsByStatus(ctx context.Context, status ...types.AlertStatus) (alert.Alerts, error) {
	var alerts alert.Alerts
	for _, alert := range r.alerts {
		if slices.Contains(status, alert.Status) {
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

func (r *Memory) FindNearestAlerts(ctx context.Context, embedding []float32, limit int) (alert.Alerts, error) {
	if len(embedding) == 0 {
		return nil, goerr.New("embedding vector is empty")
	}

	type alertWithDistance struct {
		alert    alert.Alert
		distance float32
	}

	var alertsWithDistance []alertWithDistance
	for _, alert := range r.alerts {
		if len(alert.Embedding) == 0 {
			continue
		}
		if len(alert.Embedding) != len(embedding) {
			continue
		}

		// Calculate cosine distance (1 - cosine similarity)
		var dotProduct float32
		var normA, normB float32
		for i := 0; i < len(embedding); i++ {
			dotProduct += embedding[i] * alert.Embedding[i]
			normA += embedding[i] * embedding[i]
			normB += alert.Embedding[i] * alert.Embedding[i]
		}
		normA = float32(math.Sqrt(float64(normA)))
		normB = float32(math.Sqrt(float64(normB)))

		if normA == 0 || normB == 0 {
			continue
		}

		cosineSimilarity := dotProduct / (normA * normB)
		distance := 1 - cosineSimilarity

		alertsWithDistance = append(alertsWithDistance, alertWithDistance{
			alert:    alert,
			distance: distance,
		})
	}

	// Sort by distance
	sort.Slice(alertsWithDistance, func(i, j int) bool {
		return alertsWithDistance[i].distance < alertsWithDistance[j].distance
	})

	// Take top N alerts
	var result alert.Alerts
	for i := 0; i < len(alertsWithDistance) && i < limit; i++ {
		result = append(result, &alertsWithDistance[i].alert)
	}

	return result, nil
}

func (r *Memory) GetNotes(ctx context.Context, sessionID types.SessionID) ([]*session.Note, error) {
	notes, ok := r.notes[sessionID]
	if !ok {
		return nil, nil
	}
	// 時系列順にソート
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].CreatedAt.Before(notes[j].CreatedAt)
	})
	return notes, nil
}

func (r *Memory) PutNote(ctx context.Context, note *session.Note) error {
	r.notes[note.SessionID] = append(r.notes[note.SessionID], note)
	return nil
}

func (r *Memory) SearchAlerts(ctx context.Context, path, op string, value any) (alert.Alerts, error) {
	var alerts alert.Alerts

	for _, a := range r.alerts {
		var match bool

		switch path {
		case "Status":
			status, ok := value.(types.AlertStatus)
			if !ok {
				return nil, goerr.New("invalid status type", goerr.V("value", value))
			}
			switch op {
			case "==":
				match = a.Status == status
			case "!=":
				match = a.Status != status
			default:
				return nil, goerr.New("unsupported operator for status", goerr.V("op", op))
			}

		case "Title":
			title, ok := value.(string)
			if !ok {
				return nil, goerr.New("invalid title type", goerr.V("value", value))
			}
			switch op {
			case "==":
				match = a.Title == title
			case "!=":
				match = a.Title != title
			default:
				return nil, goerr.New("unsupported operator for title", goerr.V("op", op))
			}

		case "CreatedAt":
			createdAt, ok := value.(time.Time)
			if !ok {
				return nil, goerr.New("invalid created_at type", goerr.V("value", value))
			}
			switch op {
			case "==":
				match = a.CreatedAt.Equal(createdAt)
			case "!=":
				match = !a.CreatedAt.Equal(createdAt)
			case ">":
				match = a.CreatedAt.After(createdAt)
			case "<":
				match = a.CreatedAt.Before(createdAt)
			case ">=":
				match = a.CreatedAt.After(createdAt) || a.CreatedAt.Equal(createdAt)
			case "<=":
				match = a.CreatedAt.Before(createdAt) || a.CreatedAt.Equal(createdAt)
			default:
				return nil, goerr.New("unsupported operator for created_at", goerr.V("op", op))
			}

		default:
			// 未知のフィールドパスの場合は空の結果を返す
			continue
		}

		if match {
			alerts = append(alerts, &a)
		}
	}

	return alerts, nil
}
