package repository

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
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
	collectionAlertGroups = "groups"
	collectionPolicyDiffs = "diffs"
)

func (r *Firestore) PutAlert(ctx context.Context, alert model.Alert) error {
	alert.UpdatedAt = clock.Now(ctx)
	alertDoc := r.db.Collection(collectionAlerts).Doc(alert.ID.String())
	_, err := alertDoc.Set(ctx, alert)
	if err != nil {
		return goerr.Wrap(err, "failed to put alert")
	}
	return nil
}

func (r *Firestore) GetAlert(ctx context.Context, alertID model.AlertID) (*model.Alert, error) {
	alertDoc := r.db.Collection(collectionAlerts).Doc(alertID.String())
	doc, err := alertDoc.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.New("alert not found", goerr.V("alert_id", alertID))
		}
		return nil, goerr.Wrap(err, "failed to get alert", goerr.V("alert_id", alertID))
	}

	var alert model.Alert
	if err := doc.DataTo(&alert); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to alert", goerr.V("alert_id", alertID))
	}

	return &alert, nil
}

func (r *Firestore) GetAlertBySlackThread(ctx context.Context, thread model.SlackThread) (*model.Alert, error) {
	iter := r.db.Collection(collectionAlerts).
		Where("SlackThread.ChannelID", "==", thread.ChannelID).
		Where("SlackThread.ThreadID", "==", thread.ThreadID).
		Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get alert by slack thread", goerr.V("slack_thread", thread))
	}

	var alert model.Alert
	if err := doc.DataTo(&alert); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to alert", goerr.V("slack_message_id", thread.ThreadID))
	}

	return &alert, nil
}

func (r *Firestore) GetLatestAlerts(ctx context.Context, oldest time.Time, limit int) ([]model.Alert, error) {
	iter := r.db.Collection(collectionAlerts).
		Where("CreatedAt", ">=", oldest).
		OrderBy("CreatedAt", firestore.Desc).
		Documents(ctx)

	var alerts []model.Alert
	for len(alerts) < limit {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next alert")
		}

		var alert model.Alert
		if err := doc.DataTo(&alert); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}

		if alert.Status == model.AlertStatusMerged {
			continue
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

const commentCollection = "comments"

func (r *Firestore) InsertAlertComment(ctx context.Context, comment model.AlertComment) error {
	commentDoc := r.db.Collection(collectionAlerts).Doc(comment.AlertID.String()).Collection(commentCollection).Doc(comment.Timestamp)
	_, err := commentDoc.Set(ctx, comment)
	if err != nil {
		return goerr.Wrap(err, "failed to insert alert comment")
	}
	return nil
}

func (r *Firestore) GetAlertComments(ctx context.Context, alertID model.AlertID) ([]model.AlertComment, error) {
	iter := r.db.Collection(collectionAlerts).Doc(alertID.String()).Collection(commentCollection).OrderBy("Timestamp", firestore.Desc).Documents(ctx)

	var comments []model.AlertComment
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next alert comment")
		}

		var comment model.AlertComment
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

func (r *Firestore) GetPolicy(ctx context.Context, hash string) (*model.PolicyData, error) {
	// Get all versions ordered by timestamp descending to get latest
	iter := r.db.
		Collection(collectionPolicies).Doc(hash).
		Collection(policyVersionCollection).
		OrderBy("created_at", firestore.Desc).
		Limit(1).
		Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get policy")
	}

	var policy model.PolicyData
	if err := doc.DataTo(&policy); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal policy data")
	}

	return &policy, nil
}

func (r *Firestore) SavePolicy(ctx context.Context, policy *model.PolicyData) error {
	// Save under versions collection using timestamp as document ID
	timestamp := policy.CreatedAt.Format(time.RFC3339)
	_, err := r.db.
		Collection(collectionPolicies).Doc(policy.Hash).
		Collection("versions").Doc(timestamp).
		Set(ctx, policy)
	if err != nil {
		return goerr.Wrap(err, "failed to save policy", goerr.V("policy", policy))
	}
	return nil
}

func (r *Firestore) GetAlertsByParentID(ctx context.Context, parentID model.AlertID) ([]model.Alert, error) {
	iter := r.db.Collection(collectionAlerts).Where("ParentID", "==", parentID).Documents(ctx)

	var alerts []model.Alert
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next alert")
		}

		var alert model.Alert
		if err := doc.DataTo(&alert); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (r *Firestore) GetAlertsByStatus(ctx context.Context, status model.AlertStatus) ([]model.Alert, error) {
	iter := r.db.Collection(collectionAlerts).Where("Status", "==", status).Documents(ctx)

	var alerts []model.Alert
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, goerr.Wrap(err, "failed to get next alert")
		}

		var alert model.Alert
		if err := doc.DataTo(&alert); err != nil {
			return nil, goerr.Wrap(err, "failed to convert data to alert")
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (r *Firestore) BatchGetAlerts(ctx context.Context, alertIDs []model.AlertID) ([]model.Alert, error) {
	var alerts []model.Alert

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

			var alert model.Alert
			if err := doc.DataTo(&alert); err != nil {
				return nil, goerr.Wrap(err, "failed to convert data to alert")
			}

			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

func (r *Firestore) GetPolicyDiff(ctx context.Context, id model.PolicyDiffID) (*model.PolicyDiff, error) {
	doc, err := r.db.Collection(collectionPolicyDiffs).Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get policy diff", goerr.V("id", id))
	}

	var policyDiff model.PolicyDiff
	if err := doc.DataTo(&policyDiff); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to policy diff")
	}

	return &policyDiff, nil
}

func (r *Firestore) PutPolicyDiff(ctx context.Context, diff *model.PolicyDiff) error {
	doc := r.db.Collection(collectionPolicyDiffs).Doc(diff.ID.String())
	_, err := doc.Set(ctx, diff)
	if err != nil {
		return goerr.Wrap(err, "failed to put policy diff", goerr.V("id", diff.ID))
	}
	return nil
}
