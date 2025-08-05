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

// CreateTag creates a new tag (deprecated - use CreateTagWithCustomColor)
func (s *Service) CreateTag(ctx context.Context, name string) error {
	if name == "" {
		return goerr.New("tag name cannot be empty")
	}

	// Check if tag already exists
	exists, err := s.repo.IsTagNameExists(ctx, name)
	if err != nil {
		return goerr.Wrap(err, "failed to check tag existence")
	}
	if exists {
		// Tag already exists, silently ignore for backward compatibility
		return nil
	}

	_, err = s.CreateTagWithCustomColor(ctx, name, "", tag.GenerateColor(name), "")
	if err != nil {
		return goerr.Wrap(err, "failed to create tag")
	}

	return nil
}

// DeleteTag deletes a tag and removes it from all alerts and tickets (deprecated - use DeleteTagByName)
func (s *Service) DeleteTag(ctx context.Context, name string) error {
	// Find tag by name
	existingTag, err := s.repo.GetTagByName(ctx, name)
	if err != nil {
		return goerr.Wrap(err, "failed to find tag by name")
	}
	if existingTag == nil {
		return nil // Tag doesn't exist, nothing to delete
	}

	// Use ID-based deletion
	return s.DeleteTagByID(ctx, existingTag.ID)
}

// EnsureTagsExist checks if tags exist and creates them if they don't (deprecated - use ConvertNamesToIDs)
func (s *Service) EnsureTagsExist(ctx context.Context, tags []string) error {
	_, err := s.ConvertNamesToIDs(ctx, tags)
	return err
}

// New ID-based tag management methods

func (s *Service) GetTagByID(ctx context.Context, tagID types.TagID) (*tag.Tag, error) {
	tagData, err := s.repo.GetTagByID(ctx, tagID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get tag by ID")
	}
	return tagData, nil
}

func (s *Service) GetTagsByIDs(ctx context.Context, tagIDs []types.TagID) ([]*tag.Tag, error) {
	tags, err := s.repo.GetTagsByIDs(ctx, tagIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get tags by IDs")
	}
	return tags, nil
}

func (s *Service) CreateTagWithCustomColor(ctx context.Context, name, description, color string, createdBy string) (*tag.Tag, error) {
	if name == "" {
		return nil, goerr.New("tag name cannot be empty")
	}

	// Check if tag name already exists
	exists, err := s.repo.IsTagNameExists(ctx, name)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to check tag name existence")
	}
	if exists {
		return nil, goerr.New("tag name already exists", goerr.V("name", name))
	}

	// Generate ID with collision retry
	var tagID types.TagID
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		tagID = types.NewTagID()
		existing, err := s.repo.GetTagByID(ctx, tagID)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to check tag ID collision")
		}
		if existing == nil {
			break // No collision
		}
		if i == maxRetries-1 {
			return nil, goerr.New("failed to generate unique tag ID after retries")
		}
	}

	// Use provided color or generate one
	if color == "" {
		color = tag.GenerateColor(name)
	}

	newTag := &tag.Tag{
		ID:          tagID,
		Name:        name,
		Description: description,
		Color:       color,
		CreatedBy:   createdBy,
	}

	if err := s.repo.CreateTagWithID(ctx, newTag); err != nil {
		return nil, goerr.Wrap(err, "failed to create tag")
	}

	return newTag, nil
}

func (s *Service) UpdateTagMetadata(ctx context.Context, tagID types.TagID, name, description, color string) (*tag.Tag, error) {
	// Get existing tag
	existingTag, err := s.repo.GetTagByID(ctx, tagID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get existing tag")
	}
	if existingTag == nil {
		return nil, goerr.New("tag not found", goerr.V("tagID", tagID))
	}

	// Check if new name conflicts with other tags
	if name != existingTag.Name {
		exists, err := s.repo.IsTagNameExists(ctx, name)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to check tag name existence")
		}
		if exists {
			return nil, goerr.New("tag name already exists", goerr.V("name", name))
		}
	}

	// Update the tag
	updatedTag := &tag.Tag{
		ID:          existingTag.ID,
		Name:        name,
		Description: description,
		Color:       color,
		CreatedAt:   existingTag.CreatedAt,
		CreatedBy:   existingTag.CreatedBy,
	}

	if err := s.repo.UpdateTag(ctx, updatedTag); err != nil {
		return nil, goerr.Wrap(err, "failed to update tag")
	}

	return updatedTag, nil
}

func (s *Service) DeleteTagByID(ctx context.Context, tagID types.TagID) error {
	// First, remove the tag from all alerts
	if err := s.repo.RemoveTagIDFromAllAlerts(ctx, tagID); err != nil {
		return goerr.Wrap(err, "failed to remove tag from alerts")
	}

	// Then, remove the tag from all tickets
	if err := s.repo.RemoveTagIDFromAllTickets(ctx, tagID); err != nil {
		return goerr.Wrap(err, "failed to remove tag from tickets")
	}

	// Finally, delete the tag metadata
	if err := s.repo.DeleteTagByID(ctx, tagID); err != nil {
		return goerr.Wrap(err, "failed to delete tag")
	}

	return nil
}

