package repository_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	ticketmodel "github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestTagOperations(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		t.Run("Create and list tags", func(t *testing.T) {
			// Get initial tag count
			initialTags, err := repo.ListAllTags(ctx)
			gt.NoError(t, err)
			initialCount := len(initialTags)

			// Create tags with unique names
			timestamp := time.Now().UnixNano()
			tag1Name := fmt.Sprintf("security_%d", timestamp)
			tag2Name := fmt.Sprintf("incident_%d", timestamp)
			tag3Name := fmt.Sprintf("phishing_%d", timestamp)

			tag1 := &tag.Tag{ID: tag.NewID(), Name: tag1Name, Color: "#ff0000"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag1))

			tag2 := &tag.Tag{ID: tag.NewID(), Name: tag2Name, Color: "#00ff00"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag2))

			tag3 := &tag.Tag{ID: tag.NewID(), Name: tag3Name, Color: "#0000ff"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag3))

			// List tags and verify we have at least the 3 new ones
			updatedTags, err := repo.ListAllTags(ctx)
			gt.NoError(t, err)
			gt.Number(t, len(updatedTags)).GreaterOrEqual(initialCount + 3)

			// Verify tag names exist in updated list
			tagNames := make(map[string]bool)
			for _, tag := range updatedTags {
				tagNames[tag.Name] = true
			}
			gt.True(t, tagNames[tag1Name])
			gt.True(t, tagNames[tag2Name])
			gt.True(t, tagNames[tag3Name])
		})

		t.Run("Create duplicate tag", func(t *testing.T) {
			// Use unique name to avoid conflicts with existing data
			uniqueName := fmt.Sprintf("duplicate_%d", time.Now().UnixNano())

			// Get initial count of tags with this unique name
			initialTags, err := repo.ListAllTags(ctx)
			gt.NoError(t, err)
			initialDuplicateCount := 0
			for _, tag := range initialTags {
				if tag.Name == uniqueName {
					initialDuplicateCount++
				}
			}

			// Create a tag
			tag1 := &tag.Tag{ID: tag.NewID(), Name: uniqueName, Color: "#ff0000"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag1))

			// Try to create a different tag with same name (should succeed with different ID)
			tag2 := &tag.Tag{ID: tag.NewID(), Name: uniqueName, Color: "#00ff00"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag2))

			// Verify both new tags exist
			updatedTags, err := repo.ListAllTags(ctx)
			gt.NoError(t, err)
			finalDuplicateCount := 0
			for _, tag := range updatedTags {
				if tag.Name == uniqueName {
					finalDuplicateCount++
				}
			}
			gt.Number(t, finalDuplicateCount).GreaterOrEqual(initialDuplicateCount + 2)
		})

		t.Run("Get tag by ID", func(t *testing.T) {
			// Create a tag with unique name
			uniqueName := fmt.Sprintf("gettag_%d", time.Now().UnixNano())
			testTag := &tag.Tag{ID: tag.NewID(), Name: uniqueName, Color: "#ff0000"}
			gt.NoError(t, repo.CreateTagWithID(ctx, testTag))

			// Get existing tag by ID
			retrievedTag, err := repo.GetTagByID(ctx, testTag.ID)
			gt.NoError(t, err)
			gt.NotNil(t, retrievedTag)
			gt.V(t, retrievedTag.Name).Equal(uniqueName)
			gt.V(t, retrievedTag.ID).Equal(testTag.ID)

			// Get non-existent tag
			nonExistent, err := repo.GetTagByID(ctx, tag.NewID())
			gt.NoError(t, err)
			gt.Nil(t, nonExistent)
		})

		t.Run("Get tag by name", func(t *testing.T) {
			// Use unique name to avoid conflicts with existing data
			uniqueName := fmt.Sprintf("nametest_%d", time.Now().UnixNano())
			testTag := &tag.Tag{ID: tag.NewID(), Name: uniqueName, Color: "#00ff00"}
			gt.NoError(t, repo.CreateTagWithID(ctx, testTag))

			// Verify tag was created by getting it by ID first
			verifyTag, err := repo.GetTagByID(ctx, testTag.ID)
			gt.NoError(t, err)
			gt.NotNil(t, verifyTag)
			t.Logf("Created tag with ID: %s, Name: %s", verifyTag.ID, verifyTag.Name)

			// Wait for Firestore eventual consistency

			// Get existing tag by name
			retrievedTag, err := repo.GetTagByName(ctx, uniqueName)
			gt.NoError(t, err)
			if retrievedTag == nil {
				t.Logf("Warning: GetTagByName returned nil for name: %s", uniqueName)
				t.Logf("Trying to list all tags to debug...")
				allTags, listErr := repo.ListAllTags(ctx)
				if listErr == nil {
					for _, tag := range allTags {
						if tag.Name == uniqueName {
							t.Logf("Found matching tag in list: ID=%s, Name=%s", tag.ID, tag.Name)
						}
					}
				}
			}
			gt.NoError(t, err)
			if retrievedTag == nil {
				t.Fatal("retrievedTag is nil")
			}
			gt.V(t, retrievedTag.Name).Equal(uniqueName)
			gt.V(t, retrievedTag.ID).Equal(testTag.ID)

			// Get non-existent tag
			nonExistent, err := repo.GetTagByName(ctx, "nonexistent")
			gt.NoError(t, err)
			gt.Nil(t, nonExistent)
		})

		t.Run("Delete tag by ID", func(t *testing.T) {
			// Create tags with unique names
			timestamp := time.Now().UnixNano()
			tag1Name := fmt.Sprintf("delete1_%d", timestamp)
			tag2Name := fmt.Sprintf("delete2_%d", timestamp)

			tag1 := &tag.Tag{ID: tag.NewID(), Name: tag1Name, Color: "#ff0000"}
			tag2 := &tag.Tag{ID: tag.NewID(), Name: tag2Name, Color: "#00ff00"}
			gt.NoError(t, repo.CreateTagWithID(ctx, tag1))
			gt.NoError(t, repo.CreateTagWithID(ctx, tag2))

			// Delete one tag by ID
			gt.NoError(t, repo.DeleteTagByID(ctx, tag1.ID))

			// Verify it's deleted
			deletedTag, err := repo.GetTagByID(ctx, tag1.ID)
			gt.NoError(t, err)
			gt.Nil(t, deletedTag)

			// Other tag should still exist
			remainingTag, err := repo.GetTagByID(ctx, tag2.ID)
			gt.NoError(t, err)
			gt.NotNil(t, remainingTag)
		})

		t.Run("Tag timestamps", func(t *testing.T) {
			// Create a tag with unique name
			uniqueName := fmt.Sprintf("timestamped_%d", time.Now().UnixNano())
			before := time.Now()
			testTag := &tag.Tag{ID: tag.NewID(), Name: uniqueName, Color: "#0000ff"}
			gt.NoError(t, repo.CreateTagWithID(ctx, testTag))
			after := time.Now()

			// Get the tag
			retrievedTag, err := repo.GetTagByID(ctx, testTag.ID)
			gt.NoError(t, err)
			gt.NotNil(t, retrievedTag)

			// Verify timestamps are set
			gt.True(t, !retrievedTag.CreatedAt.IsZero())
			gt.True(t, !retrievedTag.UpdatedAt.IsZero())
			gt.True(t, retrievedTag.CreatedAt.Equal(retrievedTag.UpdatedAt))

			// Verify timestamps are within expected range
			gt.True(t, retrievedTag.CreatedAt.After(before.Add(-time.Second)))
			gt.True(t, retrievedTag.CreatedAt.Before(after.Add(time.Second)))
		})
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("Firestore", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}

