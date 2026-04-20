package chatnotifier

import (
	"context"
	"encoding/json"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// WebNotifier persists Messages and pushes a deterministic JSON event to
// the WebSocket Hub for clients viewing the owning Ticket.
type WebNotifier struct {
	repo      interfaces.Repository
	publisher interfaces.WebEventPublisher
	session   *session.Session
	turnID    *types.TurnID
	ticketID  *types.TicketID
}

// NewWebNotifier constructs a WebNotifier bound to sess + turnID. publisher
// must be non-nil; it is typically the hub bridge registered at server
// startup.
func NewWebNotifier(
	repo interfaces.Repository,
	publisher interfaces.WebEventPublisher,
	sess *session.Session,
	turnID *types.TurnID,
) *WebNotifier {
	return &WebNotifier{
		repo:      repo,
		publisher: publisher,
		session:   sess,
		turnID:    turnID,
		ticketID:  sess.TicketIDOrNil(),
	}
}

// webEvent is the stable JSON envelope pushed to WebSocket clients. The
// frontend discriminates on "event" and ignores fields it does not know.
type webEvent struct {
	Event     string      `json:"event"`
	SessionID string      `json:"session_id"`
	TurnID    *string     `json:"turn_id,omitempty"`
	Message   *webMessage `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type webMessage struct {
	ID        string          `json:"id"`
	SessionID string          `json:"session_id"`
	TurnID    *string         `json:"turn_id,omitempty"`
	Type      string          `json:"type"`
	Author    *session.Author `json:"author,omitempty"`
	Content   string          `json:"content"`
	CreatedAt time.Time       `json:"created_at"`
}

func toWebTurnID(t *types.TurnID) *string {
	if t == nil {
		return nil
	}
	s := string(*t)
	return &s
}

func (w *WebNotifier) persist(ctx context.Context, msgType session.MessageType, content string, author *session.Author) *session.Message {
	msg := session.NewMessageV2(ctx, w.session.ID, w.ticketID, w.turnID, msgType, content, author)
	if err := w.repo.PutSessionMessage(ctx, msg); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to persist Web session message",
			goerr.V("session_id", w.session.ID),
			goerr.V("type", msgType),
		))
	}
	return msg
}

func (w *WebNotifier) publish(ctx context.Context, msg *session.Message) {
	if w.ticketID == nil {
		// No Ticket means no Hub routing target. The Message is still
		// persisted; polling (GraphQL) will surface it.
		return
	}
	payload := webEvent{
		Event:     "session_message_added",
		SessionID: string(msg.SessionID),
		TurnID:    toWebTurnID(msg.TurnID),
		Timestamp: clock.Now(ctx),
		Message: &webMessage{
			ID:        string(msg.ID),
			SessionID: string(msg.SessionID),
			TurnID:    toWebTurnID(msg.TurnID),
			Type:      string(msg.Type),
			Author:    msg.Author,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to marshal web event"))
		return
	}
	if err := w.publisher.PublishToTicket(ctx, *w.ticketID, data); err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to publish web event",
			goerr.V("ticket_id", *w.ticketID),
		))
	}
}

func (w *WebNotifier) emit(ctx context.Context, msgType session.MessageType, content string, author *session.Author) error {
	msg := w.persist(ctx, msgType, content, author)
	w.publish(ctx, msg)
	return nil
}

func (w *WebNotifier) Notify(ctx context.Context, content string) error {
	return w.emit(ctx, session.MessageTypeResponse, content, nil)
}
func (w *WebNotifier) Trace(ctx context.Context, content string) error {
	return w.emit(ctx, session.MessageTypeTrace, content, nil)
}
func (w *WebNotifier) Warn(ctx context.Context, content string) error {
	return w.emit(ctx, session.MessageTypeWarning, content, nil)
}
func (w *WebNotifier) Plan(ctx context.Context, content string) error {
	return w.emit(ctx, session.MessageTypePlan, content, nil)
}
func (w *WebNotifier) NotifyUser(ctx context.Context, content string, author *session.Author) error {
	if author == nil {
		return goerr.New("NotifyUser requires author")
	}
	return w.emit(ctx, session.MessageTypeUser, content, author)
}
