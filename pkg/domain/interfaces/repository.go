package interfaces

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type Repository interface {
	GetTicket(ctx context.Context, ticketID types.TicketID) (*ticket.Ticket, error)
	BatchGetTickets(ctx context.Context, ticketIDs []types.TicketID) ([]*ticket.Ticket, error)
	PutTicket(ctx context.Context, ticket ticket.Ticket) error
	BatchUpdateTicketsStatus(ctx context.Context, ticketIDs []types.TicketID, status types.TicketStatus) error
	GetTicketByThread(ctx context.Context, thread slack.Thread) (*ticket.Ticket, error)
	FindNearestTickets(ctx context.Context, embedding []float32, limit int) ([]*ticket.Ticket, error)
	FindNearestTicketsWithSpan(ctx context.Context, embedding []float32, begin, end time.Time, limit int) ([]*ticket.Ticket, error)
	GetTicketsByStatus(ctx context.Context, statuses []types.TicketStatus, offset, limit int) ([]*ticket.Ticket, error)
	CountTicketsByStatus(ctx context.Context, statuses []types.TicketStatus) (int, error)
	GetTicketsBySpan(ctx context.Context, begin, end time.Time) ([]*ticket.Ticket, error)
	GetTicketsByStatusAndSpan(ctx context.Context, status types.TicketStatus, begin, end time.Time) ([]*ticket.Ticket, error)

	// For comment management
	PutTicketComment(ctx context.Context, comment ticket.Comment) error
	GetTicketComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error)
	GetTicketCommentsPaginated(ctx context.Context, ticketID types.TicketID, offset, limit int) ([]ticket.Comment, error)
	CountTicketComments(ctx context.Context, ticketID types.TicketID) (int, error)
	GetTicketUnpromptedComments(ctx context.Context, ticketID types.TicketID) ([]ticket.Comment, error)
	PutTicketCommentsPrompted(ctx context.Context, ticketID types.TicketID, commentIDs []types.CommentID) error

	BindAlertsToTicket(ctx context.Context, alertIDs []types.AlertID, ticketID types.TicketID) error
	UnbindAlertFromTicket(ctx context.Context, alertID types.AlertID) error

	PutAlert(ctx context.Context, alert alert.Alert) error
	BatchPutAlerts(ctx context.Context, alerts alert.Alerts) error
	GetAlert(ctx context.Context, alertID types.AlertID) (*alert.Alert, error)
	GetLatestAlertByThread(ctx context.Context, thread slack.Thread) (*alert.Alert, error)
	SearchAlerts(ctx context.Context, path, op string, value any, limit int) (alert.Alerts, error)
	GetAlertWithoutTicket(ctx context.Context, offset, limit int) (alert.Alerts, error)
	CountAlertsWithoutTicket(ctx context.Context) (int, error)
	GetAlertsBySpan(ctx context.Context, begin, end time.Time) (alert.Alerts, error)
	BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) (alert.Alerts, error)
	FindNearestAlerts(ctx context.Context, embedding []float32, limit int) (alert.Alerts, error)

	GetLatestHistory(ctx context.Context, ticketID types.TicketID) (*ticket.History, error)
	PutHistory(ctx context.Context, ticketID types.TicketID, history *ticket.History) error

	// For list management
	GetAlertList(ctx context.Context, listID types.AlertListID) (*alert.List, error)
	PutAlertList(ctx context.Context, list *alert.List) error
	GetAlertListByThread(ctx context.Context, thread slack.Thread) (*alert.List, error)
	GetLatestAlertListInThread(ctx context.Context, thread slack.Thread) (*alert.List, error)
	GetAlertListsInThread(ctx context.Context, thread slack.Thread) ([]*alert.List, error)

	GetAlertWithoutEmbedding(ctx context.Context) (alert.Alerts, error)
	GetAlertsWithInvalidEmbedding(ctx context.Context) (alert.Alerts, error)
	GetTicketsWithInvalidEmbedding(ctx context.Context) ([]*ticket.Ticket, error)

	// For authentication management
	PutToken(ctx context.Context, token *auth.Token) error
	GetToken(ctx context.Context, tokenID auth.TokenID) (*auth.Token, error)
	DeleteToken(ctx context.Context, tokenID auth.TokenID) error

	// For activity management
	PutActivity(ctx context.Context, activity *activity.Activity) error
	GetActivities(ctx context.Context, offset, limit int) ([]*activity.Activity, error)
	CountActivities(ctx context.Context) (int, error)

	// For tag management
	ListTags(ctx context.Context) ([]*tag.Metadata, error)
	CreateTag(ctx context.Context, tag *tag.Metadata) error
	DeleteTag(ctx context.Context, name string) error
	GetTag(ctx context.Context, name string) (*tag.Metadata, error)
	RemoveTagFromAllAlerts(ctx context.Context, name string) error
	RemoveTagFromAllTickets(ctx context.Context, name string) error
}
