package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/firestore/apiv1/firestorepb"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/user"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Firestore struct {
	db *firestore.Client
}

var _ interfaces.Repository = &Firestore{}

func NewFirestore(ctx context.Context, projectID, databaseID string) (*Firestore, error) {
	db, err := firestore.NewClientWithDatabase(ctx, projectID, databaseID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create firestore client")
	}

	return &Firestore{
		db: db,
	}, nil
}

func (r *Firestore) Close() error {
	return r.db.Close()
}

const (
	collectionAlerts      = "alerts"
	collectionPolicies    = "policies"
	collectionPolicyDiffs = "diffs"
	collectionAlertLists  = "lists"
	collectionSessions    = "sessions"
	collectionHistories   = "histories"
	collectionNotes       = "notes"
	collectionTickets     = "tickets"
	collectionComments    = "comments"
	collectionTokens      = "tokens"
	collectionActivities  = "activities"
)

// extractCountFromAggregationResult extracts an integer count from a Firestore aggregation result.
// It handles both int64 and *firestorepb.Value types that can be returned by the Firestore client.
func extractCountFromAggregationResult(result firestore.AggregationResult, alias string) (int, error) {
	countVal, ok := result[alias]
	if !ok {
		return 0, goerr.New("count alias not found in aggregation result", goerr.V("alias", alias))
	}

	switch v := countVal.(type) {
	case int64:
		return int(v), nil
	case *firestorepb.Value:
		if v != nil && v.ValueType != nil {
			if _, okType := v.ValueType.(*firestorepb.Value_IntegerValue); okType {
				return int(v.GetIntegerValue()), nil
			}
			return 0, goerr.New("firestorepb.Value from count is not an integer type",
				goerr.V("value_type", fmt.Sprintf("%T", v.ValueType)), goerr.V("alias", alias))
		}
		return 0, goerr.New("count value is a nil or invalid *firestorepb.Value", goerr.V("alias", alias))
	default:
		return 0, goerr.New("unexpected count value type from Firestore aggregation",
			goerr.V("type", fmt.Sprintf("%T", v)), goerr.V("value", v), goerr.V("alias", alias))
	}
}

func (r *Firestore) PutAlert(ctx context.Context, alert alert.Alert) error {
	alertDoc := r.db.Collection(collectionAlerts).Doc(alert.ID.String())
	_, err := alertDoc.Set(ctx, alert)
	if err != nil {
		return goerr.Wrap(err, "failed to put alert")
	}
	return nil
}

func (r *Firestore) GetAlert(ctx context.Context, alertID types.AlertID) (*alert.Alert, error) {
	alertDoc := r.db.Collection(collectionAlerts).Doc(alertID.String())
	doc, err := alertDoc.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.New("alert not found", goerr.V("alert_id", alertID))
		}
		return nil, goerr.Wrap(err, "failed to get alert", goerr.V("alert_id", alertID))
	}

	var alert alert.Alert
	if err := doc.DataTo(&alert); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to alert", goerr.V("alert_id", alertID))
	}

	return &alert, nil
}

func (r *Firestore) GetAlertListByThread(ctx context.Context, thread slack.Thread) (*alert.List, error) {
	iter := r.db.Collection(collectionAlertLists).Where("SlackThread.ChannelID", "==", thread.ChannelID).Where("SlackThread.ThreadID", "==", thread.ThreadID).Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get alert list by thread", goerr.V("slack_thread", thread))
	}

	var alertList alert.List
	if err := doc.DataTo(&alertList); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to alert list")
	}

	return &alertList, nil
}

func (r *Firestore) GetAlertList(ctx context.Context, listID types.AlertListID) (*alert.List, error) {
	doc, err := r.db.Collection(collectionAlertLists).Doc(listID.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
	}

	var alertList alert.List
	if err := doc.DataTo(&alertList); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to alert list")
	}

	return &alertList, nil
}

func (r *Firestore) PutAlertList(ctx context.Context, list *alert.List) error {
	doc := r.db.Collection(collectionAlertLists).Doc(list.ID.String())
	_, err := doc.Set(ctx, list)
	if err != nil {
		return goerr.Wrap(err, "failed to put alert list", goerr.V("id", list.ID))
	}
	return nil
}

