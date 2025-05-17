package repository

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type Memory struct {
	mu sync.RWMutex

	alerts         map[types.AlertID]*alert.Alert
	lists          map[types.AlertListID]*alert.List
	histories      map[types.SessionID][]*session.History
	sessions       map[types.SessionID]*session.Session
	tickets        map[types.TicketID]*ticket.Ticket
	ticketComments map[types.TicketID][]ticket.Comment
}

func NewMemory() *Memory {
	return &Memory{
		alerts:         make(map[types.AlertID]*alert.Alert),
		lists:          make(map[types.AlertListID]*alert.List),
		histories:      make(map[types.SessionID][]*session.History),
		sessions:       make(map[types.SessionID]*session.Session),
		tickets:        make(map[types.TicketID]*ticket.Ticket),
		ticketComments: make(map[types.TicketID][]ticket.Comment),
	}
}

func (r *Memory) PutAlert(ctx context.Context, alert alert.Alert) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.alerts[alert.ID] = &alert
	return nil
}

func (r *Memory) GetAlert(ctx context.Context, alertID types.AlertID) (*alert.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	alert, ok := r.alerts[alertID]
	if !ok {
		return nil, goerr.New("alert not found", goerr.V("alert_id", alertID))
	}
	return alert, nil
}

func (r *Memory) GetAlertByThread(ctx context.Context, thread slack.Thread) (*alert.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, alert := range r.alerts {
		if alert.SlackThread.ChannelID == thread.ChannelID && alert.SlackThread.ThreadID == thread.ThreadID {
			return alert, nil
		}
	}
	return nil, nil
}

func (r *Memory) GetLatestHistory(ctx context.Context, sessionID types.SessionID) (*session.History, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	histories, ok := r.histories[sessionID]
	if !ok || len(histories) == 0 {
		return nil, nil
	}

	latest := histories[0]
	for _, h := range histories[1:] {
		if h.CreatedAt.After(latest.CreatedAt) {
			latest = h
		}
	}
	return latest, nil
}

func (r *Memory) PutHistory(ctx context.Context, sessionID types.SessionID, history *session.History) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.histories[sessionID] = append(r.histories[sessionID], history)
	return nil
}

func (r *Memory) GetAlertList(ctx context.Context, listID types.AlertListID) (*alert.List, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list, ok := r.lists[listID]
	if !ok {
		return nil, nil
	}
	return list, nil
}

func (r *Memory) PutAlertList(ctx context.Context, list alert.List) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lists[list.ID] = &list
	return nil
}

func (r *Memory) GetAlertListByThread(ctx context.Context, thread slack.Thread) (*alert.List, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, list := range r.lists {
		if list.SlackThread.ChannelID == thread.ChannelID && list.SlackThread.ThreadID == thread.ThreadID {
			return list, nil
		}
	}
	return nil, nil
}

func (r *Memory) GetLatestAlertListInThread(ctx context.Context, thread slack.Thread) (*alert.List, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var latestList *alert.List
	for _, list := range r.lists {
		if list.SlackThread.ChannelID == thread.ChannelID && list.SlackThread.ThreadID == thread.ThreadID {
			if latestList == nil || list.CreatedAt.After(latestList.CreatedAt) {
				latestList = list
			}
		}
	}
	return latestList, nil
}

func (r *Memory) GetSession(ctx context.Context, id types.SessionID) (*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.sessions[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}

func (r *Memory) GetSessionByThread(ctx context.Context, thread slack.Thread) (*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.sessions {
		if s.Thread.ChannelID == thread.ChannelID && s.Thread.ThreadID == thread.ThreadID {
			return s, nil
		}
	}
	return nil, nil
}

func (r *Memory) PutSession(ctx context.Context, s session.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[s.ID] = &s
	return nil
}

func (r *Memory) GetTicket(ctx context.Context, ticketID types.TicketID) (*ticket.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tickets[ticketID]
	if !ok {
		return nil, goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
	}
	return t, nil
}

func (r *Memory) PutTicket(ctx context.Context, t ticket.Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tickets[t.ID] = &t
	return nil
}

func (r *Memory) PutTicketComment(ctx context.Context, comment ticket.Comment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ticketComments[comment.TicketID] = append(r.ticketComments[comment.TicketID], comment)
	return nil
}

func (r *Memory) GetTicketComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comments, ok := r.ticketComments[ticketID]
	if !ok {
		return []ticket.Comment{}, nil
	}
	return comments, nil
}

func (r *Memory) BindAlertToTicket(ctx context.Context, alertID types.AlertID, ticketID types.TicketID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	alert, ok := r.alerts[alertID]
	if !ok {
		return goerr.New("alert not found", goerr.V("alert_id", alertID))
	}

	alert.TicketID = ticketID
	return nil
}

