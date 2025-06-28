package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// createTicketActivity creates a ticket creation activity
func createTicketActivity(ctx context.Context, repo interfaces.Repository, ticketID types.TicketID, title string) error {
	userID := user.FromContext(ctx)
	activityID := types.NewActivityID()
	act := &activity.Activity{
		ID:          activityID,
		Type:        types.ActivityTypeTicketCreated,
		Title:       "Ticket Created",
		Description: fmt.Sprintf("Created ticket: %s", title),
		UserID:      userID,
		TicketID:    ticketID,
		CreatedAt:   time.Now(),
	}

	if err := repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}

// createCommentActivity creates a comment addition activity
func createCommentActivity(ctx context.Context, repo interfaces.Repository, ticketID types.TicketID, commentID types.CommentID, title string) error {
	userID := user.FromContext(ctx)
	activityID := types.NewActivityID()
	act := &activity.Activity{
		ID:          activityID,
		Type:        types.ActivityTypeCommentAdded,
		Title:       "Comment Added",
		Description: fmt.Sprintf("Added comment to ticket: %s", title),
		UserID:      userID,
		TicketID:    ticketID,
		CommentID:   commentID,
		CreatedAt:   time.Now(),
	}

	if err := repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}

// createStatusChangeActivity creates a status change activity
func createStatusChangeActivity(ctx context.Context, repo interfaces.Repository, ticketID types.TicketID, title, oldStatus, newStatus string) error {
	userID := user.FromContext(ctx)
	activityID := types.NewActivityID()
	act := &activity.Activity{
		ID:          activityID,
		Type:        types.ActivityTypeTicketStatusChanged,
		Title:       "Status Changed",
		Description: fmt.Sprintf("Changed status of %s from %s to %s", title, oldStatus, newStatus),
		UserID:      userID,
		TicketID:    ticketID,
		CreatedAt:   time.Now(),
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

// createAlertBoundActivity creates an alert binding activity
func createAlertBoundActivity(ctx context.Context, repo interfaces.Repository, alertID types.AlertID, ticketID types.TicketID, alertTitle, ticketTitle string) error {
	userID := user.FromContext(ctx)
	activityID := types.NewActivityID()
	act := &activity.Activity{
		ID:          activityID,
		Type:        types.ActivityTypeAlertBound,
		Title:       "Alert Bound",
		Description: fmt.Sprintf("Bound alert %s to ticket: %s", alertTitle, ticketTitle),
		UserID:      userID,
		AlertID:     alertID,
		TicketID:    ticketID,
		CreatedAt:   time.Now(),
	}

	if err := repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to put activity")
	}
	return nil
}

// createBulkAlertBoundActivity creates a bulk alert binding activity
func createBulkAlertBoundActivity(ctx context.Context, repo interfaces.Repository, alertIDs []types.AlertID, ticketID types.TicketID, ticketTitle string, alertTitles []string) error {
	userID := user.FromContext(ctx)
	activityID := types.NewActivityID()
	act := &activity.Activity{
		ID:          activityID,
		Type:        types.ActivityTypeAlertsBulkBound,
		Title:       "Bulk Alert Bound",
		Description: fmt.Sprintf("Bound %d alerts to ticket: %s", len(alertIDs), ticketTitle),
		UserID:      userID,
		TicketID:    ticketID,
		CreatedAt:   time.Now(),
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
