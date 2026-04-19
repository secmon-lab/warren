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
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	hitlModel "github.com/secmon-lab/warren/pkg/domain/model/hitl"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/notifier/chatnotifier"
	slackSvc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/service/slack/testutil"
	"github.com/secmon-lab/warren/pkg/usecase"
)

// updateSlackGoldenFlag controls whether this file's snapshot assertions
// rewrite the fixture instead of comparing. The flag is declared here (not
// in a shared util) so the fixture-management concern lives with the tests
// that actually use it.
//
// Run `go test ./pkg/usecase -run TestSlackGolden -update-slack-golden` to
// regenerate all fixtures after an intentional Slack behavior change.
var updateSlackGoldenFlag = flag.Bool(
	"update-slack-golden",
	false,
	"rewrite Slack snapshot fixtures under testdata/slack_golden with the current recorder output",
)

// assertRecordedCallsMatchSnapshot compares the recorded Slack API call
// stream against the JSON fixture at path (relative to the package
// directory). When -update-slack-golden is set, or when the fixture
// does not yet exist, the fixture is written from `got` instead; in
// the latter case the test fails once to force review.
func assertRecordedCallsMatchSnapshot(t *testing.T, path string, got []byte) {
	t.Helper()

	if *updateSlackGoldenFlag {
		writeSlackSnapshot(t, path, got)
		return
	}

	want, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		writeSlackSnapshot(t, path, got)
		t.Fatalf("slack snapshot %s did not exist; wrote initial contents (review and rerun)", path)
	}
	gt.NoError(t, err).Required()

	if !bytes.Equal(bytes.TrimRight(want, "\n"), bytes.TrimRight(got, "\n")) {
		t.Errorf("slack snapshot mismatch: %s\n--- want\n%s\n--- got\n%s\n(rerun with -update-slack-golden to accept)",
			path, string(want), string(got))
	}
}

func writeSlackSnapshot(t *testing.T, path string, data []byte) {
	t.Helper()
	gt.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755)).Required()
	gt.NoError(t, os.WriteFile(path, data, 0o644)).Required()
}

// slackSnapshotPath joins the testdata prefix so individual tests do not
// have to repeat it.
func slackSnapshotPath(name string) string {
	return filepath.Join("testdata", "slack_golden", name)
}

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

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("thread_service/reply_simple.json"), rec.CallsJSON())
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

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("thread_service/post_context_block.json"), rec.CallsJSON())
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

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("thread_service/post_comment.json"), rec.CallsJSON())
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

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("thread_service/updatable_trace_chain.json"), rec.CallsJSON())
}

// TestSlackGolden_ThreadService_TraceAccumulateThenOverflow captures the
// NewTraceMessage flow end-to-end: initial Post, two short appends that
// update the same message in place, then a large append that overflows
// the 1900-byte soft limit and triggers a fresh Post for the overflow
// chunk. This is the exact path SlackChatNotifier.Trace takes in
// production, and the fixture guards against accidental changes to:
//
//   - when NewTraceMessage emits its initial PostMessageContext (should
//     happen during construction when initialMessage != "")
//   - how appends become UpdateMessageContext calls bound to the same
//     timestamp
//   - at what byte threshold a new PostMessageContext is emitted
//   - the single-element context block layout on overflow
func TestSlackGolden_ThreadService_TraceAccumulateThenOverflow(t *testing.T) {
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
	tracer := thread.NewTraceMessage(ctx, "trace start")
	tracer(ctx, "step a")
	tracer(ctx, "step b")
	// Deterministic overflow payload: 1900 is the soft limit, so a
	// single 1950-byte string guarantees the overflow branch fires on
	// the next append regardless of prior accumulation.
	big := make([]byte, 1950)
	for i := range big {
		big[i] = 'x'
	}
	tracer(ctx, string(big))

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("thread_service/trace_accumulate_then_overflow.json"), rec.CallsJSON())
}

