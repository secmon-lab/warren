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
	gt.N(t, len(updatedAlert.TagIDs)).Equal(2)

	// Get actual tag names to verify
	tagNames, err := updatedAlert.GetTagNames(ctx, func(ctx context.Context, tagIDs []string) ([]*tagmodel.Tag, error) {
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
	gt.N(t, len(updatedTicket.TagIDs)).Equal(2)

	// Get actual tag names to verify
	tagNames, err := updatedTicket.GetTagNames(ctx, func(ctx context.Context, tagIDs []string) ([]*tagmodel.Tag, error) {
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
	_ = append(colors, "test-color") // Intentionally modify the copy
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
	_ = append(colorNames, "test-color") // Intentionally modify the copy
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
	nonExistentID := tagmodel.NewID()
	_, err = service.UpdateTagMetadata(ctx, nonExistentID, "test", "desc", "bg-red-100 text-red-800")
	gt.Error(t, err)

	// Test updating with color name (should work)
	updatedTag2, err := service.UpdateTagMetadata(ctx, originalTag.ID, "test-tag-color-name", "desc with color name", "blue")
	gt.NoError(t, err)
	gt.NotNil(t, updatedTag2)
	gt.V(t, updatedTag2.Name).Equal("test-tag-color-name")
	gt.V(t, updatedTag2.Color).Equal("bg-blue-100 text-blue-800") // Should be converted to Tailwind class
}

func TestTagService_ConvertNamesToTags_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Test concurrent access to ConvertNamesToTags to ensure no duplicate tags are created
	tagName := "concurrent-test-tag"
	numGoroutines := 10

	// Channel to collect results from goroutines
	results := make(chan []string, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Launch multiple goroutines that try to convert the same tag name simultaneously
	for i := 0; i < numGoroutines; i++ {
		go func() {
			tagIDs, err := service.ConvertNamesToTags(ctx, []string{tagName})
			if err != nil {
				errors <- err
				return
			}
			results <- tagIDs
		}()
	}

	// Collect all results
	var allTagIDs []string
	for i := 0; i < numGoroutines; i++ {
		select {
		case tagIDs := <-results:
			allTagIDs = append(allTagIDs, tagIDs...)
		case err := <-errors:
			t.Fatalf("ConvertNamesToTags failed: %v", err)
		}
	}

	// Verify all goroutines got the same tag ID (no duplicates created)
	gt.N(t, len(allTagIDs)).Equal(numGoroutines)

	firstTagID := allTagIDs[0]
	for i, tagID := range allTagIDs {
		if tagID != firstTagID {
			t.Errorf("goroutine %d got different tag ID: expected %s, got %s", i, firstTagID, tagID)
		}
	}

	// Verify only one tag was actually created in the repository
	allTags, err := service.ListAllTags(ctx)
	gt.NoError(t, err)

	tagsWithName := 0
	for _, tag := range allTags {
		if tag.Name == tagName {
			tagsWithName++
		}
	}
	if tagsWithName != 1 {
		t.Errorf("expected exactly 1 tag with name %s, but found %d", tagName, tagsWithName)
	}
}

func TestTagService_DeleteTag_FullCleanup(t *testing.T) {
	// This test reproduces the Web UI tag deletion bug by verifying that
	// when a tag is deleted, it's properly removed from all associated alerts and tickets
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Create tags that will be deleted and tags that should remain
	tagToDelete := "delete-me"
	tagToKeep := "keep-me"

	// Create tags using the service
	gt.NoError(t, service.CreateTag(ctx, tagToDelete))
	gt.NoError(t, service.CreateTag(ctx, tagToKeep))

	// Get tag IDs
	tagIDs, err := service.ConvertNamesToTags(ctx, []string{tagToDelete, tagToKeep})
	gt.NoError(t, err)
	gt.N(t, len(tagIDs)).Equal(2)

	deleteTagID := tagIDs[0]
	keepTagID := tagIDs[1]

	// Create an alert with both tags
	a := alert.New(ctx, "test", map[string]string{"test": "data"}, alert.Metadata{
		Title:       "Test Alert",
		Description: "Test Description",
	})
	a.TagIDs = map[string]bool{
		deleteTagID: true,
		keepTagID:   true,
	}
	gt.NoError(t, repo.PutAlert(ctx, a))

	// Create a ticket with both tags
	tk := ticket.New(ctx, []types.AlertID{}, nil)
	tk.Metadata.Title = "Test Ticket"
	tk.TagIDs = map[string]bool{
		deleteTagID: true,
		keepTagID:   true,
	}
	gt.NoError(t, repo.PutTicket(ctx, tk))

	// Verify initial state - both alert and ticket should have 2 tags
	alertBefore, err := repo.GetAlert(ctx, a.ID)
	gt.NoError(t, err)
	gt.N(t, len(alertBefore.TagIDs)).Equal(2)
	gt.True(t, alertBefore.TagIDs[deleteTagID])
	gt.True(t, alertBefore.TagIDs[keepTagID])

	ticketBefore, err := repo.GetTicket(ctx, tk.ID)
	gt.NoError(t, err)
	gt.N(t, len(ticketBefore.TagIDs)).Equal(2)
	gt.True(t, ticketBefore.TagIDs[deleteTagID])
	gt.True(t, ticketBefore.TagIDs[keepTagID])

	// Verify that 2 tags exist in the repository
	tagsBefore, err := service.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tagsBefore)).Equal(2)

	// DELETE THE TAG - this is where the bug occurs
	gt.NoError(t, service.DeleteTag(ctx, tagToDelete))

	// CRITICAL VERIFICATION: Check that the tag was completely removed

	// 1. Tag should no longer exist in repository
	tagsAfter, err := service.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tagsAfter)).Equal(1)
	gt.V(t, tagsAfter[0].Name).Equal(tagToKeep)

	// 2. Tag should be removed from the alert
	alertAfter, err := repo.GetAlert(ctx, a.ID)
	gt.NoError(t, err)
	gt.N(t, len(alertAfter.TagIDs)).Equal(1)    // Should only have keepTagID now
	gt.False(t, alertAfter.TagIDs[deleteTagID]) // deleteTagID should be gone
	gt.True(t, alertAfter.TagIDs[keepTagID])    // keepTagID should remain

	// 3. Tag should be removed from the ticket
	ticketAfter, err := repo.GetTicket(ctx, tk.ID)
	gt.NoError(t, err)
	gt.N(t, len(ticketAfter.TagIDs)).Equal(1)    // Should only have keepTagID now
	gt.False(t, ticketAfter.TagIDs[deleteTagID]) // deleteTagID should be gone
	gt.True(t, ticketAfter.TagIDs[keepTagID])    // keepTagID should remain

	// 4. Verify the deleted tag cannot be retrieved by ID
	deletedTag, err := service.GetTagByID(ctx, deleteTagID)
	gt.True(t, err != nil || deletedTag == nil) // Should not be found

	// 5. Verify the kept tag can still be retrieved by ID
	keptTag, err := service.GetTagByID(ctx, keepTagID)
	gt.NoError(t, err)
	gt.NotNil(t, keptTag)
	gt.V(t, keptTag.Name).Equal(tagToKeep)
}

