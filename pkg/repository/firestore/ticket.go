package firestore

import (
	"context"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository/activityutil"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/user"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Ticket related methods
func (r *Firestore) GetTicket(ctx context.Context, ticketID types.TicketID) (*ticket.Ticket, error) {
	doc, err := r.db.Collection(collectionTickets).Doc(ticketID.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.New("ticket not found",
				goerr.TV(errutil.TicketIDKey, ticketID),
				goerr.T(errutil.TagNotFound))
		}
		return nil, r.eb.Wrap(err, "failed to get ticket",
			goerr.TV(errutil.TicketIDKey, ticketID),
			goerr.T(errutil.TagDatabase))
	}

	var t ticket.Ticket
	if err := doc.DataTo(&t); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to ticket",
			goerr.TV(errutil.TicketIDKey, ticketID),
			goerr.T(errutil.TagInternal))
	}

	t.NormalizeLegacyStatus()
	return &t, nil
}

func (r *Firestore) PutTicket(ctx context.Context, t ticket.Ticket) error {
	// Reject tickets with invalid embeddings (nil, empty, or zero vector)
	if isInvalidEmbedding(t.Embedding) {
		return r.eb.New("ticket has invalid embedding (nil, empty, or zero vector)",
			goerr.TV(errutil.TicketIDKey, t.ID),
			goerr.V("embedding_length", len(t.Embedding)))
	}

	// Check if ticket already exists to determine if this is create or update
	existingTicket, err := r.GetTicket(ctx, t.ID)
	isUpdate := err == nil && existingTicket != nil

	_, err = r.db.Collection(collectionTickets).Doc(t.ID.String()).Set(ctx, t)
	if err != nil {
		return r.eb.Wrap(err, "failed to put ticket",
			goerr.TV(errutil.TicketIDKey, t.ID),
			goerr.T(errutil.TagDatabase))
	}

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

func (r *Firestore) PutHistory(ctx context.Context, ticketID types.TicketID, history *ticket.History) error {
	_, err := r.db.Collection(collectionTickets).Doc(ticketID.String()).Collection(collectionHistories).Doc(history.ID.String()).Set(ctx, history)
	if err != nil {
		return goerr.Wrap(err, "failed to put history", goerr.V("ticket_id", ticketID), goerr.V("history_id", history.ID))
	}
	return nil
}

func (r *Firestore) GetLatestHistory(ctx context.Context, ticketID types.TicketID) (*ticket.History, error) {
	iter := r.db.Collection(collectionTickets).Doc(ticketID.String()).Collection(collectionHistories).OrderBy("CreatedAt", firestore.Desc).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get latest chat history")
	}

	var history ticket.History
	if err := doc.DataTo(&history); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to chat history")
	}
	return &history, nil
}

func (r *Firestore) PutTicketComment(ctx context.Context, comment ticket.Comment) error {
	_, err := r.db.Collection(collectionTickets).Doc(comment.TicketID.String()).Collection(collectionComments).Doc(comment.ID.String()).Set(ctx, comment)
	if err != nil {
		return goerr.Wrap(err, "failed to put ticket comment", goerr.V("ticket_id", comment.TicketID))
	}

	// Create activity for comment addition - only for user comments, not agent
	if !user.IsAgent(ctx) {
		// Get ticket for activity creation
		if t, err := r.GetTicket(ctx, comment.TicketID); err == nil {
			if err := activityutil.CreateCommentActivity(ctx, r, comment.TicketID, comment.ID, t.Title); err != nil {
				return goerr.Wrap(err, "failed to create comment activity", goerr.V("ticket_id", comment.TicketID), goerr.V("comment_id", comment.ID))
			}
		}
	}

	return nil
}

