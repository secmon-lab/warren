package repository

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
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
	commentCollection     = "comments"
)

func (r *Firestore) PutAlert(ctx context.Context, alert alert.Alert) error {
	alert.UpdatedAt = clock.Now(ctx)
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

func (r *Firestore) GetAlertsBySlackThread(ctx context.Context, thread slack.Thread) ([]alert.Alert, error) {
	iter := r.db.Collection(collectionAlerts).
		Where("SlackThread.ChannelID", "==", thread.ChannelID).
		Where("SlackThread.ThreadID", "==", thread.ThreadID).
		Documents(ctx)

	var alerts []alert.Alert
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get alert by slack thread", goerr.V("slack_thread", thread))
		}

		var alert alert.Alert
		if err := doc.DataTo(&alert); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert", goerr.V("slack_message_id", thread.ThreadID))
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (r *Firestore) GetLatestAlerts(ctx context.Context, oldest time.Time, limit int) ([]alert.Alert, error) {
	iter := r.db.Collection(collectionAlerts).
		Where("CreatedAt", ">=", oldest).
		OrderBy("CreatedAt", firestore.Desc).
		Documents(ctx)

	var alerts []alert.Alert
	for len(alerts) < limit {
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

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (r *Firestore) PutAlertComment(ctx context.Context, comment alert.AlertComment) error {
	commentDoc := r.db.Collection(collectionAlerts).Doc(comment.AlertID.String()).Collection(commentCollection).Doc(comment.Timestamp)
	_, err := commentDoc.Set(ctx, comment)
	if err != nil {
		return goerr.Wrap(err, "failed to insert alert comment")
	}
	return nil
}

func (r *Firestore) GetAlertComments(ctx context.Context, alertID types.AlertID) ([]alert.AlertComment, error) {
	iter := r.db.Collection(collectionAlerts).Doc(alertID.String()).Collection(commentCollection).OrderBy("Timestamp", firestore.Desc).Documents(ctx)

	var comments []alert.AlertComment
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next alert comment")
		}

		var comment alert.AlertComment
		if err := doc.DataTo(&comment); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert comment")
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

const (
	policyVersionCollection = "versions"
)

func (r *Firestore) GetAlertsByStatus(ctx context.Context, status alert.Status) ([]alert.Alert, error) {
	iter := r.db.Collection(collectionAlerts).Where("Status", "==", status).Documents(ctx)

	var alerts []alert.Alert
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

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (r *Firestore) GetAlertsWithoutStatus(ctx context.Context, status alert.Status) ([]alert.Alert, error) {
	iter := r.db.Collection(collectionAlerts).Where("Status", "!=", status).Documents(ctx)

	var alerts []alert.Alert
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

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (r *Firestore) BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) ([]alert.Alert, error) {
	var alerts []alert.Alert

	// Process in batches of 30
	for i := 0; i < len(alertIDs); i += 30 {
		end := i + 30
		if end > len(alertIDs) {
			end = len(alertIDs)
		}

		batch := alertIDs[i:end]
		iter := r.db.Collection(collectionAlerts).Where("ID", "in", batch).Documents(ctx)

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

			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

func (r *Firestore) GetPolicyDiff(ctx context.Context, id types.PolicyDiffID) (*policy.Diff, error) {
	doc, err := r.db.Collection(collectionPolicyDiffs).Doc(string(id)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get policy diff", goerr.V("id", id))
	}

	var policyDiff policy.Diff
	if err := doc.DataTo(&policyDiff); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to policy diff")
	}

	return &policyDiff, nil
}

func (r *Firestore) PutPolicyDiff(ctx context.Context, diff *policy.Diff) error {
	doc := r.db.Collection(collectionPolicyDiffs).Doc(string(diff.ID))
	_, err := doc.Set(ctx, diff)
	if err != nil {
		return goerr.Wrap(err, "failed to put policy diff", goerr.V("id", diff.ID))
	}
	return nil
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

func (r *Firestore) PutAlertList(ctx context.Context, list alert.List) error {
	doc := r.db.Collection(collectionAlertLists).Doc(list.ID.String())
	_, err := doc.Set(ctx, list)
	if err != nil {
		return goerr.Wrap(err, "failed to put alert list", goerr.V("id", list.ID))
	}
	return nil
}

func (r *Firestore) GetAlertsBySpan(ctx context.Context, begin, end time.Time) ([]alert.Alert, error) {
	iter := r.db.Collection(collectionAlerts).
		Where("CreatedAt", ">=", begin).
		Where("CreatedAt", "<=", end).
		Documents(ctx)

	var alerts []alert.Alert
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

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (r *Firestore) GetLatestAlertListInThread(ctx context.Context, thread slack.Thread) (*alert.List, error) {
	iter := r.db.Collection(collectionAlertLists).
		Where("SlackThread.ChannelID", "==", thread.ChannelID).
		Where("SlackThread.ThreadID", "==", thread.ThreadID).
		OrderBy("CreatedAt", firestore.Desc).
		Limit(1).
		Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get latest alert list in thread", goerr.V("thread", thread))
	}

	var alertList alert.List
	if err := doc.DataTo(&alertList); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to alert list")
	}

	return &alertList, nil
}

func (r *Firestore) GetHistory(ctx context.Context, thread slack.Thread) (*chat.History, error) {
	doc := r.db.Collection("chat_histories").Doc(fmt.Sprintf("%s_%s", thread.ChannelID, thread.ThreadID))
	docSnap, err := doc.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get chat history")
	}

	var history chat.History
	if err := docSnap.DataTo(&history); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to chat history")
	}

	return &history, nil
}

func (r *Firestore) PutHistory(ctx context.Context, history chat.History) error {
	doc := r.db.Collection("chat_histories").Doc(fmt.Sprintf("%s_%s", history.Thread.ChannelID, history.Thread.ThreadID))
	_, err := doc.Set(ctx, history)
	if err != nil {
		return goerr.Wrap(err, "failed to put chat history")
	}
	return nil
}

func (r *Firestore) BatchUpdateAlertStatus(ctx context.Context, alertIDs []types.AlertID, status alert.Status, reason string) error {
	writer := r.db.BulkWriter(ctx)
	defer writer.End()

	jobs := make(map[types.AlertID]*firestore.BulkWriterJob)
	for _, alertID := range alertIDs {
		ref := r.db.Collection(collectionAlerts).Doc(alertID.String())
		job, err := writer.Update(ref, []firestore.Update{
			{
				Path:  "Status",
				Value: status,
			},
			{
				Path:  "Reason",
				Value: reason,
			},
		})
		if err != nil {
			return goerr.Wrap(err, "failed to update alert status", goerr.V("alert_id", alertID))
		}
		jobs[alertID] = job
	}

	writer.End()

	for alertID, job := range jobs {
		if _, err := job.Results(); err != nil {
			return goerr.Wrap(err, "failed to update alert status", goerr.V("alert_id", alertID))
		}
	}

	return nil
}

func (r *Firestore) BatchUpdateAlertConclusion(ctx context.Context, alertIDs []types.AlertID, conclusion alert.AlertConclusion, reason string) error {
	writer := r.db.BulkWriter(ctx)
	defer writer.End()

	jobs := make(map[types.AlertID]*firestore.BulkWriterJob)
	for _, alertID := range alertIDs {
		ref := r.db.Collection(collectionAlerts).Doc(alertID.String())
		job, err := writer.Update(ref, []firestore.Update{
			{
				Path:  "Conclusion",
				Value: conclusion,
			},
			{
				Path:  "Reason",
				Value: reason,
			},
		})
		if err != nil {
			return goerr.Wrap(err, "failed to update alert conclusion", goerr.V("alert_id", alertID))
		}
		jobs[alertID] = job
	}

	writer.End()

	for alertID, job := range jobs {
		if _, err := job.Results(); err != nil {
			return goerr.Wrap(err, "failed to update alert conclusion", goerr.V("alert_id", alertID))
		}
	}

	return nil
}

func (r *Firestore) GetAlertByThread(ctx context.Context, thread slack.Thread) (*alert.Alert, error) {
	iter := r.db.Collection(collectionAlerts).
		Where("SlackThread.ChannelID", "==", thread.ChannelID).
		Where("SlackThread.ThreadID", "==", thread.ThreadID).
		Limit(1).
		Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get alert by thread", goerr.V("thread", thread))
	}

	var alert alert.Alert
	if err := doc.DataTo(&alert); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to alert")
	}

	return &alert, nil
}
