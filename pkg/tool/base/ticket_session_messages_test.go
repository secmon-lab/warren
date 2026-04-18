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

func TestWarren_GetTicketSessionMessages_ReturnsEverythingWhenNoFilters(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tid := types.TicketID("tid_fx")

	// Seed a Slack Session with a user message and a response; also a
	// Web Session with its own user message.
	slackSess := &sessModel.Session{
		ID:          "sid_slack",
		TicketID:    tid,
		TicketIDPtr: ptrTicketID(tid),
		Source:      sessModel.SessionSourceSlack,
	}
	webSess := &sessModel.Session{
		ID:          "sid_web",
		TicketID:    tid,
		TicketIDPtr: ptrTicketID(tid),
		Source:      sessModel.SessionSourceWeb,
	}
	gt.NoError(t, repo.PutSession(ctx, slackSess))
	gt.NoError(t, repo.PutSession(ctx, webSess))

	gt.NoError(t, repo.PutSessionMessage(ctx, sessModel.NewMessageV2(ctx,
		slackSess.ID, ptrTicketID(tid), nil, sessModel.MessageTypeUser, "slack hello",
		&sessModel.Author{UserID: "u-slack", DisplayName: "Alice"})))
	gt.NoError(t, repo.PutSessionMessage(ctx, sessModel.NewMessageV2(ctx,
		slackSess.ID, ptrTicketID(tid), nil, sessModel.MessageTypeResponse, "slack answer", nil)))
	gt.NoError(t, repo.PutSessionMessage(ctx, sessModel.NewMessageV2(ctx,
		webSess.ID, ptrTicketID(tid), nil, sessModel.MessageTypeUser, "web hello",
		&sessModel.Author{UserID: "u-web", DisplayName: "Bob"})))

	w := base.New(repo, tid)
	got, err := w.Run(ctx, "warren_get_ticket_session_messages", map[string]any{})
	gt.NoError(t, err)

	messages, ok := got["messages"].([]map[string]any)
	gt.V(t, ok).Equal(true)
	gt.A(t, messages).Length(3)
}

func TestWarren_GetTicketSessionMessages_FilterBySource(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tid := types.TicketID("tid_fx2")

	slackSess := &sessModel.Session{ID: "sid_s", TicketID: tid, TicketIDPtr: ptrTicketID(tid), Source: sessModel.SessionSourceSlack}
	webSess := &sessModel.Session{ID: "sid_w", TicketID: tid, TicketIDPtr: ptrTicketID(tid), Source: sessModel.SessionSourceWeb}
	gt.NoError(t, repo.PutSession(ctx, slackSess))
	gt.NoError(t, repo.PutSession(ctx, webSess))

	gt.NoError(t, repo.PutSessionMessage(ctx, sessModel.NewMessageV2(ctx,
		slackSess.ID, ptrTicketID(tid), nil, sessModel.MessageTypeUser, "slack only",
		&sessModel.Author{UserID: "u"})))
	gt.NoError(t, repo.PutSessionMessage(ctx, sessModel.NewMessageV2(ctx,
		webSess.ID, ptrTicketID(tid), nil, sessModel.MessageTypeUser, "web only",
		&sessModel.Author{UserID: "u"})))

	w := base.New(repo, tid)

	got, err := w.Run(ctx, "warren_get_ticket_session_messages", map[string]any{"source": "slack"})
	gt.NoError(t, err)
	messages := got["messages"].([]map[string]any)
	gt.A(t, messages).Length(1)
	gt.V(t, messages[0]["content"]).Equal("slack only")
}

func TestWarren_GetTicketSessionMessages_FilterByType(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tid := types.TicketID("tid_fx3")

	sess := &sessModel.Session{ID: "sid_t", TicketID: tid, TicketIDPtr: ptrTicketID(tid), Source: sessModel.SessionSourceSlack}
	gt.NoError(t, repo.PutSession(ctx, sess))
	gt.NoError(t, repo.PutSessionMessage(ctx, sessModel.NewMessageV2(ctx,
		sess.ID, ptrTicketID(tid), nil, sessModel.MessageTypeUser, "q", &sessModel.Author{UserID: "u"})))
	gt.NoError(t, repo.PutSessionMessage(ctx, sessModel.NewMessageV2(ctx,
		sess.ID, ptrTicketID(tid), nil, sessModel.MessageTypeResponse, "a", nil)))

	w := base.New(repo, tid)
	got, err := w.Run(ctx, "warren_get_ticket_session_messages", map[string]any{"type": "response"})
	gt.NoError(t, err)
	messages := got["messages"].([]map[string]any)
	gt.A(t, messages).Length(1)
	gt.V(t, messages[0]["content"]).Equal("a")
	gt.V(t, messages[0]["type"]).Equal("response")
}

func TestWarren_GetTicketSessionMessages_RejectsInvalidSource(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	w := base.New(repo, types.TicketID("tid_x"))

	_, err := w.Run(ctx, "warren_get_ticket_session_messages", map[string]any{"source": "bogus"})
	gt.Error(t, err)
}

func TestWarren_GetTicketSessionMessages_IncludesAuthor(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	tid := types.TicketID("tid_fx4")

	sess := &sessModel.Session{ID: "sid_a", TicketID: tid, TicketIDPtr: ptrTicketID(tid), Source: sessModel.SessionSourceSlack}
	gt.NoError(t, repo.PutSession(ctx, sess))

	slackID := "USLACK"
	email := "alice@example.com"
	author := &sessModel.Author{
		UserID:      "u-1",
		DisplayName: "Alice",
		SlackUserID: &slackID,
		Email:       &email,
	}
	gt.NoError(t, repo.PutSessionMessage(ctx, sessModel.NewMessageV2(ctx,
		sess.ID, ptrTicketID(tid), nil, sessModel.MessageTypeUser, "hi", author)))

	w := base.New(repo, tid)
	got, err := w.Run(ctx, "warren_get_ticket_session_messages", map[string]any{})
	gt.NoError(t, err)
	messages := got["messages"].([]map[string]any)
	gt.A(t, messages).Length(1)

	authorMap, ok := messages[0]["author"].(map[string]any)
	gt.V(t, ok).Equal(true)
	gt.V(t, authorMap["user_id"]).Equal("u-1")
	gt.V(t, authorMap["display_name"]).Equal("Alice")
	gt.V(t, authorMap["slack_user_id"]).Equal("USLACK")
	gt.V(t, authorMap["email"]).Equal("alice@example.com")
}

func ptrTicketID(t types.TicketID) *types.TicketID { return &t }
