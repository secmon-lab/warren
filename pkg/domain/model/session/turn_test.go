package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
)

func fixedClockCtx(base time.Time) context.Context {
	ctx := context.Background()
	ctx = clock.With(ctx, func() time.Time { return base })
	return ctx
}

func TestNewTurn_StartsRunningWithRequestID(t *testing.T) {
	base := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	ctx := fixedClockCtx(base)
	ctx = request_id.With(ctx, "req-xyz")

	sid := types.SessionID("ses_1")
	turn := session.NewTurn(ctx, sid)

	gt.V(t, turn.SessionID).Equal(sid)
	gt.V(t, turn.Status).Equal(session.TurnStatusRunning)
	gt.V(t, turn.RequestID).Equal("req-xyz")
	gt.V(t, turn.StartedAt.Equal(base)).Equal(true)
	gt.V(t, turn.EndedAt == nil).Equal(true)
	gt.V(t, string(turn.ID)).NotEqual("")
}

func TestTurn_Close_SetsEndedAtAndStatus(t *testing.T) {
	base := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	ctx := fixedClockCtx(base)
	turn := session.NewTurn(ctx, types.SessionID("ses_1"))

	later := base.Add(2 * time.Second)
	ctx2 := fixedClockCtx(later)
	turn.Close(ctx2, session.TurnStatusCompleted)

	gt.V(t, turn.Status).Equal(session.TurnStatusCompleted)
	gt.V(t, turn.EndedAt == nil).Equal(false)
	gt.V(t, turn.EndedAt.Equal(later)).Equal(true)
}

func TestTurn_Close_IsIdempotent(t *testing.T) {
	base := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	ctx := fixedClockCtx(base)
	turn := session.NewTurn(ctx, types.SessionID("ses_1"))

	first := base.Add(1 * time.Second)
	turn.Close(fixedClockCtx(first), session.TurnStatusCompleted)

	// Second Close should not overwrite the existing terminal state, so a
	// later panic-driven abort cleanup does not clobber a completed Turn.
	second := base.Add(2 * time.Second)
	turn.Close(fixedClockCtx(second), session.TurnStatusAborted)

	gt.V(t, turn.Status).Equal(session.TurnStatusCompleted)
	gt.V(t, turn.EndedAt.Equal(first)).Equal(true)
}

func TestTurn_Close_RejectsInvalidStatus(t *testing.T) {
	base := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	ctx := fixedClockCtx(base)
	turn := session.NewTurn(ctx, types.SessionID("ses_1"))

	turn.Close(ctx, session.TurnStatus("bogus"))
	gt.V(t, turn.Status).Equal(session.TurnStatusRunning)
	gt.V(t, turn.EndedAt == nil).Equal(true)

	// Passing running to Close is also rejected: Close only transitions to
	// terminal states.
	turn.Close(ctx, session.TurnStatusRunning)
	gt.V(t, turn.EndedAt == nil).Equal(true)
}

func TestTurn_UpdateIntent_OverwritesLatest(t *testing.T) {
	ctx := fixedClockCtx(time.Now())
	turn := session.NewTurn(ctx, types.SessionID("ses_1"))

	turn.UpdateIntent("initial")
	gt.V(t, turn.Intent).Equal("initial")

	turn.UpdateIntent("refined")
	gt.V(t, turn.Intent).Equal("refined")
}

func TestTurnStatus_Valid(t *testing.T) {
	gt.V(t, session.TurnStatusRunning.Valid()).Equal(true)
	gt.V(t, session.TurnStatusCompleted.Valid()).Equal(true)
	gt.V(t, session.TurnStatusAborted.Valid()).Equal(true)
	gt.V(t, session.TurnStatus("nope").Valid()).Equal(false)
	gt.V(t, session.TurnStatus("").Valid()).Equal(false)
}

func TestSessionSource_Valid(t *testing.T) {
	gt.V(t, session.SessionSourceSlack.Valid()).Equal(true)
	gt.V(t, session.SessionSourceWeb.Valid()).Equal(true)
	gt.V(t, session.SessionSourceCLI.Valid()).Equal(true)
	gt.V(t, session.SessionSource("foo").Valid()).Equal(false)
	gt.V(t, session.SessionSource("").Valid()).Equal(false)
}

func TestSessionLock_IsExpired(t *testing.T) {
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	active := &session.SessionLock{ExpiresAt: now.Add(time.Minute)}
	gt.V(t, active.IsExpired(now)).Equal(false)

	expired := &session.SessionLock{ExpiresAt: now}
	gt.V(t, expired.IsExpired(now)).Equal(true)

	stale := &session.SessionLock{ExpiresAt: now.Add(-time.Second)}
	gt.V(t, stale.IsExpired(now)).Equal(true)

	var nilLock *session.SessionLock
	gt.V(t, nilLock.IsExpired(now)).Equal(true)
}
