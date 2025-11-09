package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
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
				goerr.TV(errs.TicketIDKey, ticketID),
				goerr.T(errs.TagNotFound))
		}
		return nil, r.eb.Wrap(err, "failed to get ticket",
			goerr.TV(errs.TicketIDKey, ticketID),
			goerr.T(errs.TagDatabase))
	}

	var t ticket.Ticket
	if err := doc.DataTo(&t); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to ticket",
			goerr.TV(errs.TicketIDKey, ticketID),
			goerr.T(errs.TagInternal))
	}

	return &t, nil
}

func (r *Firestore) PutTicket(ctx context.Context, t ticket.Ticket) error {
	// Reject tickets with invalid embeddings (nil, empty, or zero vector)
	if isInvalidEmbedding(t.Embedding) {
		return r.eb.New("ticket has invalid embedding (nil, empty, or zero vector)",
			goerr.TV(errs.TicketIDKey, t.ID),
			goerr.V("embedding_length", len(t.Embedding)))
	}

	// Check if ticket already exists to determine if this is create or update
	existingTicket, err := r.GetTicket(ctx, t.ID)
	isUpdate := err == nil && existingTicket != nil

	_, err = r.db.Collection(collectionTickets).Doc(t.ID.String()).Set(ctx, t)
	if err != nil {
		return r.eb.Wrap(err, "failed to put ticket",
			goerr.TV(errs.TicketIDKey, t.ID),
			goerr.T(errs.TagDatabase))
	}

	// Create activity for ticket creation or update (except when called from agent)
	if !user.IsAgent(ctx) {
		if isUpdate {
			if err := createTicketUpdateActivity(ctx, r, t.ID, t.Title); err != nil {
				return goerr.Wrap(err, "failed to create ticket update activity", goerr.V("ticket_id", t.ID))
			}
		} else {
			if err := createTicketActivity(ctx, r, t.ID, t.Title); err != nil {
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
			if err := createCommentActivity(ctx, r, comment.TicketID, comment.ID, t.Title); err != nil {
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
			// Skip errors related to zero magnitude vectors
			errMsg := err.Error()
			if status.Code(err) == codes.FailedPrecondition &&
				(len(errMsg) > 0 && (errMsg[0:7] == "Cannot " || errMsg[0:7] == "Missing")) {
				// Skip this document and continue
				continue
			}
			return nil, goerr.Wrap(err, "failed to get next ticket")
		}

		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to ticket")
		}

		// Only add tickets that have embeddings
		if len(t.Embedding) > 0 {
			tickets = append(tickets, &t)
		}
	}

	return tickets, nil
}

func (r *Firestore) GetTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, offset, limit int) ([]*ticket.Ticket, error) {
	// If no statuses specified, query all tickets
	var query firestore.Query
	if len(statuses) > 0 {
		// Use "in" operator to match any of the specified statuses
		query = r.db.Collection(collectionTickets).Where("Status", "in", statuses)
	} else {
		query = r.db.Collection(collectionTickets).Query
	}

	// Order by CreatedAt in descending order (newest first)
	query = query.OrderBy("CreatedAt", firestore.Desc)

	if offset > 0 {
		query = query.Offset(offset)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var tickets []*ticket.Ticket
	iter := query.Documents(ctx)
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

		tickets = append(tickets, &t)
	}

	return tickets, nil
}

func (r *Firestore) CountTicketsByStatus(ctx context.Context, statuses []types.TicketStatus) (int, error) {
	// If no statuses specified, count all tickets
	var query firestore.Query
	if len(statuses) > 0 {
		// Use "in" operator to match any of the specified statuses
		query = r.db.Collection(collectionTickets).Where("Status", "in", statuses)
	} else {
		query = r.db.Collection(collectionTickets).Query
	}

	// Use the count aggregation query for efficiency
	countQuery := query.NewAggregationQuery().WithCount("count")
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
		if err := createStatusChangeActivity(ctx, r, t.ID, t.Title, oldStatus, newStatus); err != nil {
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

		tickets = append(tickets, &t)
	}

	return tickets, nil
}

func (r *Firestore) GetTicketsByIDs(ctx context.Context, ticketIDs []types.TicketID) ([]*ticket.Ticket, error) {
	return r.BatchGetTickets(ctx, ticketIDs)
}

func (r *Firestore) GetAllTickets(ctx context.Context, offset, limit int) ([]*ticket.Ticket, error) {
	return r.GetTicketsByStatus(ctx, nil, offset, limit)
}
