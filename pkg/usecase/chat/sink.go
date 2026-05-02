package chat

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	hitlService "github.com/secmon-lab/warren/pkg/service/hitl"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
)

// ChatSink is the narrow output surface aster/bluebell use to emit
// non-HITL progress to the user. Implementations route calls to the
// appropriate transport for the active Session.Source:
//
//   - Slack: interfaces.SlackThreadService already satisfies ChatSink
//     via structural typing; no wrapper is needed.
//   - Web: persists each call as a session.Message and publishes the
//     matching WebSocket Envelope so the Conversation UI receives it.
//   - CLI: routes through the legacy msg.Notify / msg.Trace handlers
//     already installed on the context, so stdout display is preserved.
//
// The design is deliberately transport-agnostic: Slack-specific block
// structures never cross this boundary. HITL rendering lives in
// QuestionTarget / ApprovalTarget below and their transport-specific
// Presenter implementations, not here.
type ChatSink interface {
	// PostComment posts a final-form message (Slack: regular comment,
	// Web: session_message type=response). Used by the chat pipeline
	// for planner messages, final responses, and warnings-as-comments.
	PostComment(ctx context.Context, text string) error

	// PostContextBlock posts an ephemeral status line (Slack: context
	// block, Web: session_message type=trace). Used for "Investigating",
	// reflection completion, etc.
	PostContextBlock(ctx context.Context, text string) error

	// PostSectionBlock posts a prominent block (Slack: section block,
	// Web: session_message type=response). Used for per-task completion
	// summaries.
	PostSectionBlock(ctx context.Context, text string) error

	// PostDivider posts a visual separator (Slack: divider block,
	// Web: session_message type=trace with a divider marker).
	PostDivider(ctx context.Context) error

	// NewUpdatableMessage returns a closure that updates a single
	// message in-place. The initial text is posted immediately.
	// Used by msg.Trace routing for the Slack updatable trace block
	// and by Web for a collapsible progress entry.
	NewUpdatableMessage(ctx context.Context, initial string) func(ctx context.Context, text string)
}

// ProgressHandle is a single live-updating display row attached to a
// task (aster/bluebell setupTaskMessageRouting). It is the unified
// surface for:
//
//   - Progress text updates during task execution.
//   - In-place HITL prompts (tool approval / question) that replace the
//     progress display with interactive UI, then yield it back to
//     progress once the user has responded.
//
// Transport-specific layouts are chosen inside each implementation:
// Slack builds []slack.Block via BuildToolApprovalBlocks and calls
// UpdatableBlockMessage.UpdateBlocks; Web emits a `hitl_request_pending`
// envelope; CLI blocks execution (default-deny).
type ProgressHandle interface {
	// UpdateText replaces the display row with plain text.
	UpdateText(ctx context.Context, text string)

	// PresentHITL renders a HITL request on this handle.
	// taskTitle/userID are passed through for transport-specific
	// labelling (Slack includes them in block headers).
	PresentHITL(ctx context.Context, req *hitl.Request, taskTitle, userID string) error
}

// MessageBound is an optional interface that ProgressHandle impls may
// satisfy to report the underlying SessionMessage ID backing the
// display row. Used by the WebSocket handler to bind a HITL prompt to
// an existing progress message so the UI can render approval UI in
// place. Slack / CLI impls return "".
type MessageBound interface {
	ProgressMessageID() string
}

// ProgressMessageID returns the SessionMessage ID for h when h
// implements MessageBound, or "" otherwise.
func ProgressMessageID(h ProgressHandle) string {
	if mb, ok := h.(MessageBound); ok {
		return mb.ProgressMessageID()
	}
	return ""
}

// ProgressHandleAsPresenter adapts a ProgressHandle to hitlService.Presenter.
// Each Present() call routes through the handle's PresentHITL.
type progressHandleAsPresenter struct {
	h         ProgressHandle
	taskTitle string
	userID    string
}

// NewProgressHandlePresenter returns a hitl.Presenter that renders on
// the given ProgressHandle.
func NewProgressHandlePresenter(h ProgressHandle, taskTitle, userID string) hitlService.Presenter {
	if h == nil {
		return nil
	}
	return &progressHandleAsPresenter{h: h, taskTitle: taskTitle, userID: userID}
}

func (p *progressHandleAsPresenter) Present(ctx context.Context, req *hitl.Request) error {
	return p.h.PresentHITL(ctx, req, p.taskTitle, p.userID)
}

// ResolveSink returns the ChatSink appropriate for the Session bound to
// chatCtx. Returns nil when no transport can deliver output (e.g. a
// Slack ticket with no slackService, or a ChatContext with no Session
// and no Slack thread).
//
// The explicit source-aware selection is the single policy gate that
// stops Web/CLI Sessions from leaking into Slack: on the Web path we
// never touch slackSvc even when the ticket happens to carry a
// SlackThread pointer. That pointer is structural ("this ticket
// originated from a Slack alert") and MUST NOT be conflated with
// "this chat should post to Slack".
func ResolveSink(chatCtx *chatModel.ChatContext, slackSvc *slackService.Service, repo interfaces.Repository) ChatSink {
	if chatCtx == nil {
		return nil
	}
	switch sessionSource(chatCtx) {
	case sessModel.SessionSourceWeb:
		return newWebSink(chatCtx, repo)
	case sessModel.SessionSourceCLI:
		return newCLISink(chatCtx)
	case sessModel.SessionSourceSlack:
		return slackSink(chatCtx, slackSvc)
	default:
		// No Session — legacy pre-redesign call path. Treat as Slack
		// for backward compatibility so existing Slack chats keep
		// working during the migration.
		return slackSink(chatCtx, slackSvc)
	}
}

// NewProgressHandle constructs a ProgressHandle for a task-specific
// live-updating display. Returns nil (and a no-op mark function) when
// no transport is configured for the active Session.
//
// initialText is posted immediately. chatCtx drives both the transport
// selection and the persistence binding (for Web: the SessionMessage
// is created under chatCtx.Session).
func NewProgressHandle(ctx context.Context, chatCtx *chatModel.ChatContext, slackSvc *slackService.Service, repo interfaces.Repository, initialText string) ProgressHandle {
	if chatCtx == nil {
		return nil
	}
	switch sessionSource(chatCtx) {
	case sessModel.SessionSourceWeb:
		return newWebProgressHandle(ctx, chatCtx, repo, initialText)
	case sessModel.SessionSourceCLI:
		return newCLIProgressHandle(ctx, initialText)
	case sessModel.SessionSourceSlack:
		return newSlackProgressHandle(ctx, chatCtx, slackSvc, initialText)
	default:
		return newSlackProgressHandle(ctx, chatCtx, slackSvc, initialText)
	}
}

// sessionSource returns the active session source, or empty if no
// Session is attached to chatCtx.
func sessionSource(chatCtx *chatModel.ChatContext) sessModel.SessionSource {
	if chatCtx == nil || chatCtx.Session == nil {
		return ""
	}
	return chatCtx.Session.Source
}
