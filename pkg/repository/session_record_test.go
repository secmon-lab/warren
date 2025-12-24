package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func TestSessionRecordRepository(t *testing.T) {
	ctx := context.Background()

	testFn := func(t *testing.T, repo interfaces.Repository) {
		now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		ctx = clock.With(ctx, func() time.Time { return now })

		// Use random session ID to avoid conflicts
		sessionID := types.SessionID(time.Now().Format("test-session-20060102-150405.000000000"))

		t.Run("PutSessionRecord and GetSessionRecords", func(t *testing.T) {
			// Create records with different timestamps
			ctx1 := clock.With(ctx, func() time.Time { return now })
			ctx2 := clock.With(ctx, func() time.Time { return now.Add(1 * time.Millisecond) })

			record1 := session.NewSessionRecord(ctx1, sessionID, "First message")
			record2 := session.NewSessionRecord(ctx2, sessionID, "Second message")

			gt.NoError(t, repo.PutSessionRecord(ctx, record1))
			gt.NoError(t, repo.PutSessionRecord(ctx, record2))

			records, err := repo.GetSessionRecords(ctx, sessionID)
			gt.NoError(t, err)
			gt.Equal(t, len(records), 2)

			// Verify records are in time series order
			gt.Equal(t, records[0].ID, record1.ID)
			gt.Equal(t, records[0].Content, "First message")
			gt.Equal(t, records[1].ID, record2.ID)
			gt.Equal(t, records[1].Content, "Second message")
		})

		t.Run("GetSessionRecords with different sessions", func(t *testing.T) {
			sessionID2 := types.SessionID(time.Now().Format("test-session2-20060102-150405.000000000"))

			record3 := session.NewSessionRecord(ctx, sessionID2, "Session 2 message")
			gt.NoError(t, repo.PutSessionRecord(ctx, record3))

			// Get records for session 2
			records2, err := repo.GetSessionRecords(ctx, sessionID2)
			gt.NoError(t, err)
			gt.Equal(t, len(records2), 1)
			gt.Equal(t, records2[0].Content, "Session 2 message")
		})

		t.Run("GetSessionRecords for empty session", func(t *testing.T) {
			sessionID3 := types.SessionID(time.Now().Format("test-session3-20060102-150405.000000000"))

			records, err := repo.GetSessionRecords(ctx, sessionID3)
			gt.NoError(t, err)
			gt.Equal(t, len(records), 0)
		})

		t.Run("Multiple records in time series order", func(t *testing.T) {
			sessionID4 := types.SessionID(time.Now().Format("test-session4-20060102-150405.000000000"))

			// Create records with different timestamps
			ctx1 := clock.With(ctx, func() time.Time { return now })
			ctx2 := clock.With(ctx, func() time.Time { return now.Add(1 * time.Second) })
			ctx3 := clock.With(ctx, func() time.Time { return now.Add(2 * time.Second) })

			record1 := session.NewSessionRecord(ctx1, sessionID4, "Message at T+0")
			record2 := session.NewSessionRecord(ctx2, sessionID4, "Message at T+1")
			record3 := session.NewSessionRecord(ctx3, sessionID4, "Message at T+2")

			gt.NoError(t, repo.PutSessionRecord(ctx, record1))
			gt.NoError(t, repo.PutSessionRecord(ctx, record2))
			gt.NoError(t, repo.PutSessionRecord(ctx, record3))

			records, err := repo.GetSessionRecords(ctx, sessionID4)
			gt.NoError(t, err)
			gt.Equal(t, len(records), 3)

			// Verify time series order
			gt.Equal(t, records[0].Content, "Message at T+0")
			gt.Equal(t, records[1].Content, "Message at T+1")
			gt.Equal(t, records[2].Content, "Message at T+2")

			// Verify timestamps are in ascending order
			gt.True(t, records[0].CreatedAt.Before(records[1].CreatedAt))
			gt.True(t, records[1].CreatedAt.Before(records[2].CreatedAt))
		})
	}

	t.Run("Memory", func(t *testing.T) {
		testFn(t, repository.NewMemory())
	})

	t.Run("Firestore", func(t *testing.T) {
		testFn(t, newFirestoreClient(t))
	})
}
