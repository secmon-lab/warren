package activityutil

import (
	"context"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// CreateTicketActivity creates a ticket creation activity
func CreateTicketActivity(ctx context.Context, repo interfaces.Repository, ticketID types.TicketID, title string) error {
	return createTicketActivityWithType(ctx, repo, ticketID, title, types.ActivityTypeTicketCreated)
}

// CreateTicketUpdateActivity creates a ticket update activity
func CreateTicketUpdateActivity(ctx context.Context, repo interfaces.Repository, ticketID types.TicketID, title string) error {
	return createTicketActivityWithType(ctx, repo, ticketID, title, types.ActivityTypeTicketUpdated)
}

// createTicketActivityWithType creates a ticket activity with specified type
func createTicketActivityWithType(ctx context.Context, repo interfaces.Repository, ticketID types.TicketID, title string, activityType types.ActivityType) error {
	userID := user.FromContext(ctx)
	activityID := types.NewActivityID()
	act := &activity.Activity{
		ID:        activityID,
		Type:      activityType,
		UserID:    userID,
		TicketID:  ticketID,
		CreatedAt: time.Now(),
	}

	if err := repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}

// CreateCommentActivity creates a comment addition activity
func CreateCommentActivity(ctx context.Context, repo interfaces.Repository, ticketID types.TicketID, commentID types.CommentID, title string) error {
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

	if err := repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}

// CreateStatusChangeActivity creates a status change activity
func CreateStatusChangeActivity(ctx context.Context, repo interfaces.Repository, ticketID types.TicketID, title, oldStatus, newStatus string) error {
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

	if err := repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}

// CreateAlertBoundActivity creates an alert binding activity
func CreateAlertBoundActivity(ctx context.Context, repo interfaces.Repository, alertID types.AlertID, ticketID types.TicketID, alertTitle, ticketTitle string) error {
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

	if err := repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}

// CreateBulkAlertBoundActivity creates a bulk alert binding activity
func CreateBulkAlertBoundActivity(ctx context.Context, repo interfaces.Repository, alertIDs []types.AlertID, ticketID types.TicketID, ticketTitle string, alertTitles []string) error {
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

	if err := repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}
