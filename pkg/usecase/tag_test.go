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

// Note: TestAlertHandling_WithTags would require mocking the policy client
// to return metadata with tags. This would be done in integration tests.

// Note: TestTicketCreation_InheritsTags would require mocking the LLM client
// to generate ticket metadata. This would be done in integration tests.
