package usecase

import (
	"context"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/service/command"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
)

// slackNotifierAdapter adapts slack.Service to SlackNotifier interface
type slackNotifierAdapter struct {
	service *slackService.Service
}

// NewSlackNotifier creates a SlackNotifier from slack.Service
func NewSlackNotifier(service *slackService.Service) SlackNotifier {
	return &slackNotifierAdapter{service: service}
}

func (s *slackNotifierAdapter) IsEnabled() bool {
	return true
}

func (s *slackNotifierAdapter) PostAlert(ctx context.Context, alert *alert.Alert) (SlackThreadService, error) {
	threadSvc, err := s.service.PostAlert(ctx, *alert)
	if err != nil {
		return nil, err
	}
	return &slackThreadServiceAdapter{threadSvc: threadSvc}, nil
}

func (s *slackNotifierAdapter) NewThread(thread slack.Thread) SlackThreadService {
	threadSvc := s.service.NewThread(thread)
	return &slackThreadServiceAdapter{threadSvc: threadSvc}
}

func (s *slackNotifierAdapter) PostTicket(ctx context.Context, ticket *ticket.Ticket, alerts alert.Alerts) (SlackThreadService, string, error) {
	threadSvc, messageID, err := s.service.PostTicket(ctx, *ticket, alerts)
	if err != nil {
		return nil, "", err
	}
	return &slackThreadServiceAdapter{threadSvc: threadSvc}, messageID, nil
}

func (s *slackNotifierAdapter) GetUserIcon(ctx context.Context, userID string) ([]byte, string, error) {
	return s.service.GetUserIcon(ctx, userID)
}

func (s *slackNotifierAdapter) GetUserProfile(ctx context.Context, userID string) (string, error) {
	return s.service.GetUserProfile(ctx, userID)
}

func (s *slackNotifierAdapter) IsBotUser(userID string) bool {
	return s.service.IsBotUser(userID)
}

func (s *slackNotifierAdapter) BotID() string {
	return s.service.BotID()
}

func (s *slackNotifierAdapter) DefaultChannelID() string {
	return s.service.DefaultChannelID()
}

func (s *slackNotifierAdapter) ToMsgURL(channelID, threadID string) string {
	return s.service.ToMsgURL(channelID, threadID)
}

func (s *slackNotifierAdapter) ShowBindToTicketModal(ctx context.Context, callbackID slack.CallbackID, tickets []*ticket.Ticket, triggerID string, metadata string) error {
	return s.service.ShowBindToTicketModal(ctx, callbackID, tickets, triggerID, metadata)
}

func (s *slackNotifierAdapter) ShowResolveTicketModal(ctx context.Context, ticket *ticket.Ticket, triggerID string) error {
	return s.service.ShowResolveTicketModal(ctx, ticket, triggerID)
}

func (s *slackNotifierAdapter) ShowSalvageModal(ctx context.Context, ticket *ticket.Ticket, unboundAlerts alert.Alerts, triggerID string) error {
	return s.service.ShowSalvageModal(ctx, ticket, unboundAlerts, triggerID)
}

func (s *slackNotifierAdapter) UpdateSalvageModal(ctx context.Context, ticket *ticket.Ticket, unboundAlerts alert.Alerts, viewID string, threshold float64, keyword string) error {
	return s.service.UpdateSalvageModal(ctx, ticket, unboundAlerts, viewID, threshold, keyword)
}

func (s *slackNotifierAdapter) ExecuteCommand(ctx context.Context, slackMsg *slack.Message, thread slack.Thread, commandStr string, repository interfaces.Repository, llmClient gollem.LLMClient) error {
	concreteThreadSvc := s.service.NewThread(thread)
	cmdSvc := command.New(repository, llmClient, concreteThreadSvc)
	return cmdSvc.Execute(ctx, slackMsg, commandStr)
}

// slackThreadServiceAdapter adapts slack.ThreadService to SlackThreadService interface
type slackThreadServiceAdapter struct {
	threadSvc *slackService.ThreadService
}

func (s *slackThreadServiceAdapter) ChannelID() string {
	return s.threadSvc.ChannelID()
}

func (s *slackThreadServiceAdapter) ThreadID() string {
	return s.threadSvc.ThreadID()
}

func (s *slackThreadServiceAdapter) Entity() *slack.Thread {
	return s.threadSvc.Entity()
}

func (s *slackThreadServiceAdapter) PostAlert(ctx context.Context, alert *alert.Alert) error {
	return s.threadSvc.PostAlert(ctx, *alert)
}

func (s *slackThreadServiceAdapter) PostComment(ctx context.Context, comment string) error {
	return s.threadSvc.PostComment(ctx, comment)
}

func (s *slackThreadServiceAdapter) PostCommentWithMessageID(ctx context.Context, comment string) (string, error) {
	return s.threadSvc.PostCommentWithMessageID(ctx, comment)
}

func (s *slackThreadServiceAdapter) PostTicket(ctx context.Context, ticket *ticket.Ticket, alerts alert.Alerts) (string, error) {
	return s.threadSvc.PostTicket(ctx, *ticket, alerts)
}

func (s *slackThreadServiceAdapter) PostLinkToTicket(ctx context.Context, ticketURL, ticketTitle string) error {
	return s.threadSvc.PostLinkToTicket(ctx, ticketURL, ticketTitle)
}

