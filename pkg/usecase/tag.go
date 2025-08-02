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
func (u *TagUseCase) ListTags(ctx context.Context) ([]*tagmodel.Metadata, error) {
	tags, err := u.tagService.ListTags(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list tags")
	}
	return tags, nil
}

// CreateTag creates a new tag
func (u *TagUseCase) CreateTag(ctx context.Context, name string) (*tagmodel.Metadata, error) {
	if name == "" {
		return nil, goerr.New("tag name cannot be empty")
	}

	tagName := tagmodel.Tag(name)
	if err := u.tagService.CreateTag(ctx, tagName); err != nil {
		return nil, goerr.Wrap(err, "failed to create tag")
	}

	// Return the created tag metadata
	// Since CreateTag is idempotent, we can safely get the tag
	tags, err := u.tagService.ListTags(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get created tag")
	}

	for _, tag := range tags {
		if tag.Name == tagName {
			return tag, nil
		}
	}

	return nil, goerr.New("created tag not found")
}

// DeleteTag deletes a tag
func (u *TagUseCase) DeleteTag(ctx context.Context, name string) error {
	if name == "" {
		return goerr.New("tag name cannot be empty")
	}

	if err := u.tagService.DeleteTag(ctx, tagmodel.Tag(name)); err != nil {
		return goerr.Wrap(err, "failed to delete tag")
	}

	return nil
}

// UpdateAlertTags updates tags for an alert
func (u *TagUseCase) UpdateAlertTags(ctx context.Context, alertID types.AlertID, tags []string) (*alert.Alert, error) {
	a, err := u.tagService.UpdateAlertTags(ctx, alertID, tags)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to update alert tags")
	}
	return a, nil
}

// UpdateTicketTags updates tags for a ticket
func (u *TagUseCase) UpdateTicketTags(ctx context.Context, ticketID types.TicketID, tags []string) (*ticket.Ticket, error) {
	t, err := u.tagService.UpdateTicketTags(ctx, ticketID, tags)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to update ticket tags")
	}
	return t, nil
}