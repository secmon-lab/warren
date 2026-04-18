// slack_golden_test.go captures the existing Slack API call patterns used by
// the usecase layer as "golden" JSON fixtures. The Phase 0 regression
// protection spec (see .spec/chat-session-redesign/spec.md) requires that
// refactoring work in Phase 1-7 produce zero diffs against these fixtures.
//
// Layout:
//
//	testdata/slack_golden/<scenario>.json
//
// When running locally, use `-update-slack-golden` to rewrite all fixture
// files with the current observed output. Any diff in a subsequent run is a
// regression that must be investigated before merging.
package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	slackSvc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/service/slack/testutil"
	"github.com/secmon-lab/warren/pkg/usecase"
)

// TestSlackGolden_ThreadService_Reply exercises the basic Reply path used by
// msg.Notify handlers in the existing Slack chat pipeline. This is the
// simplest end-to-end path through ThreadService that hits a real Slack API
// method, and serves as the infrastructure smoke test for the golden
// framework.
func TestSlackGolden_ThreadService_Reply(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()

	// Discard the constructor's AuthTest / GetTeamInfo calls from the
	// captured stream; those are covered separately in the "construction"
	// golden case and including them in every scenario would create noise.
	rec.Reset()

	thread := svc.NewThread(slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
	})

	ctx := context.Background()
	thread.Reply(ctx, "*hello* from warren")

	testutil.AssertGolden(t, "thread_service/reply_simple.json", rec.CallsJSON())
}

// TestSlackGolden_ThreadService_PostContextBlock captures how the existing
// "status" context block is posted at the start of a chat session
// (pkg/usecase/chat/usecase.go:185).
func TestSlackGolden_ThreadService_PostContextBlock(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	thread := svc.NewThread(slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
	})

	ctx := context.Background()
	err = thread.PostContextBlock(ctx, "Investigating ...")
	gt.NoError(t, err)

	testutil.AssertGolden(t, "thread_service/post_context_block.json", rec.CallsJSON())
}

// TestSlackGolden_ThreadService_PostComment captures how a regular comment is
// posted (used by msg.Notify and by the chat final response path).
func TestSlackGolden_ThreadService_PostComment(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	thread := svc.NewThread(slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
	})

	ctx := context.Background()
	err = thread.PostComment(ctx, "final analysis result")
	gt.NoError(t, err)

	testutil.AssertGolden(t, "thread_service/post_comment.json", rec.CallsJSON())
}

// TestSlackGolden_ThreadService_UpdatableTraceChain captures the trace-message
// update chain: initial post followed by in-place updates. This is how the
// existing chat code (setupSlackMessageFuncs in pkg/usecase/chat/usecase.go)
// surfaces live trace output to the Slack thread.
func TestSlackGolden_ThreadService_UpdatableTraceChain(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	thread := svc.NewThread(slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
	})

	ctx := context.Background()
	updater := thread.NewUpdatableMessage(ctx, "starting ...")
	updater(ctx, "step 1 complete")
	updater(ctx, "step 2 complete")

	testutil.AssertGolden(t, "thread_service/updatable_trace_chain.json", rec.CallsJSON())
}

// TestSlackGolden_Service_Construction records the API calls performed during
// slackSvc.New construction so that future refactoring does not silently
// change startup behavior (AuthTest / GetTeamInfo ordering and arguments).
func TestSlackGolden_Service_Construction(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	_, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()

	testutil.AssertGolden(t, "service/construction.json", rec.CallsJSON())
}

// TestSlackGolden_HandleSlackMessage_UserMessageSaved captures the case where a
// regular Slack user posts into a thread that has an associated Ticket. The
// existing behavior is to save the message as a ticket.Comment WITHOUT
// invoking any Slack API (the thread is already displaying the message).
func TestSlackGolden_HandleSlackMessage_UserMessageSaved(t *testing.T) {
	ctx := context.Background()

	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	repo := repository.NewMemory()
	thread := slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
		TeamID:    "T_TEAM",
	}
	t1 := ticket.Ticket{
		ID:          types.TicketID("TCKT_FIXED_0001"),
		SlackThread: &thread,
		Status:      types.TicketStatusOpen,
	}
	gt.NoError(t, repo.PutTicket(ctx, t1)).Required()

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(svc),
	)

	msg := slackModel.NewTestMessage(
		thread.ChannelID,
		thread.ThreadID,
		thread.TeamID,
		"1700000001.000100",
		"U_USER",
		"additional context from the on-call",
	)
	gt.NoError(t, uc.HandleSlackMessage(ctx, msg))

	testutil.AssertGolden(t, "handle_slack_message/user_message_saved.json", rec.CallsJSON())
}