func (r *Firestore) GetAlertsBySpan(ctx context.Context, begin, end time.Time) (alert.Alerts, error) {
	iter := r.db.Collection(collectionAlerts).
		Where("CreatedAt", ">=", begin).
		Where("CreatedAt", "<=", end).
		Documents(ctx)

	var alerts alert.Alerts
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next alert")
		}

		var alert alert.Alert
		if err := doc.DataTo(&alert); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

func (r *Firestore) GetLatestAlertListInThread(ctx context.Context, thread slack.Thread) (*alert.List, error) {
	iter := r.db.Collection(collectionAlertLists).
		Where("SlackThread.ChannelID", "==", thread.ChannelID).
		Where("SlackThread.ThreadID", "==", thread.ThreadID).
		Documents(ctx)

	var lists []*alert.List
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get alert lists in thread", goerr.V("thread", thread))
		}

		var alertList alert.List
		if err := doc.DataTo(&alertList); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert list")
		}

		lists = append(lists, &alertList)
	}

	if len(lists) == 0 {
		return nil, nil
	}

	// Sort by CreatedAt in descending order
	sort.Slice(lists, func(i, j int) bool {
		return lists[i].CreatedAt.After(lists[j].CreatedAt)
	})

	return lists[0], nil
}

func (r *Firestore) GetAlertListsInThread(ctx context.Context, thread slack.Thread) ([]*alert.List, error) {
	iter := r.db.Collection(collectionAlertLists).
		Where("SlackThread.ChannelID", "==", thread.ChannelID).
		Where("SlackThread.ThreadID", "==", thread.ThreadID).
		Documents(ctx)

	var lists []*alert.List
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get alert lists in thread", goerr.V("thread", thread))
		}

		var alertList alert.List
		if err := doc.DataTo(&alertList); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert list")
		}

		lists = append(lists, &alertList)
	}

	// Sort by CreatedAt in ascending order (oldest first)
	sort.Slice(lists, func(i, j int) bool {
		return lists[i].CreatedAt.Before(lists[j].CreatedAt)
	})

	return lists, nil
}

func (r *Firestore) GetLatestAlertByThread(ctx context.Context, thread slack.Thread) (*alert.Alert, error) {
	iter := r.db.Collection(collectionAlerts).
		Where("SlackThread.ChannelID", "==", thread.ChannelID).
		Where("SlackThread.ThreadID", "==", thread.ThreadID).
		Limit(1).
		Documents(ctx)

	var resp *alert.Alert
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get alert by thread", goerr.V("thread", thread))
		}

		var v alert.Alert
		if err := doc.DataTo(&v); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}
		if resp == nil {
			resp = &v
		} else if v.CreatedAt.After(resp.CreatedAt) {
			resp = &v
		}
	}

	return resp, nil
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

func (r *Firestore) SearchAlerts(ctx context.Context, path, op string, value any, limit int) (alert.Alerts, error) {
	iter := r.db.Collection(collectionAlerts).Where(path, op, value).Limit(limit).Documents(ctx)

	var alerts alert.Alerts
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next alert")
		}

		var alert alert.Alert
		if err := doc.DataTo(&alert); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// Ticket related methods
func (r *Firestore) GetTicket(ctx context.Context, ticketID types.TicketID) (*ticket.Ticket, error) {
	doc, err := r.db.Collection(collectionTickets).Doc(ticketID.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
		}
		return nil, goerr.Wrap(err, "failed to get ticket", goerr.V("ticket_id", ticketID))
	}

	var t ticket.Ticket
	if err := doc.DataTo(&t); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to ticket", goerr.V("ticket_id", ticketID))
	}

	return &t, nil
}