func (r *Firestore) GetTicketComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error) {
	iter := r.db.Collection(collectionTickets).Doc(ticketID.String()).Collection(collectionComments).OrderBy("CreatedAt", firestore.Desc).Documents(ctx)
	var comments []ticket.Comment
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get ticket comments", goerr.V("ticket_id", ticketID))
		}

		var comment ticket.Comment
		if err := doc.DataTo(&comment); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket comment", goerr.V("ticket_id", ticketID))
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (r *Firestore) GetTicketCommentsPaginated(ctx context.Context, ticketID types.TicketID, offset, limit int) ([]ticket.Comment, error) {
	iter := r.db.Collection(collectionTickets).
		Doc(ticketID.String()).
		Collection(collectionComments).
		OrderBy("CreatedAt", firestore.Desc).
		Offset(offset).
		Limit(limit).
		Documents(ctx)

	var comments []ticket.Comment
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get paginated ticket comments", goerr.V("ticket_id", ticketID))
		}

		var comment ticket.Comment
		if err := doc.DataTo(&comment); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket comment", goerr.V("ticket_id", ticketID))
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (r *Firestore) CountTicketComments(ctx context.Context, ticketID types.TicketID) (int, error) {
	// Use Firestore aggregation query to count documents efficiently
	result, err := r.db.Collection(collectionTickets).
		Doc(ticketID.String()).
		Collection(collectionComments).
		NewAggregationQuery().
		WithCount("total").
		Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to count ticket comments", goerr.V("ticket_id", ticketID))
	}

	return extractCountFromAggregationResult(result, "total")
}

func (r *Firestore) GetTicketUnpromptedComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error) {
	iter := r.db.Collection(collectionTickets).
		Doc(ticketID.String()).
		Collection(collectionComments).
		Where("Prompted", "==", false).
		Documents(ctx)

	var comments []ticket.Comment
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get ticket unprompted comments", goerr.V("ticket_id", ticketID))
		}

		var comment ticket.Comment
		if err := doc.DataTo(&comment); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket comment", goerr.V("ticket_id", ticketID))
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (r *Firestore) PutTicketCommentsPrompted(ctx context.Context, ticketID types.TicketID, commentIDs []types.CommentID) error {
	bw := r.db.BulkWriter(ctx)
	var jobs []*firestore.BulkWriterJob

	for _, commentID := range commentIDs {
		commentDoc := r.db.Collection(collectionTickets).
			Doc(ticketID.String()).
			Collection(collectionComments).
			Doc(commentID.String())

		job, err := bw.Update(commentDoc, []firestore.Update{
			{
				Path:  "Prompted",
				Value: true,
			},
		})
		if err != nil {
			return goerr.Wrap(err, "failed to update comment prompted status", goerr.V("ticket_id", ticketID), goerr.V("comment_id", commentID))
		}
		jobs = append(jobs, job)
	}

	bw.End()

	for _, job := range jobs {
		if _, err := job.Results(); err != nil {
			return goerr.Wrap(err, "failed to commit bulk writer job")
		}
	}

	return nil
}

func (r *Firestore) GetTicketByThread(ctx context.Context, thread slack.Thread) (*ticket.Ticket, error) {
	iter := r.db.Collection(collectionTickets).
		Where("SlackThread.ChannelID", "==", thread.ChannelID).
		Where("SlackThread.ThreadID", "==", thread.ThreadID).
		Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get ticket by thread", goerr.V("slack_thread", thread))
	}

	var t ticket.Ticket
	if err := doc.DataTo(&t); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to ticket")
	}

	t.NormalizeLegacyStatus()
	return &t, nil
}

// BatchGetTickets gets tickets by their IDs. If some tickets are not found, it will be ignored.
func (r *Firestore) BatchGetTickets(ctx context.Context, ticketIDs []types.TicketID) ([]*ticket.Ticket, error) {
	var tickets []*ticket.Ticket
	var docRefs []*firestore.DocumentRef
	for _, id := range ticketIDs {
		docRef := r.db.Collection(collectionTickets).Doc(id.String())
		docRefs = append(docRefs, docRef)
	}

	docs, err := r.db.GetAll(ctx, docRefs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get tickets")
	}

	for _, doc := range docs {
		if !doc.Exists() {
			continue
		}

		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket", goerr.V("doc.ref.id", doc.Ref.ID))
		}

		t.NormalizeLegacyStatus()
		tickets = append(tickets, &t)
	}

	return tickets, nil
}

