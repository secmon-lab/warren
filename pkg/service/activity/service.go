package activity

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type Service struct {
	repo interfaces.Repository
}

func New(repo interfaces.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateTicketActivity(ctx context.Context, ticketID types.TicketID, title, userID string) error {
	act := activity.New(
		types.ActivityTypeTicketCreated,
		"Ticket was created",
		title,
	).WithTicketID(ticketID).WithUserID(userID)

	if err := s.repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to create ticket activity")
	}

	return nil
}

func (s *Service) CreateTicketStatusChangedActivity(ctx context.Context, ticketID types.TicketID, title, fromStatus, toStatus, userID string) error {
	act := activity.New(
		types.ActivityTypeTicketStatusChanged,
		"Ticket status was changed",
		title,
	).WithTicketID(ticketID).WithUserID(userID).WithMetadata("from_status", fromStatus).WithMetadata("to_status", toStatus)

	if err := s.repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to create ticket status changed activity")
	}

	return nil
}

func (s *Service) CreateCommentActivity(ctx context.Context, ticketID types.TicketID, commentID types.CommentID, ticketTitle, userID string) error {
	act := activity.New(
		types.ActivityTypeCommentAdded,
		"Comment was added",
		ticketTitle,
	).WithTicketID(ticketID).WithCommentID(commentID).WithUserID(userID)

	if err := s.repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to create comment activity")
	}

	return nil
}

func (s *Service) CreateAlertBoundActivity(ctx context.Context, alertID types.AlertID, ticketID types.TicketID, alertTitle, ticketTitle, userID string) error {
	act := activity.New(
		types.ActivityTypeAlertBound,
		"Alert was bound to ticket",
		alertTitle+" → "+ticketTitle,
	).WithAlertID(alertID).WithTicketID(ticketID).WithUserID(userID)

	if err := s.repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to create alert bound activity")
	}

	return nil
}

func (s *Service) CreateAlertsBulkBoundActivity(ctx context.Context, alertIDs []types.AlertID, ticketID types.TicketID, ticketTitle, userID string, alertTitles []string) error {
	metadata := map[string]any{
		"alert_count": len(alertIDs),
		"alert_ids":   alertIDs,
	}
	if len(alertTitles) > 0 {
		metadata["alert_titles"] = alertTitles
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal metadata")
	}

	act := activity.New(
		types.ActivityTypeAlertsBulkBound,
		"Multiple alerts were bulk bound to ticket",
		ticketTitle,
	).WithTicketID(ticketID).WithUserID(userID)

	// Set metadata as JSON string
	act.Metadata = map[string]any{
		"bulk_operation": string(metadataJSON),
	}

	if err := s.repo.PutActivity(ctx, act); err != nil {
		return goerr.Wrap(err, "failed to create alerts bulk bound activity")
	}

	return nil
}
