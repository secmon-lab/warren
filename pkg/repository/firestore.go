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
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
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
)

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
	_, err := r.db.Collection(collectionTickets).Doc(t.ID.String()).Set(ctx, t)
	if err != nil {
		return goerr.Wrap(err, "failed to put ticket", goerr.V("ticket_id", t.ID))
	}
	return nil
}

func (r *Firestore) PutTicketComment(ctx context.Context, comment ticket.Comment) error {
	_, err := r.db.Collection(collectionTickets).Doc(comment.TicketID.String()).Collection(collectionComments).Doc(comment.ID.String()).Set(ctx, comment)
	if err != nil {
		return goerr.Wrap(err, "failed to put ticket comment", goerr.V("ticket_id", comment.TicketID))
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
func (r *Firestore) BindAlertToTicket(ctx context.Context, alertID types.AlertID, ticketID types.TicketID) error {
	alertDoc := r.db.Collection(collectionAlerts).Doc(alertID.String())
	_, err := alertDoc.Update(ctx, []firestore.Update{
		{
			Path:  "TicketID",
			Value: ticketID,
		},
	})
	if err != nil {
		return goerr.Wrap(err, "failed to bind alert to ticket", goerr.V("alert_id", alertID), goerr.V("ticket_id", ticketID))
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

func (r *Firestore) GetAlertWithoutTicket(ctx context.Context) (alert.Alerts, error) {
	iter := r.db.Collection(collectionAlerts).Where("TicketID", "==", "").Documents(ctx)

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

func (r *Firestore) BatchBindAlertsToTicket(ctx context.Context, alertIDs []types.AlertID, ticketID types.TicketID) error {
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

	return nil
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

func (r *Firestore) FindSimilarTickets(ctx context.Context, ticketID types.TicketID, limit int) ([]*ticket.Ticket, error) {
	// Get target ticket
	targetDoc := r.db.Collection(collectionTickets).Doc(ticketID.String())
	targetSnapshot, err := targetDoc.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
		}
		return nil, goerr.Wrap(err, "failed to get ticket", goerr.V("ticket_id", ticketID))
	}

	var target ticket.Ticket
	if err := targetSnapshot.DataTo(&target); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to ticket", goerr.V("ticket_id", ticketID))
	}

	// Build vector search query
	query := r.db.Collection(collectionTickets).
		FindNearest("Embedding",
			target.Embedding,
			limit+1, // Add 1 to exclude target itself
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

		// Exclude the same ticket
		if t.ID == target.ID {
			continue
		}

		// Only add tickets that have embeddings
		if len(t.Embedding) > 0 {
			tickets = append(tickets, &t)
		}
	}

	// Apply limit
	if limit > 0 && limit < len(tickets) {
		tickets = tickets[:limit]
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

	// The Firestore client library for Go typically returns int64 for count aggregations.
	// See: https://pkg.go.dev/cloud.google.com/go/firestore#AggregationResult
	// It's good practice to handle the specific expected type.
	countVal, ok := result["count"]
	if !ok {
		return 0, goerr.New("count alias not found in aggregation result")
	}

	switch v := countVal.(type) {
	case int64:
		return int(v), nil
	// It's less common for *firestorepb.Value to appear here directly from AggregationResult values,
	// but if it does, this handles it by checking the inner value type.
	case *firestorepb.Value:
		if v != nil && v.ValueType != nil {
			if _, okType := v.ValueType.(*firestorepb.Value_IntegerValue); okType {
				return int(v.GetIntegerValue()), nil
			}
			return 0, goerr.New("firestorepb.Value from count is not an integer type", goerr.V("value_type", fmt.Sprintf("%T", v.ValueType)))
		}
		return 0, goerr.New("count value is a nil or invalid *firestorepb.Value")
	default:
		// This case helps catch unexpected types if Firestore's behavior changes or if there's a misunderstanding.
		return 0, goerr.New("unexpected count value type from Firestore aggregation", goerr.V("type", fmt.Sprintf("%T", v)), goerr.V("value", v))
	}
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

	return nil
}

