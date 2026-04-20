package chat_test

import (
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	adapter "github.com/secmon-lab/warren/pkg/adapter/storage"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/usecase/chat"
)

// chat-session-redesign Phase 7 (confinement): the legacy LoadHistory /
// SaveHistory surface was removed together with the ticket.History
// records they relied on. Only the Session-scoped helpers remain.

func TestLoadSessionHistory_ReturnsPersistedHistory(t *testing.T) {
	ctx := t.Context()
	mockStorage := adapter.NewMock()
	svc := storage.New(mockStorage)
	sid := types.SessionID("sid-load")

	h := &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}, {Role: gollem.RoleAssistant}},
	}
	gt.NoError(t, svc.PutSessionHistory(ctx, sid, h))

	loaded, err := chat.LoadSessionHistory(ctx, sid, svc)
	gt.NoError(t, err)
	gt.V(t, loaded).NotNil()
	gt.V(t, loaded.ToCount()).Equal(2)
}

func TestLoadSessionHistory_MissingReturnsNil(t *testing.T) {
	ctx := t.Context()
	mockStorage := adapter.NewMock()
	svc := storage.New(mockStorage)

	loaded, err := chat.LoadSessionHistory(ctx, types.SessionID("sid-missing"), svc)
	gt.NoError(t, err)
	gt.V(t, loaded).Nil()
}

func TestSaveSessionHistory_RejectsNilHistory(t *testing.T) {
	ctx := t.Context()
	mockStorage := adapter.NewMock()
	svc := storage.New(mockStorage)

	err := chat.SaveSessionHistory(ctx, types.SessionID("sid"), svc, nil)
	gt.Error(t, err)
}
