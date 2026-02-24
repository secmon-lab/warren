package memory

import (
	"context"
	"math"
	"reflect"
	"sort"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func (r *Memory) PutAlert(ctx context.Context, alert alert.Alert) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.alerts[alert.ID] = &alert
	return nil
}

func (r *Memory) GetAlert(ctx context.Context, alertID types.AlertID) (*alert.Alert, error) {
	r.incrementCallCount("GetAlert")
	r.mu.RLock()
	defer r.mu.RUnlock()

	alert, ok := r.alerts[alertID]
	if !ok {
		return nil, goerr.New("alert not found", goerr.V("alert_id", alertID))
	}
	return alert, nil
}

func (r *Memory) GetLatestAlertByThread(ctx context.Context, thread slack.Thread) (*alert.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest *alert.Alert
	for _, alert := range r.alerts {
		if alert.SlackThread.ChannelID == thread.ChannelID && alert.SlackThread.ThreadID == thread.ThreadID {
			if latest == nil || alert.CreatedAt.After(latest.CreatedAt) {
				latest = alert
			}
		}
	}
	return latest, nil
}

func (r *Memory) GetAlertsByThread(ctx context.Context, thread slack.Thread) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, alert := range r.alerts {
		if alert.SlackThread != nil && alert.SlackThread.ChannelID == thread.ChannelID && alert.SlackThread.ThreadID == thread.ThreadID {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
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

func (r *Memory) PutAlertList(ctx context.Context, list *alert.List) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lists[list.ID] = list
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

func (r *Memory) GetAlertListsInThread(ctx context.Context, thread slack.Thread) ([]*alert.List, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var lists []*alert.List
	for _, list := range r.lists {
		if list.SlackThread.ChannelID == thread.ChannelID && list.SlackThread.ThreadID == thread.ThreadID {
			lists = append(lists, list)
		}
	}

	// Sort by CreatedAt in ascending order (oldest first)
	sort.Slice(lists, func(i, j int) bool {
		return lists[i].CreatedAt.Before(lists[j].CreatedAt)
	})

	return lists, nil
}

func (r *Memory) GetAlertWithoutTicket(ctx context.Context, offset, limit int) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, a := range r.alerts {
		a.Normalize()
		if a.TicketID == types.EmptyTicketID && (a.Status == alert.AlertStatusActive || a.Status == "") {
			alerts = append(alerts, a)
		}
	}

	// Apply offset
	if offset >= len(alerts) {
		return alert.Alerts{}, nil
	}
	if offset > 0 {
		alerts = alerts[offset:]
	}

	// Apply limit
	if limit > 0 && limit < len(alerts) {
		alerts = alerts[:limit]
	}

	if alerts == nil {
		return alert.Alerts{}, nil
	}
	return alerts, nil
}

func (r *Memory) CountAlertsWithoutTicket(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, a := range r.alerts {
		a.Normalize()
		if a.TicketID == types.EmptyTicketID && (a.Status == alert.AlertStatusActive || a.Status == "") {
			count++
		}
	}

	return count, nil
}

func (r *Memory) GetDeclinedAlerts(ctx context.Context, offset, limit int) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, a := range r.alerts {
		a.Normalize()
		if a.TicketID == types.EmptyTicketID && a.Status == alert.AlertStatusDeclined {
			alerts = append(alerts, a)
		}
	}

	// Apply offset
	if offset >= len(alerts) {
		return alert.Alerts{}, nil
	}
	if offset > 0 {
		alerts = alerts[offset:]
	}

	// Apply limit
	if limit > 0 && limit < len(alerts) {
		alerts = alerts[:limit]
	}

	if alerts == nil {
		return alert.Alerts{}, nil
	}
	return alerts, nil
}

func (r *Memory) CountDeclinedAlerts(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, a := range r.alerts {
		a.Normalize()
		if a.TicketID == types.EmptyTicketID && a.Status == alert.AlertStatusDeclined {
			count++
		}
	}

	return count, nil
}

func (r *Memory) UpdateAlertStatus(ctx context.Context, alertID types.AlertID, status alert.AlertStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	a, ok := r.alerts[alertID]
	if !ok {
		return goerr.New("alert not found", goerr.V("alert_id", alertID))
	}

	a.Status = status
	return nil
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

		// Use reflection to access fields dynamically
		alertValue := reflect.ValueOf(alert).Elem()
		fieldValue := alertValue.FieldByName(path)
		if !fieldValue.IsValid() {
			continue
		}

		// For pointer types, get the value
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				continue
			}
			fieldValue = fieldValue.Elem()
		}

		// Compare values based on comparison operator
		cmpValue := value
		if fieldValue.Type() != reflect.TypeOf(value) {
			// Convert from string type to type aliases like string
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
	r.incrementCallCount("BatchGetAlerts")
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

func (r *Memory) BatchPutAlerts(ctx context.Context, alerts alert.Alerts) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, alert := range alerts {
		r.alerts[alert.ID] = alert
	}
	return nil
}

func (r *Memory) GetAlertWithoutEmbedding(ctx context.Context) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, alert := range r.alerts {
		if len(alert.Embedding) == 0 {
			alerts = append(alerts, alert)
		}
	}
	if alerts == nil {
		return alert.Alerts{}, nil
	}
	return alerts, nil
}

func (r *Memory) GetAlertsWithInvalidEmbedding(ctx context.Context) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, alert := range r.alerts {
		if isInvalidEmbeddingMemory(alert.Embedding) {
			alerts = append(alerts, alert)
		}
	}
	if alerts == nil {
		return alert.Alerts{}, nil
	}
	return alerts, nil
}

// isInvalidEmbeddingMemory checks if embedding is invalid (helper for memory repository)
func isInvalidEmbeddingMemory(embedding []float32) bool {
	if len(embedding) == 0 {
		return true
	}

	// Check if all values are zero
	for _, v := range embedding {
		if v != 0 {
			return false
		}
	}
	return true
}

func (r *Memory) FindNearestAlerts(ctx context.Context, embedding []float32, limit int) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, a := range r.alerts {
		// Only add alerts that have embeddings
		if len(a.Embedding) > 0 {
			alerts = append(alerts, a)
		}
	}

	// Sort by similarity
	sort.Slice(alerts, func(i, j int) bool {
		simI := cosineSimilarity(alerts[i].Embedding, embedding)
		simJ := cosineSimilarity(alerts[j].Embedding, embedding)
		return simI > simJ
	})

	// Apply limit
	if limit > 0 && limit < len(alerts) {
		alerts = alerts[:limit]
	}

	return alerts, nil
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

func cosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
