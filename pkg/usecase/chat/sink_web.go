package chat

import (
	"context"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// webSink persists every emitted item as a session.Message and
// publishes the matching Envelope via chatCtx.OnSessionEvent so the
// WebSocket client sees progress identical to what a Slack user sees
// in the thread (context blocks → trace rows, section blocks →
// response rows, divider → trace marker).
//
// No Slack API is ever touched. Callers are responsible for ensuring
// that webSink is only returned when chatCtx.Session.Source is
// SessionSourceWeb; ResolveSink enforces this gate.
type webSink struct {
	chatCtx *chatModel.ChatContext
	repo    interfaces.Repository
}

func newWebSink(chatCtx *chatModel.ChatContext, repo interfaces.Repository) ChatSink {
	if chatCtx == nil || chatCtx.Session == nil || repo == nil {
		return nil
	}
	return &webSink{chatCtx: chatCtx, repo: repo}
}

func (s *webSink) PostComment(ctx context.Context, text string) error {
	_, err := s.persist(ctx, sessModel.MessageTypeResponse, text)
	return err
}

func (s *webSink) PostContextBlock(ctx context.Context, text string) error {
	_, err := s.persist(ctx, sessModel.MessageTypeTrace, text)
	return err
}

func (s *webSink) PostSectionBlock(ctx context.Context, text string) error {
	_, err := s.persist(ctx, sessModel.MessageTypeResponse, text)
	return err
}

// PostDivider persists a divider marker as a trace row. Web UI renders
// "---" markers as a visual separator between exchanges the same way
// Slack renders divider blocks.
func (s *webSink) PostDivider(ctx context.Context) error {
	_, err := s.persist(ctx, sessModel.MessageTypeTrace, dividerMarker)
	return err
}

// NewUpdatableMessage returns a closure that updates a single persisted
// Message in-place by appending revisions. The initial content is
// persisted eagerly so the UI sees something immediately.
func (s *webSink) NewUpdatableMessage(ctx context.Context, initial string) func(ctx context.Context, text string) {
	m, err := s.persist(ctx, sessModel.MessageTypeTrace, initial)
	if err != nil {
		errutil.Handle(ctx, err)
		return func(context.Context, string) {}
	}
	var mu sync.Mutex
	current := m
	return func(ctx context.Context, text string) {
		mu.Lock()
		defer mu.Unlock()
		if current == nil {
			return
		}
		current.AppendRevision(ctx, text)
		if err := s.repo.PutSessionMessage(ctx, current); err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "failed to update session message", goerr.V("id", current.ID)))
			return
		}
		if s.chatCtx.OnSessionEvent != nil {
			s.chatCtx.OnSessionEvent(eventKindSessionMessageUpdated, current)
		}
	}
}

// persist creates a new Message, writes it, and publishes the
// added-event. Returns the persisted Message for callers that need to
// retain its ID (e.g. updatable messages).
func (s *webSink) persist(ctx context.Context, t sessModel.MessageType, content string) (*sessModel.Message, error) {
	var tidPtr *types.TicketID
	if s.chatCtx.Ticket != nil && s.chatCtx.Ticket.ID != "" {
		tid := s.chatCtx.Ticket.ID
		tidPtr = &tid
	}
	m := sessModel.NewMessageV2(ctx, s.chatCtx.Session.ID, tidPtr, s.chatCtx.CurrentTurnID, t, content, nil)
	if err := s.repo.PutSessionMessage(ctx, m); err != nil {
		return nil, goerr.Wrap(err, "failed to persist session message", goerr.V("type", t))
	}
	if s.chatCtx.OnSessionEvent != nil {
		s.chatCtx.OnSessionEvent(eventKindSessionMessageAdded, m)
	}
	return m, nil
}

// webProgressHandle persists a task's live display row as one Message
// and appends revisions on each UpdateText / PresentHITL call. The UI
// shows the latest content in the collapsed state and the full revision
// history when expanded, mirroring the Slack updatable-block UX.
type webProgressHandle struct {
	chatCtx *chatModel.ChatContext
	repo    interfaces.Repository
	mu      sync.Mutex
	msg     *sessModel.Message
}

func newWebProgressHandle(ctx context.Context, chatCtx *chatModel.ChatContext, repo interfaces.Repository, initialText string) ProgressHandle {
	if chatCtx == nil || chatCtx.Session == nil || repo == nil {
		return nil
	}
	h := &webProgressHandle{chatCtx: chatCtx, repo: repo}
	h.msg = h.createInitial(ctx, initialText)
	return h
}

func (h *webProgressHandle) createInitial(ctx context.Context, text string) *sessModel.Message {
	var tidPtr *types.TicketID
	if h.chatCtx.Ticket != nil && h.chatCtx.Ticket.ID != "" {
		tid := h.chatCtx.Ticket.ID
		tidPtr = &tid
	}
	m := sessModel.NewMessageV2(ctx, h.chatCtx.Session.ID, tidPtr, h.chatCtx.CurrentTurnID, sessModel.MessageTypeTrace, text, nil)
	if err := h.repo.PutSessionMessage(ctx, m); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to persist progress message"))
		return nil
	}
	if h.chatCtx.OnSessionEvent != nil {
		h.chatCtx.OnSessionEvent(eventKindSessionMessageAdded, m)
	}
	return m
}

func (h *webProgressHandle) UpdateText(ctx context.Context, text string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.msg == nil {
		return
	}
	h.msg.AppendRevision(ctx, text)
	if err := h.repo.PutSessionMessage(ctx, h.msg); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to update progress message", goerr.V("id", h.msg.ID)))
		return
	}
	if h.chatCtx.OnSessionEvent != nil {
		h.chatCtx.OnSessionEvent(eventKindSessionMessageUpdated, h.msg)
	}
}

// PresentHITL emits a hitl_request_pending envelope carrying the
// request payload so the Web UI can render approval/question UI in
// place of the current progress row. No Slack API is touched.
func (h *webProgressHandle) PresentHITL(ctx context.Context, req *hitl.Request, _ /* taskTitle */, _ /* userID */ string) error {
	if h == nil || h.chatCtx == nil {
		return nil
	}
	if h.chatCtx.OnHITLEvent == nil {
		return goerr.New("web HITL requires OnHITLEvent but none is configured",
			goerr.V("request_id", req.ID))
	}
	var msgID string
	h.mu.Lock()
	if h.msg != nil {
		msgID = string(h.msg.ID)
	}
	h.mu.Unlock()
	h.chatCtx.OnHITLEvent(hitlKindPending, req, msgID)
	return nil
}

// dividerMarker is the text content used to represent a divider in the
// Web timeline. Kept as a module-level constant so the UI and backend
// can agree on the marker.
const dividerMarker = "---"

// Envelope kind labels — kept as strings to avoid importing the
// pkg/controller/websocket package (domain layer must not depend on
// the controller layer). The websocket handler maps these strings to
// EventKind constants.
const (
	eventKindSessionMessageAdded   = "session_message_added"
	eventKindSessionMessageUpdated = "session_message_updated"
	hitlKindPending                = "pending"
	hitlKindResolved               = "resolved"
)
