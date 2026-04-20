package chat

import (
	"context"
	"fmt"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// slackSink returns a ChatSink backed by the real Slack ThreadService.
// Returns nil when the session is not Slack-capable (no slackSvc, or
// no SlackThread on the ticket).
//
// interfaces.SlackThreadService already exposes PostComment /
// PostContextBlock / PostSectionBlock / PostDivider /
// NewUpdatableMessage, so the real implementation satisfies ChatSink
// without a wrapper. A tiny adapter struct is used only so that
// ResolveSink can hand out nil cleanly when Slack is not available;
// the alternative (returning a typed-nil interface) is a common Go
// nil-trap.
func slackSink(chatCtx *chatModel.ChatContext, slackSvc *slackService.Service) ChatSink {
	if slackSvc == nil {
		return nil
	}
	if chatCtx == nil || chatCtx.Ticket == nil || chatCtx.Ticket.SlackThread == nil {
		return nil
	}
	thread := slackSvc.NewThread(*chatCtx.Ticket.SlackThread)
	return &slackSinkImpl{thread: thread}
}

type slackSinkImpl struct {
	thread interfaces.SlackThreadService
}

func (s *slackSinkImpl) PostComment(ctx context.Context, text string) error {
	return s.thread.PostComment(ctx, text)
}

func (s *slackSinkImpl) PostContextBlock(ctx context.Context, text string) error {
	return s.thread.PostContextBlock(ctx, text)
}

func (s *slackSinkImpl) PostSectionBlock(ctx context.Context, text string) error {
	return s.thread.PostSectionBlock(ctx, text)
}

func (s *slackSinkImpl) PostDivider(ctx context.Context) error {
	return s.thread.PostDivider(ctx)
}

func (s *slackSinkImpl) NewUpdatableMessage(ctx context.Context, initial string) func(ctx context.Context, text string) {
	return s.thread.NewUpdatableMessage(ctx, initial)
}

// slackProgressHandle backs a live-updating task display with a Slack
// UpdatableBlockMessage. HITL prompts replace the progress text with
// approval or question blocks built by the existing Slack helpers, so
// the wire format stays byte-identical to the pre-refactor path
// (captured by TestSlackGolden_HITL_*).
type slackProgressHandle struct {
	ubm *slackService.UpdatableBlockMessage
}

// newSlackProgressHandle posts the initial message and wraps the
// returned UpdatableBlockMessage. Returns nil when the session is
// not Slack-capable (no slackSvc, or no SlackThread on the ticket).
func newSlackProgressHandle(ctx context.Context, chatCtx *chatModel.ChatContext, slackSvc *slackService.Service, initialText string) ProgressHandle {
	if slackSvc == nil {
		return nil
	}
	if chatCtx == nil || chatCtx.Ticket == nil || chatCtx.Ticket.SlackThread == nil {
		return nil
	}
	threadSvc, ok := slackSvc.NewThread(*chatCtx.Ticket.SlackThread).(*slackService.ThreadService)
	if !ok {
		return nil
	}
	return &slackProgressHandle{
		ubm: threadSvc.NewUpdatableBlockMessage(ctx, initialText),
	}
}

func (h *slackProgressHandle) UpdateText(ctx context.Context, text string) {
	if h == nil || h.ubm == nil {
		return
	}
	h.ubm.UpdateText(ctx, text)
}

// PresentHITL renders the HITL request as Slack blocks on the shared
// UpdatableBlockMessage. The block builders live in pkg/service/slack
// and are guarded by golden fixtures — no rewrite here is allowed to
// alter their output.
func (h *slackProgressHandle) PresentHITL(ctx context.Context, req *hitl.Request, taskTitle, userID string) error {
	if h == nil || h.ubm == nil {
		return nil
	}
	switch req.Type {
	case hitl.RequestTypeToolApproval:
		h.ubm.UpdateBlocks(ctx, slackService.BuildToolApprovalBlocks(taskTitle, userID, req))
	case hitl.RequestTypeQuestion:
		h.ubm.UpdateBlocks(ctx, slackService.BuildQuestionBlocks(taskTitle, userID, req))
	default:
		logging.From(ctx).Warn("unknown HITL request type", "type", req.Type, "id", req.ID)
		h.ubm.UpdateText(ctx, fmt.Sprintf("❓ %s", req.Type))
	}
	logging.From(ctx).Info("HITL request presented on Slack",
		"request_id", req.ID,
		"task_title", taskTitle,
		"type", req.Type,
	)
	return nil
}
