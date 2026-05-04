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

// buildTicketBaseQuery builds a Firestore query filtered by statuses and assigneeID.
// Composite index required when both status and Assignee.ID are specified:
//
//	Collection: tickets, Fields: Status (ASC), Assignee.ID (ASC), CreatedAt (DESC)
func (r *Firestore) buildTicketBaseQuery(statuses []types.TicketStatus, assigneeID string) firestore.Query {
	var q firestore.Query
	if len(statuses) > 0 {
		q = r.db.Collection(collectionTickets).Where("Status", "in", statuses)
	} else {
		q = r.db.Collection(collectionTickets).Query
	}
	if assigneeID != "" {
		q = q.Where("Assignee.ID", "==", assigneeID)
	}
	return q
}

// filterTicketsByKeyword filters tickets in-memory by keyword (title or description).
// This is only needed for keyword search since Firestore does not support
// full-text CONTAINS queries natively.
func filterTicketsByKeyword(tickets []*ticket.Ticket, keyword string) []*ticket.Ticket {
	if keyword == "" {
		return tickets
	}
	kw := strings.ToLower(keyword)
	var result []*ticket.Ticket
	for _, t := range tickets {
		if strings.Contains(strings.ToLower(t.Title), kw) ||
			strings.Contains(strings.ToLower(t.Description), kw) {
			result = append(result, t)
		}
	}
	return result
}

// fetchAllByQuery retrieves all documents for the given query without offset/limit,
// sorted by CreatedAt DESC in-memory. Sorting is done in Go to avoid requiring a
// composite index on (Status, Assignee.ID, CreatedAt).
func (r *Firestore) fetchAllByQuery(ctx context.Context, q firestore.Query) ([]*ticket.Ticket, error) {
	var tickets []*ticket.Ticket
	iter := q.Documents(ctx)
	defer iter.Stop()
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
	sort.Slice(tickets, func(i, j int) bool { return tickets[i].CreatedAt.After(tickets[j].CreatedAt) })
	return tickets, nil
}

func (r *Firestore) GetTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, keyword, assigneeID string, offset, limit int) ([]*ticket.Ticket, error) {
	q := r.buildTicketBaseQuery(statuses, assigneeID)

	// In-memory sort/paginate to avoid composite index on (Status, Assignee.ID, CreatedAt).
	all, err := r.fetchAllByQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	if keyword != "" {
		all = filterTicketsByKeyword(all, keyword)
	}
	if offset > 0 {
		if offset >= len(all) {
			return []*ticket.Ticket{}, nil
		}
		all = all[offset:]
	}
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	return all, nil
}

func (r *Firestore) CountTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, keyword, assigneeID string) (int, error) {
	q := r.buildTicketBaseQuery(statuses, assigneeID)

	// keyword requires Go-side counting.
	if keyword != "" {
		all, err := r.fetchAllByQuery(ctx, q)
		if err != nil {
			return 0, err
		}
		return len(filterTicketsByKeyword(all, keyword)), nil
	}

	// Fast path: use Firestore aggregation.
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

// GetAllTickets returns all tickets for full-scan diagnosis checks.
func (r *Firestore) GetAllTickets(ctx context.Context) ([]*ticket.Ticket, error) {
	iter := r.db.Collection(collectionTickets).Documents(ctx)

	var tickets []*ticket.Ticket
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, r.eb.Wrap(err, "failed to iterate tickets")
		}
		var t ticket.Ticket
		if err := doc.DataTo(&t); err != nil {
			return nil, r.eb.Wrap(err, "failed to unmarshal ticket", goerr.V("id", doc.Ref.ID))
		}
		tickets = append(tickets, &t)
	}
	return tickets, nil
}
