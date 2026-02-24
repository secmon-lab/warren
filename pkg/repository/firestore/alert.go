package firestore

import (
	"context"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository/activityutil"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *Firestore) PutAlert(ctx context.Context, a alert.Alert) error {
	// Reject alerts with invalid embeddings (nil, empty, or zero vector)
	if isInvalidEmbedding(a.Embedding) {
		return r.eb.New("alert has invalid embedding (nil, empty, or zero vector)",
			goerr.TV(errutil.AlertIDKey, a.ID),
			goerr.V("embedding_length", len(a.Embedding)))
	}

	alertDoc := r.db.Collection(collectionAlerts).Doc(a.ID.String())
	_, err := alertDoc.Set(ctx, a)
	if err != nil {
		return r.eb.Wrap(err, "failed to put alert",
			goerr.TV(errutil.AlertIDKey, a.ID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
}

func (r *Firestore) GetAlert(ctx context.Context, alertID types.AlertID) (*alert.Alert, error) {
	alertDoc := r.db.Collection(collectionAlerts).Doc(alertID.String())
	doc, err := alertDoc.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.New("alert not found",
				goerr.TV(errutil.AlertIDKey, alertID),
				goerr.T(errutil.TagNotFound))
		}
		return nil, r.eb.Wrap(err, "failed to get alert",
			goerr.TV(errutil.AlertIDKey, alertID),
			goerr.T(errutil.TagDatabase))
	}

	var a alert.Alert
	if err := doc.DataTo(&a); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to alert",
			goerr.TV(errutil.AlertIDKey, alertID),
			goerr.T(errutil.TagInternal))
	}

	return &a, nil
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

		var alertData alert.Alert
		if err := doc.DataTo(&alertData); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}
		alerts = append(alerts, &alertData)
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

func (r *Firestore) GetAlertsByThread(ctx context.Context, thread slack.Thread) (alert.Alerts, error) {
	iter := r.db.Collection(collectionAlerts).
		Where("SlackThread.ChannelID", "==", thread.ChannelID).
		Where("SlackThread.ThreadID", "==", thread.ThreadID).
		Documents(ctx)

	var alerts alert.Alerts
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get alerts by thread", goerr.V("thread", thread))
		}

		var v alert.Alert
		if err := doc.DataTo(&v); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}
		alerts = append(alerts, &v)
	}

	return alerts, nil
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

		var alertData alert.Alert
		if err := doc.DataTo(&alertData); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}
		alerts = append(alerts, &alertData)
	}

	return alerts, nil
}

func (r *Firestore) GetAlertWithoutTicket(ctx context.Context, offset, limit int) (alert.Alerts, error) {
	query := r.db.Collection(collectionAlerts).
		Where("TicketID", "==", "").
		Where("Status", "in", []string{string(alert.AlertStatusActive), "unbound", ""})

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
		v.Normalize()
		alerts = append(alerts, &v)
	}

	return alerts, nil
}

func (r *Firestore) CountAlertsWithoutTicket(ctx context.Context) (int, error) {
	query := r.db.Collection(collectionAlerts).
		Where("TicketID", "==", "").
		Where("Status", "in", []string{string(alert.AlertStatusActive), "unbound", ""})

	result, err := query.NewAggregationQuery().WithCount("total").Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to count alerts")
	}

	return extractCountFromAggregationResult(result, "total")
}

func (r *Firestore) GetDeclinedAlerts(ctx context.Context, offset, limit int) (alert.Alerts, error) {
	query := r.db.Collection(collectionAlerts).
		Where("TicketID", "==", "").
		Where("Status", "==", string(alert.AlertStatusDeclined))

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
		v.Normalize()
		alerts = append(alerts, &v)
	}

	return alerts, nil
}

func (r *Firestore) CountDeclinedAlerts(ctx context.Context) (int, error) {
	query := r.db.Collection(collectionAlerts).
		Where("TicketID", "==", "").
		Where("Status", "==", string(alert.AlertStatusDeclined))

	result, err := query.NewAggregationQuery().WithCount("total").Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to count declined alerts")
	}

	return extractCountFromAggregationResult(result, "total")
}

func (r *Firestore) UpdateAlertStatus(ctx context.Context, alertID types.AlertID, status alert.AlertStatus) error {
	alertDoc := r.db.Collection(collectionAlerts).Doc(alertID.String())
	_, err := alertDoc.Update(ctx, []firestore.Update{
		{
			Path:  "Status",
			Value: string(status),
		},
	})
	if err != nil {
		return r.eb.Wrap(err, "failed to update alert status",
			goerr.TV(errutil.AlertIDKey, alertID),
			goerr.T(errutil.TagDatabase))
	}
	return nil
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

		var alertData alert.Alert
		if err := doc.DataTo(&alertData); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert", goerr.V("doc.ref.id", doc.Ref.ID))
		}
		alerts = append(alerts, &alertData)
	}
	return alerts, nil
}