func (r *Firestore) PutTicket(ctx context.Context, t ticket.Ticket) error {
	// Check if ticket already exists to determine if this is create or update
	existingTicket, err := r.GetTicket(ctx, t.ID)
	isUpdate := err == nil && existingTicket != nil

	_, err = r.db.Collection(collectionTickets).Doc(t.ID.String()).Set(ctx, t)
	if err != nil {
		return goerr.Wrap(err, "failed to put ticket", goerr.V("ticket_id", t.ID))
	}

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

func (r *Firestore) PutTicketComment(ctx context.Context, comment ticket.Comment) error {
	_, err := r.db.Collection(collectionTickets).Doc(comment.TicketID.String()).Collection(collectionComments).Doc(comment.ID.String()).Set(ctx, comment)
	if err != nil {
		return goerr.Wrap(err, "failed to put ticket comment", goerr.V("ticket_id", comment.TicketID))
	}

	// Create activity for comment addition - only for user comments, not agent
	if !user.IsAgent(ctx) {
		// Get ticket for activity creation
		if t, err := r.GetTicket(ctx, comment.TicketID); err == nil {
			if err := createCommentActivity(ctx, r, comment.TicketID, comment.ID, t.Metadata.Title); err != nil {
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

// Alert-Ticket binding methods
func (r *Firestore) BindAlertsToTicket(ctx context.Context, alertIDs []types.AlertID, ticketID types.TicketID) error {
	// Update alerts using BulkWriter for performance
	bw := r.db.BulkWriter(ctx)
	var jobs []*firestore.BulkWriterJob
	for _, alertID := range alertIDs {
		alertDoc := r.db.Collection(collectionAlerts).Doc(alertID.String())
		job, err := bw.Update(alertDoc, []firestore.Update{
			{
				Path:  "TicketID",
				Value: ticketID,
			},
		})
		if err != nil {
			return goerr.Wrap(err, "failed to bind alert to ticket", goerr.V("alert_id", alertID), goerr.V("ticket_id", ticketID))
		}
		jobs = append(jobs, job)
	}
	bw.End()

	for _, job := range jobs {
		if _, err := job.Results(); err != nil {
			return goerr.Wrap(err, "failed to commit bulk writer job")
		}
	}

	// Update ticket's AlertIDs array using transaction for consistency
	err := r.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		ticketDoc := r.db.Collection(collectionTickets).Doc(ticketID.String())

		// Verify the ticket exists
		ticketSnap, err := tx.Get(ticketDoc)
		if err != nil {
			return goerr.Wrap(err, "failed to get ticket in transaction", goerr.V("ticket_id", ticketID))
		}
		if !ticketSnap.Exists() {
			return goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
		}

		// Update ticket's AlertIDs array to include the newly bound alerts
		err = tx.Update(ticketDoc, []firestore.Update{
			{
				Path:  "AlertIDs",
				Value: firestore.ArrayUnion(alertIDsToInterface(alertIDs)...),
			},
		})
		if err != nil {
			return goerr.Wrap(err, "failed to update ticket AlertIDs in transaction", goerr.V("ticket_id", ticketID))
		}

		return nil
	})

	if err != nil {
		return goerr.Wrap(err, "transaction failed for updating ticket AlertIDs")
	}

	// Create activity for bulk alert binding (outside transaction to avoid conflicts)
	// Get ticket for activity creation
	ticket, ticketErr := r.GetTicket(ctx, ticketID)
	if ticketErr == nil {
		// Get alerts for activity creation
		var alertTitles []string
		for _, alertID := range alertIDs {
			if alert, err := r.GetAlert(ctx, alertID); err == nil {
				alertTitles = append(alertTitles, alert.Metadata.Title)
			}
		}

		if len(alertIDs) > 1 {
			if err := createBulkAlertBoundActivity(ctx, r, alertIDs, ticketID, ticket.Metadata.Title, alertTitles); err != nil {
				return goerr.Wrap(err, "failed to create bulk alert bound activity", goerr.V("ticket_id", ticketID))
			}
		} else if len(alertIDs) == 1 {
			alertTitle := ""
			if len(alertTitles) > 0 {
				alertTitle = alertTitles[0]
			}
			if err := createAlertBoundActivity(ctx, r, alertIDs[0], ticketID, alertTitle, ticket.Metadata.Title); err != nil {
				return goerr.Wrap(err, "failed to create alert bound activity", goerr.V("alert_id", alertIDs[0]), goerr.V("ticket_id", ticketID))
			}
		}
	}

	return nil
}

func (r *Firestore) UnbindAlertFromTicket(ctx context.Context, alertID types.AlertID) error {
	alertDoc := r.db.Collection(collectionAlerts).Doc(alertID.String())
	_, err := alertDoc.Update(ctx, []firestore.Update{
		{
			Path:  "TicketID",
			Value: "",
		},
	})
	if err != nil {
		return goerr.Wrap(err, "failed to unbind alert from ticket", goerr.V("alert_id", alertID))
	}
	return nil
}

func (r *Firestore) GetAlertWithoutTicket(ctx context.Context, offset, limit int) (alert.Alerts, error) {
	query := r.db.Collection(collectionAlerts).Where("TicketID", "==", "")

	// Apply offset and limit to the query
	if offset > 0 {
		query = query.Offset(offset)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	iter := query.Documents(ctx)

	var alerts alert.Alerts
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next alert")
		}

		var v alert.Alert
		if err := doc.DataTo(&v); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}

		alerts = append(alerts, &v)
	}

	return alerts, nil
}

func (r *Firestore) CountAlertsWithoutTicket(ctx context.Context) (int, error) {
	query := r.db.Collection(collectionAlerts).Where("TicketID", "==", "")

	result, err := query.NewAggregationQuery().WithCount("total").Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to count alerts")
	}

	return extractCountFromAggregationResult(result, "total")
}

