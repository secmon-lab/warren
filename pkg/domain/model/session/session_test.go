package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
)

func TestNewSession(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket-001")
	requestID := "test-request-123"

	// Set request ID in context
	ctx = request_id.With(ctx, requestID)

	// Set fixed time for testing
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	sess := session.NewSession(ctx, ticketID)

	if sess.ID == "" {
		t.Error("SessionID should not be empty")
	}

	if sess.TicketID != ticketID {
		t.Errorf("TicketID = %v, want %v", sess.TicketID, ticketID)
	}

	if sess.RequestID != requestID {
		t.Errorf("RequestID = %v, want %v", sess.RequestID, requestID)
	}

	if sess.Status != types.SessionStatusRunning {
		t.Errorf("Status = %v, want %v", sess.Status, types.SessionStatusRunning)
	}

	if !sess.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", sess.CreatedAt, now)
	}

	if !sess.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", sess.UpdatedAt, now)
	}
}

func TestNewSession_NoRequestID(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket-001")

	sess := session.NewSession(ctx, ticketID)

	if sess.RequestID != "unknown" {
		t.Errorf("RequestID = %v, want %v", sess.RequestID, "unknown")
	}
}

func TestUpdateStatus(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket-001")

	// Create session at time T1
	t1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx1 := clock.With(ctx, func() time.Time { return t1 })
	sess := session.NewSession(ctx1, ticketID)

	// Update status at time T2
	t2 := time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC)
	ctx2 := clock.With(ctx, func() time.Time { return t2 })
	sess.UpdateStatus(ctx2, types.SessionStatusCompleted)

	if sess.Status != types.SessionStatusCompleted {
		t.Errorf("Status = %v, want %v", sess.Status, types.SessionStatusCompleted)
	}

	if !sess.CreatedAt.Equal(t1) {
		t.Errorf("CreatedAt = %v, want %v (should not change)", sess.CreatedAt, t1)
	}

	if !sess.UpdatedAt.Equal(t2) {
		t.Errorf("UpdatedAt = %v, want %v", sess.UpdatedAt, t2)
	}
}
