package tag_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	tagmodel "github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/tag"
)

func TestTagService_CreateAndListTags(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Create tags
	gt.NoError(t, service.CreateTag(ctx, "security"))
	gt.NoError(t, service.CreateTag(ctx, "incident"))
	gt.NoError(t, service.CreateTag(ctx, "phishing"))

	// List tags
	tags, err := service.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(3)

	// Verify tag names
	tagNames := make(map[string]bool)
	for _, tag := range tags {
		tagNames[tag.Name] = true
	}
	gt.True(t, tagNames["security"])
	gt.True(t, tagNames["incident"])
	gt.True(t, tagNames["phishing"])
}

func TestTagService_CreateDuplicateTag(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Create tag
	gt.NoError(t, service.CreateTag(ctx, "security"))

	// Try to create duplicate tag (should not error)
	gt.NoError(t, service.CreateTag(ctx, "security"))

	// List tags - should still have only one
	tags, err := service.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(1)
}

func TestTagService_DeleteTag(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Create tags
	gt.NoError(t, service.CreateTag(ctx, "security"))
	gt.NoError(t, service.CreateTag(ctx, "incident"))

	// Delete one tag
	gt.NoError(t, service.DeleteTag(ctx, "security"))

	// List tags - should have only one
	tags, err := service.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(1)
	gt.V(t, tags[0].Name).Equal("incident")
}

func TestTagService_EnsureTagsExist(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Ensure tags exist (should create them)
	gt.NoError(t, service.EnsureTagsExist(ctx, []string{"tag1", "tag2", "tag3"}))

	// List tags
	tags, err := service.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(3)

	// Ensure tags exist again (should not create duplicates)
	gt.NoError(t, service.EnsureTagsExist(ctx, []string{"tag1", "tag2", "tag4"}))

	// List tags - should have 4 now
	tags, err = service.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(4)
}

func TestTagService_UpdateAlertTags(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Create an alert
	a := alert.New(ctx, "test", map[string]string{"test": "data"}, alert.Metadata{
		Title:       "Test Alert",
		Description: "Test Description",
	})
	gt.NoError(t, repo.PutAlert(ctx, a))

	// Update alert tags
	updatedAlert, err := service.UpdateAlertTags(ctx, a.ID, []string{"security", "incident"})
	gt.NoError(t, err)
	gt.NotNil(t, updatedAlert)
	gt.N(t, len(updatedAlert.Tags)).Equal(2)

	// Get actual tag names to verify
	tagNames, err := updatedAlert.GetTagNames(ctx, func(ctx context.Context, tagIDs []types.TagID) ([]*tagmodel.Tag, error) {
		return service.GetTagsByIDs(ctx, tagIDs)
	})
	gt.NoError(t, err)
	gt.N(t, len(tagNames)).Equal(2)

	// Check that the expected tags are present
	tagMap := make(map[string]bool)
	for _, name := range tagNames {
		tagMap[name] = true
	}
	gt.True(t, tagMap["security"])
	gt.True(t, tagMap["incident"])

	// Verify tags were created
	tags, err := service.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(2)
}

func TestTagService_UpdateTicketTags(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Create a ticket
	tk := ticket.New(ctx, []types.AlertID{}, nil)
	tk.Metadata.Title = "Test Ticket"
	gt.NoError(t, repo.PutTicket(ctx, tk))

	// Update ticket tags
	updatedTicket, err := service.UpdateTicketTags(ctx, tk.ID, []string{"resolved", "false-positive"})
	gt.NoError(t, err)
	gt.NotNil(t, updatedTicket)
	gt.N(t, len(updatedTicket.Tags)).Equal(2)

	// Get actual tag names to verify
	tagNames, err := updatedTicket.GetTagNames(ctx, func(ctx context.Context, tagIDs []types.TagID) ([]*tagmodel.Tag, error) {
		return service.GetTagsByIDs(ctx, tagIDs)
	})
	gt.NoError(t, err)
	gt.N(t, len(tagNames)).Equal(2)

	// Check that the expected tags are present
	tagMap := make(map[string]bool)
	for _, name := range tagNames {
		tagMap[name] = true
	}
	gt.True(t, tagMap["resolved"])
	gt.True(t, tagMap["false-positive"])

	// Verify tags were created
	tags, err := service.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(2)
}

func TestTagService_EmptyTagHandling(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Try to create empty tag
	err := service.CreateTag(ctx, "")
	gt.Error(t, err)

	// Ensure tags with empty string in array
	gt.NoError(t, service.EnsureTagsExist(ctx, []string{"valid", "", "tag"}))

	// Should only create valid tags
	tags, err := service.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(2)
}