func (r *Firestore) BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) (alert.Alerts, error) {
	var alerts alert.Alerts
	var docRefs []*firestore.DocumentRef
	for _, id := range alertIDs {
		docRef := r.db.Collection(collectionAlerts).Doc(id.String())
		docRefs = append(docRefs, docRef)
	}

	docs, err := r.db.GetAll(ctx, docRefs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alerts")
	}

	for _, doc := range docs {
		if !doc.Exists() {
			continue
		}

		var alert alert.Alert
		if err := doc.DataTo(&alert); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert", goerr.V("doc.ref.id", doc.Ref.ID))
		}

		alerts = append(alerts, &alert)
	}
	return alerts, nil
}

func (r *Firestore) FindSimilarAlerts(ctx context.Context, target alert.Alert, limit int) (alert.Alerts, error) {
	// Build vector search query
	query := r.db.Collection(collectionAlerts).
		FindNearest("Embedding",
			target.Embedding,
			limit+1, // Add 1 to exclude target itself
			firestore.DistanceMeasureEuclidean,
			&firestore.FindNearestOptions{
				DistanceResultField: "vector_distance",
			})

	iter := query.Documents(ctx)
	var alerts alert.Alerts
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next alert")
		}

		var a alert.Alert
		if err := doc.DataTo(&a); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}

		// Exclude the same alert
		if a.ID == target.ID {
			continue
		}

		// Only add alerts that have embeddings
		if len(a.Embedding) > 0 {
			alerts = append(alerts, &a)
		}
	}

	// Apply limit
	if limit > 0 && limit < len(alerts) {
		alerts = alerts[:limit]
	}

	return alerts, nil
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
			firestore.DistanceMeasureEuclidean,
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

		// Only add tickets that have embeddings
		if len(t.Embedding) > 0 {
			tickets = append(tickets, &t)
		}
	}

	return tickets, nil
}

func (r *Firestore) FindNearestAlerts(ctx context.Context, embedding []float32, limit int) (alert.Alerts, error) {
	// Convert []float32 to firestore.Vector32
	vector32 := firestore.Vector32(embedding[:])

	// Build vector search query
	query := r.db.Collection(collectionAlerts).
		FindNearest("Embedding",
			vector32,
			limit,
			firestore.DistanceMeasureEuclidean,
			&firestore.FindNearestOptions{
				DistanceResultField: "vector_distance",
			})

	iter := query.Documents(ctx)
	var alerts alert.Alerts
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next alert")
		}

		var a alert.Alert
		if err := doc.DataTo(&a); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}

		// Only add alerts that have embeddings
		if len(a.Embedding) > 0 {
			alerts = append(alerts, &a)
		}
	}

	return alerts, nil
}

