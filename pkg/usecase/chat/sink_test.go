package chat_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository/memory"
	slackSvc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/usecase/chat"
	slackSDK "github.com/slack-go/slack"
)

// slackFailClient is a SlackClient implementation whose every method
// fails the test when invoked. It is installed into a Service and then
// a Service-level ChatSink resolution is performed for a non-Slack
// Session — the test passes only if the mock never fires.
func slackFailClient(t *testing.T) *mock.SlackClientMock {
	t.Helper()
	m := &mock.SlackClientMock{}
	m.AuthTestFunc = func() (*slackSDK.AuthTestResponse, error) {
		return &slackSDK.AuthTestResponse{
			UserID: "U_BOT", User: "warren", TeamID: "T", Team: "t", URL: "https://t.slack.com/",
		}, nil
	}
	m.GetTeamInfoFunc = func() (*slackSDK.TeamInfo, error) {
		return &slackSDK.TeamInfo{ID: "T", Name: "t", Domain: "t"}, nil
	}
	m.GetBotInfoContextFunc = func(context.Context, slackSDK.GetBotInfoParameters) (*slackSDK.Bot, error) {
		return &slackSDK.Bot{ID: "B_BOT", AppID: "A_APP", Name: "warren"}, nil
	}
	m.PostMessageContextFunc = func(context.Context, string, ...slackSDK.MsgOption) (string, string, error) {
		t.Fatal("Slack PostMessageContext must not be called for non-Slack Sessions")
		return "", "", nil
	}
	m.UpdateMessageContextFunc = func(context.Context, string, string, ...slackSDK.MsgOption) (string, string, string, error) {
		t.Fatal("Slack UpdateMessageContext must not be called for non-Slack Sessions")
		return "", "", "", nil
	}
	return m
}

// webChatCtx constructs a minimal ChatContext for a Web Session on a
// ticket that has a SlackThread attached. This is the exact shape the
// Session leak bug reproduced under: the ticket still references a
// Slack thread (origin), but the caller is on the Web UI.
func webChatCtx(t *testing.T) *chatModel.ChatContext {
	t.Helper()
	tid := types.NewTicketID()
	return &chatModel.ChatContext{
		Ticket: &ticket.Ticket{
			ID: tid,
			SlackThread: &slackModel.Thread{
				ChannelID: "C_ORIGIN",
				ThreadID:  "1700000000.000000",
			},
		},
		Session: &sessModel.Session{
			ID:        types.NewSessionID(),
			Source:    sessModel.SessionSourceWeb,
			TicketID:  tid,
			UserID:    "U_WEB_USER",
			CreatedAt: time.Now(),
		},
	}
}

func TestResolveSink_WebSession_IgnoresSlackService(t *testing.T) {
	// Slack client that fails the test on any call.
	client := slackFailClient(t)
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()

	repo := memory.New()
	ctx := context.Background()
	cc := webChatCtx(t)

	sink := chat.ResolveSink(cc, svc, repo)
	gt.NotNil(t, sink)

	// Every surface method must persist+publish through the Web path
	// and must not hit Slack. slackFailClient would fatal the test
	// if any Slack API were called.
	var events atomic.Int32
	cc.OnSessionEvent = func(string, *sessModel.Message) { events.Add(1) }

	gt.NoError(t, sink.PostComment(ctx, "hello"))
	gt.NoError(t, sink.PostContextBlock(ctx, "status"))
	gt.NoError(t, sink.PostSectionBlock(ctx, "section"))
	gt.NoError(t, sink.PostDivider(ctx))

	updater := sink.NewUpdatableMessage(ctx, "initial")
	updater(ctx, "update 1")
	updater(ctx, "update 2")

	gt.V(t, events.Load() > 0).Equal(true) // OnSessionEvent fired for Web
}

func TestResolveSink_CLISession_IgnoresSlackService(t *testing.T) {
	client := slackFailClient(t)
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	repo := memory.New()
	ctx := context.Background()

	cc := webChatCtx(t)
	cc.Session.Source = sessModel.SessionSourceCLI

	sink := chat.ResolveSink(cc, svc, repo)
	gt.NotNil(t, sink)

	// CLI path routes to msg.Notify/Trace — no Slack; slackFailClient
	// would fire t.Fatal if anything leaked.
	gt.NoError(t, sink.PostComment(ctx, "hello"))
	gt.NoError(t, sink.PostContextBlock(ctx, "status"))
	gt.NoError(t, sink.PostSectionBlock(ctx, "section"))
	gt.NoError(t, sink.PostDivider(ctx))

	updater := sink.NewUpdatableMessage(ctx, "initial")
	updater(ctx, "update 1")
}