func TestTagService_DeleteTag_MultipleAlertsAndTickets(t *testing.T) {
	// Test deleting a tag that's used across multiple alerts and tickets
	ctx := context.Background()
	repo := repository.NewMemory()
	service := tag.New(repo)

	// Create a tag that will be used across multiple entities
	sharedTagName := "shared-tag"
	gt.NoError(t, service.CreateTag(ctx, sharedTagName))

	tagIDs, err := service.ConvertNamesToTags(ctx, []string{sharedTagName})
	gt.NoError(t, err)
	sharedTagID := tagIDs[0]

	// Create multiple alerts with the shared tag
	var alertIDs []types.AlertID
	for i := 0; i < 3; i++ {
		a := alert.New(ctx, "test", map[string]interface{}{"index": i}, alert.Metadata{
			Title: "Test Alert " + string(rune(i)),
		})
		a.TagIDs = map[string]bool{sharedTagID: true}
		gt.NoError(t, repo.PutAlert(ctx, a))
		alertIDs = append(alertIDs, a.ID)
	}

	// Create multiple tickets with the shared tag
	var ticketIDs []types.TicketID
	for i := 0; i < 3; i++ {
		tk := ticket.New(ctx, []types.AlertID{}, nil)
		tk.Metadata.Title = "Test Ticket " + string(rune(i))
		tk.TagIDs = map[string]bool{sharedTagID: true}
		gt.NoError(t, repo.PutTicket(ctx, tk))
		ticketIDs = append(ticketIDs, tk.ID)
	}

	// Verify initial state - all alerts and tickets should have the tag
	for _, alertID := range alertIDs {
		a, err := repo.GetAlert(ctx, alertID)
		gt.NoError(t, err)
		gt.True(t, a.TagIDs[sharedTagID])
	}

	for _, ticketID := range ticketIDs {
		tk, err := repo.GetTicket(ctx, ticketID)
		gt.NoError(t, err)
		gt.True(t, tk.TagIDs[sharedTagID])
	}

	// Delete the shared tag
	gt.NoError(t, service.DeleteTag(ctx, sharedTagName))

	// Verify the tag was removed from ALL alerts
	for _, alertID := range alertIDs {
		a, err := repo.GetAlert(ctx, alertID)
		gt.NoError(t, err)
		gt.N(t, len(a.TagIDs)).Equal(0)    // No tags should remain
		gt.False(t, a.TagIDs[sharedTagID]) // Shared tag should be gone
	}

	// Verify the tag was removed from ALL tickets
	for _, ticketID := range ticketIDs {
		tk, err := repo.GetTicket(ctx, ticketID)
		gt.NoError(t, err)
		gt.N(t, len(tk.TagIDs)).Equal(0)    // No tags should remain
		gt.False(t, tk.TagIDs[sharedTagID]) // Shared tag should be gone
	}

	// Verify the tag no longer exists in the repository
	tags, err := service.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.N(t, len(tags)).Equal(0)
}
