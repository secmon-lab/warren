package usecase

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
)

// SlackNotifier defines the interface for Slack notification operations
type SlackNotifier interface {
	// IsEnabled returns whether Slack functionality is enabled
	IsEnabled() bool

	// Alert operations
	PostAlert(ctx context.Context, alert *alert.Alert) (SlackThreadService, error)
	
	// Thread operations
	NewThread(thread slack.Thread) SlackThreadService

	// Ticket operations
	PostTicket(ctx context.Context, ticket *ticket.Ticket, alerts alert.Alerts) (SlackThreadService, string, error)

	// User operations
	GetUserIcon(ctx context.Context, userID string) ([]byte, string, error)
	GetUserProfile(ctx context.Context, userID string) (string, error)

	// Utility operations
	IsBotUser(userID string) bool
	BotID() string
	DefaultChannelID() string
	ToMsgURL(channelID, threadID string) string

	// Modal operations
	ShowBindToTicketModal(ctx context.Context, callbackID slack.CallbackID, tickets []*ticket.Ticket, triggerID string, metadata string) error
	ShowResolveTicketModal(ctx context.Context, ticket *ticket.Ticket, triggerID string) error
	ShowSalvageModal(ctx context.Context, ticket *ticket.Ticket, unboundAlerts alert.Alerts, triggerID string) error
	UpdateSalvageModal(ctx context.Context, ticket *ticket.Ticket, unboundAlerts alert.Alerts, viewID string, threshold float64, keyword string) error
}

// SlackThreadService defines the interface for Slack thread operations
type SlackThreadService interface {
	// Thread information
	ChannelID() string
	ThreadID() string
	Entity() *slack.Thread

	// Posting operations
	PostAlert(ctx context.Context, alert *alert.Alert) error
	PostComment(ctx context.Context, comment string) error
	PostCommentWithMessageID(ctx context.Context, comment string) (string, error)
	PostTicket(ctx context.Context, ticket *ticket.Ticket, alerts alert.Alerts) (string, error)
	PostLinkToTicket(ctx context.Context, ticketURL, ticketTitle string) error
	PostFinding(ctx context.Context, finding *ticket.Finding) error

	// Update operations
	UpdateAlert(ctx context.Context, alert alert.Alert) error
	UpdateAlertList(ctx context.Context, list *alert.List, status string) error

	// List operations
	PostAlerts(ctx context.Context, alerts alert.Alerts) error
	PostAlertList(ctx context.Context, list *alert.List) (string, error)
	PostAlertLists(ctx context.Context, clusters []*alert.List) error
	PostTicketList(ctx context.Context, tickets []*ticket.Ticket) error

	// Interactive operations
	Reply(ctx context.Context, message string)
	NewStateFunc(ctx context.Context, message string) func(ctx context.Context, msg string)
	NewUpdatableMessage(ctx context.Context, initialMessage string) func(ctx context.Context, newMsg string)

	// File operations
	AttachFile(ctx context.Context, title, fileName string, data []byte) error
}