func TestResolveSink_SlackSession_UsesSlackService(t *testing.T) {
	// Slack client that counts but does not fatal. A Slack Session
	// must route through the real Slack API — confirm the count > 0.
	var postCalls atomic.Int32
	var updateCalls atomic.Int32
	client := &mock.SlackClientMock{
		AuthTestFunc: func() (*slackSDK.AuthTestResponse, error) {
			return &slackSDK.AuthTestResponse{UserID: "U_BOT", TeamID: "T", URL: "https://t/"}, nil
		},
		GetTeamInfoFunc: func() (*slackSDK.TeamInfo, error) {
			return &slackSDK.TeamInfo{ID: "T"}, nil
		},
		GetBotInfoContextFunc: func(context.Context, slackSDK.GetBotInfoParameters) (*slackSDK.Bot, error) {
			return &slackSDK.Bot{ID: "B"}, nil
		},
		PostMessageContextFunc: func(context.Context, string, ...slackSDK.MsgOption) (string, string, error) {
			postCalls.Add(1)
			return "C", "1700000000.000001", nil
		},
		UpdateMessageContextFunc: func(context.Context, string, string, ...slackSDK.MsgOption) (string, string, string, error) {
			updateCalls.Add(1)
			return "", "", "", nil
		},
	}
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	repo := memory.New()
	ctx := context.Background()

	cc := webChatCtx(t)
	cc.Session.Source = sessModel.SessionSourceSlack

	sink := chat.ResolveSink(cc, svc, repo)
	gt.NotNil(t, sink)

	gt.NoError(t, sink.PostComment(ctx, "hello"))
	gt.NoError(t, sink.PostContextBlock(ctx, "status"))

	gt.V(t, postCalls.Load() > 0).Equal(true)
}

// TestWebProgressHandle_PresentHITL_EmitsEnvelope_NoSlack asserts that
// when aster/bluebell present a HITL request on a Web ProgressHandle,
// the hitl_request_pending callback fires and the Slack client is
// never touched.
func TestWebProgressHandle_PresentHITL_EmitsEnvelope_NoSlack(t *testing.T) {
	client := slackFailClient(t)
	svc, err := slackSvc.New(client, "C_DEFAULT")
	gt.NoError(t, err).Required()
	repo := memory.New()
	ctx := context.Background()

	cc := webChatCtx(t)

	var hitlKind atomic.Value // string
	var hitlReq atomic.Pointer[hitl.Request]
	cc.OnHITLEvent = func(kind string, req *hitl.Request, _ string) {
		hitlKind.Store(kind)
		hitlReq.Store(req)
	}

	h := chat.NewProgressHandle(ctx, cc, svc, repo, "🕐 Waiting...")
	gt.NotNil(t, h)

	req := &hitl.Request{
		ID:        types.HITLRequestID("req-1"),
		SessionID: cc.Session.ID,
		Type:      hitl.RequestTypeToolApproval,
		Payload:   hitl.NewToolApprovalPayload("web_fetch", map[string]any{"url": "https://x"}),
		Status:    hitl.StatusPending,
		UserID:    "U_WEB_USER",
	}
	gt.NoError(t, h.PresentHITL(ctx, req, "title", "U_WEB_USER"))

	gt.V(t, hitlKind.Load()).Equal("pending")
	loaded := hitlReq.Load()
	gt.NotNil(t, loaded)
	gt.V(t, loaded.ID).Equal(req.ID)
}

// TestCLIProgressHandle_PresentHITL_DefaultDeny asserts that the CLI
// transport blocks HITL tools rather than silently allowing them.
func TestCLIProgressHandle_PresentHITL_DefaultDeny(t *testing.T) {
	cc := webChatCtx(t)
	cc.Session.Source = sessModel.SessionSourceCLI
	repo := memory.New()

	h := chat.NewProgressHandle(context.Background(), cc, nil, repo, "🕐 Waiting...")
	gt.NotNil(t, h)

	req := &hitl.Request{
		ID:      types.HITLRequestID("req-cli"),
		Type:    hitl.RequestTypeToolApproval,
		Payload: hitl.NewToolApprovalPayload("web_fetch", nil),
	}
	err := h.PresentHITL(context.Background(), req, "t", "U")
	gt.Error(t, err)
}
