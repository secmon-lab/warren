package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	tagmodel "github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/tag"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestTagUseCase_Operations(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tagService := tag.New(repo)
	tagUC := usecase.NewTagUseCase(tagService)

	// Create tags
	tag1, err := tagUC.CreateTag(ctx, "security")
	gt.NoError(t, err)
	gt.NotNil(t, tag1)
	gt.V(t, tag1.Name).Equal("security")

	tag2, err := tagUC.CreateTag(ctx, "incident")
	gt.NoError(t, err)
	gt.NotNil(t, tag2)

	// List tags
	tags, err := tagUC.ListTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(2)

	// Update alert tags
	a := alert.New(ctx, "test", map[string]string{"test": "data"}, alert.Metadata{
		Title:       "Test Alert",
		Description: "Test Description",
	})
	gt.NoError(t, repo.PutAlert(ctx, a))

	updatedAlert, err := tagUC.UpdateAlertTags(ctx, a.ID, []string{"security", "critical"})
	gt.NoError(t, err)
	gt.NotNil(t, updatedAlert)
	gt.N(t, len(updatedAlert.Tags)).Equal(2)

	// Get actual tag names to verify
	tagNames, err := updatedAlert.GetTagNames(ctx, func(ctx context.Context, tagIDs []types.TagID) ([]*tagmodel.Tag, error) {
		return tagService.GetTagsByIDs(ctx, tagIDs)
	})
	gt.NoError(t, err)
	gt.N(t, len(tagNames)).Equal(2)

	// Check that the expected tags are present
	tagMap := make(map[string]bool)
	for _, name := range tagNames {
		tagMap[name] = true
	}
	gt.True(t, tagMap["security"])
	gt.True(t, tagMap["critical"])

	// Verify new tag was created
	tags, err = tagUC.ListTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(3) // security, incident, critical

	// Delete tag
	gt.NoError(t, tagUC.DeleteTag(ctx, "incident"))

	tags, err = tagUC.ListTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(2) // security, critical
}

func TestTagUseCase_UpdateTag(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tagService := tag.New(repo)
	tagUC := usecase.NewTagUseCase(tagService)

	// Create a tag first
	originalTag, err := tagUC.CreateTag(ctx, "test-tag")
	gt.NoError(t, err)
	gt.NotNil(t, originalTag)

	// Test successful update
	updatedTag, err := tagUC.UpdateTag(ctx, originalTag.ID.String(), "updated-tag", "bg-blue-100 text-blue-800", "updated description")
	gt.NoError(t, err)
	gt.NotNil(t, updatedTag)
	gt.V(t, updatedTag.Name).Equal("updated-tag")
	gt.V(t, updatedTag.Description).Equal("updated description")
	gt.V(t, updatedTag.Color).Equal("bg-blue-100 text-blue-800")
	gt.V(t, updatedTag.ID).Equal(originalTag.ID)

	// Test validation errors
	tests := []struct {
		name        string
		tagID       string
		tagName     string
		color       string
		description string
		expectError bool
	}{
		{
			name:        "empty tag name",
			tagID:       originalTag.ID.String(),
			tagName:     "",
			color:       "bg-red-100 text-red-800",
			description: "test",
			expectError: true,
		},
		{
			name:        "invalid color",
			tagID:       originalTag.ID.String(),
			tagName:     "valid-name",
			color:       "invalid-color",
			description: "test",
			expectError: true,
		},
		{
			name:        "non-existent tag ID",
			tagID:       types.NewTagID().String(),
			tagName:     "valid-name",
			color:       "bg-red-100 text-red-800",
			description: "test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tagUC.UpdateTag(ctx, tt.tagID, tt.tagName, tt.color, tt.description)
			if tt.expectError {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
			}
		})
	}
}

func TestTagUseCase_GetAvailableColors(t *testing.T) {
	repo := repository.NewMemory()
	tagService := tag.New(repo)
	tagUC := usecase.NewTagUseCase(tagService)

	colors, err := tagUC.GetAvailableColors(context.Background())
	gt.NoError(t, err)
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
}

// Note: TestAlertHandling_WithTags would require mocking the policy client
// to return metadata with tags. This would be done in integration tests.

// Note: TestTicketCreation_InheritsTags would require mocking the LLM client
// to generate ticket metadata. This would be done in integration tests.