func (r *Firestore) FindNearestTickets(ctx context.Context, embedding []float32, limit int) ([]*ticket.Ticket, error) {
	// Convert []float32 to firestore.Vector32
	vector32 := firestore.Vector32(embedding[:])

	// Build vector search query
	query := r.db.Collection(collectionTickets).
		FindNearest("Embedding",
			vector32,
			limit,
			firestore.DistanceMeasureCosine,
			&firestore.FindNearestOptions{
				DistanceResultField: "vector_distance",
			})

	iter := query.Documents(ctx)
	var tickets []*ticket.Ticket
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next ticket")
		}

		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket")
		}

		t.NormalizeLegacyStatus()
		// Only add tickets that have embeddings
		if len(t.Embedding) > 0 {
			tickets = append(tickets, &t)
		}
	}

	return tickets, nil
}

// fetchAllTicketsByStatus retrieves all tickets matching the given statuses from Firestore
// without applying offset/limit (used when Go-side filtering is needed).
func (r *Firestore) fetchAllTicketsByStatus(ctx context.Context, statuses []types.TicketStatus) ([]*ticket.Ticket, error) {
	var q firestore.Query
	if len(statuses) > 0 {
		q = r.db.Collection(collectionTickets).Where("Status", "in", statuses)
	} else {
		q = r.db.Collection(collectionTickets).Query
	}
	q = q.OrderBy("CreatedAt", firestore.Desc)

	var tickets []*ticket.Ticket
	iter := q.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get ticket documents")
		}
		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket")
		}
		t.NormalizeLegacyStatus()
		tickets = append(tickets, &t)
	}
	return tickets, nil
}

func filterTickets(tickets []*ticket.Ticket, keyword, assigneeID string) []*ticket.Ticket {
	if keyword == "" && assigneeID == "" {
		return tickets
	}
	kw := strings.ToLower(keyword)
	var result []*ticket.Ticket
	for _, t := range tickets {
		if assigneeID != "" {
			if t.Assignee == nil || string(t.Assignee.ID) != assigneeID {
				continue
			}
		}
		if kw != "" {
			if !strings.Contains(strings.ToLower(t.Title), kw) &&
				!strings.Contains(strings.ToLower(t.Description), kw) {
				continue
			}
		}
		result = append(result, t)
	}
	return result
}

func (r *Firestore) GetTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, keyword, assigneeID string, offset, limit int) ([]*ticket.Ticket, error) {
	// When extra filters are needed, fetch all matching tickets and filter in Go.
	if keyword != "" || assigneeID != "" {
		all, err := r.fetchAllTicketsByStatus(ctx, statuses)
		if err != nil {
			return nil, err
		}
		filtered := filterTickets(all, keyword, assigneeID)
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
		})
		if offset > 0 {
			if offset >= len(filtered) {
				return []*ticket.Ticket{}, nil
			}
			filtered = filtered[offset:]
		}
		if limit > 0 && limit < len(filtered) {
			filtered = filtered[:limit]
		}
		return filtered, nil
	}

	// Fast path: delegate offset/limit to Firestore.
	var q firestore.Query
	if len(statuses) > 0 {
		q = r.db.Collection(collectionTickets).Where("Status", "in", statuses)
	} else {
		q = r.db.Collection(collectionTickets).Query
	}
	q = q.OrderBy("CreatedAt", firestore.Desc)
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}

	var tickets []*ticket.Ticket
	iter := q.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get offset documents")
		}
		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket")
		}
		t.NormalizeLegacyStatus()
		tickets = append(tickets, &t)
	}
	return tickets, nil
}

func (r *Firestore) CountTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, keyword, assigneeID string) (int, error) {
	// When extra filters are needed, fetch all and count in Go.
	if keyword != "" || assigneeID != "" {
		all, err := r.fetchAllTicketsByStatus(ctx, statuses)
		if err != nil {
			return 0, err
		}
		return len(filterTickets(all, keyword, assigneeID)), nil
	}

	// Fast path: use Firestore aggregation.
	var q firestore.Query
	if len(statuses) > 0 {
		q = r.db.Collection(collectionTickets).Where("Status", "in", statuses)
	} else {
		q = r.db.Collection(collectionTickets).Query
	}
	countQuery := q.NewAggregationQuery().WithCount("count")
	result, err := countQuery.Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to get ticket count")
	}
	return extractCountFromAggregationResult(result, "count")
}

