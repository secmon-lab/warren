package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository/activityutil"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

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
			if err := activityutil.CreateTicketUpdateActivity(ctx, r, t.ID, t.Title); err != nil {
				return goerr.Wrap(err, "failed to create ticket update activity", goerr.V("ticket_id", t.ID))
			}
		} else {
			if err := activityutil.CreateTicketActivity(ctx, r, t.ID, t.Title); err != nil {
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
		ticketTitle = t.Title
		hasTicket = true
	}
	r.mu.Unlock()

	// Create activity for comment addition - only for user comments, not agent
	if !user.IsAgent(ctx) && hasTicket {
		if err := activityutil.CreateCommentActivity(ctx, r, comment.TicketID, comment.ID, ticketTitle); err != nil {
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
		alertTitles = append(alertTitles, alert.Title)
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

	ticketTitle := t.Title
	r.mu.Unlock()

	// Create activity for bulk alert binding
	if len(alertIDs) > 1 {
		if err := activityutil.CreateBulkAlertBoundActivity(ctx, r, alertIDs, ticketID, ticketTitle, alertTitles); err != nil {
			return goerr.Wrap(err, "failed to create bulk alert bound activity", goerr.V("ticket_id", ticketID))
		}
	} else if len(alertIDs) == 1 {
		alertTitle := ""
		if len(alertTitles) > 0 {
			alertTitle = alertTitles[0]
		}
		if err := activityutil.CreateAlertBoundActivity(ctx, r, alertIDs[0], ticketID, alertTitle, ticketTitle); err != nil {
			return goerr.Wrap(err, "failed to create alert bound activity", goerr.V("alert_id", alertIDs[0]), goerr.V("ticket_id", ticketID))
		}
	}

	return nil
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

func ticketMatchesFilter(t *ticket.Ticket, statuses []types.TicketStatus, keyword, assigneeID string) bool {
	if len(statuses) > 0 {
		matched := false
		for _, status := range statuses {
			if t.Status == status {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if assigneeID != "" {
		if t.Assignee == nil || string(t.Assignee.ID) != assigneeID {
			return false
		}
	}
	if keyword != "" {
		kw := strings.ToLower(keyword)
		if !strings.Contains(strings.ToLower(t.Title), kw) &&
			!strings.Contains(strings.ToLower(t.Description), kw) {
			return false
		}
	}
	return true
}

func (r *Memory) GetTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, keyword, assigneeID string, offset, limit int) ([]*ticket.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tickets []*ticket.Ticket
	for _, t := range r.tickets {
		if ticketMatchesFilter(t, statuses, keyword, assigneeID) {
			tickets = append(tickets, t)
		}
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

func (r *Memory) CountTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, keyword, assigneeID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, t := range r.tickets {
		if ticketMatchesFilter(t, statuses, keyword, assigneeID) {
			count++
		}
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
			title:     ticket.Title,
			oldStatus: oldStatus,
			newStatus: newStatus,
		})
	}
	r.mu.Unlock()

	// Create activities after releasing the main mutex
	for _, update := range updates {
		if err := activityutil.CreateStatusChangeActivity(ctx, r, update.id, update.title, update.oldStatus, update.newStatus); err != nil {
			return goerr.Wrap(err, "failed to create status change activity", goerr.V("ticket_id", update.id))
		}
	}

	return nil
}