func (r *Memory) UnbindAlertFromTicket(ctx context.Context, alertID types.AlertID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	alert, ok := r.alerts[alertID]
	if !ok {
		return goerr.New("alert not found", goerr.V("alert_id", alertID))
	}

	alert.TicketID = types.EmptyTicketID
	return nil
}

func (r *Memory) GetAlertWithoutTicket(ctx context.Context) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, alert := range r.alerts {
		if alert.TicketID == types.EmptyTicketID {
			alerts = append(alerts, alert)
		}
	}
	if alerts == nil {
		return alert.Alerts{}, nil
	}
	return alerts, nil
}

func (r *Memory) GetAlertsBySpan(ctx context.Context, begin, end time.Time) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, alert := range r.alerts {
		if alert.CreatedAt.After(begin) && alert.CreatedAt.Before(end) {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

func (r *Memory) SearchAlerts(ctx context.Context, path, op string, value any, limit int) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, alert := range r.alerts {
		if limit > 0 && len(alerts) >= limit {
			break
		}

		// リフレクションを使用して動的にフィールドにアクセス
		alertValue := reflect.ValueOf(alert).Elem()
		fieldValue := alertValue.FieldByName(path)
		if !fieldValue.IsValid() {
			continue
		}

		// ポインタ型の場合は、その値を取得
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				continue
			}
			fieldValue = fieldValue.Elem()
		}

		// 比較演算子に基づいて値を比較
		var cmpValue any = value
		if fieldValue.Type() != reflect.TypeOf(value) {
			// string型→type alias(string)などの変換
			if fieldValue.Type().Kind() == reflect.String && reflect.TypeOf(value).Kind() == reflect.String {
				cmpValue = fieldValue.Convert(fieldValue.Type()).Interface()
			} else if fieldValue.Type().Kind() == reflect.Int && reflect.TypeOf(value).Kind() == reflect.Int {
				cmpValue = fieldValue.Convert(fieldValue.Type()).Interface()
			}
		}
		switch op {
		case "==":
			if reflect.DeepEqual(fieldValue.Interface(), cmpValue) {
				alerts = append(alerts, alert)
			}
		case "!=":
			if !reflect.DeepEqual(fieldValue.Interface(), cmpValue) {
				alerts = append(alerts, alert)
			}
		case ">":
			if fieldValue.Interface().(time.Time).After(value.(time.Time)) {
				alerts = append(alerts, alert)
			}
		case ">=":
			if fieldValue.Interface().(time.Time).After(value.(time.Time)) || reflect.DeepEqual(fieldValue.Interface(), cmpValue) {
				alerts = append(alerts, alert)
			}
		case "<":
			if fieldValue.Interface().(time.Time).Before(value.(time.Time)) {
				alerts = append(alerts, alert)
			}
		case "<=":
			if fieldValue.Interface().(time.Time).Before(value.(time.Time)) || reflect.DeepEqual(fieldValue.Interface(), cmpValue) {
				alerts = append(alerts, alert)
			}
		}
	}
	return alerts, nil
}

func (r *Memory) BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, id := range alertIDs {
		if alert, ok := r.alerts[id]; ok {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}
func (r *Memory) FindSimilarAlerts(ctx context.Context, target alert.Alert, limit int) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, a := range r.alerts {
		// Skip the same alert
		if a.ID == target.ID {
			continue
		}

		// Only add alerts that have embeddings
		if len(a.Embedding) > 0 {
			alerts = append(alerts, a)
		}
	}

	// Sort by similarity
	sort.Slice(alerts, func(i, j int) bool {
		simI := alert.CosineSimilarity(alerts[i].Embedding, target.Embedding)
		simJ := alert.CosineSimilarity(alerts[j].Embedding, target.Embedding)
		return simI > simJ
	})

	// Apply limit
	if limit > 0 && limit < len(alerts) {
		alerts = alerts[:limit]
	}

	return alerts, nil
}

func (r *Memory) GetTicketByThread(ctx context.Context, thread slack.Thread) (*ticket.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, t := range r.tickets {
		if t.SlackThread != nil && t.SlackThread.ChannelID == thread.ChannelID && t.SlackThread.ThreadID == thread.ThreadID {
			return t, nil
		}
	}
	return nil, nil
}

func (r *Memory) BatchBindAlertsToTicket(ctx context.Context, alertIDs []types.AlertID, ticketID types.TicketID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, alertID := range alertIDs {
		alert, ok := r.alerts[alertID]
		if !ok {
			return goerr.New("alert not found", goerr.V("alert_id", alertID))
		}
		alert.TicketID = ticketID
	}
	return nil
}