func (r *Firestore) GetTicketsBySpan(ctx context.Context, begin, end time.Time) ([]*ticket.Ticket, error) {
	iter := r.db.Collection(collectionTickets).
		Where("CreatedAt", ">=", begin).
		Where("CreatedAt", "<=", end).
		OrderBy("CreatedAt", firestore.Desc).
		Documents(ctx)

	var tickets []*ticket.Ticket
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next ticket")
		}

		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket")
		}

		t.NormalizeLegacyStatus()
		tickets = append(tickets, &t)
	}

	return tickets, nil
}

func (r *Firestore) GetTicketsWithInvalidEmbedding(ctx context.Context) ([]*ticket.Ticket, error) {
	// Get all tickets and filter for invalid embeddings
	iter := r.db.Collection(collectionTickets).Documents(ctx)

	var tickets []*ticket.Ticket
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next ticket")
		}

		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket")
		}

		t.NormalizeLegacyStatus()
		// Check if embedding is invalid (nil, empty, or zero vector)
		if isInvalidEmbedding(t.Embedding) {
			tickets = append(tickets, &t)
		}
	}

	return tickets, nil
}

func (r *Firestore) FindNearestTicketsWithSpan(ctx context.Context, embedding []float32, begin, end time.Time, limit int) ([]*ticket.Ticket, error) {
	iter := r.db.Collection(collectionTickets).
		OrderBy("CreatedAt", firestore.Desc).
		Where("CreatedAt", "<=", end).
		Where("CreatedAt", ">=", begin).
		FindNearest("Embedding",
			firestore.Vector32(embedding[:]),
			limit,
			firestore.DistanceMeasureCosine,
			&firestore.FindNearestOptions{
				DistanceResultField: "vector_distance",
			}).Documents(ctx)

	var tickets []*ticket.Ticket
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next ticket")
		}

		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket")
		}

		t.NormalizeLegacyStatus()
		tickets = append(tickets, &t)
	}

	if len(tickets) == 0 {
		return []*ticket.Ticket{}, nil
	}

	if len(tickets) > limit {
		tickets = tickets[:limit]
	}

	return tickets, nil
}

func (r *Firestore) GetTicketsByStatusAndSpan(ctx context.Context, status types.TicketStatus, begin, end time.Time) ([]*ticket.Ticket, error) {
	iter := r.db.Collection(collectionTickets).
		Where("Status", "==", status).
		Where("CreatedAt", ">=", begin).
		Where("CreatedAt", "<=", end).
		OrderBy("CreatedAt", firestore.Desc).
		Documents(ctx)

	var tickets []*ticket.Ticket
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next ticket")
		}

		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket")
		}

		t.NormalizeLegacyStatus()
		tickets = append(tickets, &t)
	}

	return tickets, nil
}

func (r *Firestore) BatchUpdateTicketsStatus(ctx context.Context, ticketIDs []types.TicketID, status types.TicketStatus) error {
	// Get current tickets for activity creation
	var ticketsForActivity []*ticket.Ticket
	for _, ticketID := range ticketIDs {
		if t, err := r.GetTicket(ctx, ticketID); err == nil {
			ticketsForActivity = append(ticketsForActivity, t)
		}
	}

	bw := r.db.BulkWriter(ctx)
	var jobs []*firestore.BulkWriterJob

	now := time.Now()
	for _, ticketID := range ticketIDs {
		ticketDoc := r.db.Collection(collectionTickets).Doc(ticketID.String())

		job, err := bw.Update(ticketDoc, []firestore.Update{
			{
				Path:  "Status",
				Value: status,
			},
			{
				Path:  "UpdatedAt",
				Value: now,
			},
		})
		if err != nil {
			return goerr.Wrap(err, "failed to update ticket status", goerr.V("ticket_id", ticketID), goerr.V("status", status))
		}
		jobs = append(jobs, job)
	}

	bw.End()

	for _, job := range jobs {
		if _, err := job.Results(); err != nil {
			return goerr.Wrap(err, "failed to commit bulk writer job")
		}
	}

	// Create activity for status changes
	for _, t := range ticketsForActivity {
		oldStatus := string(t.Status)
		newStatus := string(status)
		if err := activityutil.CreateStatusChangeActivity(ctx, r, t.ID, t.Title, oldStatus, newStatus); err != nil {
			return goerr.Wrap(err, "failed to create status change activity", goerr.V("ticket_id", t.ID))
		}
	}

	return nil
}