func (r *Firestore) BatchPutAlerts(ctx context.Context, alerts alert.Alerts) error {
	bw := r.db.BulkWriter(ctx)
	var jobs []*firestore.BulkWriterJob

	for _, alert := range alerts {
		alertDoc := r.db.Collection(collectionAlerts).Doc(alert.ID.String())
		job, err := bw.Set(alertDoc, alert)
		if err != nil {
			return goerr.Wrap(err, "failed to put alert", goerr.V("alert_id", alert.ID))
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

func (r *Firestore) GetAlertWithoutEmbedding(ctx context.Context) (alert.Alerts, error) {
	iter := r.db.Collection(collectionAlerts).Where("Embedding", "==", nil).Documents(ctx)

	var alerts alert.Alerts
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next alert")
		}

		var v alert.Alert
		if err := doc.DataTo(&v); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}

		if len(v.Embedding) == 0 {
			alerts = append(alerts, &v)
		}
	}

	return alerts, nil
}

func (r *Firestore) FindNearestTicketsWithSpan(ctx context.Context, embedding []float32, begin, end time.Time, limit int) ([]*ticket.Ticket, error) {
	iter := r.db.Collection(collectionTickets).
		OrderBy("CreatedAt", firestore.Desc).
		Where("CreatedAt", "<=", end).
		Where("CreatedAt", ">=", begin).
		FindNearest("Embedding",
			firestore.Vector32(embedding[:]),
			limit,
			firestore.DistanceMeasureEuclidean,
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

// Token related methods
func (r *Firestore) PutToken(ctx context.Context, token *auth.Token) error {
	doc := r.db.Collection(collectionTokens).Doc(token.ID.String())
	_, err := doc.Set(ctx, token)
	if err != nil {
		return goerr.Wrap(err, "failed to put token", goerr.V("token_id", token.ID))
	}
	return nil
}

func (r *Firestore) GetToken(ctx context.Context, tokenID auth.TokenID) (*auth.Token, error) {
	doc, err := r.db.Collection(collectionTokens).Doc(tokenID.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.New("token not found", goerr.V("token_id", tokenID))
		}
		return nil, goerr.Wrap(err, "failed to get token", goerr.V("token_id", tokenID))
	}

	var token auth.Token
	if err := doc.DataTo(&token); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to token", goerr.V("token_id", tokenID))
	}

	token.ID = tokenID // Set the ID manually since it's not stored in the document
	return &token, nil
}

func (r *Firestore) DeleteToken(ctx context.Context, tokenID auth.TokenID) error {
	doc := r.db.Collection(collectionTokens).Doc(tokenID.String())
	_, err := doc.Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete token", goerr.V("token_id", tokenID))
	}
	return nil
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
		if err := createStatusChangeActivity(ctx, r, t.ID, t.Metadata.Title, oldStatus, newStatus); err != nil {
			return goerr.Wrap(err, "failed to create status change activity", goerr.V("ticket_id", t.ID))
		}
	}

	return nil
}

// Activity related methods
func (r *Firestore) PutActivity(ctx context.Context, activity *activity.Activity) error {
	doc := r.db.Collection(collectionActivities).Doc(activity.ID.String())
	_, err := doc.Set(ctx, activity)
	if err != nil {
		return goerr.Wrap(err, "failed to put activity", goerr.V("activity_id", activity.ID))
	}

	return nil
}

func (r *Firestore) GetActivities(ctx context.Context, offset, limit int) ([]*activity.Activity, error) {
	iter := r.db.Collection(collectionActivities).
		OrderBy("CreatedAt", firestore.Desc).
		Offset(offset).
		Limit(limit).
		Documents(ctx)

	var activities []*activity.Activity
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next activity")
		}

		var a activity.Activity
		if err := doc.DataTo(&a); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to activity")
		}

		activities = append(activities, &a)
	}

	return activities, nil
}

func (r *Firestore) CountActivities(ctx context.Context) (int, error) {
	countQuery := r.db.Collection(collectionActivities).NewAggregationQuery().WithCount("count")
	result, err := countQuery.Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to get activity count")
	}

	return extractCountFromAggregationResult(result, "count")
}

// alertIDsToInterface converts []types.AlertID to []interface{} for Firestore ArrayUnion
func alertIDsToInterface(alertIDs []types.AlertID) []interface{} {
	interfaces := make([]interface{}, len(alertIDs))
	for i, id := range alertIDs {
		interfaces[i] = id.String()
	}
	return interfaces
}
