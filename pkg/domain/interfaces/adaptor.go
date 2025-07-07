package interfaces

import (
	"context"
	"io"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/opaq"
	"github.com/slack-go/slack"
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
	PostMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error)
	UpdateMessageContext(ctx context.Context, channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error)
	AuthTest() (*slack.AuthTestResponse, error)
	GetTeamInfo() (*slack.TeamInfo, error)
	OpenView(triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error)
	UpdateView(view slack.ModalViewRequest, externalID, hash, viewID string) (*slack.ViewResponse, error)
	UploadFileV2Context(ctx context.Context, params slack.UploadFileV2Parameters) (*slack.FileSummary, error)
	GetUserInfo(userID string) (*slack.User, error)
	GetUsersInfo(users ...string) (*[]slack.User, error)
	GetConversationInfo(input *slack.GetConversationInfoInput) (*slack.Channel, error)
	GetUserGroups(options ...slack.GetUserGroupsOption) ([]slack.UserGroup, error)
	GetBotInfoContext(ctx context.Context, parameters slack.GetBotInfoParameters) (*slack.Bot, error)
}

type SlackThreadService interface {
	Reply(ctx context.Context, message string)
	NewStateFunc(ctx context.Context, message string) func(ctx context.Context, msg string)
}

type LLMClient interface {
	gollem.LLMClient
}

type LLMSession interface {
	gollem.Session
}
