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
	collectionAlerts = "alerts"
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
			return nil, goerr.New("alert not found", goerr.V("slack_channel", thread.ChannelID), goerr.V("slack_message_id", thread.ThreadID))
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