func TestAlertAndTicketTags(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := t.Context()

		t.Run("Alert with tags", func(t *testing.T) {
			// Create tags first in the new system
			securityTag := &tag.Tag{
				ID:   tag.NewID(),
				Name: "security",
			}
			incidentTag := &tag.Tag{
				ID:   tag.NewID(),
				Name: "incident",
			}
			criticalTag := &tag.Tag{
				ID:   tag.NewID(),
				Name: "critical",
			}

			// Store tags in repository
			gt.NoError(t, repo.CreateTagWithID(ctx, securityTag))
			gt.NoError(t, repo.CreateTagWithID(ctx, incidentTag))
			gt.NoError(t, repo.CreateTagWithID(ctx, criticalTag))

			// Create an alert with tags
			a := alert.New(ctx, "test", map[string]string{"test": "data"}, alert.Metadata{
				Title:       "Test Alert",
				Description: "Test Description",
			})
			// Generate random embedding
			emb := make([]float32, 256)
			for i := range emb {
				emb[i] = rand.Float32()
			}
			a.Embedding = emb
			if a.TagIDs == nil {
				a.TagIDs = make(map[string]bool)
			}
			a.TagIDs[securityTag.ID] = true
			a.TagIDs[incidentTag.ID] = true
			a.TagIDs[criticalTag.ID] = true

			// Save the alert
			gt.NoError(t, repo.PutAlert(ctx, a))

			// Retrieve the alert
			retrievedAlert, err := repo.GetAlert(ctx, a.ID)
			gt.NoError(t, err)
			gt.NotNil(t, retrievedAlert)

			// Verify tags are preserved
			gt.Number(t, len(retrievedAlert.TagIDs)).Equal(3)
			// Check tags are present in map
			gt.True(t, retrievedAlert.TagIDs[securityTag.ID])
			gt.True(t, retrievedAlert.TagIDs[incidentTag.ID])
			gt.True(t, retrievedAlert.TagIDs[criticalTag.ID])
		})

		t.Run("Ticket with tags", func(t *testing.T) {
			// Create tags first
			resolvedTag := &tag.Tag{
				ID:   tag.NewID(),
				Name: "resolved",
			}
			fpTag := &tag.Tag{
				ID:   tag.NewID(),
				Name: "false-positive",
			}

			// Store tags in repository
			gt.NoError(t, repo.CreateTagWithID(ctx, resolvedTag))
			gt.NoError(t, repo.CreateTagWithID(ctx, fpTag))

			// Create a ticket with tags
			tk := ticketmodel.New(ctx, []types.AlertID{}, nil)
			tk.Title = "Test Ticket"
			// Generate random embedding
			tkEmb := make([]float32, 256)
			for i := range tkEmb {
				tkEmb[i] = rand.Float32()
			}
			tk.Embedding = tkEmb
			if tk.TagIDs == nil {
				tk.TagIDs = make(map[string]bool)
			}
			tk.TagIDs[resolvedTag.ID] = true
			tk.TagIDs[fpTag.ID] = true

			// Save the ticket
			gt.NoError(t, repo.PutTicket(ctx, tk))

			// Retrieve the ticket
			retrievedTicket, err := repo.GetTicket(ctx, tk.ID)
			gt.NoError(t, err)
			gt.NotNil(t, retrievedTicket)

			// Verify tags are preserved
			gt.Number(t, len(retrievedTicket.TagIDs)).Equal(2)
			gt.True(t, retrievedTicket.TagIDs[resolvedTag.ID])
			gt.True(t, retrievedTicket.TagIDs[fpTag.ID])
		})

		t.Run("Empty tags", func(t *testing.T) {
			// Create alert without tags
			a := alert.New(ctx, "test", map[string]string{"test": "data"}, alert.Metadata{
				Title:       "No Tags Alert",
				Description: "Test Description",
			})
			// a.Tags should be nil by default
			// Generate random embedding
			emb := make([]float32, 256)
			for i := range emb {
				emb[i] = rand.Float32()
			}
			a.Embedding = emb

			gt.NoError(t, repo.PutAlert(ctx, a))

			// Retrieve and verify
			retrievedAlert, err := repo.GetAlert(ctx, a.ID)
			gt.NoError(t, err)
			gt.NotNil(t, retrievedAlert)
			// TagIDs should be nil or empty
			if retrievedAlert.TagIDs != nil {
				gt.Number(t, len(retrievedAlert.TagIDs)).Equal(0)
			}
		})

		t.Run("Tag persistence in batch operations", func(t *testing.T) {
			// Create common tag and individual tags
			commonTag := &tag.Tag{
				ID:   tag.NewID(),
				Name: "common",
			}
			gt.NoError(t, repo.CreateTagWithID(ctx, commonTag))

			var individualTags []*tag.Tag
			for i := range 3 {
				individualTag := &tag.Tag{
					ID:   tag.NewID(),
					Name: fmt.Sprintf("tag%d", i),
				}
				gt.NoError(t, repo.CreateTagWithID(ctx, individualTag))
				individualTags = append(individualTags, individualTag)
			}

			// Create multiple alerts with tags
			alerts := make(alert.Alerts, 3)
			for i := range 3 {
				a := alert.New(ctx, "test", map[string]string{"index": fmt.Sprintf("%d", i)}, alert.Metadata{
					Title:       fmt.Sprintf("Batch Alert %d", i),
					Description: "Test Description",
				})
				// Generate random embedding
				emb := make([]float32, 256)
				for j := range emb {
					emb[j] = rand.Float32()
				}
				a.Embedding = emb
				if a.TagIDs == nil {
					a.TagIDs = make(map[string]bool)
				}
				a.TagIDs[individualTags[i].ID] = true
				a.TagIDs[commonTag.ID] = true
				alerts[i] = &a
			}

			// Batch save
			gt.NoError(t, repo.BatchPutAlerts(ctx, alerts))

			// Batch retrieve
			alertIDs := make([]types.AlertID, len(alerts))
			for i, a := range alerts {
				alertIDs[i] = a.ID
			}
			retrievedAlerts, err := repo.BatchGetAlerts(ctx, alertIDs)
			gt.NoError(t, err)
			gt.Array(t, retrievedAlerts).Length(3)

			// Verify tags
			for i, a := range retrievedAlerts {
				gt.True(t, a.TagIDs[individualTags[i].ID])
				gt.True(t, a.TagIDs[commonTag.ID])
			}
		})
	}

	t.Run("Memory", func(t *testing.T) {
		repo := repository.NewMemory()
		testFn(t, repo)
	})

	t.Run("Firestore", func(t *testing.T) {
		repo := newFirestoreClient(t)
		testFn(t, repo)
	})
}
