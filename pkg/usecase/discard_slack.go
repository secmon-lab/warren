package usecase

import (
	"context"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/service/command"
)

// discardSlackNotifier is a no-op implementation of SlackNotifier
type discardSlackNotifier struct{}

// NewDiscardSlackNotifier creates a no-op SlackNotifier
func NewDiscardSlackNotifier() interfaces.SlackNotifier {
	return &discardSlackNotifier{}
}

func (d *discardSlackNotifier) IsEnabled() bool {
	return false
}

func (d *discardSlackNotifier) PostAlert(ctx context.Context, alert *alert.Alert) (interfaces.SlackThreadService, error) {
	return &discardSlackThreadService{}, nil
}

func (d *discardSlackNotifier) NewThread(thread slack.Thread) interfaces.SlackThreadService {
	return &discardSlackThreadService{}
}

func (d *discardSlackNotifier) PostTicket(ctx context.Context, ticket *ticket.Ticket, alerts alert.Alerts) (interfaces.SlackThreadService, string, error) {
	return &discardSlackThreadService{}, "", nil
}

func (d *discardSlackNotifier) GetUserIcon(ctx context.Context, userID string) ([]byte, string, error) {
	return nil, "", ErrSlackServiceNotConfigured
}

func (d *discardSlackNotifier) GetUserProfile(ctx context.Context, userID string) (string, error) {
	return "", ErrSlackServiceNotConfigured
}

func (d *discardSlackNotifier) IsBotUser(userID string) bool {
	return false
}

func (d *discardSlackNotifier) BotID() string {
	return ""
}

func (d *discardSlackNotifier) DefaultChannelID() string {
	return ""
}

func (d *discardSlackNotifier) ToMsgURL(channelID, threadID string) string {
	return ""
}

func (d *discardSlackNotifier) ShowBindToTicketModal(ctx context.Context, callbackID slack.CallbackID, tickets []*ticket.Ticket, triggerID string, metadata string) error {
	return ErrSlackServiceNotConfigured
}

func (d *discardSlackNotifier) ShowResolveTicketModal(ctx context.Context, ticket *ticket.Ticket, triggerID string) error {
	return ErrSlackServiceNotConfigured
}

func (d *discardSlackNotifier) ShowSalvageModal(ctx context.Context, ticket *ticket.Ticket, unboundAlerts alert.Alerts, triggerID string) error {
	return ErrSlackServiceNotConfigured
}

func (d *discardSlackNotifier) UpdateSalvageModal(ctx context.Context, ticket *ticket.Ticket, unboundAlerts alert.Alerts, viewID string, threshold float64, keyword string) error {
	return ErrSlackServiceNotConfigured
}

func (d *discardSlackNotifier) ExecuteCommand(ctx context.Context, slackMsg *slack.Message, thread slack.Thread, commandStr string, repository interfaces.Repository, llmClient gollem.LLMClient) error {
	return command.ErrUnknownCommand // Commands are not available when Slack is disabled
}

// discardSlackThreadService is a no-op implementation of SlackThreadService
type discardSlackThreadService struct{}

func (d *discardSlackThreadService) ChannelID() string {
	return ""
}

func (d *discardSlackThreadService) ThreadID() string {
	return ""
}

func (d *discardSlackThreadService) Entity() *slack.Thread {
	return nil
}

func (d *discardSlackThreadService) PostAlert(ctx context.Context, alert *alert.Alert) error {
	return nil // Silent no-op
}

func (d *discardSlackThreadService) PostComment(ctx context.Context, comment string) error {
	return nil // Silent no-op
}

func (d *discardSlackThreadService) PostCommentWithMessageID(ctx context.Context, comment string) (string, error) {
	return "", nil // Silent no-op
}

func (d *discardSlackThreadService) PostTicket(ctx context.Context, ticket *ticket.Ticket, alerts alert.Alerts) (string, error) {
	return "", nil // Silent no-op
}

func (d *discardSlackThreadService) PostLinkToTicket(ctx context.Context, ticketURL, ticketTitle string) error {
	return nil // Silent no-op
}

func (d *discardSlackThreadService) PostFinding(ctx context.Context, finding *ticket.Finding) error {
	return nil // Silent no-op
}

func (d *discardSlackThreadService) UpdateAlert(ctx context.Context, alert alert.Alert) error {
	return nil // Silent no-op
}

func (d *discardSlackThreadService) UpdateAlertList(ctx context.Context, list *alert.List, status string) error {
	return nil // Silent no-op
}

func (d *discardSlackThreadService) PostAlerts(ctx context.Context, alerts alert.Alerts) error {
	return nil // Silent no-op
}

func (d *discardSlackThreadService) PostAlertList(ctx context.Context, list *alert.List) (string, error) {
	return "", nil // Silent no-op
}

func (d *discardSlackThreadService) PostAlertLists(ctx context.Context, clusters []*alert.List) error {
	return nil // Silent no-op
}

func (d *discardSlackThreadService) PostTicketList(ctx context.Context, tickets []*ticket.Ticket) error {
	return nil // Silent no-op
}

func (d *discardSlackThreadService) Reply(ctx context.Context, message string) {
	// Silent no-op
}

func (d *discardSlackThreadService) NewStateFunc(ctx context.Context, message string) func(ctx context.Context, msg string) {
	return func(ctx context.Context, msg string) {
		// Silent no-op
	}
}

func (d *discardSlackThreadService) NewUpdatableMessage(ctx context.Context, initialMessage string) func(ctx context.Context, newMsg string) {
	return func(ctx context.Context, newMsg string) {
		// Silent no-op
	}
}

func (d *discardSlackThreadService) AttachFile(ctx context.Context, title, fileName string, data []byte) error {
	return nil // Silent no-op
}