// TestSlackGolden_ChatNotifier_SlackTraceChain exercises
// SlackChatNotifier.Trace from the chat-session-redesign notifier path
// end-to-end: the Session+Turn are created, the notifier is bound to a
// real slackSvc.ThreadService, and two Trace() calls plus one Notify()
// are invoked. The captured API stream validates:
//
//   - first Trace() emits the initial PostMessageContext (trace context
//     block with the initial text)
//   - second Trace() emits an UpdateMessageContext binding to the same
//     timestamp with accumulated text
//   - Notify() emits a plain PostMessageContext for the final response
//     comment, NOT a context block update
//
// This is the canonical AI trace+response flow that ships to Slack.
func TestSlackGolden_ChatNotifier_SlackTraceChain(t *testing.T) {
	ctx := context.Background()

	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	thread := svc.NewThread(slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
	})

	tid := types.TicketID("TCKT_NOTIFIER_0001")
	sess := &sessModel.Session{
		ID:          types.SessionID("sid_notifier"),
		TicketIDPtr: &tid,
		Source:      sessModel.SessionSourceSlack,
	}
	repo := repository.NewMemory()

	n := chatnotifier.NewSlackChatNotifier(repo, thread, sess, nil)
	gt.NoError(t, n.Trace(ctx, "analyzing alert fingerprint"))
	gt.NoError(t, n.Trace(ctx, "checking VirusTotal"))
	gt.NoError(t, n.Notify(ctx, "final summary: alert is a true positive"))

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("chatnotifier/slack_trace_chain.json"), rec.CallsJSON())
}

// TestSlackGolden_Service_Construction records the API calls performed during
// slackSvc.New construction so that future refactoring does not silently
// change startup behavior (AuthTest / GetTeamInfo ordering and arguments).
func TestSlackGolden_Service_Construction(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	_, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("service/construction.json"), rec.CallsJSON())
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

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("handle_slack_message/user_message_saved.json"), rec.CallsJSON())
}

// TestSlackGolden_HandleSlackMessage_DedupesOnRetry asserts that Slack
// retries (Events API re-deliveries) and `message_changed` follow-ups
// collapse onto a single SessionMessage row instead of duplicating
// the user's input in the Conversation timeline. This reproduces the
// bug screenshot where the same mention appeared twice.
//
// The test drives HandleSlackMessage twice with the same Slack ts and
// asserts the ticket's SessionMessages contain exactly one type=user
// row with that content.
func TestSlackGolden_HandleSlackMessage_DedupesOnRetry(t *testing.T) {
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
		ID:          types.TicketID("TCKT_FIXED_0002"),
		SlackThread: &thread,
		Status:      types.TicketStatusOpen,
	}
	gt.NoError(t, repo.PutTicket(ctx, t1)).Required()

	uc := usecase.New(
		usecase.WithRepository(repo),
		usecase.WithSlackService(svc),
	)

	sameTs := "1700000001.000200"
	sameContent := "<@U08A3TTRENS> is this related?"

	// First delivery (original `message` event).
	gt.NoError(t, uc.HandleSlackMessage(ctx,
		slackModel.NewTestMessage(thread.ChannelID, thread.ThreadID, thread.TeamID,
			sameTs, "U_USER", sameContent),
	))
	// Second delivery (retry OR `message_changed` follow-up). The
	// deterministic MessageID derived from (session_id, slack_ts)
	// makes the write idempotent so we do not grow a second row.
	gt.NoError(t, uc.HandleSlackMessage(ctx,
		slackModel.NewTestMessage(thread.ChannelID, thread.ThreadID, thread.TeamID,
			sameTs, "U_USER", sameContent),
	))

	slackSource := sessModel.SessionSourceSlack
	userType := sessModel.MessageTypeUser
	msgs, err := repo.GetTicketSessionMessages(ctx, t1.ID, &slackSource, &userType, 0, 0)
	gt.NoError(t, err).Required()

	matching := 0
	for _, m := range msgs {
		if m.Content == sameContent {
			matching++
		}
	}
	gt.V(t, matching).Equal(1)
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

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("handle_slack_message/bot_message_skipped.json"), rec.CallsJSON())
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

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("thread_service/post_finding.json"), rec.CallsJSON())
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

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("thread_service/post_session_actions.json"), rec.CallsJSON())
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

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("thread_service/post_resolve_details.json"), rec.CallsJSON())
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

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("thread_service/post_link_to_ticket.json"), rec.CallsJSON())
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

	assertRecordedCallsMatchSnapshot(t, slackSnapshotPath("handle_slack_message/no_ticket_for_thread.json"), rec.CallsJSON())
}

