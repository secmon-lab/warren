package repository

import (
	"context"
	"math"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

type Memory struct {
	mu         sync.RWMutex
	activityMu sync.RWMutex
	tagMu      sync.RWMutex
	noticeMu   sync.RWMutex

	alerts         map[types.AlertID]*alert.Alert
	lists          map[types.AlertListID]*alert.List
	histories      map[types.TicketID][]*ticket.History
	tickets        map[types.TicketID]*ticket.Ticket
	ticketComments map[types.TicketID][]ticket.Comment
	tokens         map[auth.TokenID]*auth.Token
	activities     map[types.ActivityID]*activity.Activity
	tagsV2         map[string]*tag.Tag // New ID-based tags
	notices        map[types.NoticeID]*notice.Notice

	// Call counter for tracking method invocations
	callCounts map[string]int
	callMu     sync.RWMutex

	eb *goerr.Builder
}

var _ interfaces.Repository = &Memory{}

func NewMemory() *Memory {
	return &Memory{
		alerts:         make(map[types.AlertID]*alert.Alert),
		lists:          make(map[types.AlertListID]*alert.List),
		histories:      make(map[types.TicketID][]*ticket.History),
		tickets:        make(map[types.TicketID]*ticket.Ticket),
		ticketComments: make(map[types.TicketID][]ticket.Comment),
		tokens:         make(map[auth.TokenID]*auth.Token),
		activities:     make(map[types.ActivityID]*activity.Activity),
		tagsV2:         make(map[string]*tag.Tag),
		notices:        make(map[types.NoticeID]*notice.Notice),
		callCounts:     make(map[string]int),
		eb:             goerr.NewBuilder(goerr.TV(errs.RepositoryKey, "memory")),
	}
}

// incrementCallCount safely increments the call counter for a method
func (r *Memory) incrementCallCount(methodName string) {
	r.callMu.Lock()
	defer r.callMu.Unlock()
	r.callCounts[methodName]++
}

// GetCallCount returns the number of times a method has been called
func (r *Memory) GetCallCount(methodName string) int {
	r.callMu.RLock()
	defer r.callMu.RUnlock()
	return r.callCounts[methodName]
}

// GetAllCallCounts returns a copy of all call counts
func (r *Memory) GetAllCallCounts() map[string]int {
	r.callMu.RLock()
	defer r.callMu.RUnlock()

	counts := make(map[string]int)
	for k, v := range r.callCounts {
		counts[k] = v
	}
	return counts
}

// ResetCallCounts clears all call counters
func (r *Memory) ResetCallCounts() {
	r.callMu.Lock()
	defer r.callMu.Unlock()
	r.callCounts = make(map[string]int)
}

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

func (r *Memory) GetLatestHistory(ctx context.Context, ticketID types.TicketID) (*ticket.History, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	histories, ok := r.histories[ticketID]
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

func (r *Memory) PutHistory(ctx context.Context, ticketID types.TicketID, history *ticket.History) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.histories[ticketID] = append(r.histories[ticketID], history)
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

func (r *Memory) GetTicket(ctx context.Context, ticketID types.TicketID) (*ticket.Ticket, error) {
	r.incrementCallCount("GetTicket")
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

	// Check if ticket already exists to determine if this is create or update
	_, isUpdate := r.tickets[t.ID]

	r.tickets[t.ID] = &t
	r.mu.Unlock()

	// Create activity for ticket creation or update (except when called from agent)
	if !user.IsAgent(ctx) {
		if isUpdate {
			if err := createTicketUpdateActivity(ctx, r, t.ID, t.Metadata.Title); err != nil {
				return goerr.Wrap(err, "failed to create ticket update activity", goerr.V("ticket_id", t.ID))
			}
		} else {
			if err := createTicketActivity(ctx, r, t.ID, t.Metadata.Title); err != nil {
				return goerr.Wrap(err, "failed to create ticket activity", goerr.V("ticket_id", t.ID))
			}
		}
	}

	return nil
}

func (r *Memory) PutTicketComment(ctx context.Context, comment ticket.Comment) error {
	// Store comment first
	r.mu.Lock()
	r.ticketComments[comment.TicketID] = append(r.ticketComments[comment.TicketID], comment)

	// Get ticket title for activity creation
	var ticketTitle string
	var hasTicket bool
	if t, exists := r.tickets[comment.TicketID]; exists {
		ticketTitle = t.Metadata.Title
		hasTicket = true
	}
	r.mu.Unlock()

	// Create activity for comment addition - only for user comments, not agent
	if !user.IsAgent(ctx) && hasTicket {
		if err := createCommentActivity(ctx, r, comment.TicketID, comment.ID, ticketTitle); err != nil {
			return goerr.Wrap(err, "failed to create comment activity", goerr.V("ticket_id", comment.TicketID), goerr.V("comment_id", comment.ID))
		}
	}

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

func (r *Memory) GetTicketCommentsPaginated(ctx context.Context, ticketID types.TicketID, offset, limit int) ([]ticket.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comments, ok := r.ticketComments[ticketID]
	if !ok {
		return []ticket.Comment{}, nil
	}

	// Sort comments by CreatedAt in descending order (newest first)
	sortedComments := make([]ticket.Comment, len(comments))
	copy(sortedComments, comments)
	sort.Slice(sortedComments, func(i, j int) bool {
		return sortedComments[i].CreatedAt.After(sortedComments[j].CreatedAt)
	})

	// Apply pagination
	start := offset
	if start > len(sortedComments) {
		return []ticket.Comment{}, nil
	}

	end := start + limit
	if end > len(sortedComments) {
		end = len(sortedComments)
	}

	return sortedComments[start:end], nil
}

func (r *Memory) CountTicketComments(ctx context.Context, ticketID types.TicketID) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comments, ok := r.ticketComments[ticketID]
	if !ok {
		return 0, nil
	}
	return len(comments), nil
}

func (r *Memory) GetTicketUnpromptedComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comments, ok := r.ticketComments[ticketID]
	if !ok {
		return []ticket.Comment{}, nil
	}

	var unpromptedComments []ticket.Comment
	for _, comment := range comments {
		if !comment.Prompted {
			unpromptedComments = append(unpromptedComments, comment)
		}
	}
	return unpromptedComments, nil
}

func (r *Memory) PutTicketCommentsPrompted(ctx context.Context, ticketID types.TicketID, commentIDs []types.CommentID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	comments, ok := r.ticketComments[ticketID]
	if !ok {
		return goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
	}

	// Create a map for faster lookup
	commentIDMap := make(map[types.CommentID]bool)
	for _, id := range commentIDs {
		commentIDMap[id] = true
	}

	// Update prompted status for matching comments
	for i := range comments {
		if commentIDMap[comments[i].ID] {
			comments[i].Prompted = true
		}
	}

	r.ticketComments[ticketID] = comments
	return nil
}

func (r *Memory) BindAlertsToTicket(ctx context.Context, alertIDs []types.AlertID, ticketID types.TicketID) error {
	// Bind alerts to ticket first
	r.mu.Lock()

	// Get ticket for activity creation
	t, ticketExists := r.tickets[ticketID]
	if !ticketExists {
		r.mu.Unlock()
		return goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
	}

	// Get alerts for activity creation and bind them to ticket
	var alertTitles []string
	for _, alertID := range alertIDs {
		alert, ok := r.alerts[alertID]
		if !ok {
			r.mu.Unlock()
			return goerr.New("alert not found", goerr.V("alert_id", alertID))
		}
		alert.TicketID = ticketID
		alertTitles = append(alertTitles, alert.Metadata.Title)
	}

	// Update ticket's AlertIDs array to include newly bound alerts
	for _, alertID := range alertIDs {
		// Check if alert is already in the ticket's AlertIDs to avoid duplicates
		found := false
		for _, existingID := range t.AlertIDs {
			if existingID == alertID {
				found = true
				break
			}
		}
		if !found {
			t.AlertIDs = append(t.AlertIDs, alertID)
		}
	}

	ticketTitle := t.Metadata.Title
	r.mu.Unlock()

	// Create activity for bulk alert binding
	if len(alertIDs) > 1 {
		if err := createBulkAlertBoundActivity(ctx, r, alertIDs, ticketID, ticketTitle, alertTitles); err != nil {
			return goerr.Wrap(err, "failed to create bulk alert bound activity", goerr.V("ticket_id", ticketID))
		}
	} else if len(alertIDs) == 1 {
		alertTitle := ""
		if len(alertTitles) > 0 {
			alertTitle = alertTitles[0]
		}
		if err := createAlertBoundActivity(ctx, r, alertIDs[0], ticketID, alertTitle, ticketTitle); err != nil {
			return goerr.Wrap(err, "failed to create alert bound activity", goerr.V("alert_id", alertIDs[0]), goerr.V("ticket_id", ticketID))
		}
	}

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

func (r *Memory) GetAlertWithoutTicket(ctx context.Context, offset, limit int) (alert.Alerts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alerts alert.Alerts
	for _, alert := range r.alerts {
		if alert.TicketID == types.EmptyTicketID {
			alerts = append(alerts, alert)
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
	for _, alert := range r.alerts {
		if alert.TicketID == types.EmptyTicketID {
			count++
		}
	}

	return count, nil
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
		var cmpValue any = value
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

func (r *Memory) BatchGetTickets(ctx context.Context, ticketIDs []types.TicketID) ([]*ticket.Ticket, error) {
	r.incrementCallCount("BatchGetTickets")
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ticket.Ticket
	for _, id := range ticketIDs {
		if ticket, ok := r.tickets[id]; ok {
			tickets = append(tickets, ticket)
		}
	}
	return tickets, nil
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

func (r *Memory) FindNearestTickets(ctx context.Context, embedding []float32, limit int) ([]*ticket.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ticket.Ticket
	for _, t := range r.tickets {
		// Only add tickets that have embeddings
		if len(t.Embedding) > 0 {
			tickets = append(tickets, t)
		}
	}

	// Sort by similarity
	sort.Slice(tickets, func(i, j int) bool {
		simI := cosineSimilarity(tickets[i].Embedding, embedding)
		simJ := cosineSimilarity(tickets[j].Embedding, embedding)
		return simI > simJ
	})

	// Apply limit
	if limit > 0 && limit < len(tickets) {
		tickets = tickets[:limit]
	}

	return tickets, nil
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

func (r *Memory) BatchPutAlerts(ctx context.Context, alerts alert.Alerts) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, alert := range alerts {
		r.alerts[alert.ID] = alert
	}
	return nil
}

func (r *Memory) GetTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, offset, limit int) ([]*ticket.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ticket.Ticket
	for _, t := range r.tickets {
		// Filter by status if specified
		if len(statuses) > 0 {
			matched := false
			for _, status := range statuses {
				if t.Status == status {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		tickets = append(tickets, t)
	}

	// Sort tickets by CreatedAt in descending order (newest first)
	sort.Slice(tickets, func(i, j int) bool {
		return tickets[i].CreatedAt.After(tickets[j].CreatedAt)
	})

	// Apply offset and limit
	if offset > 0 && offset < len(tickets) {
		tickets = tickets[offset:]
	} else if offset >= len(tickets) {
		return []*ticket.Ticket{}, nil
	}

	if limit > 0 && limit < len(tickets) {
		tickets = tickets[:limit]
	}

	return tickets, nil
}

func (r *Memory) CountTicketsByStatus(ctx context.Context, statuses []types.TicketStatus) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, t := range r.tickets {
		// Filter by status if specified
		if len(statuses) > 0 {
			matched := false
			for _, status := range statuses {
				if t.Status == status {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		count++
	}

	return count, nil
}

func (r *Memory) GetTicketsBySpan(ctx context.Context, start, end time.Time) ([]*ticket.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ticket.Ticket
	for _, t := range r.tickets {
		if t.CreatedAt.After(start) && t.CreatedAt.Before(end) {
			tickets = append(tickets, t)
		}
	}
	return tickets, nil
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

func (r *Memory) GetTicketsWithInvalidEmbedding(ctx context.Context) ([]*ticket.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ticket.Ticket
	for _, t := range r.tickets {
		if isInvalidEmbeddingMemory(t.Embedding) {
			tickets = append(tickets, t)
		}
	}
	return tickets, nil
}

func (r *Memory) FindNearestTicketsWithSpan(ctx context.Context, embedding []float32, begin, end time.Time, limit int) ([]*ticket.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ticket.Ticket
	for _, t := range r.tickets {
		if t.CreatedAt.After(begin) && t.CreatedAt.Before(end) {
			tickets = append(tickets, t)
		}
	}

	// Sort by cosine similarity
	sort.Slice(tickets, func(i, j int) bool {
		simI := cosineSimilarity(embedding, tickets[i].Embedding)
		simJ := cosineSimilarity(embedding, tickets[j].Embedding)
		return simI > simJ
	})

	if len(tickets) > limit {
		tickets = tickets[:limit]
	}

	return tickets, nil
}

func (r *Memory) GetTicketsByStatusAndSpan(ctx context.Context, status types.TicketStatus, begin, end time.Time) ([]*ticket.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ticket.Ticket
	for _, t := range r.tickets {
		if t.Status == status && t.CreatedAt.After(begin) && t.CreatedAt.Before(end) {
			tickets = append(tickets, t)
		}
	}

	return tickets, nil
}

// Token related methods
func (r *Memory) PutToken(ctx context.Context, token *auth.Token) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tokens[token.ID] = token
	return nil
}

func (r *Memory) GetToken(ctx context.Context, tokenID auth.TokenID) (*auth.Token, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	token, ok := r.tokens[tokenID]
	if !ok {
		return nil, goerr.New("token not found", goerr.V("token_id", tokenID))
	}
	return token, nil
}

func (r *Memory) DeleteToken(ctx context.Context, tokenID auth.TokenID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tokens, tokenID)
	return nil
}

func (r *Memory) BatchUpdateTicketsStatus(ctx context.Context, ticketIDs []types.TicketID, status types.TicketStatus) error {
	// Collect ticket information for activity creation
	type ticketUpdate struct {
		id        types.TicketID
		title     string
		oldStatus string
		newStatus string
	}
	var updates []ticketUpdate

	// Update tickets first
	r.mu.Lock()
	for _, ticketID := range ticketIDs {
		ticket, ok := r.tickets[ticketID]
		if !ok {
			r.mu.Unlock()
			return goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
		}

		oldStatus := string(ticket.Status)
		newStatus := string(status)

		ticket.Status = status
		ticket.UpdatedAt = time.Now()
		r.tickets[ticketID] = ticket

		updates = append(updates, ticketUpdate{
			id:        ticketID,
			title:     ticket.Metadata.Title,
			oldStatus: oldStatus,
			newStatus: newStatus,
		})
	}
	r.mu.Unlock()

	// Create activities after releasing the main mutex
	for _, update := range updates {
		if err := createStatusChangeActivity(ctx, r, update.id, update.title, update.oldStatus, update.newStatus); err != nil {
			return goerr.Wrap(err, "failed to create status change activity", goerr.V("ticket_id", update.id))
		}
	}

	return nil
}

// Activity related methods
func (r *Memory) PutActivity(ctx context.Context, activity *activity.Activity) error {
	r.activityMu.Lock()
	defer r.activityMu.Unlock()

	r.activities[activity.ID] = activity
	return nil
}

func (r *Memory) GetActivities(ctx context.Context, offset, limit int) ([]*activity.Activity, error) {
	r.activityMu.RLock()
	defer r.activityMu.RUnlock()

	var activities []*activity.Activity
	for _, a := range r.activities {
		activities = append(activities, a)
	}

	// Sort by CreatedAt in descending order (newest first)
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].CreatedAt.After(activities[j].CreatedAt)
	})

	// Apply offset and limit
	start := offset
	if start >= len(activities) {
		return []*activity.Activity{}, nil
	}

	end := start + limit
	if end > len(activities) {
		end = len(activities)
	}

	return activities[start:end], nil
}

func (r *Memory) CountActivities(ctx context.Context) (int, error) {
	r.activityMu.RLock()
	defer r.activityMu.RUnlock()

	return len(r.activities), nil
}

// Tag management methods

func (r *Memory) RemoveTagFromAllAlerts(ctx context.Context, name string) error {
	// First, look up the tag by name to get its ID
	tag, err := r.GetTagByName(ctx, name)
	if err != nil {
		return goerr.Wrap(err, "failed to get tag by name")
	}
	if tag == nil {
		// Tag doesn't exist, nothing to remove
		return nil
	}

	// Use the new ID-based removal method
	return r.RemoveTagIDFromAllAlerts(ctx, tag.ID)
}

func (r *Memory) RemoveTagFromAllTickets(ctx context.Context, name string) error {
	// First, look up the tag by name to get its ID
	tag, err := r.GetTagByName(ctx, name)
	if err != nil {
		return goerr.Wrap(err, "failed to get tag by name")
	}
	if tag == nil {
		// Tag doesn't exist, nothing to remove
		return nil
	}

	// Use the new ID-based removal method
	return r.RemoveTagIDFromAllTickets(ctx, tag.ID)
}

// New ID-based tag management methods

func (r *Memory) GetTagByID(ctx context.Context, tagID string) (*tag.Tag, error) {
	r.tagMu.RLock()
	defer r.tagMu.RUnlock()

	if tagData, exists := r.tagsV2[tagID]; exists {
		// Return a copy to prevent external modification
		tagCopy := *tagData
		return &tagCopy, nil
	}

	return nil, nil
}

func (r *Memory) GetTagsByIDs(ctx context.Context, tagIDs []string) ([]*tag.Tag, error) {
	r.tagMu.RLock()
	defer r.tagMu.RUnlock()

	tags := make([]*tag.Tag, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		if tagData, exists := r.tagsV2[tagID]; exists {
			// Return a copy to prevent external modification
			tagCopy := *tagData
			tags = append(tags, &tagCopy)
		}
	}

	return tags, nil
}

func (r *Memory) CreateTagWithID(ctx context.Context, tag *tag.Tag) error {
	r.tagMu.Lock()
	defer r.tagMu.Unlock()

	if tag.ID == "" {
		return goerr.New("tag ID is required")
	}

	if _, exists := r.tagsV2[tag.ID]; exists {
		return goerr.New("tag ID already exists", goerr.V("tagID", tag.ID))
	}

	// Set timestamps if not already set
	now := time.Now()
	tagCopy := *tag
	if tagCopy.CreatedAt.IsZero() {
		tagCopy.CreatedAt = now
	}
	if tagCopy.UpdatedAt.IsZero() {
		tagCopy.UpdatedAt = now
	}

	r.tagsV2[tag.ID] = &tagCopy

	return nil
}

func (r *Memory) UpdateTag(ctx context.Context, tag *tag.Tag) error {
	r.tagMu.Lock()
	defer r.tagMu.Unlock()

	if tag.ID == "" {
		return goerr.New("tag ID is required")
	}

	// Set UpdatedAt timestamp
	tagCopy := *tag
	tagCopy.UpdatedAt = time.Now()
	r.tagsV2[tag.ID] = &tagCopy

	return nil
}

func (r *Memory) DeleteTagByID(ctx context.Context, tagID string) error {
	r.tagMu.Lock()
	defer r.tagMu.Unlock()

	delete(r.tagsV2, tagID)
	return nil
}

func (r *Memory) RemoveTagIDFromAllAlerts(ctx context.Context, tagID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Iterate through all alerts and remove the tag ID
	for _, alert := range r.alerts {
		if alert.TagIDs != nil {
			// Remove tagID from map
			delete(alert.TagIDs, tagID)
		}
	}

	return nil
}

func (r *Memory) RemoveTagIDFromAllTickets(ctx context.Context, tagID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Iterate through all tickets and remove the tag ID
	for _, ticket := range r.tickets {
		if ticket.TagIDs != nil {
			// Remove tagID from map
			delete(ticket.TagIDs, tagID)
		}
	}

	return nil
}

func (r *Memory) GetTagByName(ctx context.Context, name string) (*tag.Tag, error) {
	r.tagMu.RLock()
	defer r.tagMu.RUnlock()

	for _, tagData := range r.tagsV2 {
		if tagData.Name == name {
			// Return a copy to prevent external modification
			tagCopy := *tagData
			return &tagCopy, nil
		}
	}

	return nil, nil
}

func (r *Memory) IsTagNameExists(ctx context.Context, name string) (bool, error) {
	r.tagMu.RLock()
	defer r.tagMu.RUnlock()

	for _, tagData := range r.tagsV2 {
		if tagData.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// GetOrCreateTagByName atomically gets an existing tag or creates a new one
func (r *Memory) GetOrCreateTagByName(ctx context.Context, name, description, color, createdBy string) (*tag.Tag, error) {
	r.tagMu.Lock()
	defer r.tagMu.Unlock()

	// First check if tag already exists by name
	for _, tagData := range r.tagsV2 {
		if tagData.Name == name {
			return tagData, nil
		}
	}

	// Tag doesn't exist, create it
	// Generate unique ID with collision retry
	var tagID string
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		tagID = tag.NewID()
		if _, exists := r.tagsV2[tagID]; !exists {
			break // No collision
		}
		if i == maxRetries-1 {
			return nil, goerr.New("failed to generate unique tag ID after retries")
		}
	}

	// Use provided color or generate one
	if color == "" {
		color = tag.GenerateColor(name)
	}

	// Create the new tag
	now := time.Now()
	newTag := &tag.Tag{
		ID:          tagID,
		Name:        name,
		Description: description,
		Color:       color,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Store the tag
	r.tagsV2[tagID] = newTag

	return newTag, nil
}

func (r *Memory) ListAllTags(ctx context.Context) ([]*tag.Tag, error) {
	r.tagMu.RLock()
	defer r.tagMu.RUnlock()

	tags := make([]*tag.Tag, 0, len(r.tagsV2))
	for _, tagData := range r.tagsV2 {
		// Return a copy to prevent external modification
		tagCopy := *tagData
		tags = append(tags, &tagCopy)
	}

	return tags, nil
}

// Notice management methods

func (r *Memory) CreateNotice(ctx context.Context, notice *notice.Notice) error {
	r.noticeMu.Lock()
	defer r.noticeMu.Unlock()

	if notice.ID == types.EmptyNoticeID {
		return r.eb.New("notice ID is empty")
	}

	// Check if notice already exists
	if _, exists := r.notices[notice.ID]; exists {
		return r.eb.New("notice already exists", goerr.V("notice_id", notice.ID))
	}

	// Store a copy to prevent external modification
	noticeCopy := *notice
	r.notices[notice.ID] = &noticeCopy

	return nil
}

func (r *Memory) GetNotice(ctx context.Context, id types.NoticeID) (*notice.Notice, error) {
	r.noticeMu.RLock()
	defer r.noticeMu.RUnlock()

	notice, exists := r.notices[id]
	if !exists {
		return nil, r.eb.New("notice not found", goerr.V("notice_id", id))
	}

	// Return a copy to prevent external modification
	noticeCopy := *notice
	return &noticeCopy, nil
}

func (r *Memory) UpdateNotice(ctx context.Context, notice *notice.Notice) error {
	r.noticeMu.Lock()
	defer r.noticeMu.Unlock()

	if notice.ID == types.EmptyNoticeID {
		return r.eb.New("notice ID is empty")
	}

	// Check if notice exists
	if _, exists := r.notices[notice.ID]; !exists {
		return r.eb.New("notice not found", goerr.V("notice_id", notice.ID))
	}

	// Store a copy to prevent external modification
	noticeCopy := *notice
	r.notices[notice.ID] = &noticeCopy

	return nil
}
