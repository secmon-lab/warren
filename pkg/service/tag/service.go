package tag

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// Service provides tag management functionality
type Service struct {
	repo interfaces.Repository
}

// New creates a new tag service
func New(repo interfaces.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// ListTags returns all tags in the system
func (s *Service) ListTags(ctx context.Context) ([]*tag.Metadata, error) {
	tags, err := s.repo.ListTags(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list tags")
	}
	return tags, nil
}

// GetTag returns a tag by name
func (s *Service) GetTag(ctx context.Context, name tag.Tag) (*tag.Metadata, error) {
	tag, err := s.repo.GetTag(ctx, name)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get tag")
	}
	return tag, nil
}

// CreateTag creates a new tag
func (s *Service) CreateTag(ctx context.Context, name tag.Tag) error {
	if name == "" {
		return goerr.New("tag name cannot be empty")
	}

	tagMeta := &tag.Metadata{
		Name:  name,
		Color: tag.GenerateColor(string(name)),
	}

	if err := s.repo.CreateTag(ctx, tagMeta); err != nil {
		return goerr.Wrap(err, "failed to create tag")
	}

	return nil
}

// DeleteTag deletes a tag and removes it from all alerts and tickets
func (s *Service) DeleteTag(ctx context.Context, name tag.Tag) error {
	// First, remove the tag from all alerts
	if err := s.repo.RemoveTagFromAllAlerts(ctx, name); err != nil {
		return goerr.Wrap(err, "failed to remove tag from alerts")
	}

	// Then, remove the tag from all tickets
	if err := s.repo.RemoveTagFromAllTickets(ctx, name); err != nil {
		return goerr.Wrap(err, "failed to remove tag from tickets")
	}

	// Finally, delete the tag metadata
	if err := s.repo.DeleteTag(ctx, name); err != nil {
		return goerr.Wrap(err, "failed to delete tag")
	}

	return nil
}

// EnsureTagsExist checks if tags exist and creates them if they don't
func (s *Service) EnsureTagsExist(ctx context.Context, tags []string) error {
	for _, tagName := range tags {
		if tagName == "" {
			continue
		}

		tag := tag.Tag(tagName)

		// Check if tag exists
		existingTag, err := s.repo.GetTag(ctx, tag)
		if err != nil {
			return goerr.Wrap(err, "failed to check tag existence", goerr.V("tag", tagName))
		}

		// Create tag if it doesn't exist
		if existingTag == nil {
			if err := s.CreateTag(ctx, tag); err != nil {
				return goerr.Wrap(err, "failed to create tag", goerr.V("tag", tagName))
			}
		}
	}

	return nil
}

// UpdateAlertTags updates tags for an alert
func (s *Service) UpdateAlertTags(ctx context.Context, alertID types.AlertID, tags []string) (*alert.Alert, error) {
	// Ensure all tags exist
	if err := s.EnsureTagsExist(ctx, tags); err != nil {
		return nil, goerr.Wrap(err, "failed to ensure tags exist")
	}

	// Get the alert
	a, err := s.repo.GetAlert(ctx, alertID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alert")
	}
	if a == nil {
		return nil, goerr.New("alert not found", goerr.V("alert_id", alertID))
	}

	// Update tags
	a.Tags = tag.NewSet(tags)

	// Save the alert
	if err := s.repo.PutAlert(ctx, *a); err != nil {
		return nil, goerr.Wrap(err, "failed to update alert")
	}

	return a, nil
}

// UpdateTicketTags updates tags for a ticket
func (s *Service) UpdateTicketTags(ctx context.Context, ticketID types.TicketID, tags []string) (*ticket.Ticket, error) {
	// Ensure all tags exist
	if err := s.EnsureTagsExist(ctx, tags); err != nil {
		return nil, goerr.Wrap(err, "failed to ensure tags exist")
	}

	// Get the ticket
	t, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get ticket")
	}
	if t == nil {
		return nil, goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
	}

	// Update tags
	t.Tags = tag.NewSet(tags)

	// Save the ticket
	if err := s.repo.PutTicket(ctx, *t); err != nil {
		return nil, goerr.Wrap(err, "failed to update ticket")
	}

	return t, nil
}