func (r *Firestore) FindSimilarAlerts(ctx context.Context, target alert.Alert, limit int) (alert.Alerts, error) {
	// Build vector search query
	query := r.db.Collection(collectionAlerts).
		FindNearest("Embedding",
			target.Embedding,
			limit+1, // Add 1 to exclude target itself
			firestore.DistanceMeasureCosine,
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

func (r *Firestore) FindNearestAlerts(ctx context.Context, embedding []float32, limit int) (alert.Alerts, error) {
	// Convert []float32 to firestore.Vector32
	vector32 := firestore.Vector32(embedding[:])

	// Check if the input embedding is zero vector
	isZeroVector := true
	for _, v := range embedding {
		if v != 0 {
			isZeroVector = false
			break
		}
	}
	if isZeroVector {
		return alert.Alerts{}, nil
	}

	// Build vector search query
	query := r.db.Collection(collectionAlerts).
		FindNearest("Embedding",
			vector32,
			limit,
			firestore.DistanceMeasureCosine,
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
		// Check if embedding is zero vector and skip it to prevent Firestore vector search errors
		if len(alert.Embedding) > 0 {
			isZeroVector := true
			for _, v := range alert.Embedding {
				if v != 0 {
					isZeroVector = false
					break
				}
			}
			if isZeroVector {
				// Clear the embedding if it's a zero vector
				alert.Embedding = nil
			}
		}

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

func (r *Firestore) GetAlertsWithInvalidEmbedding(ctx context.Context) (alert.Alerts, error) {
	// Get all alerts and filter for invalid embeddings
	// This is necessary because Firestore doesn't support complex queries for array fields
	iter := r.db.Collection(collectionAlerts).Documents(ctx)

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
		// Check if embedding is invalid (nil, empty, or zero vector)
		if isInvalidEmbedding(v.Embedding) {
			alerts = append(alerts, &v)
		}
	}

	return alerts, nil
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
				alertTitles = append(alertTitles, alert.Title)
			}
		}

		if len(alertIDs) > 1 {
			if err := activityutil.CreateBulkAlertBoundActivity(ctx, r, alertIDs, ticketID, ticket.Title, alertTitles); err != nil {
				return goerr.Wrap(err, "failed to create bulk alert bound activity", goerr.V("ticket_id", ticketID))
			}
		} else if len(alertIDs) == 1 {
			alertTitle := ""
			if len(alertTitles) > 0 {
				alertTitle = alertTitles[0]
			}
			if err := activityutil.CreateAlertBoundActivity(ctx, r, alertIDs[0], ticketID, alertTitle, ticket.Title); err != nil {
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

func (r *Firestore) GetAlertsByTicketID(ctx context.Context, ticketID types.TicketID) (alert.Alerts, error) {
	iter := r.db.Collection(collectionAlerts).Where("TicketID", "==", ticketID).Documents(ctx)

	var alerts alert.Alerts
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get alerts by ticket ID", goerr.V("ticket_id", ticketID))
		}

		var v alert.Alert
		if err := doc.DataTo(&v); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}
		alerts = append(alerts, &v)
	}

	return alerts, nil
}

func (r *Firestore) GetAlertsByIDs(ctx context.Context, alertIDs []types.AlertID) (alert.Alerts, error) {
	return r.BatchGetAlerts(ctx, alertIDs)
}

func (r *Firestore) GetAllAlerts(ctx context.Context, offset, limit int) (alert.Alerts, error) {
	query := r.db.Collection(collectionAlerts).OrderBy("CreatedAt", firestore.Desc)

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

func (r *Firestore) CountAlerts(ctx context.Context) (int, error) {
	result, err := r.db.Collection(collectionAlerts).NewAggregationQuery().WithCount("total").Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to count alerts")
	}

	return extractCountFromAggregationResult(result, "total")
}

func (r *Firestore) CountAlertsBySchema(ctx context.Context, schema types.AlertSchema) (int, error) {
	query := r.db.Collection(collectionAlerts).Where("Schema", "==", schema)
	result, err := query.NewAggregationQuery().WithCount("total").Get(ctx)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to count alerts by schema", goerr.V("schema", schema))
	}

	return extractCountFromAggregationResult(result, "total")
}

func (r *Firestore) DeleteAlertList(ctx context.Context, listID types.AlertListID) error {
	_, err := r.db.Collection(collectionAlertLists).Doc(listID.String()).Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete alert list", goerr.V("list_id", listID))
	}
	return nil
}

func (r *Firestore) AddAlertsToList(ctx context.Context, listID types.AlertListID, alertIDs []types.AlertID) error {
	listDoc := r.db.Collection(collectionAlertLists).Doc(listID.String())
	_, err := listDoc.Update(ctx, []firestore.Update{
		{
			Path:  "AlertIDs",
			Value: firestore.ArrayUnion(alertIDsToInterface(alertIDs)...),
		},
	})
	if err != nil {
		return goerr.Wrap(err, "failed to add alerts to list", goerr.V("list_id", listID))
	}
	return nil
}

func (r *Firestore) RemoveAlertsFromList(ctx context.Context, listID types.AlertListID, alertIDs []types.AlertID) error {
	listDoc := r.db.Collection(collectionAlertLists).Doc(listID.String())
	_, err := listDoc.Update(ctx, []firestore.Update{
		{
			Path:  "AlertIDs",
			Value: firestore.ArrayRemove(alertIDsToInterface(alertIDs)...),
		},
	})
	if err != nil {
		return goerr.Wrap(err, "failed to remove alerts from list", goerr.V("list_id", listID))
	}
	return nil
}

func (r *Firestore) GetAlertLists(ctx context.Context) ([]*alert.List, error) {
	iter := r.db.Collection(collectionAlertLists).Documents(ctx)

	var lists []*alert.List
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get alert lists")
		}

		var alertList alert.List
		if err := doc.DataTo(&alertList); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert list")
		}

		lists = append(lists, &alertList)
	}

	return lists, nil
}

// alertIDsToInterface converts []types.AlertID to []any for Firestore ArrayUnion
func alertIDsToInterface(alertIDs []types.AlertID) []any {
	interfaces := make([]any, len(alertIDs))
	for i, id := range alertIDs {
		interfaces[i] = id.String()
	}
	return interfaces
}
