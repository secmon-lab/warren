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

func TestTagService_UpdateAlertTagsByName(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Create an alert
	a := alert.New(ctx, "test", map[string]string{"test": "data"}, alert.Metadata{
		Title:       "Test Alert",
		Description: "Test Description",
	})
	gt.NoError(t, repo.PutAlert(ctx, a))

	// Update alert tags using name-based method
	updatedAlert, err := service.UpdateAlertTagsByName(ctx, a.ID, []string{"security", "incident"})
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

func TestTagService_UpdateTicketTagsByName(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Create a ticket
	tk := ticket.New(ctx, []types.AlertID{}, nil)
	tk.Metadata.Title = "Test Ticket"
	gt.NoError(t, repo.PutTicket(ctx, tk))

	// Update ticket tags using name-based method
	updatedTicket, err := service.UpdateTicketTagsByName(ctx, tk.ID, []string{"resolved", "false-positive"})
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

func TestTagService_GetAvailableColors(t *testing.T) {
	repo := repository.NewMemory()
	service := tag.New(repo)

	colors := service.GetAvailableColors()
	gt.True(t, len(colors) > 0)

	// Check that it contains expected color format
	hasRedColor := false
	for _, color := range colors {
		if color == "bg-red-100 text-red-800" {
			hasRedColor = true
			break
		}
	}
	gt.True(t, hasRedColor)

	// Verify returned slice is a copy (modifying it shouldn't affect original)
	originalLen := len(colors)
	colors = append(colors, "test-color")
	newColors := service.GetAvailableColors()
	gt.N(t, len(newColors)).Equal(originalLen)
}

func TestTagService_GetAvailableColorNames(t *testing.T) {
	repo := repository.NewMemory()
	service := tag.New(repo)

	colorNames := service.GetAvailableColorNames()
	gt.True(t, len(colorNames) > 0)

	// Check that it contains expected color names
	hasRedColor := false
	for _, colorName := range colorNames {
		if colorName == "red" {
			hasRedColor = true
			break
		}
	}
	gt.True(t, hasRedColor)

	// Verify returned slice is a copy
	originalLen := len(colorNames)
	colorNames = append(colorNames, "test-color")
	newColorNames := service.GetAvailableColorNames()
	gt.N(t, len(newColorNames)).Equal(originalLen)
}

func TestTagService_UpdateTagMetadata(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Create a tag first
	originalTag, err := service.CreateTagWithCustomColor(ctx, "test-tag", "original description", "bg-red-100 text-red-800", "test-user")
	gt.NoError(t, err)
	gt.NotNil(t, originalTag)

	// Test successful update
	updatedTag, err := service.UpdateTagMetadata(ctx, originalTag.ID, "updated-tag", "updated description", "bg-blue-100 text-blue-800")
	gt.NoError(t, err)
	gt.NotNil(t, updatedTag)
	gt.V(t, updatedTag.Name).Equal("updated-tag")
	gt.V(t, updatedTag.Description).Equal("updated description")
	gt.V(t, updatedTag.Color).Equal("bg-blue-100 text-blue-800")
	gt.V(t, updatedTag.ID).Equal(originalTag.ID)
	gt.V(t, updatedTag.CreatedBy).Equal(originalTag.CreatedBy)
	gt.True(t, updatedTag.UpdatedAt.After(updatedTag.CreatedAt))

	// Test updating with invalid color
	_, err = service.UpdateTagMetadata(ctx, originalTag.ID, "test-tag-2", "desc", "completely-invalid-color-name")
	gt.Error(t, err)

	// Create another tag to test name collision
	_, err = service.CreateTagWithCustomColor(ctx, "another-tag", "", "bg-green-100 text-green-800", "")
	gt.NoError(t, err)

	// Test name collision (try to update first tag to use name of second tag)
	_, err = service.UpdateTagMetadata(ctx, originalTag.ID, "another-tag", "desc", "bg-red-100 text-red-800")
	gt.Error(t, err)

	// Test updating non-existent tag
	nonExistentID := types.NewTagID()
	_, err = service.UpdateTagMetadata(ctx, nonExistentID, "test", "desc", "bg-red-100 text-red-800")
	gt.Error(t, err)

	// Test updating with color name (should work)
	updatedTag2, err := service.UpdateTagMetadata(ctx, originalTag.ID, "test-tag-color-name", "desc with color name", "blue")
	gt.NoError(t, err)
	gt.NotNil(t, updatedTag2)
	gt.V(t, updatedTag2.Name).Equal("test-tag-color-name")
	gt.V(t, updatedTag2.Color).Equal("bg-blue-100 text-blue-800") // Should be converted to Tailwind class
}
