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
	userID := types.UserID("user-123")
	query := "investigate this alert"
	slackURL := "https://slack.com/archives/C123/p456"
	requestID := "test-request-123"

	// Set request ID in context
	ctx = request_id.With(ctx, requestID)

	// Set fixed time for testing
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	sess := session.NewSession(ctx, ticketID, userID, query, slackURL)

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

	if sess.UserID != userID {
		t.Errorf("UserID = %v, want %v", sess.UserID, userID)
	}

	if sess.Query != query {
		t.Errorf("Query = %v, want %v", sess.Query, query)
	}

	if sess.SlackURL != slackURL {
		t.Errorf("SlackURL = %v, want %v", sess.SlackURL, slackURL)
	}

	if sess.Intent != "" {
		t.Errorf("Intent = %v, want empty", sess.Intent)
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
	userID := types.UserID("user-123")

	sess := session.NewSession(ctx, ticketID, userID, "test query", "")

	if sess.RequestID != "(unknown)" {
		t.Errorf("RequestID = %v, want %v", sess.RequestID, "(unknown)")
	}
}

func TestUpdateStatus(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket-001")
	userID := types.UserID("user-123")

	// Create session at time T1
	t1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx1 := clock.With(ctx, func() time.Time { return t1 })
	sess := session.NewSession(ctx1, ticketID, userID, "test query", "")

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

func TestUpdateIntent(t *testing.T) {
	ctx := context.Background()
	ticketID := types.TicketID("test-ticket-001")
	userID := types.UserID("user-123")

	// Create session at time T1
	t1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx1 := clock.With(ctx, func() time.Time { return t1 })
	sess := session.NewSession(ctx1, ticketID, userID, "test query", "")

	// Update intent at time T2
	t2 := time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC)
	ctx2 := clock.With(ctx, func() time.Time { return t2 })
	intent := "Investigate suspicious network activity"
	sess.UpdateIntent(ctx2, intent)

	if sess.Intent != intent {
		t.Errorf("Intent = %v, want %v", sess.Intent, intent)
	}

	if !sess.CreatedAt.Equal(t1) {
		t.Errorf("CreatedAt = %v, want %v (should not change)", sess.CreatedAt, t1)
	}

	if !sess.UpdatedAt.Equal(t2) {
		t.Errorf("UpdatedAt = %v, want %v", sess.UpdatedAt, t2)
	}
}

// --- chat-session-redesign additions ---

func TestNewSessionV2_WithTicket_SetsRedesignFields(t *testing.T) {
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	ctx := clock.With(context.Background(), func() time.Time { return now })

	tid := types.TicketID("tid_1")
	sess := session.NewSessionV2(ctx,
		types.SessionID("sid_1"),
		session.SessionSourceSlack,
		&tid,
		&session.ChannelRef{},
		types.UserID("user-1"),
	)

	if sess.ID != "sid_1" {
		t.Errorf("ID = %v, want sid_1", sess.ID)
	}
	if sess.Source != session.SessionSourceSlack {
		t.Errorf("Source = %v", sess.Source)
	}
	if sess.TicketIDPtr == nil || *sess.TicketIDPtr != tid {
		t.Errorf("TicketIDPtr = %v, want %v", sess.TicketIDPtr, tid)
	}
	if sess.TicketID != tid {
		t.Errorf("TicketID legacy shadow = %v, want %v", sess.TicketID, tid)
	}
	if !sess.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v", sess.CreatedAt)
	}
	if !sess.LastActiveAt.Equal(now) {
		t.Errorf("LastActiveAt = %v", sess.LastActiveAt)
	}
	// Legacy fields should be zero.
	if sess.Status != "" {
		t.Errorf("Status = %v, want zero", sess.Status)
	}
	if sess.Query != "" {
		t.Errorf("Query = %v, want zero", sess.Query)
	}
}

func TestNewSessionV2_Ticketless_AllowsNil(t *testing.T) {
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	ctx := clock.With(context.Background(), func() time.Time { return now })

	sess := session.NewSessionV2(ctx,
		types.SessionID("sid_tl"),
		session.SessionSourceSlack,
		nil, // ticketless
		&session.ChannelRef{},
		types.UserID("user-1"),
	)
	if sess.TicketIDPtr != nil {
		t.Errorf("TicketIDPtr = %v, want nil", sess.TicketIDPtr)
	}
	if sess.TicketID != "" {
		t.Errorf("legacy TicketID = %q, want empty", sess.TicketID)
	}
}

func TestNewSessionV2_AutoGeneratesIDWhenEmpty(t *testing.T) {
	ctx := clock.With(context.Background(), func() time.Time { return time.Now() })
	sess := session.NewSessionV2(ctx, "", session.SessionSourceWeb, nil, nil, types.UserID("u"))
	if sess.ID == "" {
		t.Fatal("ID should be auto-generated when empty")
	}
}

func TestSession_TicketIDOrNil(t *testing.T) {
	tid := types.TicketID("tid_x")

	// Only legacy field set (simulates pre-redesign data).
	legacy := &session.Session{TicketID: tid}
	got := legacy.TicketIDOrNil()
	if got == nil || *got != tid {
		t.Errorf("legacy fallback = %v, want %v", got, tid)
	}

	// Both set; TicketIDPtr wins.
	newPtr := types.TicketID("tid_y")
	both := &session.Session{TicketID: tid, TicketIDPtr: &newPtr}
	got2 := both.TicketIDOrNil()
	if got2 == nil || *got2 != newPtr {
		t.Errorf("TicketIDPtr priority = %v, want %v", got2, newPtr)
	}

	// Neither set → nil.
	empty := &session.Session{}
	if empty.TicketIDOrNil() != nil {
		t.Error("empty session should return nil")
	}
}

func TestSession_TouchLastActive(t *testing.T) {
	t1 := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	ctx := clock.With(context.Background(), func() time.Time { return t1 })
	sess := session.NewSessionV2(ctx, "", session.SessionSourceWeb, nil, nil, types.UserID("u"))

	t2 := t1.Add(5 * time.Minute)
	ctx2 := clock.With(context.Background(), func() time.Time { return t2 })
	sess.TouchLastActive(ctx2)

	if !sess.LastActiveAt.Equal(t2) {
		t.Errorf("LastActiveAt = %v, want %v", sess.LastActiveAt, t2)
	}
}