func (s *slackThreadServiceAdapter) PostFinding(ctx context.Context, finding *ticket.Finding) error {
	return s.threadSvc.PostFinding(ctx, *finding)
}

func (s *slackThreadServiceAdapter) UpdateAlert(ctx context.Context, alert alert.Alert) error {
	return s.threadSvc.UpdateAlert(ctx, alert)
}

func (s *slackThreadServiceAdapter) UpdateAlertList(ctx context.Context, list *alert.List, status string) error {
	return s.threadSvc.UpdateAlertList(ctx, list, status)
}

func (s *slackThreadServiceAdapter) PostAlerts(ctx context.Context, alerts alert.Alerts) error {
	return s.threadSvc.PostAlerts(ctx, alerts)
}

func (s *slackThreadServiceAdapter) PostAlertList(ctx context.Context, list *alert.List) (string, error) {
	return s.threadSvc.PostAlertList(ctx, list)
}

func (s *slackThreadServiceAdapter) PostAlertLists(ctx context.Context, clusters []*alert.List) error {
	return s.threadSvc.PostAlertLists(ctx, clusters)
}

func (s *slackThreadServiceAdapter) PostTicketList(ctx context.Context, tickets []*ticket.Ticket) error {
	return s.threadSvc.PostTicketList(ctx, tickets)
}

func (s *slackThreadServiceAdapter) Reply(ctx context.Context, message string) {
	s.threadSvc.Reply(ctx, message)
}

func (s *slackThreadServiceAdapter) NewStateFunc(ctx context.Context, message string) func(ctx context.Context, msg string) {
	return s.threadSvc.NewStateFunc(ctx, message)
}

func (s *slackThreadServiceAdapter) NewUpdatableMessage(ctx context.Context, initialMessage string) func(ctx context.Context, newMsg string) {
	return s.threadSvc.NewUpdatableMessage(ctx, initialMessage)
}

func (s *slackThreadServiceAdapter) AttachFile(ctx context.Context, title, fileName string, data []byte) error {
	return s.threadSvc.AttachFile(ctx, title, fileName, data)
}

// discardSlackNotifier is a no-op implementation of SlackNotifier
type discardSlackNotifier struct{}

// NewDiscardSlackNotifier creates a no-op SlackNotifier
func NewDiscardSlackNotifier() SlackNotifier {
	return &discardSlackNotifier{}
}

func (d *discardSlackNotifier) IsEnabled() bool {
	return false
}

func (d *discardSlackNotifier) PostAlert(ctx context.Context, alert *alert.Alert) (SlackThreadService, error) {
	// Return nil, nil to indicate no thread was created
	return nil, nil
}

func (d *discardSlackNotifier) NewThread(thread slack.Thread) SlackThreadService {
	return &discardSlackThreadService{}
}

func (d *discardSlackNotifier) PostTicket(ctx context.Context, ticket *ticket.Ticket, alerts alert.Alerts) (SlackThreadService, string, error) {
	// Return nil, "", nil to indicate no thread was created
	return nil, "", nil
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
	// No-op
	return nil
}

func (d *discardSlackThreadService) PostComment(ctx context.Context, comment string) error {
	// No-op
	return nil
}

func (d *discardSlackThreadService) PostCommentWithMessageID(ctx context.Context, comment string) (string, error) {
	// No-op
	return "", nil
}

func (d *discardSlackThreadService) PostTicket(ctx context.Context, ticket *ticket.Ticket, alerts alert.Alerts) (string, error) {
	// No-op
	return "", nil
}

func (d *discardSlackThreadService) PostLinkToTicket(ctx context.Context, ticketURL, ticketTitle string) error {
	// No-op
	return nil
}

func (d *discardSlackThreadService) PostFinding(ctx context.Context, finding *ticket.Finding) error {
	// No-op
	return nil
}

func (d *discardSlackThreadService) UpdateAlert(ctx context.Context, alert alert.Alert) error {
	// No-op
	return nil
}

func (d *discardSlackThreadService) UpdateAlertList(ctx context.Context, list *alert.List, status string) error {
	// No-op
	return nil
}

func (d *discardSlackThreadService) PostAlerts(ctx context.Context, alerts alert.Alerts) error {
	// No-op
	return nil
}

func (d *discardSlackThreadService) PostAlertList(ctx context.Context, list *alert.List) (string, error) {
	// No-op
	return "", nil
}

func (d *discardSlackThreadService) PostAlertLists(ctx context.Context, clusters []*alert.List) error {
	// No-op
	return nil
}

func (d *discardSlackThreadService) PostTicketList(ctx context.Context, tickets []*ticket.Ticket) error {
	// No-op
	return nil
}

func (d *discardSlackThreadService) Reply(ctx context.Context, message string) {
	// No-op
}

func (d *discardSlackThreadService) NewStateFunc(ctx context.Context, message string) func(ctx context.Context, msg string) {
	// Return a no-op function
	return func(ctx context.Context, msg string) {}
}

func (d *discardSlackThreadService) NewUpdatableMessage(ctx context.Context, initialMessage string) func(ctx context.Context, newMsg string) {
	// Return a no-op function
	return func(ctx context.Context, newMsg string) {}
}

func (d *discardSlackThreadService) AttachFile(ctx context.Context, title, fileName string, data []byte) error {
	// No-op
	return nil
}