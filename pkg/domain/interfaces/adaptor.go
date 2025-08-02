package interfaces

import (
	"context"
	"io"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	slackSDK "github.com/slack-go/slack"
)

type EmbeddingClient interface {
	Embeddings(ctx context.Context, texts []string, dimensionality int) ([][]float32, error)
}

type PolicyClient interface {
	Query(context.Context, string, any, any, ...opaq.QueryOption) error
	Sources() map[string]string
}

type StorageClient interface {
	PutObject(ctx context.Context, object string) io.WriteCloser
	GetObject(ctx context.Context, object string) (io.ReadCloser, error)
	Close(ctx context.Context)
}

type SlackClient interface {
	PostMessageContext(ctx context.Context, channelID string, options ...slackSDK.MsgOption) (string, string, error)
	UpdateMessageContext(ctx context.Context, channelID, timestamp string, options ...slackSDK.MsgOption) (string, string, string, error)
	AuthTest() (*slackSDK.AuthTestResponse, error)
	GetTeamInfo() (*slackSDK.TeamInfo, error)
	OpenView(triggerID string, view slackSDK.ModalViewRequest) (*slackSDK.ViewResponse, error)
	UpdateView(view slackSDK.ModalViewRequest, externalID, hash, viewID string) (*slackSDK.ViewResponse, error)
	UploadFileV2Context(ctx context.Context, params slackSDK.UploadFileV2Parameters) (*slackSDK.FileSummary, error)
	GetUserInfo(userID string) (*slackSDK.User, error)
	GetUsersInfo(users ...string) (*[]slackSDK.User, error)
	GetConversationInfo(input *slackSDK.GetConversationInfoInput) (*slackSDK.Channel, error)
	GetUserGroups(options ...slackSDK.GetUserGroupsOption) ([]slackSDK.UserGroup, error)
	GetBotInfoContext(ctx context.Context, parameters slackSDK.GetBotInfoParameters) (*slackSDK.Bot, error)
}

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
	NewTraceMessage(ctx context.Context, initialMessage string) func(ctx context.Context, traceMsg string)

	// File operations
	AttachFile(ctx context.Context, title, fileName string, data []byte) error
}


// CommandExecutor defines the interface for executing Slack commands
type CommandExecutor interface {
	Execute(ctx context.Context, slackMsg *slack.Message, thread slack.Thread, command string) error
}

type LLMClient interface {
	gollem.LLMClient
}

type LLMSession interface {
	gollem.Session
}
