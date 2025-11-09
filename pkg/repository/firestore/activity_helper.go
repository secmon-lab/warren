package firestore

import (
	"context"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// createTicketActivity creates a ticket creation activity
func createTicketActivity(ctx context.Context, r *Firestore, ticketID types.TicketID, title string) error {
	return createTicketActivityWithType(ctx, r, ticketID, title, types.ActivityTypeTicketCreated)
}

// createTicketUpdateActivity creates a ticket update activity
func createTicketUpdateActivity(ctx context.Context, r *Firestore, ticketID types.TicketID, title string) error {
	return createTicketActivityWithType(ctx, r, ticketID, title, types.ActivityTypeTicketUpdated)
}

// createTicketActivityWithType creates a ticket activity with specified type
func createTicketActivityWithType(ctx context.Context, r *Firestore, ticketID types.TicketID, title string, activityType types.ActivityType) error {
	userID := user.FromContext(ctx)
	activityID := types.NewActivityID()
	act := &activity.Activity{
		ID:        activityID,
		Type:      activityType,
		UserID:    userID,
		TicketID:  ticketID,
		CreatedAt: time.Now(),
	}

	if err := r.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}

// createCommentActivity creates a comment addition activity
func createCommentActivity(ctx context.Context, r *Firestore, ticketID types.TicketID, commentID types.CommentID, title string) error {
	userID := user.FromContext(ctx)
	activityID := types.NewActivityID()
	act := &activity.Activity{
		ID:        activityID,
		Type:      types.ActivityTypeCommentAdded,
		UserID:    userID,
		TicketID:  ticketID,
		CommentID: commentID,
		CreatedAt: time.Now(),
	}

	if err := r.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}

// createStatusChangeActivity creates a status change activity
func createStatusChangeActivity(ctx context.Context, r *Firestore, ticketID types.TicketID, title, oldStatus, newStatus string) error {
	userID := user.FromContext(ctx)
	activityID := types.NewActivityID()
	act := &activity.Activity{
		ID:        activityID,
		Type:      types.ActivityTypeTicketStatusChanged,
		UserID:    userID,
		TicketID:  ticketID,
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"old_status": oldStatus,
			"new_status": newStatus,
		},
	}

	if err := r.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}

// createAlertBoundActivity creates an alert binding activity
func createAlertBoundActivity(ctx context.Context, r *Firestore, alertID types.AlertID, ticketID types.TicketID, alertTitle, ticketTitle string) error {
	userID := user.FromContext(ctx)
	activityID := types.NewActivityID()
	act := &activity.Activity{
		ID:        activityID,
		Type:      types.ActivityTypeAlertBound,
		UserID:    userID,
		AlertID:   alertID,
		TicketID:  ticketID,
		CreatedAt: time.Now(),
	}

	if err := r.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}

// createBulkAlertBoundActivity creates a bulk alert binding activity
func createBulkAlertBoundActivity(ctx context.Context, r *Firestore, alertIDs []types.AlertID, ticketID types.TicketID, ticketTitle string, alertTitles []string) error {
	userID := user.FromContext(ctx)
	activityID := types.NewActivityID()
	act := &activity.Activity{
		ID:        activityID,
		Type:      types.ActivityTypeAlertsBulkBound,
		UserID:    userID,
		TicketID:  ticketID,
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"alert_count":  len(alertIDs),
			"alert_titles": strings.Join(alertTitles, ", "),
		},
	}

	if err := r.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}
