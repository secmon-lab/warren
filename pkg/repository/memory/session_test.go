package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository/memory"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func TestSessionRepository(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	ticketID := types.TicketID("test-ticket-001")
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
	})

	t.Run("GetSessionByTicket", func(t *testing.T) {
		retrieved, err := repo.GetSessionByTicket(ctx, ticketID)
		if err != nil {
			t.Fatalf("GetSessionByTicket failed: %v", err)
		}

		if retrieved == nil {
			t.Fatal("GetSessionByTicket returned nil")
		}

		if retrieved.ID != sess.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, sess.ID)
		}

		if retrieved.Status != types.SessionStatusRunning {
			t.Errorf("Status = %v, want %v", retrieved.Status, types.SessionStatusRunning)
		}
	})

	t.Run("UpdateStatus to aborted removes from index", func(t *testing.T) {
		sess.UpdateStatus(ctx, types.SessionStatusAborted)
		if err := repo.PutSession(ctx, sess); err != nil {
			t.Fatalf("PutSession failed: %v", err)
		}

		// Should not be returned by GetSessionByTicket
		retrieved, err := repo.GetSessionByTicket(ctx, ticketID)
		if err != nil {
			t.Fatalf("GetSessionByTicket failed: %v", err)
		}

		if retrieved != nil {
			t.Errorf("GetSessionByTicket should return nil for aborted session, got %v", retrieved)
		}

		// But should still be retrievable by ID
		byID, err := repo.GetSession(ctx, sess.ID)
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
		if err := repo.DeleteSession(ctx, sess.ID); err != nil {
			t.Fatalf("DeleteSession failed: %v", err)
		}

		retrieved, err := repo.GetSession(ctx, sess.ID)
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
			t.Fatalf("GetSessionByTicket failed: %v", err)
		}

		if retrieved != nil {
			t.Errorf("GetSessionByTicket should return nil for nonexistent ticket, got %v", retrieved)
		}
	})

	t.Run("DeleteSession_NotFound", func(t *testing.T) {
		err := repo.DeleteSession(ctx, types.SessionID("nonexistent"))
		if err == nil {
			t.Fatal("DeleteSession should return error for nonexistent session")
		}
	})
}

func TestSessionRepository_MultipleSessionsPerTicket(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()

	ticketID := types.TicketID("test-ticket-001")

	// Create first session
	sess1 := session.NewSession(ctx, ticketID)
	if err := repo.PutSession(ctx, sess1); err != nil {
		t.Fatalf("PutSession failed: %v", err)
	}

	// GetSessionByTicket should return sess1
	retrieved, err := repo.GetSessionByTicket(ctx, ticketID)
	if err != nil {
		t.Fatalf("GetSessionByTicket failed: %v", err)
	}

	if retrieved.ID != sess1.ID {
		t.Errorf("GetSessionByTicket returned wrong session: %v, want %v", retrieved.ID, sess1.ID)
	}

	// Complete sess1
	sess1.UpdateStatus(ctx, types.SessionStatusCompleted)
	if err := repo.PutSession(ctx, sess1); err != nil {
		t.Fatalf("PutSession failed: %v", err)
	}

	// Create second session
	sess2 := session.NewSession(ctx, ticketID)
	if err := repo.PutSession(ctx, sess2); err != nil {
		t.Fatalf("PutSession failed: %v", err)
	}

	// GetSessionByTicket should now return sess2
	retrieved, err = repo.GetSessionByTicket(ctx, ticketID)
	if err != nil {
		t.Fatalf("GetSessionByTicket failed: %v", err)
	}

	if retrieved.ID != sess2.ID {
		t.Errorf("GetSessionByTicket returned wrong session: %v, want %v", retrieved.ID, sess2.ID)
	}

	// Both sessions should still be retrievable by ID
	s1, err := repo.GetSession(ctx, sess1.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if s1 == nil {
		t.Error("sess1 should still be retrievable")
	}

	s2, err := repo.GetSession(ctx, sess2.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if s2 == nil {
		t.Error("sess2 should still be retrievable")
	}
}
