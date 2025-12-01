package firestore_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository/firestore"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

// isIndexRequiredError checks if the error is due to missing Firestore index
func isIndexRequiredError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "requires an index")
}

func TestFirestoreSessionRepository(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_FIRESTORE_PROJECT_ID", "TEST_FIRESTORE_DATABASE_ID")

	ctx := context.Background()
	repo, err := firestore.New(ctx, vars.Get("TEST_FIRESTORE_PROJECT_ID"), vars.Get("TEST_FIRESTORE_DATABASE_ID"))
	gt.NoError(t, err).Required()
	defer func() {
		_ = repo.Close()
	}()

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	// Use unique ticket ID to avoid conflicts
	ticketID := types.TicketID(time.Now().Format("test-ticket-20060102-150405.000000"))
	sess := session.NewSession(ctx, ticketID)

	t.Run("PutSession and GetSession", func(t *testing.T) {
		if err := repo.PutSession(ctx, sess); err != nil {
			t.Fatalf("PutSession failed: %v", err)
		}

		retrieved, err := repo.GetSession(ctx, sess.ID)
		if err != nil {
			t.Fatalf("GetSession failed: %v", err)
		}

		if retrieved == nil {
			t.Fatal("GetSession returned nil")
		}

		if retrieved.ID != sess.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, sess.ID)
		}

		if retrieved.TicketID != sess.TicketID {
			t.Errorf("TicketID = %v, want %v", retrieved.TicketID, sess.TicketID)
		}

		if retrieved.Status != sess.Status {
			t.Errorf("Status = %v, want %v", retrieved.Status, sess.Status)
		}

		if retrieved.RequestID != sess.RequestID {
			t.Errorf("RequestID = %v, want %v", retrieved.RequestID, sess.RequestID)
		}

		// Clean up
		defer func() {
			_ = repo.DeleteSession(ctx, sess.ID)
		}()
	})

	t.Run("GetSessionByTicket", func(t *testing.T) {
		// Create new session for this test
		ticketID2 := types.TicketID(time.Now().Format("test-ticket2-20060102-150405.000000"))
		sess2 := session.NewSession(ctx, ticketID2)

		if err := repo.PutSession(ctx, sess2); err != nil {
			t.Fatalf("PutSession failed: %v", err)
		}
		defer func() {
			_ = repo.DeleteSession(ctx, sess2.ID)
		}()

		retrieved, err := repo.GetSessionByTicket(ctx, ticketID2)
		if err != nil {
			// Skip if composite index is not created yet
			if isIndexRequiredError(err) {
				t.Skip("Firestore composite index not created yet - see spec.md for index creation")
			}
			t.Fatalf("GetSessionByTicket failed: %v", err)
		}

		if retrieved == nil {
			t.Fatal("GetSessionByTicket returned nil")
		}

		if retrieved.ID != sess2.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, sess2.ID)
		}

		if retrieved.Status != types.SessionStatusRunning {
			t.Errorf("Status = %v, want %v", retrieved.Status, types.SessionStatusRunning)
		}
	})

	t.Run("UpdateStatus to aborted", func(t *testing.T) {
		// Create new session for this test
		ticketID3 := types.TicketID(time.Now().Format("test-ticket3-20060102-150405.000000"))
		sess3 := session.NewSession(ctx, ticketID3)

		if err := repo.PutSession(ctx, sess3); err != nil {
			t.Fatalf("PutSession failed: %v", err)
		}
		defer func() {
			_ = repo.DeleteSession(ctx, sess3.ID)
		}()

		sess3.UpdateStatus(ctx, types.SessionStatusAborted)
		if err := repo.PutSession(ctx, sess3); err != nil {
			t.Fatalf("PutSession failed: %v", err)
		}

		// Should not be returned by GetSessionByTicket (status != running)
		retrieved, err := repo.GetSessionByTicket(ctx, ticketID3)
		if err != nil {
			// Skip if composite index is not created yet
			if isIndexRequiredError(err) {
				t.Skip("Firestore composite index not created yet - see spec.md for index creation")
			}
			t.Fatalf("GetSessionByTicket failed: %v", err)
		}

		if retrieved != nil {
			t.Errorf("GetSessionByTicket should return nil for aborted session, got %v", retrieved)
		}

		// But should still be retrievable by ID
		byID, err := repo.GetSession(ctx, sess3.ID)
		if err != nil {
			t.Fatalf("GetSession failed: %v", err)
		}

		if byID == nil {
			t.Fatal("GetSession should still return aborted session")
		}

		if byID.Status != types.SessionStatusAborted {
			t.Errorf("Status = %v, want %v", byID.Status, types.SessionStatusAborted)
		}
	})

	t.Run("DeleteSession", func(t *testing.T) {
		// Create new session for this test
		ticketID4 := types.TicketID(time.Now().Format("test-ticket4-20060102-150405.000000"))
		sess4 := session.NewSession(ctx, ticketID4)

		if err := repo.PutSession(ctx, sess4); err != nil {
			t.Fatalf("PutSession failed: %v", err)
		}

		if err := repo.DeleteSession(ctx, sess4.ID); err != nil {
			t.Fatalf("DeleteSession failed: %v", err)
		}

		retrieved, err := repo.GetSession(ctx, sess4.ID)
		if err != nil {
			t.Fatalf("GetSession failed: %v", err)
		}

		if retrieved != nil {
			t.Errorf("GetSession should return nil after deletion, got %v", retrieved)
		}
	})

	t.Run("GetSession_NotFound", func(t *testing.T) {
		retrieved, err := repo.GetSession(ctx, types.SessionID("nonexistent"))
		if err != nil {
			t.Fatalf("GetSession failed: %v", err)
		}

		if retrieved != nil {
			t.Errorf("GetSession should return nil for nonexistent session, got %v", retrieved)
		}
	})

	t.Run("GetSessionByTicket_NotFound", func(t *testing.T) {
		retrieved, err := repo.GetSessionByTicket(ctx, types.TicketID("nonexistent"))
		if err != nil {
			// Skip if composite index is not created yet
			if isIndexRequiredError(err) {
				t.Skip("Firestore composite index not created yet - see spec.md for index creation")
			}
			t.Fatalf("GetSessionByTicket failed: %v", err)
		}

		if retrieved != nil {
			t.Errorf("GetSessionByTicket should return nil for nonexistent ticket, got %v", retrieved)
		}
	})
}
