package base_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/tool/base"
)

// seedMessagesForSearch populates the memory repository with a mixed
// set of SessionMessages bound to the given ticket so each test can
// focus on assertion logic.
func seedMessagesForSearch(t *testing.T, repo *repository.Memory, ticketID types.TicketID) {
	t.Helper()
	ctx := context.Background()

	sid := types.SessionID("sid_search_test")
	gt.NoError(t, repo.PutSession(ctx, &sessModel.Session{
		ID:          sid,
		TicketIDPtr: &ticketID,
		Source:      sessModel.SessionSourceSlack,
	})).Required()

	texts := []struct {
		t       sessModel.MessageType
		content string
	}{
		{sessModel.MessageTypeUser, "please check VirusTotal for this hash"},
		{sessModel.MessageTypeTrace, "fetched VirusTotal report for hash abc"},
		{sessModel.MessageTypeResponse, "the hash is flagged by 12 vendors"},
		{sessModel.MessageTypeUser, "what about the WHOIS record?"},
	}
	for _, entry := range texts {
		m := sessModel.NewMessageV2(ctx, sid, &ticketID, nil, entry.t, entry.content, nil)
		gt.NoError(t, repo.PutSessionMessage(ctx, m)).Required()
	}
}

func TestWarren_SearchSessionMessages_ReturnsMatchingMessages(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tid := types.NewTicketID()
	seedMessagesForSearch(t, repo, tid)

	tool := base.New(repo, tid)
	res, err := tool.Run(ctx, "warren_search_session_messages", map[string]any{
		"query": "VirusTotal",
		"limit": int64(10),
	})
	gt.NoError(t, err)

	messages := res["messages"].([]map[string]any)
	gt.N(t, len(messages)).Equal(2)
	gt.V(t, res["query"]).Equal("VirusTotal")
	gt.V(t, res["ticket_id"]).Equal(string(tid))
}

func TestWarren_SearchSessionMessages_RejectsEmptyQuery(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tid := types.NewTicketID()
	seedMessagesForSearch(t, repo, tid)

	tool := base.New(repo, tid)
	_, err := tool.Run(ctx, "warren_search_session_messages", map[string]any{
		"query": "",
		"limit": int64(10),
	})
	gt.Error(t, err)
}

func TestWarren_SearchSessionMessages_AppliesDefaultLimit(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tid := types.NewTicketID()
	seedMessagesForSearch(t, repo, tid)

	tool := base.New(repo, tid)
	res, err := tool.Run(ctx, "warren_search_session_messages", map[string]any{
		"query": "hash",
		"limit": int64(0),
	})
	gt.NoError(t, err)
	gt.V(t, res["limit"]).Equal(base.DefaultCommentsLimit)
}