// TestSlackGolden_HandleSlackMessage_BotMessageSkipped captures the short-
// circuit path: when Slack delivers a message whose author is warren itself,
// we drop it before any repository or API work happens.
func TestSlackGolden_HandleSlackMessage_BotMessageSkipped(t *testing.T) {
	ctx := context.Background()

	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	repo := repository.NewMemory()
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(svc),
	)

	// U_WARREN_BOT is the bot user ID returned by the testutil AuthTest stub.
	msg := slackModel.NewTestMessage(
		"C_TICKET",
		"1700000000.000000",
		"T_TEAM",
		"1700000001.000100",
		"U_WARREN_BOT",
		"bot speaking to itself",
	)
	gt.NoError(t, uc.HandleSlackMessage(ctx, msg))

	testutil.AssertGolden(t, "handle_slack_message/bot_message_skipped.json", rec.CallsJSON())
}

// TestSlackGolden_ThreadService_PostFinding captures the PostFinding path
// invoked by the chat Finding-update tool (pkg/tool/base/ticket.go slackUpdate
// closure in pkg/usecase/chat.go:121-127).
func TestSlackGolden_ThreadService_PostFinding(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	thread := svc.NewThread(slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
	})

	finding := &ticket.Finding{
		Severity:       types.AlertSeverityHigh,
		Summary:        "Suspicious login from new geography",
		Reason:         "The source IP is in an unusual region and authentication succeeded on the first try.",
		Recommendation: "Force password reset and enable MFA.",
	}
	gt.NoError(t, thread.PostFinding(context.Background(), finding))

	testutil.AssertGolden(t, "thread_service/post_finding.json", rec.CallsJSON())
}

// TestSlackGolden_ThreadService_PostSessionActions captures the session
// actions block (Resolve / Edit buttons) that follows a chat session's final
// response (pkg/usecase/chat/usecase.go finishSession path).
func TestSlackGolden_ThreadService_PostSessionActions(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	thread := svc.NewThread(slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
	})

	ctx := context.Background()
	// Case 1: open ticket → both Resolve and Edit buttons
	gt.NoError(t, thread.PostSessionActions(ctx,
		types.TicketID("TCKT_FIXED_0001"),
		types.TicketStatusOpen,
		"https://warren.example.com/tickets/TCKT_FIXED_0001/sessions/SESS_FIXED_0001",
	))
	// Case 2: resolved ticket → only Edit button
	gt.NoError(t, thread.PostSessionActions(ctx,
		types.TicketID("TCKT_FIXED_0002"),
		types.TicketStatusResolved,
		"",
	))

	testutil.AssertGolden(t, "thread_service/post_session_actions.json", rec.CallsJSON())
}

// TestSlackGolden_ThreadService_PostResolveDetails captures the "ticket
// resolved" summary block posted after a user resolves a ticket via
// interactive modal (pkg/usecase/slack_itx_submit.go).
func TestSlackGolden_ThreadService_PostResolveDetails(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	thread := svc.NewThread(slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
	})

	t1 := &ticket.Ticket{
		ID:         types.TicketID("TCKT_FIXED_0001"),
		Status:     types.TicketStatusResolved,
		Conclusion: types.AlertConclusionTruePositive,
		Reason:     "Confirmed real attack, credentials rotated.",
	}
	gt.NoError(t, thread.PostResolveDetails(context.Background(), t1))

	testutil.AssertGolden(t, "thread_service/post_resolve_details.json", rec.CallsJSON())
}

// TestSlackGolden_ThreadService_PostLinkToTicket captures the "ticket created"
// link message posted after a new ticket is created from an alert thread.
func TestSlackGolden_ThreadService_PostLinkToTicket(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	thread := svc.NewThread(slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
	})

	gt.NoError(t, thread.PostLinkToTicket(context.Background(),
		"https://warren.example.com/tickets/TCKT_FIXED_0001",
		"Suspicious login pattern",
	))

	testutil.AssertGolden(t, "thread_service/post_link_to_ticket.json", rec.CallsJSON())
}

// TestSlackGolden_HandleSlackMessage_NoTicketForThread captures the case where
// Slack delivers a message for a thread that is not currently tracked by any
// Ticket. The handler logs and returns without calling Slack.
func TestSlackGolden_HandleSlackMessage_NoTicketForThread(t *testing.T) {
	ctx := context.Background()

	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	repo := repository.NewMemory()
	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(svc),
	)

	msg := slackModel.NewTestMessage(
		"C_UNKNOWN",
		"1700000000.999999",
		"T_TEAM",
		"1700000001.000100",
		"U_USER",
		"lone message with no ticket",
	)
	gt.NoError(t, uc.HandleSlackMessage(ctx, msg))

	testutil.AssertGolden(t, "handle_slack_message/no_ticket_for_thread.json", rec.CallsJSON())
}
