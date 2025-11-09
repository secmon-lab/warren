package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestNoticeRepository(t *testing.T) {
	testFn := func(t *testing.T, repo interfaces.Repository) {
		ctx := context.Background()

		t.Run("create and get notice", func(t *testing.T) {
			// Use random ID to avoid test conflicts (CLAUDE.md requirement)
			noticeID := types.NoticeID(fmt.Sprintf("notice-%d", time.Now().UnixNano()))
			alertID := types.NewAlertID()

			testNotice := &notice.Notice{
				ID: noticeID,
				Alert: alert.Alert{
					ID: alertID,
					Metadata: alert.Metadata{
						Title:       "Test Security Notice",
						Description: "This is a test notice for repository testing",
					},
					Schema: "test.schema",
					Data: map[string]any{
						"severity": "medium",
						"source":   "test-system",
					},
				},
				CreatedAt: time.Now(),
				Escalated: false,
			}

			// Create notice
			err := repo.CreateNotice(ctx, testNotice)
			gt.NoError(t, err)

			// Get notice and verify ALL fields (CLAUDE.md requirement)
			retrievedNotice, err := repo.GetNotice(ctx, noticeID)
			gt.NoError(t, err)

			// Verify all fields match what was saved
			gt.Equal(t, retrievedNotice.ID, noticeID)
			gt.Equal(t, retrievedNotice.Alert.ID, alertID)
			gt.S(t, retrievedNotice.Alert.Metadata.Title).Equal("Test Security Notice")
			gt.S(t, retrievedNotice.Alert.Metadata.Description).Equal("This is a test notice for repository testing")
			gt.V(t, retrievedNotice.Alert.Schema).Equal("test.schema")
			gt.V(t, retrievedNotice.Alert.Data).Equal(testNotice.Alert.Data)
			gt.False(t, retrievedNotice.Escalated)

			// Verify timestamp with tolerance (CLAUDE.md requirement for timestamp comparisons)
			timeDiff := retrievedNotice.CreatedAt.Sub(testNotice.CreatedAt)
			if timeDiff < 0 {
				timeDiff = -timeDiff
			}
			gt.True(t, timeDiff < time.Second)
		})

		t.Run("update notice escalation status", func(t *testing.T) {
			// Use random ID to avoid test conflicts
			noticeID := types.NoticeID(fmt.Sprintf("notice-%d", time.Now().UnixNano()))

			originalNotice := &notice.Notice{
				ID: noticeID,
				Alert: alert.Alert{
					ID: types.NewAlertID(),
					Metadata: alert.Metadata{
						Title: "Notice to Escalate",
					},
				},
				CreatedAt: time.Now(),
				Escalated: false,
			}

			// Create notice
			err := repo.CreateNotice(ctx, originalNotice)
			gt.NoError(t, err)

			// Update escalation status
			originalNotice.Escalated = true
			err = repo.UpdateNotice(ctx, originalNotice)
			gt.NoError(t, err)

			// Verify update
			updatedNotice, err := repo.GetNotice(ctx, noticeID)
			gt.NoError(t, err)
			gt.True(t, updatedNotice.Escalated)
			gt.Equal(t, updatedNotice.ID, noticeID)
			gt.S(t, updatedNotice.Alert.Metadata.Title).Equal("Notice to Escalate")
		})

		t.Run("get nonexistent notice", func(t *testing.T) {
			// Use random ID that doesn't exist
			nonexistentID := types.NoticeID(fmt.Sprintf("notice-%d", time.Now().UnixNano()))

			_, err := repo.GetNotice(ctx, nonexistentID)
			gt.Error(t, err)
			gt.S(t, err.Error()).Contains("notice not found")
		})

		t.Run("update nonexistent notice", func(t *testing.T) {
			// Try to update notice that doesn't exist
			nonexistentNotice := &notice.Notice{
				ID: types.NoticeID(fmt.Sprintf("notice-%d", time.Now().UnixNano())),
				Alert: alert.Alert{
					ID: types.NewAlertID(),
					Metadata: alert.Metadata{
						Title: "Nonexistent Notice",
					},
				},
				CreatedAt: time.Now(),
				Escalated: false,
			}

			err := repo.UpdateNotice(ctx, nonexistentNotice)
			gt.Error(t, err)
			gt.S(t, err.Error()).Contains("notice not found")
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