func (r *Firestore) UpdateTicketStatus(ctx context.Context, ticketID types.TicketID, status types.TicketStatus) error {
	return r.BatchUpdateTicketsStatus(ctx, []types.TicketID{ticketID}, status)
}

func (r *Firestore) DeleteTicket(ctx context.Context, ticketID types.TicketID) error {
	_, err := r.db.Collection(collectionTickets).Doc(ticketID.String()).Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete ticket", goerr.V("ticket_id", ticketID))
	}
	return nil
}

func (r *Firestore) PutTicketHistory(ctx context.Context, ticketID types.TicketID, history *ticket.History) error {
	return r.PutHistory(ctx, ticketID, history)
}

func (r *Firestore) GetTicketHistories(ctx context.Context, ticketID types.TicketID) ([]*ticket.History, error) {
	iter := r.db.Collection(collectionTickets).Doc(ticketID.String()).Collection(collectionHistories).OrderBy("CreatedAt", firestore.Desc).Documents(ctx)

	var histories []*ticket.History
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get ticket histories", goerr.V("ticket_id", ticketID))
		}

		var history ticket.History
		if err := doc.DataTo(&history); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket history", goerr.V("ticket_id", ticketID))
		}
		histories = append(histories, &history)
	}

	return histories, nil
}

func (r *Firestore) DeleteTicketComment(ctx context.Context, ticketID types.TicketID, commentID types.CommentID) error {
	_, err := r.db.Collection(collectionTickets).Doc(ticketID.String()).Collection(collectionComments).Doc(commentID.String()).Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete ticket comment", goerr.V("ticket_id", ticketID), goerr.V("comment_id", commentID))
	}
	return nil
}

func (r *Firestore) UpdateTicketComment(ctx context.Context, comment ticket.Comment) error {
	_, err := r.db.Collection(collectionTickets).Doc(comment.TicketID.String()).Collection(collectionComments).Doc(comment.ID.String()).Set(ctx, comment)
	if err != nil {
		return goerr.Wrap(err, "failed to update ticket comment", goerr.V("ticket_id", comment.TicketID), goerr.V("comment_id", comment.ID))
	}
	return nil
}

func (r *Firestore) CountTickets(ctx context.Context) (int, error) {
	result, err := r.db.Collection(collectionTickets).NewAggregationQuery().WithCount("total").Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to count tickets")
	}

	return extractCountFromAggregationResult(result, "total")
}

func (r *Firestore) QueryTickets(ctx context.Context, query string, offset, limit int) ([]*ticket.Ticket, error) {
	// For now, this is a placeholder implementation
	// In a real implementation, this would use full-text search or similar
	iter := r.db.Collection(collectionTickets).
		OrderBy("CreatedAt", firestore.Desc).
		Offset(offset).
		Limit(limit).
		Documents(ctx)

	var tickets []*ticket.Ticket
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to query tickets")
		}

		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket")
		}

		t.NormalizeLegacyStatus()
		tickets = append(tickets, &t)
	}

	return tickets, nil
}

func (r *Firestore) GetTicketsByIDs(ctx context.Context, ticketIDs []types.TicketID) ([]*ticket.Ticket, error) {
	return r.BatchGetTickets(ctx, ticketIDs)
}

func (r *Firestore) GetAllTickets(ctx context.Context, offset, limit int) ([]*ticket.Ticket, error) {
	return r.GetTicketsByStatus(ctx, nil, "", "", offset, limit)
}