func (s *Service) UpdateAlertTagsByID(ctx context.Context, alertID types.AlertID, tagIDs []types.TagID) (*alert.Alert, error) {
	// Ensure all tags exist
	for _, tagID := range tagIDs {
		existing, err := s.repo.GetTagByID(ctx, tagID)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to check tag existence", goerr.V("tagID", tagID))
		}
		if existing == nil {
			return nil, goerr.New("tag not found", goerr.V("tagID", tagID))
		}
	}

	// Get the alert
	a, err := s.repo.GetAlert(ctx, alertID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alert")
	}
	if a == nil {
		return nil, goerr.New("alert not found", goerr.V("alert_id", alertID))
	}

	// Merge existing tags with new tags
	a.Tags = mergeTagIDs(a.Tags, tagIDs)

	// Save the alert
	if err := s.repo.PutAlert(ctx, *a); err != nil {
		return nil, goerr.Wrap(err, "failed to update alert")
	}

	return a, nil
}

func (s *Service) UpdateTicketTagsByID(ctx context.Context, ticketID types.TicketID, tagIDs []types.TagID) (*ticket.Ticket, error) {
	// Ensure all tags exist
	for _, tagID := range tagIDs {
		existing, err := s.repo.GetTagByID(ctx, tagID)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to check tag existence", goerr.V("tagID", tagID))
		}
		if existing == nil {
			return nil, goerr.New("tag not found", goerr.V("tagID", tagID))
		}
	}

	// Get the ticket
	t, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get ticket")
	}
	if t == nil {
		return nil, goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
	}

	// Merge existing tags with new tags
	t.Tags = mergeTagIDs(t.Tags, tagIDs)

	// Save the ticket
	if err := s.repo.PutTicket(ctx, *t); err != nil {
		return nil, goerr.Wrap(err, "failed to update ticket")
	}

	return t, nil
}

// Helper methods for tag name ↔ ID conversion

// ConvertNamesToIDs converts tag names to tag IDs, creating tags if they don't exist
func (s *Service) ConvertNamesToIDs(ctx context.Context, tagNames []string) ([]types.TagID, error) {
	if len(tagNames) == 0 {
		return []types.TagID{}, nil
	}

	tagIDs := make([]types.TagID, 0, len(tagNames))

	for _, name := range tagNames {
		if name == "" {
			continue
		}

		// Try to get existing tag by name
		existingTag, err := s.repo.GetTagByName(ctx, name)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to check existing tag", goerr.V("name", name))
		}

		if existingTag != nil {
			// Tag exists, use its ID
			tagIDs = append(tagIDs, existingTag.ID)
		} else {
			// Tag doesn't exist, create it
			newTag, err := s.CreateTagWithCustomColor(ctx, name, "", "", "")
			if err != nil {
				return nil, goerr.Wrap(err, "failed to create new tag", goerr.V("name", name))
			}
			tagIDs = append(tagIDs, newTag.ID)
		}
	}

	return tagIDs, nil
}

// UpdateAlertTagsByName provides compatibility for name-based tag updates
func (s *Service) UpdateAlertTagsByName(ctx context.Context, alertID types.AlertID, tagNames []string) (*alert.Alert, error) {
	// Convert names to IDs
	tagIDs, err := s.ConvertNamesToIDs(ctx, tagNames)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to convert tag names to IDs")
	}

	// Use ID-based method
	return s.UpdateAlertTagsByID(ctx, alertID, tagIDs)
}

// UpdateTicketTagsByName provides compatibility for name-based tag updates
func (s *Service) UpdateTicketTagsByName(ctx context.Context, ticketID types.TicketID, tagNames []string) (*ticket.Ticket, error) {
	// Convert names to IDs
	tagIDs, err := s.ConvertNamesToIDs(ctx, tagNames)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to convert tag names to IDs")
	}

	// Use ID-based method
	return s.UpdateTicketTagsByID(ctx, ticketID, tagIDs)
}

// ListAllTags returns all tags using the new ID-based system
func (s *Service) ListAllTags(ctx context.Context) ([]*tag.Tag, error) {
	tags, err := s.repo.ListAllTags(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list all tags")
	}
	return tags, nil
}

// mergeTagIDs merges existing tags with new tags, avoiding duplicates
func mergeTagIDs(existingTags, newTags []types.TagID) []types.TagID {
	// Create a map to avoid duplicates
	tagMap := make(map[types.TagID]bool)

	// Add existing tags
	for _, tagID := range existingTags {
		tagMap[tagID] = true
	}

	// Add new tags
	for _, tagID := range newTags {
		tagMap[tagID] = true
	}

	// Convert back to slice
	mergedTags := make([]types.TagID, 0, len(tagMap))
	for tagID := range tagMap {
		mergedTags = append(mergedTags, tagID)
	}

	return mergedTags
}
