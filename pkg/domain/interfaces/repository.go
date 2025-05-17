package interfaces

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type Repository interface {
	GetTicket(ctx context.Context, ticketID types.TicketID) (*ticket.Ticket, error)
	PutTicket(ctx context.Context, ticket ticket.Ticket) error
	PutTicketComment(ctx context.Context, comment ticket.Comment) error
	GetTicketComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error)

	BindAlertToTicket(ctx context.Context, alertID types.AlertID, ticketID types.TicketID) error
	UnbindAlertFromTicket(ctx context.Context, alertID types.AlertID) error

	PutAlert(ctx context.Context, alert alert.Alert) error
	GetAlert(ctx context.Context, alertID types.AlertID) (*alert.Alert, error)
	GetAlertByThread(ctx context.Context, thread slack.Thread) (*alert.Alert, error)
	SearchAlerts(ctx context.Context, path, op string, value any) (alert.Alerts, error)
	GetAlertWithoutTicket(ctx context.Context) (alert.Alerts, error)
	GetAlertsBySpan(ctx context.Context, begin, end time.Time) (alert.Alerts, error)
	BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) (alert.Alerts, error)
	FindSimilarAlerts(ctx context.Context, alert alert.Alert, limit int) (alert.Alerts, error)

	// For list management
	GetAlertList(ctx context.Context, listID types.AlertListID) (*alert.List, error)
	PutAlertList(ctx context.Context, list alert.List) error
	GetAlertListByThread(ctx context.Context, thread slack.Thread) (*alert.List, error)
	GetLatestAlertListInThread(ctx context.Context, thread slack.Thread) (*alert.List, error)

	// For session management
	GetSession(ctx context.Context, id types.SessionID) (*session.Session, error)
	GetSessionByThread(ctx context.Context, thread slack.Thread) (*session.Session, error)
	PutSession(ctx context.Context, session session.Session) error
	// For chat
	GetLatestHistory(ctx context.Context, sessionID types.SessionID) (*session.History, error)
	PutHistory(ctx context.Context, sessionID types.SessionID, history *session.History) error
}
