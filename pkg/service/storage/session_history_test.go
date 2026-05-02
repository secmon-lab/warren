package storage_test

import (
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	adapterStorage "github.com/secmon-lab/warren/pkg/adapter/storage"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
)

func TestPutAndGetSessionHistory_RoundTrip(t *testing.T) {
	ctx := t.Context()
	client := adapterStorage.NewMemoryClient()
	svc := storage.New(client)

	sid := types.SessionID("sid_round_trip")
	h := &gollem.History{Version: gollem.HistoryVersion}

	gt.NoError(t, svc.PutSessionHistory(ctx, sid, h))

	got, err := svc.GetSessionHistory(ctx, sid)
	gt.NoError(t, err)
	gt.V(t, got == nil).Equal(false)
	gt.V(t, got.Version).Equal(h.Version)
}

func TestGetSessionHistory_Missing(t *testing.T) {
	ctx := t.Context()
	client := adapterStorage.NewMemoryClient()
	svc := storage.New(client)

	got, err := svc.GetSessionHistory(ctx, types.SessionID("sid_missing"))
	gt.NoError(t, err)
	gt.V(t, got == nil).Equal(true)
}

func TestPutSessionHistory_Overwrites(t *testing.T) {
	ctx := t.Context()
	client := adapterStorage.NewMemoryClient()
	svc := storage.New(client)
	sid := types.SessionID("sid_over")

	h1 := &gollem.History{Version: gollem.HistoryVersion, Messages: []gollem.Message{{Role: gollem.RoleUser}}}
	h2 := &gollem.History{Version: gollem.HistoryVersion, Messages: []gollem.Message{{Role: gollem.RoleUser}, {Role: gollem.RoleAssistant}}}
	gt.NoError(t, svc.PutSessionHistory(ctx, sid, h1))
	gt.NoError(t, svc.PutSessionHistory(ctx, sid, h2))

	got, err := svc.GetSessionHistory(ctx, sid)
	gt.NoError(t, err)
	gt.V(t, got == nil).Equal(false)
	gt.V(t, got.ToCount()).Equal(2)
}

func TestGetSessionHistory_IsolatedPerSession(t *testing.T) {
	ctx := t.Context()
	client := adapterStorage.NewMemoryClient()
	svc := storage.New(client)

	hA := &gollem.History{Version: gollem.HistoryVersion, Messages: []gollem.Message{{Role: gollem.RoleUser}}}
	hB := &gollem.History{Version: gollem.HistoryVersion, Messages: []gollem.Message{
		{Role: gollem.RoleUser}, {Role: gollem.RoleAssistant}, {Role: gollem.RoleUser},
	}}
	gt.NoError(t, svc.PutSessionHistory(ctx, "sid_a", hA))
	gt.NoError(t, svc.PutSessionHistory(ctx, "sid_b", hB))

	a, err := svc.GetSessionHistory(ctx, "sid_a")
	gt.NoError(t, err)
	gt.V(t, a == nil).Equal(false)
	gt.V(t, a.ToCount()).Equal(1)

	b, err := svc.GetSessionHistory(ctx, "sid_b")
	gt.NoError(t, err)
	gt.V(t, b == nil).Equal(false)
	gt.V(t, b.ToCount()).Equal(3)
}