// ---------------------------------------------------------------------------
// HITL presenter golden fixtures.
//
// The chat-session-redesign work refactors HITL rendering so that Web/CLI
// sessions never touch Slack. Slack's presenter path must stay byte-identical
// to the pre-refactor behavior — Slack is the only HITL-capable transport
// today, and any drift in block structure breaks live interactive approvals.
//
// These fixtures capture the exact Slack API calls produced by:
//
//   - UpdatableBlockMessage construction (initial PostMessageContext)
//   - HITLPresenter.Present for a tool_approval request (UpdateMessageContext
//     carrying the approval blocks built by BuildToolApprovalBlocks)
//   - QuestionPresenter.Present for a question request (UpdateMessageContext
//     carrying the question blocks built by BuildQuestionBlocks)
//
// If the refactor changes any of these bytes the tests fail and
// -update-slack-golden must be used consciously to accept the change.
// ---------------------------------------------------------------------------

// TestSlackGolden_HITL_ToolApprovalPresenter freezes the Slack wire format
// for a tool approval HITL request: the initial UpdatableBlockMessage post
// plus the Present() update with approval buttons.
func TestSlackGolden_HITL_ToolApprovalPresenter(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	thread := svc.NewThread(slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
	}).(*slackSvc.ThreadService)

	ctx := context.Background()
	ubm := thread.NewUpdatableBlockMessage(ctx, "⏳ *[Fetch evidence]*\n\nWaiting...")

	presenter := slackSvc.NewHITLPresenter(ubm, "Fetch evidence", "U12345")

	req := &hitlModel.Request{
		ID:        types.HITLRequestID("HITL_FIXED_0001"),
		SessionID: types.SessionID("SSN_FIXED_0001"),
		Type:      hitlModel.RequestTypeToolApproval,
		Payload: hitlModel.NewToolApprovalPayload("web_fetch", map[string]any{
			"url": "https://example.com/evidence",
		}),
		Status: hitlModel.StatusPending,
		UserID: "U12345",
	}

	gt.NoError(t, presenter.Present(ctx, req)).Required()

	assertRecordedCallsMatchSnapshot(t,
		slackSnapshotPath("hitl/tool_approval_presenter.json"),
		rec.CallsJSON(),
	)
}

// TestSlackGolden_HITL_QuestionPresenter freezes the Slack wire format for a
// question HITL request: the initial UpdatableBlockMessage post plus the
// Present() update with radio-button options and submit button.
func TestSlackGolden_HITL_QuestionPresenter(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	rec.Reset()

	thread := svc.NewThread(slackModel.Thread{
		ChannelID: "C_TICKET",
		ThreadID:  "1700000000.000000",
	}).(*slackSvc.ThreadService)

	ctx := context.Background()
	ubm := thread.NewUpdatableBlockMessage(ctx, "❓ *Question*\n\nIs 10.0.0.5 an internal IP?")

	presenter := slackSvc.NewQuestionPresenter(ubm, "Correlating ...", "U12345")

	req := &hitlModel.Request{
		ID:        types.HITLRequestID("HITL_FIXED_0002"),
		SessionID: types.SessionID("SSN_FIXED_0002"),
		Type:      hitlModel.RequestTypeQuestion,
		Payload: hitlModel.NewQuestionPayload(
			"Is 10.0.0.5 an internal IP?",
			[]string{"Yes, VPN GW", "No", "None of the above"},
		),
		Status: hitlModel.StatusPending,
		UserID: "U12345",
	}

	gt.NoError(t, presenter.Present(ctx, req)).Required()

	assertRecordedCallsMatchSnapshot(t,
		slackSnapshotPath("hitl/question_presenter.json"),
		rec.CallsJSON(),
	)
}
