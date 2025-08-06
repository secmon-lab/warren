package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	tagmodel "github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/tag"
)

// TagUseCase provides tag management use cases
type TagUseCase struct {
	tagService *tag.Service
}

// NewTagUseCase creates a new tag use case
func NewTagUseCase(tagService *tag.Service) *TagUseCase {
	return &TagUseCase{
		tagService: tagService,
	}
}

// ListTags returns all tags in the system
func (u *TagUseCase) ListTags(ctx context.Context) ([]*tagmodel.Tag, error) {
	tags, err := u.tagService.ListAllTags(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list tags")
	}
	return tags, nil
}

// CreateTag creates a new tag
func (u *TagUseCase) CreateTag(ctx context.Context, name string) (*tagmodel.Tag, error) {
	if name == "" {
		return nil, goerr.New("tag name cannot be empty")
	}

	tag, err := u.tagService.CreateTagWithCustomColor(ctx, name, "", "", "")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create tag")
	}

	return tag, nil
}

// DeleteTag deletes a tag
func (u *TagUseCase) DeleteTag(ctx context.Context, name string) error {
	if name == "" {
		return goerr.New("tag name cannot be empty")
	}

	if err := u.tagService.DeleteTag(ctx, name); err != nil {
		return goerr.Wrap(err, "failed to delete tag")
	}

	return nil
}

// UpdateAlertTags updates tags for an alert
func (u *TagUseCase) UpdateAlertTags(ctx context.Context, alertID types.AlertID, tags []string) (*alert.Alert, error) {
	// Convert tag names to IDs
	tagIDs, err := u.tagService.ConvertNamesToIDs(ctx, tags)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to convert tag names to IDs")
	}

	// Use ID-based method
	a, err := u.tagService.UpdateAlertTagsByID(ctx, alertID, tagIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to update alert tags")
	}
	return a, nil
}

// UpdateTicketTags updates tags for a ticket
func (u *TagUseCase) UpdateTicketTags(ctx context.Context, ticketID types.TicketID, tags []string) (*ticket.Ticket, error) {
	// Convert tag names to IDs
	tagIDs, err := u.tagService.ConvertNamesToIDs(ctx, tags)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to convert tag names to IDs")
	}

	// Use ID-based method
	t, err := u.tagService.UpdateTicketTagsByID(ctx, ticketID, tagIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to update ticket tags")
	}
	return t, nil
}

// UpdateTag updates tag metadata (name, color, description)
func (u *TagUseCase) UpdateTag(ctx context.Context, tagID string, name, color, description string) (*tagmodel.Tag, error) {
	id := types.TagID(tagID)

	// Validate input
	if err := u.validateUpdateTagInput(name, color); err != nil {
		return nil, err
	}

	// Call service layer
	tag, err := u.tagService.UpdateTagMetadata(ctx, id, name, description, color)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to update tag")
	}

	return tag, nil
}

// GetAvailableColors returns available color options for tags (Tailwind classes)
func (u *TagUseCase) GetAvailableColors() ([]string, error) {
	return u.tagService.GetAvailableColors(), nil
}

// GetAvailableColorNames returns user-friendly color names for tags
func (u *TagUseCase) GetAvailableColorNames() ([]string, error) {
	return u.tagService.GetAvailableColorNames(), nil
}

// validateUpdateTagInput validates input parameters for tag updates
func (u *TagUseCase) validateUpdateTagInput(name, color string) error {
	if name == "" {
		return goerr.New("tag name cannot be empty")
	}

	// Validate color (support both color names and Tailwind classes)
	availableColors := u.tagService.GetAvailableColors()
	availableNames := u.tagService.GetAvailableColorNames()
	
	colorValid := false
	// Check if it's a valid Tailwind class
	for _, c := range availableColors {
		if c == color {
			colorValid = true
			break
		}
	}
	// Check if it's a valid color name
	if !colorValid {
		for _, n := range availableNames {
			if n == color {
				colorValid = true
				break
			}
		}
	}
	
	if !colorValid {
		return goerr.New("invalid color selection", goerr.V("color", color))
	}

	return nil
}
