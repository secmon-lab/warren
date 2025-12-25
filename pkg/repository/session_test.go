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

func TestSessionRepository(t *testing.T) {
	ctx := context.Background()

	testFn := func(t *testing.T, repo interfaces.Repository) {
		now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		ctx = clock.With(ctx, func() time.Time { return now })

		ticketID := types.TicketID(time.Now().Format("test-ticket-20060102-150405.000000"))
		sess := session.NewSession(ctx, ticketID, types.UserID("user-test"), "test query", "")

		t.Run("PutSession and GetSession", func(t *testing.T) {
			gt.NoError(t, repo.PutSession(ctx, sess))

			retrieved, err := repo.GetSession(ctx, sess.ID)
			gt.NoError(t, err)
			gt.V(t, retrieved).NotNil()
			gt.Equal(t, retrieved.ID, sess.ID)
			gt.Equal(t, retrieved.TicketID, sess.TicketID)
			gt.Equal(t, retrieved.Status, sess.Status)
			gt.Equal(t, retrieved.RequestID, sess.RequestID)

			// Clean up
			defer func() {
				_ = repo.DeleteSession(ctx, sess.ID)
			}()
		})

		t.Run("GetSessionsByTicket", func(t *testing.T) {
			ticketID2 := types.TicketID(time.Now().Format("test-ticket2-20060102-150405.000000"))
			sess2 := session.NewSession(ctx, ticketID2, "", "", "")

			gt.NoError(t, repo.PutSession(ctx, sess2))
			defer func() {
				_ = repo.DeleteSession(ctx, sess2.ID)
			}()

			sessions, err := repo.GetSessionsByTicket(ctx, ticketID2)
			gt.NoError(t, err)
			gt.Equal(t, len(sessions), 1)
			gt.Equal(t, sessions[0].ID, sess2.ID)
			gt.Equal(t, sessions[0].Status, types.SessionStatusRunning)
		})

		t.Run("GetSessionsByTicket with multiple sessions", func(t *testing.T) {
			ticketID3 := types.TicketID(time.Now().Format("test-ticket3-20060102-150405.000000"))
			sess3a := session.NewSession(ctx, ticketID3, "", "", "")
			sess3b := session.NewSession(ctx, ticketID3, "", "", "")

			gt.NoError(t, repo.PutSession(ctx, sess3a))
			gt.NoError(t, repo.PutSession(ctx, sess3b))
			defer func() {
				_ = repo.DeleteSession(ctx, sess3a.ID)
				_ = repo.DeleteSession(ctx, sess3b.ID)
			}()

			sessions, err := repo.GetSessionsByTicket(ctx, ticketID3)
			gt.NoError(t, err)
			gt.Equal(t, len(sessions), 2)

			// Verify both sessions are present
			foundIDs := make(map[types.SessionID]bool)
			for _, s := range sessions {
				foundIDs[s.ID] = true
			}
			gt.True(t, foundIDs[sess3a.ID])
			gt.True(t, foundIDs[sess3b.ID])
		})

		t.Run("UpdateStatus to aborted", func(t *testing.T) {
			ticketID4 := types.TicketID(time.Now().Format("test-ticket4-20060102-150405.000000"))
			sess4 := session.NewSession(ctx, ticketID4, "", "", "")

			gt.NoError(t, repo.PutSession(ctx, sess4))
			defer func() {
				_ = repo.DeleteSession(ctx, sess4.ID)
			}()

			sess4.UpdateStatus(ctx, types.SessionStatusAborted)
			gt.NoError(t, repo.PutSession(ctx, sess4))

			// Should still be returned by GetSessionsByTicket
			sessions, err := repo.GetSessionsByTicket(ctx, ticketID4)
			gt.NoError(t, err)
			gt.Equal(t, len(sessions), 1)
			gt.Equal(t, sessions[0].Status, types.SessionStatusAborted)

			// Should still be retrievable by ID
			byID, err := repo.GetSession(ctx, sess4.ID)
			gt.NoError(t, err)
			gt.V(t, byID).NotNil()
			gt.Equal(t, byID.Status, types.SessionStatusAborted)
		})

		t.Run("DeleteSession", func(t *testing.T) {
			ticketID5 := types.TicketID(time.Now().Format("test-ticket5-20060102-150405.000000"))
			sess5 := session.NewSession(ctx, ticketID5, "", "", "")

			gt.NoError(t, repo.PutSession(ctx, sess5))
			gt.NoError(t, repo.DeleteSession(ctx, sess5.ID))

			retrieved, err := repo.GetSession(ctx, sess5.ID)
			gt.NoError(t, err)
			gt.V(t, retrieved).Nil()
		})

		t.Run("GetSession_NotFound", func(t *testing.T) {
			retrieved, err := repo.GetSession(ctx, types.SessionID("nonexistent"))
			gt.NoError(t, err)
			gt.V(t, retrieved).Nil()
		})

		t.Run("GetSessionsByTicket_NotFound", func(t *testing.T) {
			sessions, err := repo.GetSessionsByTicket(ctx, types.TicketID("nonexistent"))
			gt.NoError(t, err)
			gt.Equal(t, len(sessions), 0)
		})
	}

	t.Run("Memory", func(t *testing.T) {
		testFn(t, repository.NewMemory())
	})

	t.Run("Firestore", func(t *testing.T) {
		testFn(t, newFirestoreClient(t))
	})
}
