package repository

import (
	"context"
	"sync"
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

func (r *Firestore) FetchLatestAlerts(ctx context.Context, oldest time.Time, limit int) ([]model.Alert, error) {
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
	iter := r.db.Collection(collectionAlerts).Where("ID", "in", alertIDs).Documents(ctx)

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

func (r *Firestore) PutAlertGroups(ctx context.Context, groups []model.AlertGroup) error {
	batch := r.db.BulkWriter(ctx)
	wg := sync.WaitGroup{}
	errCh := make(chan error, len(groups))

	for _, group := range groups {
		groupDoc := r.db.Collection(collectionAlertGroups).Doc(group.ID.String())
		job, err := batch.Create(groupDoc, group)
		if err != nil {
			return goerr.Wrap(err, "failed to create alert group", goerr.V("group", group))
		}

		wg.Add(1)
		go func(group model.AlertGroup) {
			defer wg.Done()
			if _, err := job.Results(); err != nil {
				errCh <- goerr.Wrap(err, "failed to save alert group", goerr.V("group", group))
			}
		}(group)
	}

	batch.End()
	wg.Wait()

	close(errCh)

	for err := range errCh {
		return err
	}

	return nil
}

func (r *Firestore) GetAlertGroup(ctx context.Context, groupID model.AlertGroupID) (*model.AlertGroup, error) {
	groupDoc := r.db.Collection(collectionAlertGroups).Doc(groupID.String())
	doc, err := groupDoc.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get alert group", goerr.V("group_id", groupID))
	}

	var group model.AlertGroup
	if err := doc.DataTo(&group); err != nil {
		return nil, goerr.Wrap(err, "failed to convert data to alert group")
	}

	return &group, nil
}
